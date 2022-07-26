package runtime

import (
	"unsafe"
)

// setEncoreG sets the encoreG value on the running g to the given value.
func setEncoreG(val unsafe.Pointer) {
	g := getg().m.curg
	g.encore = val
}

// getEncoreG gets the encoreG value from the running g.
func getEncoreG() unsafe.Pointer {
	return getg().m.curg.encore
}

// encoreCallers is like runtime.Callers but also returns the offset
// of the text segment to make the PCs ASLR-independent.
func encoreCallers(skip int, pc []uintptr) (n int, off uintptr) {
	n = Callers(skip+1, pc)
	return n, firstmoduledata.text
}
