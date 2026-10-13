// Package board has the rules of mahjong solitaire: layouts, which tiles are
// free, and deals that are always solvable.
package board

import (
	"errors"
	"math/rand/v2"
	"sort"
)

// Pos is a tile's place in half-tile units: the tile covers X..X+2, Y..Y+2 on level Z.
type Pos struct{ X, Y, Z int }

// Layout is an arrangement of tiles.
type Layout struct {
	Name  string
	Title string
	Tiles []Pos // drawing order: lower levels first, then top to bottom, left to right
	// Bounding box in half-tile units, and the number of levels.
	Width, Height, Levels int

	above, left, right [][]int // tiles that block each tile
}

func newLayout(name, title string, tiles []Pos) *Layout {
	sort.Slice(tiles, func(i, j int) bool {
		a, b := tiles[i], tiles[j]
		if a.Z != b.Z {
			return a.Z < b.Z
		}
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		return a.X < b.X
	})
	l := &Layout{Name: name, Title: title, Tiles: tiles}
	n := len(tiles)
	l.above, l.left, l.right = make([][]int, n), make([][]int, n), make([][]int, n)
	for i, a := range tiles {
		l.Width = max(l.Width, a.X+2)
		l.Height = max(l.Height, a.Y+2)
		l.Levels = max(l.Levels, a.Z+1)
		for j, b := range tiles {
			dx, dy := b.X-a.X, b.Y-a.Y
			overlapY := dy > -2 && dy < 2
			switch {
			case b.Z > a.Z && dx > -2 && dx < 2 && overlapY:
				l.above[i] = append(l.above[i], j)
			case b.Z == a.Z && dx == -2 && overlapY:
				l.left[i] = append(l.left[i], j)
			case b.Z == a.Z && dx == 2 && overlapY:
				l.right[i] = append(l.right[i], j)
			}
		}
	}
	return l
}

func anyPresent(tiles []int, present []bool) bool {
	for _, t := range tiles {
		if present[t] {
			return true
		}
	}
	return false
}

// Free reports whether tile i can be taken: nothing on top of it, and nothing
// next to it on at least one side.
func (l *Layout) Free(i int, present []bool) bool {
	return present[i] && !anyPresent(l.above[i], present) &&
		(!anyPresent(l.left[i], present) || !anyPresent(l.right[i], present))
}

// Faces: 0-8 dots, 9-17 bamboo, 18-26 characters, 27-30 winds, 31-33 dragons,
// 34-37 flowers, 38-41 seasons. Any flower matches any flower, any season any season.
const (
	firstFlower = 34
	firstSeason = 38
	faceCount   = 42
)

func class(f int8) int8 {
	switch {
	case f >= firstSeason:
		return firstSeason
	case f >= firstFlower:
		return firstFlower
	}
	return f
}

// Match reports whether two faces can be removed together.
func Match(a, b int8) bool { return class(a) == class(b) }

// standardPairs is the 144 tile set as 72 matching pairs.
func standardPairs() [][2]int8 {
	var pairs [][2]int8
	for f := int8(0); f < firstFlower; f++ {
		pairs = append(pairs, [2]int8{f, f}, [2]int8{f, f})
	}
	for f := int8(firstFlower); f < faceCount; f += 2 {
		pairs = append(pairs, [2]int8{f, f + 1})
	}
	return pairs
}

// removalOrder takes away random free pairs until nothing is left, which
// proves the position can be cleared. It fails if it gets stuck.
func (l *Layout) removalOrder(rng *rand.Rand, present []bool) ([][2]int, bool) {
	p := append([]bool(nil), present...)
	left := 0
	for _, ok := range p {
		if ok {
			left++
		}
	}
	var order [][2]int
	free := make([]int, 0, len(p))
	for left > 0 {
		free = free[:0]
		for i := range p {
			if l.Free(i, p) {
				free = append(free, i)
			}
		}
		if len(free) < 2 {
			return nil, false
		}
		a := rng.IntN(len(free))
		b := rng.IntN(len(free) - 1)
		if b >= a {
			b++
		}
		order = append(order, [2]int{free[a], free[b]})
		p[free[a]], p[free[b]] = false, false
		left -= 2
	}
	return order, true
}

