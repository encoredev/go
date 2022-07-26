//go:build encore
// +build encore

package runtime

import "unsafe"

// startEncoreG starts a new g, copying the src encore data to a new encoreG value.
// It must be defined by the Encore runtime and linked using
// go:linkname.
func startEncoreG(src unsafe.Pointer) unsafe.Pointer

// exitEncoreG marks a goroutine as having exited.
// It must be defined by the Encore runtime and linked using
// go:linkname.
func exitEncoreG(src unsafe.Pointer)
