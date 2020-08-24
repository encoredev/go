// +build encore

package http

import (
	"context"
	_ "unsafe"
)

// encoreBeginRoundTrip is called by net/http when a RoundTrip begins.
func encoreBeginRoundTrip(req *Request) (context.Context, error)

// encoreFinishRoundTrip is called by net/http when a RoundTrip completes.
func encoreFinishRoundTrip(req *Request, resp *Response, err error)
