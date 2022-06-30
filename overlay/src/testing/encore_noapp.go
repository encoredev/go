//go:build !encore
// +build !encore

package testing

// encoreStartTest is called when a test starts running. This allows Encore's testing framework to
// isolate behavior between different tests on global state.
//
// This implementation is simply a no-op as it's used when the tests are not being run against an Encore application
func encoreStartTest(t *T) {}

// encoreEndTest is called when a test ends. This allows Encore's testing framework to clear down any state from the test
// and to perform any assertions on that state that it needs to. It is linked to the Encore runtime via go:linkname.
//
// This implementation is simply a no-op as it's used when the tests are not being run against an Encore application
func encoreEndTest(t *T) {}

// encorePauseTest is called when a test is paused. This allows Encore's testing framework to
// isolate behavior between different tests on global state. It is linked to the Encore runtime via go:linkname.
func encorePauseTest(t *T) {}

// encoreResumeTest is called when a test is resumed after being paused. This allows Encore's testing framework to clear down any state from the test
// and to perform any assertions on that state that it needs to. It is linked to the Encore runtime via go:linkname.
func encoreResumeTest(t *T) {}
