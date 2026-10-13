// Package hwr recognizes a single handwritten capital letter.
//
// Pen strokes are drawn into a 28x28 image the way the EMNIST dataset was
// made (centered in a square, 2 pixels of padding, blurred), then classified
// by a small convolutional network trained on EMNIST letters plus synthetic
// pen strokes (see games/crossword/hwrtrain).
package hwr

import "math"

// Point is a position in any unit; only the shape of the strokes matters.
type Point struct{ X, Y float64 }

// Stroke is what the pen drew between touching down and lifting off.
type Stroke []Point

const (
	Size    = 28 // network input
	canvas  = 112
	padding = 2 // EMNIST's border, in input pixels
)

// Image converts strokes into the network input: 28x28, ink 1, paper 0.
// It returns false if there is nothing to recognize.
func Image(strokes []Stroke) ([Size * Size]float32, bool) {
	var out [Size * Size]float32
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	n := 0
	for _, s := range strokes {
		for _, p := range s {
			minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
			minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
			n++
		}
	}
	if n == 0 {
		return out, false
	}
	// Fit the letter into a square, centered, keeping its proportions.
	side := math.Max(maxX-minX, maxY-minY)
	if side == 0 {
		side = 1 // a dot
	}
	inner := float64(canvas) * float64(Size-2*padding) / Size
	scale := inner / side
	offX := (canvas-(maxX-minX)*scale)/2 - minX*scale
	offY := (canvas-(maxY-minY)*scale)/2 - minY*scale

	// Pen width similar to EMNIST's (about 2.5 input pixels)
	img := make([]float32, canvas*canvas)
	radius := float64(canvas) / Size * 1.3
	for _, s := range strokes {
		if len(s) == 1 {
			disc(img, s[0].X*scale+offX, s[0].Y*scale+offY, radius)
		}
		for i := 1; i < len(s); i++ {
			line(img,
				s[i-1].X*scale+offX, s[i-1].Y*scale+offY,
				s[i].X*scale+offX, s[i].Y*scale+offY, radius)
		}
	}

	// Downsample by averaging 4x4 blocks, then blur a little.
	f := canvas / Size
	var small [Size * Size]float32
	for y := range Size {
		for x := range Size {
			var sum float32
			for dy := range f {
				for dx := range f {
					sum += img[(y*f+dy)*canvas+x*f+dx]
				}
			}
			small[y*Size+x] = sum / float32(f*f)
		}
	}
	out = blur(small)
	var peak float32
	for _, v := range out {
		peak = max(peak, v)
	}
	if peak > 0 {
		for i := range out {
			out[i] /= peak
		}
	}
	return out, true
}

func disc(img []float32, cx, cy, r float64) {
	for y := int(cy - r - 1); y <= int(cy+r+1); y++ {
		for x := int(cx - r - 1); x <= int(cx+r+1); x++ {
			if x < 0 || y < 0 || x >= canvas || y >= canvas {
				continue
			}
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			if v := float32(math.Min(1, math.Max(0, r-d+0.5))); v > img[y*canvas+x] {
				img[y*canvas+x] = v
			}
		}
	}
}

func line(img []float32, x0, y0, x1, y1, r float64) {
	steps := int(math.Hypot(x1-x0, y1-y0)/(r/2)) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		disc(img, x0+(x1-x0)*t, y0+(y1-y0)*t, r)
	}
}

// blur applies a 3x3 Gaussian (sigma about 0.85).
func blur(in [Size * Size]float32) [Size * Size]float32 {
	k := [3]float32{0.25, 0.5, 0.25}
	var tmp, out [Size * Size]float32
	for y := range Size {
		for x := range Size {
			var s float32
			for i := -1; i <= 1; i++ {
				if xx := x + i; xx >= 0 && xx < Size {
					s += in[y*Size+xx] * k[i+1]
				}
			}
			tmp[y*Size+x] = s
		}
	}
	for y := range Size {
		for x := range Size {
			var s float32
			for i := -1; i <= 1; i++ {
				if yy := y + i; yy >= 0 && yy < Size {
					s += tmp[yy*Size+x] * k[i+1]
				}
			}
			out[y*Size+x] = s
		}
	}
	return out
}
