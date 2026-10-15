package hwr

// Implemented in axpy_arm64.s.
//
//go:noescape
func axpyKernel(dst, src []float32, k float32)
