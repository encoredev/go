//go:build encore
// +build encore

package testing

import _ "unsafe"

// encoreTestStart is called when a test starts running. This allows Encore's testing framework to
// isolate behaviour between different tests on global state. It is linked to the Encore runtime via go:linkname.
func encoreTestStart(t *T)

// encoreTestEnd is called when a test ends. This allows Encore's testing framework to clear down any state from the test
// and to perform any assertions on that state that it needs to. It is linked to the Encore runtime via go:linkname.
func encoreTestEnd(t *T)

// encoreTestPaused is called when a test is paused. This allows Encore's testing framework to
// isolate behaviour between different tests on global state. It is linked to the Encore runtime via go:linkname.
func encoreTestPaused(t *T)

// encoreTestResumed is called when a test is resumed after being paused. This allows Encore's testing framework to clear down any state from the test
// and to perform any assertions on that state that it needs to. It is linked to the Encore runtime via go:linkname.
func encoreTestResumed(t *T)
