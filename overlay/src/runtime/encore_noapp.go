//go:build !encore
// +build !encore

package runtime

// encoreSendTrace is called by Encore's go runtime to send a trace.
// It is defined here as a no-op for Encore binaries that were built
// without linking in the Encore runtime to avoid strange build errors.
func encoreSendTrace(log []byte) {}
