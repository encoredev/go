//go:build !encore
// +build !encore

package http

import (
	"context"
)

// encoreBeginRoundTrip is called by net/http when a RoundTrip begins.
func encoreBeginRoundTrip(req *Request) (context.Context, error) { return req.Context(), nil }

// encoreFinishRoundTrip is called by net/http when a RoundTrip completes.
func encoreFinishRoundTrip(req *Request, resp *Response, err error) {}
