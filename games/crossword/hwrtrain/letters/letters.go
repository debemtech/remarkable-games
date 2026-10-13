// Package letters has capital letters A-Z as pen strokes, for making
// synthetic training data and for testing the recognizer with pen input.
package letters

import (
	"math"
	"math/rand/v2"

	"appload-games/games/crossword/backend/hwr"
)

type pt = hwr.Point

func arc(cx, cy, rx, ry, from, to float64) []pt {
	var out []pt
	steps := int(math.Abs(to-from)/10) + 2
	for i := 0; i <= steps; i++ {
		a := (from + (to-from)*float64(i)/float64(steps)) * math.Pi / 180
		out = append(out, pt{X: cx + rx*math.Cos(a), Y: cy + ry*math.Sin(a)})
	}
	return out
}

func path(parts ...[]pt) hwr.Stroke {
	var s hwr.Stroke
	for _, p := range parts {
		s = append(s, p...)
	}
	return s
}

func p(coords ...float64) []pt {
	var out []pt
	for i := 0; i+1 < len(coords); i += 2 {
		out = append(out, pt{X: coords[i], Y: coords[i+1]})
	}
	return out
}

// Strokes of each letter in a 10x10 box, y pointing down. Some letters have
// several common ways of writing them.
var Shapes = map[byte][][]hwr.Stroke{
	'A': {{path(p(0, 10, 5, 0, 10, 10)), path(p(2.2, 6, 7.8, 6))}},
	'B': {{path(p(1, 10, 1, 0, 6, 0), arc(6, 2.5, 2.7, 2.5, -90, 90), p(1, 5)), path(p(6, 5), arc(6, 7.5, 3, 2.5, -90, 90), p(1, 10))},
		{path(p(1, 0, 1, 10)), path(p(1, 0, 6, 0), arc(6, 2.5, 2.7, 2.5, -90, 90), p(1, 5, 6, 5), arc(6, 7.5, 3, 2.5, -90, 90), p(1, 10))}},
	'C': {{path(arc(5.5, 5, 4.5, 5, -40, -320))}},
	'D': {{path(p(1, 0, 1, 10)), path(p(1, 0, 4, 0), arc(4, 5, 5, 5, -90, 90), p(1, 10))},
		{path(p(1, 10, 1, 0, 4, 0), arc(4, 5, 5, 5, -90, 90), p(1, 10))}},
	'E': {{path(p(9, 0, 1, 0, 1, 10, 9, 10)), path(p(1, 5, 7, 5))},
		{path(p(1, 0, 1, 10)), path(p(1, 0, 9, 0)), path(p(1, 5, 7, 5)), path(p(1, 10, 9, 10))}},
	'F': {{path(p(9, 0, 1, 0, 1, 10)), path(p(1, 5, 7, 5))}},
	'G': {{path(arc(5.5, 5, 4.5, 5, -40, -330), p(9.5, 5.5, 6, 5.5))},
		{path(arc(5.5, 5, 4.5, 5, -40, -340), p(9.8, 8, 9.8, 5.5, 6.5, 5.5))}},
	'H': {{path(p(1, 0, 1, 10)), path(p(9, 0, 9, 10)), path(p(1, 5, 9, 5))}},
	'I': {{path(p(5, 0, 5, 10)), path(p(3, 0, 7, 0)), path(p(3, 10, 7, 10))}, {path(p(5, 0, 5, 10))}},
	'J': {{path(p(8, 0, 8, 7), arc(5, 7, 3, 3, 0, 180))}, {path(p(4, 0, 10, 0)), path(p(8, 0, 8, 7), arc(5, 7, 3, 3, 0, 160))}},
	'K': {{path(p(1, 0, 1, 10)), path(p(9, 0, 1, 6)), path(p(3.5, 4.5, 9, 10))},
		{path(p(1, 0, 1, 10)), path(p(9, 0, 1.5, 5, 9, 10))}},
	'L': {{path(p(1, 0, 1, 10, 8, 10))}},
	'M': {{path(p(1, 10, 1, 0, 5, 6, 9, 0, 9, 10))}, {path(p(0, 10, 1.5, 0, 5, 8, 8.5, 0, 10, 10))}},
	'N': {{path(p(1, 10, 1, 0, 9, 10, 9, 0))}},
	'O': {{path(arc(5, 5, 4.5, 5, -90, 270))}},
	'P': {{path(p(1, 10, 1, 0, 6, 0), arc(6, 2.7, 3, 2.7, -90, 90), p(1, 5.4))},
		{path(p(1, 0, 1, 10)), path(p(1, 0, 6, 0), arc(6, 2.7, 3, 2.7, -90, 90), p(1, 5.4))}},
	'Q': {{path(arc(5, 5, 4.5, 5, -90, 270)), path(p(6, 7, 10, 10.5))}},
	'R': {{path(p(1, 10, 1, 0, 6, 0), arc(6, 2.7, 3, 2.7, -90, 90), p(1, 5.4)), path(p(5, 5.4, 9, 10))},
		{path(p(1, 10, 1, 0, 6, 0), arc(6, 2.7, 3, 2.7, -90, 90), p(1, 5.4, 9, 10))}},
	'S': {{path(p(8.5, 1.5, 7, 0.2, 4.5, 0, 2, 0.8, 1.5, 2.5, 2.5, 4, 5, 5, 7.5, 6, 8.7, 7.7, 8, 9.3, 5.5, 10, 3, 9.8, 1.3, 8.5))}},
	'T': {{path(p(0, 0, 10, 0)), path(p(5, 0, 5, 10))}},
	'U': {{path(p(1, 0, 1, 6.5), arc(5, 6.5, 4, 3.5, 180, 0), p(9, 0))}, {path(p(1, 0, 1, 6.5), arc(5, 6.5, 4, 3.5, 180, 0), p(9, 0, 9, 10))}},
	'V': {{path(p(0, 0, 5, 10, 10, 0))}},
	'W': {{path(p(0, 0, 2.5, 10, 5, 3, 7.5, 10, 10, 0))}},
	'X': {{path(p(0, 0, 10, 10)), path(p(10, 0, 0, 10))}},
	'Y': {{path(p(0, 0, 5, 5, 10, 0)), path(p(5, 5, 5, 10))}, {path(p(0, 0, 5, 5)), path(p(10, 0, 3, 10))}},
	'Z': {{path(p(0, 0, 10, 0, 0, 10, 10, 10))}, {path(p(0, 0, 10, 0, 0, 10, 10, 10)), path(p(2.5, 5, 7.5, 5))}},
}

