package runtime

// This file implements Encore's request tracking and runtime tracing.
//
// Request Tracking
//
// Encore tracks the current request being processed, in order for code
// to seamlessly access the right infrastructure resources and do request logging.
//
// The request tracking spans goroutines: when a goroutine starts it copies the
// current request from its parent goroutine. The Encore runtime can then look up
// the current request at will.
//
// During processing, a request can call other APIs which yield child requests.
// This group of requests is collectively called an operation. Operations
// are ref-counted (number of active requests + 1). The +1 allows for multiple
// root requests to be a part of a single operation; the creator of the operation
// marks the operation as completed which decrements the refcount by one.
// Note that a finished operation may continue being processed while child requests
// are still running.
//
// Tracing
//
// Tracing extends the request tracking machinery to implement performant tracing.
// Traces are associated with an operation at creation time. The runtime maintains
// an in-memory buffer which is appended to to log events. When the operation
// refcount reaches zero, the trace is marked completed and sent.

import (
	"sync/atomic"
	"unsafe"
)

// encoreG tracks per-goroutine Encore-specific data.
type encoreG struct {
	// op is the current operation the goroutine is a part of.
	op *encoreOp

	// req holds Encore-specific request data, or nil if the g
	// is not processing a request.
	req *encoreReq

	// goid is the per-op goroutine id.
	goid uint32
}

// encoreOp represents an Encore operation.
type encoreOp struct {
	// start is the start time of the operation
	start int64 // start time of trace from nanotime()

	// trace is the trace log; it is nil if the op is not traced.
	trace *encoreTraceLog

	// refs is the op refcount. It is 1 + number of requests
	// that reference this op (see doc comment above).
	// It is accessed atomically.
	refs int32

	// goidCtr is a per-operation goroutine counter, for telling
	// apart goroutines participating in the operation.
	goidCtr uint32
}

// encoreSpanID represents a span id in Encore's tracing framework.
type encoreSpanID [8]byte

// encoreReq represents an Encore API request.
type encoreReq struct {
	// spanID is the request span id.
	spanID encoreSpanID
	// data is request-specific data defined in the Encore runtime.
	data unsafe.Pointer
}

// encoreBeginOp begins a new Encore operation.
// The trace parameter determines if the op is traced.
//
// It tags the current goroutine with the op.
// It panics if the goroutine is already part of an op.
func encoreBeginOp(trace bool) *encoreOp {
	op := encoreNewOp(trace)
	g := getg().m.curg
	encoreTagG(g, op, nil)
	return op
}

// encoreNewOp creates a new encoreOp.
func encoreNewOp(trace bool) *encoreOp {
	op := &encoreOp{
		start: nanotime(),
		refs:  1,
	}
	if trace {
		op.trace = new(encoreTraceLog)
	}
	return op
}

// encoreTagG tags the g as participating in op, and with req
// as its request data.
// It does not increment the ref count, which means req
// must already be an active request.
// g must not already be part of an op.
func encoreTagG(g *g, op *encoreOp, req *encoreReq) (goid uint32) {
	if g.encore != nil {
		panic("encore.tagG: goroutine already part of another operation")
	}
	goid = atomic.AddUint32(&op.goidCtr, 1)
	g.encore = &encoreG{
		op:   op,
		req:  req,
		goid: goid,
	}
	return goid
}

// encoreGetG gets the encore data for the current g, or nil.
func encoreGetG() *encoreG {
	return getg().m.curg.encore
}

// encoreFinishOp marks an operation as finished
// and unsets the operation tag on the g.
// It must be part of an operation.
func encoreFinishOp() {
	g := getg().m.curg
	e := g.encore
	if e == nil {
		panic("encore.completeReq: goroutine not in an operation")
	}
	e.op.decRef()
	g.encore = nil
}

// incRef increases the op's refcount by one.
func (op *encoreOp) incRef() int32 {
	return atomic.AddInt32(&op.refs, 1)
}

