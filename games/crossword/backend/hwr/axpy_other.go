//go:build !arm64

package hwr

func axpyKernel(dst, src []float32, k float32) {
	src = src[:len(dst)]
	for i, v := range src {
		dst[i] += k * v
	}
}
