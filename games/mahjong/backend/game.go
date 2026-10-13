package main

import (
	"errors"
	"math/rand/v2"

	"appload-games/framework/appload"
	"appload-games/framework/stats"
	"appload-games/games/mahjong/backend/board"
)

type snapshot struct {
	faces   []int8
	present []bool
}

// game is a deal in progress. Exported fields are saved.
type game struct {
	Layout   string `json:"layout"`
	Faces    []int8 `json:"faces"`
	Present  []bool `json:"present"`
	Hints    int    `json:"hints"`
	Shuffles int    `json:"shuffles"`
	Solved   bool   `json:"solved"`
	stats.Clock

	layout *board.Layout
	undo   []snapshot
	hint   []int // pair shown by "hint", until the next change
}

func newGame(l *board.Layout, rng *rand.Rand) *game {
	present := make([]bool, len(l.Tiles))
	for i := range present {
		present[i] = true
	}
	return &game{Layout: l.Name, Faces: l.Deal(rng), Present: present, layout: l}
}

// valid reports whether a loaded game fits its layout.
func (g *game) valid() bool {
	g.layout = board.LayoutByName(g.Layout)
	return g.layout != nil && len(g.Faces) == len(g.layout.Tiles) && len(g.Present) == len(g.Faces)
}

func (g *game) remaining() int {
	n := 0
	for _, p := range g.Present {
		if p {
			n++
		}
	}
	return n
}

func (g *game) save() {
	g.undo = append(g.undo, snapshot{append([]int8(nil), g.Faces...), append([]bool(nil), g.Present...)})
	g.hint = nil
}

func (g *game) remove(a, b int) error {
	n := len(g.Faces)
	switch {
	case g.Solved:
		return appload.Info("The board is cleared.")
	case a < 0 || b < 0 || a >= n || b >= n || a == b || !g.Present[a] || !g.Present[b]:
		return appload.Errorf("No such tiles")
	case !g.layout.Free(a, g.Present) || !g.layout.Free(b, g.Present):
		return appload.Info("That tile is blocked.")
	case !board.Match(g.Faces[a], g.Faces[b]):
		return appload.Info("These tiles don't match.")
	}
	g.save()
	g.Present[a], g.Present[b] = false, false
	if g.remaining() == 0 {
		g.Pause()
		g.Solved = true
	}
	return nil
}

func (g *game) undoStep() error {
	if g.Solved {
		return appload.Info("The board is cleared.")
	}
	n := len(g.undo)
	if n == 0 {
		return appload.Info("There's nothing to undo.")
	}
	last := g.undo[n-1]
	g.Faces, g.Present, g.undo, g.hint = last.faces, last.present, g.undo[:n-1], nil
	return nil
}

func (g *game) showHint() error {
	if g.Solved {
		return nil
	}
	_, pair := g.layout.Moves(g.Faces, g.Present)
	if pair == nil {
		return appload.Info("No moves left. Shuffle or undo.")
	}
	g.hint = pair
	g.Hints++
	return nil
}

func (g *game) shuffle(rng *rand.Rand) error {
	if g.Solved {
		return nil
	}
	faces, err := g.layout.Shuffle(rng, g.Faces, g.Present)
	if errors.Is(err, board.ErrNoSolvableShuffle) {
		return appload.Info("These tiles can't be cleared in any order. Undo a few moves.")
	}
	if err != nil {
		return err
	}
	g.save()
	g.Faces = faces
	g.Shuffles++
	return nil
}

func (g *game) state() gameState {
	l := g.layout
	s := gameState{
		Active:    true,
		Layout:    l.Name,
		Width:     l.Width,
		Height:    l.Height,
		Levels:    l.Levels,
		X:         make([]int, len(l.Tiles)),
		Y:         make([]int, len(l.Tiles)),
		Z:         make([]int, len(l.Tiles)),
		Faces:     g.Faces,
		Present:   g.Present,
		Free:      make([]bool, len(l.Tiles)),
		Hint:      g.hint,
		Remaining: g.remaining(),
		Solved:    g.Solved,
		Seconds:   g.Now(),
		Hints:     g.Hints,
		Shuffles:  g.Shuffles,
		CanUndo:   len(g.undo) > 0 && !g.Solved,
	}
	for i, p := range l.Tiles {
		s.X[i], s.Y[i], s.Z[i] = p.X, p.Y, p.Z
		s.Free[i] = l.Free(i, g.Present)
	}
	s.Moves, _ = l.Moves(g.Faces, g.Present)
	if s.Hint == nil {
		s.Hint = []int{}
	}
	return s
}
