package board

import (
	"math/rand/v2"
	"testing"
)

func all(n int) []bool {
	p := make([]bool, n)
	for i := range p {
		p[i] = true
	}
	return p
}

func freeCount(l *Layout, present []bool) int {
	n := 0
	for i := range present {
		if l.Free(i, present) {
			n++
		}
	}
	return n
}

func TestLayouts(t *testing.T) {
	want := map[string][2]int{"turtle": {144, 35}, "pyramid": {120, 40}, "quick": {56, 24}}
	for _, l := range Layouts {
		seen := map[Pos]bool{}
		for _, p := range l.Tiles {
			if seen[p] {
				t.Fatalf("%s: two tiles at %v", l.Name, p)
			}
			seen[p] = true
		}
		got := [2]int{len(l.Tiles), freeCount(l, all(len(l.Tiles)))}
		if got != want[l.Name] {
			t.Errorf("%s: %d tiles, %d free; want %v", l.Name, got[0], got[1], want[l.Name])
		}
	}
	if l := LayoutByName("turtle"); l.Width != 30 || l.Height != 16 || l.Levels != 5 {
		t.Errorf("turtle is %dx%d with %d levels", l.Width, l.Height, l.Levels)
	}
}

// replay checks that taking the pairs off in order is legal and clears the board.
func replay(t *testing.T, l *Layout, faces []int8, present []bool, order [][2]int) {
	t.Helper()
	p := append([]bool(nil), present...)
	for _, pair := range order {
		a, b := pair[0], pair[1]
		if !l.Free(a, p) || !l.Free(b, p) || !Match(faces[a], faces[b]) {
			t.Fatalf("%s: illegal step %v", l.Name, pair)
		}
		p[a], p[b] = false, false
	}
	for i, ok := range p {
		if ok {
			t.Fatalf("%s: tile %d left over", l.Name, i)
		}
	}
}

func TestDealsCanBeCleared(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	for _, l := range Layouts {
		for range 50 {
			present := all(len(l.Tiles))
			pairs := standardPairs()[:len(l.Tiles)/2]
			faces := make([]int8, len(l.Tiles))
			order, ok := l.arrange(rng, present, pairs, faces)
			if !ok {
				t.Fatalf("%s: no deal", l.Name)
			}
			replay(t, l, faces, present, order)
		}
		faces := l.Deal(rng)
		counts := map[int8]int{}
		for _, f := range faces {
			counts[class(f)]++
		}
		for f, n := range counts {
			if n%2 != 0 {
				t.Fatalf("%s: odd number of tiles of face %d", l.Name, f)
			}
		}
	}
}

func TestShuffle(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 6))
	l := LayoutByName("turtle")
	faces := l.Deal(rng)
	present := all(len(faces))
	// Play greedily until stuck or done, shuffling when stuck.
	for shuffles := 0; ; {
		n, pair := l.Moves(faces, present)
		if pair != nil {
			if n < 1 || !Match(faces[pair[0]], faces[pair[1]]) {
				t.Fatal("Moves returned a bad pair")
			}
			present[pair[0]], present[pair[1]] = false, false
			continue
		}
		if freeCount(l, present) == 0 {
			break // cleared
		}
		shuffled, err := l.Shuffle(rng, faces, present)
		if err != nil {
			t.Fatal(err)
		}
		for i := range faces {
			if !present[i] && shuffled[i] != faces[i] {
				t.Fatal("shuffle changed a removed tile")
			}
		}
		faces = shuffled
		if shuffles++; shuffles > 50 {
			t.Fatal("too many shuffles")
		}
	}
	for i, ok := range present {
		if ok {
			t.Fatalf("tile %d left", i)
		}
	}
}

func TestMatch(t *testing.T) {
	if !Match(34, 37) || !Match(38, 41) || Match(37, 38) || Match(0, 1) || !Match(5, 5) {
		t.Fatal("wrong matching rules")
	}
}