// Variant draws a letter the way a person might: shaken, slanted, squashed.
func Variant(rng *rand.Rand, letter byte) []hwr.Stroke {
	forms := Shapes[letter]
	form := forms[rng.IntN(len(forms))]

	angle := (rng.Float64()*2 - 1) * 12 * math.Pi / 180
	shear := (rng.Float64()*2 - 1) * 0.3
	sx := 0.65 + rng.Float64()*0.55
	sy := 0.75 + rng.Float64()*0.45
	cos, sin := math.Cos(angle), math.Sin(angle)
	// A slow wobble along the stroke, plus some jitter per point
	wobble := rng.Float64() * 0.4
	phase := rng.Float64() * 2 * math.Pi

	var out []hwr.Stroke
	for _, s := range form {
		var ns hwr.Stroke
		for i := 1; i < len(s); i++ {
			a, b := s[i-1], s[i]
			n := int(math.Hypot(b.X-a.X, b.Y-a.Y)/0.5) + 1
			for k := 0; k < n; k++ {
				t := float64(k) / float64(n)
				ns = append(ns, pt{X: a.X + (b.X-a.X)*t, Y: a.Y + (b.Y-a.Y)*t})
			}
		}
		ns = append(ns, s[len(s)-1])
		for i := range ns {
			q := ns[i]
			q.X += wobble*math.Sin(float64(i)*0.15+phase) + rng.NormFloat64()*0.08
			q.Y += wobble*math.Cos(float64(i)*0.13+phase) + rng.NormFloat64()*0.08
			x, y := (q.X-5)*sx+shear*(q.Y-5), (q.Y-5)*sy
			ns[i] = pt{X: x*cos - y*sin, Y: x*sin + y*cos}
		}
		// Sometimes a stroke stops a little short
		if len(ns) > 10 && rng.IntN(6) == 0 {
			ns = ns[:len(ns)-rng.IntN(len(ns)/12+1)]
		}
		out = append(out, ns)
	}
	return out
}
