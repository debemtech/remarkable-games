package hwr

// axpy adds k times src to dst: dst[i] += k * src[i]. src must be at least as
// long as dst. On arm64 it's NEON assembly (axpy_arm64.s), 4 lanes at a time.
func axpy(dst, src []float32, k float32) {
	if len(src) < len(dst) {
		panic("hwr: axpy: src shorter than dst")
	}
	axpyKernel(dst, src, k)
}
