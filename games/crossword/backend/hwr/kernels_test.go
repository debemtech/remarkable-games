package hwr

import (
	"math"
	"math/rand/v2"
	"testing"
)

// The straight versions of conv3x3 and dense, to check the fast ones against.

func conv3x3Plain(in []float32, cin, h, w int, weight, bias []float32, cout int) []float32 {
	out := make([]float32, cout*h*w)
	for o := range cout {
		for y := range h {
			for x := range w {
				sum := bias[o]
				for c := range cin {
					wo := (o*cin + c) * 9
					for ky := -1; ky <= 1; ky++ {
						yy := y + ky
						if yy < 0 || yy >= h {
							continue
						}
						for kx := -1; kx <= 1; kx++ {
							xx := x + kx
							if xx < 0 || xx >= w {
								continue
							}
							sum += in[(c*h+yy)*w+xx] * weight[wo+(ky+1)*3+kx+1]
						}
					}
				}
				out[(o*h+y)*w+x] = max(sum, 0)
			}
		}
	}
	return out
}

func densePlain(in, weight, bias []float32, relu bool) []float32 {
	out := make([]float32, len(bias))
	for o := range out {
		sum := bias[o]
		row := weight[o*len(in) : (o+1)*len(in)]
		for i, v := range in {
			sum += v * row[i]
		}
		if relu {
			sum = max(sum, 0)
		}
		out[o] = sum
	}
	return out
}

func random(rng *rand.Rand, n int, zeros float64) []float32 {
	s := make([]float32, n)
	for i := range s {
		if rng.Float64() >= zeros {
			s[i] = float32(rng.NormFloat64())
		}
	}
	return s
}

func same(t *testing.T, what string, got, want []float32) {
	t.Helper()
	for i := range want {
		if math.Float32bits(got[i]) != math.Float32bits(want[i]) {
			t.Fatalf("%s: value %d is %v, the plain version gives %v", what, i, got[i], want[i])
		}
	}
}

// The fast kernels give exactly the same numbers, bit for bit.
func TestKernelsExact(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for _, shape := range [][4]int{{1, 28, 28, 8}, {6, 14, 14, 5}, {3, 7, 9, 4}, {2, 1, 1, 3}} {
		cin, h, w, cout := shape[0], shape[1], shape[2], shape[3]
		in, weight, bias := random(rng, cin*h*w, 0.3), random(rng, cout*cin*9, 0), random(rng, cout, 0)
		same(t, "conv3x3", conv3x3(in, cin, h, w, weight, bias, cout), conv3x3Plain(in, cin, h, w, weight, bias, cout))
	}
	for _, zeros := range []float64{0, 0.7, 1} {
		in, weight, bias := random(rng, 300, zeros), random(rng, 300*17, 0), random(rng, 17, 0)
		same(t, "dense", dense(in, weight, bias, true), densePlain(in, weight, bias, true))
		same(t, "dense", dense(in, weight, bias, false), densePlain(in, weight, bias, false))
	}
	// And the whole model, on real letters
	m, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for k := range 5 {
		img := random(rng, Size*Size, 0.6)
		for i := range img {
			img[i] = float32(math.Abs(float64(img[i]))) / 3
		}
		var x [Size * Size]float32
		copy(x[:], img)
		plain := conv3x3Plain(x[:], 1, 28, 28, m.c1w, m.c1b, m.conv1)
		plain = pool2(plain, m.conv1, 28, 28)
		plain = conv3x3Plain(plain, m.conv1, 14, 14, m.c2w, m.c2b, m.conv2)
		plain = pool2(plain, m.conv2, 14, 14)
		plain = densePlain(plain, m.f1w, m.f1b, true)
		plain = densePlain(plain, m.f2w, m.f2b, false)
		same(t, "model "+string(rune('0'+k)), m.Logits(x), plain)
	}
}