// decRef decreases the op's refcount by one.
// If it reaches zero and the op is traced, it sends off the trace.
func (op *encoreOp) decRef() int32 {
	n := atomic.AddInt32(&op.refs, -1)
	if n == 0 && op.trace != nil {
		op.trace.send()
	}
	return n
}

// encoreTraceEvent adds the event to the trace.
// The g must already be part of an operation.
func encoreTraceEvent(event byte, data []byte) {
	e := getg().m.curg.encore
	if e == nil {
		println("encore.traceEvent: goroutine not in an operation, skipping")
		return
	}
	e.op.trace.log(event, data)
}

// encoreBeginReq sets the request data for the current g,
// and increases the ref count on the operation.
// If the g is not part of an op, it creates a new op
// that is bound to the request lifetime.
func encoreBeginReq(spanID encoreSpanID, data unsafe.Pointer, trace bool) {
	g := getg().m.curg
	e := g.encore
	req := &encoreReq{spanID: spanID, data: data}
	if e == nil {
		op := encoreNewOp(trace)
		encoreTagG(g, op, req)
		// Don't increment the op refcount since it starts at one,
		// and this is not a standalone op.
	} else {
		if e.req != nil {
			panic("encore.beginReq: request already running")
		}
		e.op.incRef()
		e.req = req
	}
}

// encoreCompleteReq completes the request and decreases the
// ref count on the operation.
// The g must be processing a request.
func encoreCompleteReq() {
	e := getg().m.curg.encore
	if e == nil {
		panic("encore.completeReq: goroutine not in an operation")
	} else if e.req == nil {
		panic("encore.completeReq: no current request")
	}
	e.op.decRef()
	e.req = nil
}

// encoreCleareq clears request data from the running g
// without decrementing the ref count.
// The g must be processing a request.
func encoreClearReq() {
	e := getg().m.curg.encore
	if e == nil {
		panic("encore.replaceReq: goroutine not in an operation")
	} else if e.req == nil {
		panic("encore.replaceReq: no current request")
	}
	spanID := e.req.spanID
	e.req = nil
	if e.op.trace != nil {
		e.op.trace.log(0x05, []byte{
			spanID[0],
			spanID[1],
			spanID[2],
			spanID[3],
			spanID[4],
			spanID[5],
			spanID[6],
			spanID[7],
			byte(e.goid),
			byte(e.goid >> 8),
			byte(e.goid >> 16),
			byte(e.goid >> 24),
		})
	}
}

type encoreTraceLog struct {
	mu   mutex
	data []byte
}

func (tl *encoreTraceLog) send() {
	lock(&tl.mu)
	data := tl.data
	tl.data = tl.data[len(tl.data):]
	unlock(&tl.mu)
	encoreSendTrace(data)
}

// log logs a new event in the trace.
// If tr is nil, it does nothing.
func (tl *encoreTraceLog) log(event byte, data []byte) {
	if tl == nil {
		return
	}
	ln := len(data)
	if ln > (1<<32 - 1) {
		println("encore.traceEvent: event too large, dropping")
		return
	}

	lock(&tl.mu)
	defer unlock(&tl.mu)

	// Do this in the critical section to ensure we don't get
	// out-of-order timestamps.
	t := nanotime()
	var b [13]byte
	b[0] = event
	b[1] = byte(t)
	b[2] = byte(t >> 8)
	b[3] = byte(t >> 16)
	b[4] = byte(t >> 24)
	b[5] = byte(t >> 32)
	b[6] = byte(t >> 40)
	b[7] = byte(t >> 48)
	b[8] = byte(t >> 56)
	b[9] = byte(ln)
	b[10] = byte(ln >> 8)
	b[11] = byte(ln >> 16)
	b[12] = byte(ln >> 24)
	tl.data = append(tl.data, append(b[:], data...)...)
}

// encoreCallers is like runtime.Callers but also returns the offset
// of the text segment to make the PCs ASLR-independent.
func encoreCallers(skip int, pc []uintptr) (n int, off uintptr) {
	n = Callers(skip+1, pc)
	return n, firstmoduledata.text
}