// arrange puts the pairs onto the present tiles so that the position can be
// cleared, by taking the pairs off in the returned order.
func (l *Layout) arrange(rng *rand.Rand, present []bool, pairs [][2]int8, faces []int8) ([][2]int, bool) {
	for range 100 {
		order, ok := l.removalOrder(rng, present)
		if !ok {
			continue
		}
		rng.Shuffle(len(pairs), func(i, j int) { pairs[i], pairs[j] = pairs[j], pairs[i] })
		for k, tiles := range order {
			a, b := pairs[k][0], pairs[k][1]
			if rng.IntN(2) == 0 {
				a, b = b, a
			}
			faces[tiles[0]], faces[tiles[1]] = a, b
		}
		return order, true
	}
	return nil, false
}

// Deal returns faces for all tiles of the layout. The deal can always be cleared.
func (l *Layout) Deal(rng *rand.Rand) []int8 {
	pairs := standardPairs()
	rng.Shuffle(len(pairs), func(i, j int) { pairs[i], pairs[j] = pairs[j], pairs[i] })
	pairs = pairs[:len(l.Tiles)/2]
	present := make([]bool, len(l.Tiles))
	for i := range present {
		present[i] = true
	}
	faces := make([]int8, len(l.Tiles))
	if _, ok := l.arrange(rng, present, pairs, faces); !ok {
		panic("layout " + l.Name + " can't be cleared")
	}
	return faces
}

// ErrNoSolvableShuffle means the remaining tiles can't be cleared in any order,
// e.g. when the last two tiles are stacked on each other.
var ErrNoSolvableShuffle = errors.New("the remaining tiles can't be cleared")

// Shuffle rearranges the faces of the remaining tiles into a position that can be cleared.
func (l *Layout) Shuffle(rng *rand.Rand, faces []int8, present []bool) ([]int8, error) {
	byClass := map[int8][]int8{}
	for i, ok := range present {
		if ok {
			byClass[class(faces[i])] = append(byClass[class(faces[i])], faces[i])
		}
	}
	var pairs [][2]int8
	for _, fs := range byClass {
		for i := 0; i+1 < len(fs); i += 2 {
			pairs = append(pairs, [2]int8{fs[i], fs[i+1]})
		}
	}
	out := append([]int8(nil), faces...)
	if _, ok := l.arrange(rng, present, pairs, out); !ok {
		return nil, ErrNoSolvableShuffle
	}
	return out, nil
}

// Moves returns how many matching pairs are free, and one of them (or nil).
func (l *Layout) Moves(faces []int8, present []bool) (int, []int) {
	byClass := map[int8][]int{}
	for i := range present {
		if l.Free(i, present) {
			byClass[class(faces[i])] = append(byClass[class(faces[i])], i)
		}
	}
	count := 0
	var example []int
	for _, tiles := range byClass {
		n := len(tiles)
		count += n * (n - 1) / 2
		if n >= 2 && (example == nil || tiles[0] < example[0]) {
			example = tiles[:2]
		}
	}
	return count, example
}

func rect(z, x0, y0, cols, rows int) []Pos {
	var out []Pos
	for r := range rows {
		for c := range cols {
			out = append(out, Pos{x0 + 2*c, y0 + 2*r, z})
		}
	}
	return out
}

func turtle() []Pos {
	var t []Pos
	// Level 0: rows of 12, 8, 10, 12, 12, 10, 8, 12 tiles
	for row, span := range [][2]int{{1, 12}, {3, 8}, {2, 10}, {1, 12}, {1, 12}, {2, 10}, {3, 8}, {1, 12}} {
		t = append(t, rect(0, 2*span[0], 2*row, span[1], 1)...)
	}
	// The single tile on the left and the two on the right, between the middle rows
	t = append(t, Pos{0, 7, 0}, Pos{26, 7, 0}, Pos{28, 7, 0})
	t = append(t, rect(1, 8, 2, 6, 6)...)
	t = append(t, rect(2, 10, 4, 4, 4)...)
	t = append(t, rect(3, 12, 6, 2, 2)...)
	return append(t, Pos{13, 7, 4})
}

// pyramid stacks squares of n, n-2, ... tiles, each centered on the one below.
func pyramid(sizes ...int) []Pos {
	var t []Pos
	for z, n := range sizes {
		t = append(t, rect(z, 2*z, 2*z, n, n)...)
	}
	return t
}

// Layouts, in menu order.
var Layouts = []*Layout{
	newLayout("turtle", "Turtle", turtle()),
	newLayout("pyramid", "Pyramid", pyramid(8, 6, 4, 2)),
	newLayout("quick", "Quick", pyramid(6, 4, 2)),
}

// LayoutByName returns the layout with that name.
func LayoutByName(name string) *Layout {
	for _, l := range Layouts {
		if l.Name == name {
			return l
		}
	}
	return nil
}
