package hwr_test

import (
	"encoding/binary"
	"math"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"appload-games/games/crossword/backend/hwr"
	"appload-games/games/crossword/hwrtrain/letters"
)

// The Go model must match PyTorch's on the same inputs (testdata/reference.bin
// is written by hwrtrain/train.py).
func TestMatchesPyTorch(t *testing.T) {
	data, err := os.ReadFile("testdata/reference.bin")
	if err != nil {
		t.Fatal(err)
	}
	n := int(binary.LittleEndian.Uint32(data))
	images, logits := data[4:4+n*784], data[4+n*784:]
	m, err := hwr.Load()
	if err != nil {
		t.Fatal(err)
	}
	for i := range n {
		var img [784]float32
		for k := range img {
			img[k] = float32(images[i*784+k]) / 255
		}
		got := m.Logits(img)
		for k, v := range got {
			want := math.Float32frombits(binary.LittleEndian.Uint32(logits[(i*26+k)*4:]))
			if math.Abs(float64(v-want)) > 1e-3 {
				t.Fatalf("sample %d logit %d: got %f, PyTorch %f", i, k, v, want)
			}
		}
	}
}

// Letters written with the pen (synthetic strokes, a different seed than the
// training data) should mostly be recognized, and fast.
func TestRecognizeStrokes(t *testing.T) {
	m, err := hwr.Load()
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewPCG(99, 99))
	correct, top3, total := 0, 0, 0
	start := time.Now()
	for range 20 {
		for l := byte('A'); l <= 'Z'; l++ {
			guesses := m.Recognize(letters.Variant(rng, l), 3)
			total++
			if guesses[0].Letter == string(l) {
				correct++
			}
			for _, g := range guesses {
				if g.Letter == string(l) {
					top3++
				}
			}
		}
	}
	per := time.Since(start) / time.Duration(total)
	t.Logf("%d/%d right, %d in the top 3, %v per letter", correct, total, top3, per)
	if float64(correct)/float64(total) < 0.95 {
		t.Fatalf("only %d of %d letters recognized", correct, total)
	}
	if got := m.Recognize(nil, 3); got != nil {
		t.Fatal("recognized an empty drawing")
	}
	// Size and position don't matter.
	small := []hwr.Stroke{{{X: 1000, Y: 1000}, {X: 1001, Y: 1010}}}
	if g := m.Recognize(small, 1); len(g) != 1 {
		t.Fatal("no guess for a tiny stroke")
	}
}

func BenchmarkLogits(b *testing.B) {
	m, err := hwr.Load()
	if err != nil {
		b.Fatal(err)
	}
	img, _ := hwr.Image(letters.Variant(rand.New(rand.NewPCG(1, 1)), 'A'))
	b.ResetTimer()
	for range b.N {
		m.Logits(img)
	}
}

func BenchmarkImage(b *testing.B) {
	strokes := letters.Variant(rand.New(rand.NewPCG(1, 1)), 'A')
	for range b.N {
		hwr.Image(strokes)
	}
}

func BenchmarkVariant(b *testing.B) {
	rng := rand.New(rand.NewPCG(1, 1))
	for range b.N {
		letters.Variant(rng, 'A')
	}
}
