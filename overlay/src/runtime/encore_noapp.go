//go:build !encore
// +build !encore

package runtime

import "unsafe"

// startEncoreG starts a new g, copying the src encore data to a new encoreG value.
// It is defined here as a no-op for Encore binaries that were built
// without linking in the Encore runtime to avoid strange build errors.
func startEncoreG(src unsafe.Pointer) unsafe.Pointer {
	return nil
}

// exitEncoreG marks a goroutine as having exited.
// It is defined here as a no-op for Encore binaries that were built
// without linking in the Encore runtime to avoid strange build errors.
func exitEncoreG(src unsafe.Pointer) {
}
