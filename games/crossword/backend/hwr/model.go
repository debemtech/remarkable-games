package hwr

import (
	_ "embed"
	"encoding/binary"
	"errors"
	"math"
	"sort"
)

//go:embed model.bin
var modelData []byte

type tensor struct {
	values []float32
}

// Model is the letter classifier: conv 3x3, pool, conv 3x3, pool, dense,
// dense 26. How wide the layers are comes from the file.
type Model struct {
	c1w, c1b, c2w, c2b, f1w, f1b, f2w, f2b []float32
	conv1, conv2, hidden                   int // channels and units
}

// Load reads the embedded model.
func Load() (*Model, error) { return parse(modelData) }

func parse(data []byte) (*Model, error) {
	if len(data) < 4 || string(data[:4]) != "HWR1" {
		return nil, errors.New("hwr: not a model file")
	}
	data = data[4:]
	var ts [8][]float32
	for i := range ts {
		if len(data) < 8 {
			return nil, errors.New("hwr: truncated model")
		}
		n := int(binary.LittleEndian.Uint32(data))
		scale := math.Float32frombits(binary.LittleEndian.Uint32(data[4:]))
		data = data[8:]
		if len(data) < n {
			return nil, errors.New("hwr: truncated model")
		}
		t := make([]float32, n)
		for k := range t {
			t[k] = float32(int8(data[k])) * scale
		}
		ts[i], data = t, data[n:]
	}
	m := &Model{c1w: ts[0], c1b: ts[1], c2w: ts[2], c2b: ts[3], f1w: ts[4], f1b: ts[5], f2w: ts[6], f2b: ts[7]}
	m.conv1, m.conv2, m.hidden = len(m.c1b), len(m.c2b), len(m.f1b)
	if m.conv1 == 0 || len(m.c1w) != m.conv1*9 || len(m.c2w) != m.conv2*m.conv1*9 ||
		len(m.f1w) != m.hidden*m.conv2*7*7 || len(m.f2b) != 26 || len(m.f2w) != 26*m.hidden {
		return nil, errors.New("hwr: unexpected model shape")
	}
	return m, nil
}

// conv3x3 with padding 1, followed by ReLU. in is [cin][h][w].
//
// Every output plane starts at the bias and gets each input plane added in,
// shifted for each of the 9 kernel taps, over the part where the shifted plane
// lies inside the image (so there are no edge checks per multiply). Each
// output pixel adds up the same terms in the same order as a straight sum
// over channels and taps would.
func conv3x3(in []float32, cin, h, w int, weight, bias []float32, cout int) []float32 {
	plane := h * w
	out := make([]float32, cout*plane)
	for o := range cout {
		acc := out[o*plane : (o+1)*plane]
		for i := range acc {
			acc[i] = bias[o]
		}
		for c := range cin {
			src := in[c*plane : (c+1)*plane]
			kernel := weight[(o*cin+c)*9 : (o*cin+c+1)*9]
			for ky := -1; ky <= 1; ky++ {
				y0, y1 := max(0, -ky), min(h, h-ky)
				for kx := -1; kx <= 1; kx++ {
					k := kernel[(ky+1)*3+kx+1]
					x0, x1 := max(0, -kx), min(w, w-kx)
					for y := y0; y < y1; y++ {
						dst := acc[y*w+x0 : y*w+x1]
						from := src[(y+ky)*w+x0+kx:]
						from = from[:len(dst)]
						for i, v := range from {
							dst[i] += k * v
						}
					}
				}
			}
		}
		for i, v := range acc {
			acc[i] = max(v, 0)
		}
	}
	return out
}

func pool2(in []float32, c, h, w int) []float32 {
	oh, ow := h/2, w/2
	out := make([]float32, c*oh*ow)
	for k := range c {
		for y := range oh {
			for x := range ow {
				i := (k*h+2*y)*w + 2*x
				out[(k*oh+y)*ow+x] = max(in[i], in[i+1], in[i+w], in[i+w+1])
			}
		}
	}
	return out
}

// dense is a fully connected layer. Inputs that are zero (most of them, after
// ReLU) are skipped: they would only add zeros.
func dense(in, weight, bias []float32, relu bool) []float32 {
	idx := make([]int, 0, len(in))
	for i, v := range in {
		if v != 0 {
			idx = append(idx, i)
		}
	}
	out := make([]float32, len(bias))
	for o := range out {
		sum := bias[o]
		row := weight[o*len(in) : (o+1)*len(in)]
		for _, i := range idx {
			sum += in[i] * row[i]
		}
		if relu {
			sum = max(sum, 0)
		}
		out[o] = sum
	}
	return out
}

// Logits returns the raw scores for A-Z.
func (m *Model) Logits(img [Size * Size]float32) []float32 {
	x := conv3x3(img[:], 1, 28, 28, m.c1w, m.c1b, m.conv1)
	x = pool2(x, m.conv1, 28, 28)
	x = conv3x3(x, m.conv1, 14, 14, m.c2w, m.c2b, m.conv2)
	x = pool2(x, m.conv2, 14, 14)
	x = dense(x, m.f1w, m.f1b, true)
	return dense(x, m.f2w, m.f2b, false)
}

// Guess is a letter with its probability.
type Guess struct {
	Letter      string  `json:"letter"`
	Probability float32 `json:"p"`
}

// Recognize returns the most likely letters, best first.
func (m *Model) Recognize(strokes []Stroke, n int) []Guess {
	img, ok := Image(strokes)
	if !ok {
		return nil
	}
	logits := m.Logits(img)
	peak := logits[0]
	for _, v := range logits {
		peak = max(peak, v)
	}
	var total float32
	probs := make([]float32, len(logits))
	for i, v := range logits {
		probs[i] = float32(math.Exp(float64(v - peak)))
		total += probs[i]
	}
	guesses := make([]Guess, len(probs))
	for i := range probs {
		guesses[i] = Guess{string(rune('A' + i)), probs[i] / total}
	}
	sort.Slice(guesses, func(i, j int) bool { return guesses[i].Probability > guesses[j].Probability })
	return guesses[:min(n, len(guesses))]
}
