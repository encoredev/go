//go:build encore
// +build encore

package runtime

// encoreSendTrace is called by Encore's go runtime to send a trace.
// It must be defined by the Encore runtime and linked using
// go:linkname.
func encoreSendTrace(log []byte)
