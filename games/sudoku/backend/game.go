package main

import (
	"appload-games/framework/appload"
	"appload-games/framework/stats"
	"appload-games/games/sudoku/backend/sudoku"
)

// cell is one cell's state, used to undo changes.
type cell struct {
	Index int
	Value int8
	Notes uint16
}

// game is a puzzle in progress. Exported fields are saved.
type game struct {
	Difficulty string      `json:"difficulty"`
	Givens     sudoku.Grid `json:"givens"`
	Solution   sudoku.Grid `json:"solution"`
	Values     sudoku.Grid `json:"values"` // includes the givens
	Notes      [81]uint16  `json:"notes"`  // bit d-1 set: pencil mark d
	Hints      int         `json:"hints"`
	Solved     bool        `json:"solved"`
	stats.Clock

	undo  [][]cell // each step restores a list of cells
	wrong []int    // cells marked by "check", until the next change
}

func newGame(difficulty string, puzzle, solution sudoku.Grid) *game {
	return &game{Difficulty: difficulty, Givens: puzzle, Solution: solution, Values: puzzle}
}

func (g *game) given(i int) bool { return g.Givens[i] != 0 }

func (g *game) snapshot(cells ...int) []cell {
	step := make([]cell, len(cells))
	for i, c := range cells {
		step[i] = cell{c, g.Values[c], g.Notes[c]}
	}
	return step
}

func (g *game) editable(i int) error {
	switch {
	case g.Solved:
		return appload.Info("The puzzle is solved.")
	case i < 0 || i >= 81:
		return appload.Errorf("No such cell %d", i)
	case g.given(i):
		return appload.Info("That number is part of the puzzle.")
	}
	return nil
}

// set puts digit d (0 = erase) into a cell and removes it from the pencil
// marks of the cell's row, column and box.
func (g *game) set(i int, d int8) error {
	if err := g.editable(i); err != nil {
		return err
	}
	if d < 0 || d > 9 {
		return appload.Errorf("No such digit %d", d)
	}
	if g.Values[i] == d && g.Notes[i] == 0 {
		return nil
	}
	changed := []int{i}
	if d != 0 {
		for _, p := range sudoku.Peers(i) {
			if g.Notes[p]&(1<<(d-1)) != 0 {
				changed = append(changed, p)
			}
		}
	}
	g.undo = append(g.undo, g.snapshot(changed...))
	g.Values[i], g.Notes[i] = d, 0
	for _, p := range changed[1:] {
		g.Notes[p] &^= 1 << (d - 1)
	}
	g.changed()
	return nil
}

// note toggles pencil mark d in an empty cell.
func (g *game) note(i int, d int8) error {
	if err := g.editable(i); err != nil {
		return err
	}
	if d < 1 || d > 9 {
		return appload.Errorf("No such digit %d", d)
	}
	if g.Values[i] != 0 {
		return appload.Info("Erase the number first to add notes.")
	}
	g.undo = append(g.undo, g.snapshot(i))
	g.Notes[i] ^= 1 << (d - 1)
	g.changed()
	return nil
}

func (g *game) undoStep() error {
	if g.Solved {
		return appload.Info("The puzzle is solved.")
	}
	n := len(g.undo)
	if n == 0 {
		return appload.Info("There's nothing to undo.")
	}
	for _, c := range g.undo[n-1] {
		g.Values[c.Index], g.Notes[c.Index] = c.Value, c.Notes
	}
	g.undo = g.undo[:n-1]
	g.changed()
	return nil
}

// hint fills in a cell: the given one if it's empty or wrong, otherwise the
// first one that is.
func (g *game) hint(i int) error {
	if g.Solved {
		return appload.Info("The puzzle is solved.")
	}
	if i < 0 || i >= 81 || g.Values[i] == g.Solution[i] {
		i = -1
		for c := range 81 {
			if g.Values[c] != g.Solution[c] {
				i = c
				break
			}
		}
	}
	if i < 0 {
		return nil
	}
	g.Hints++
	return g.set(i, g.Solution[i])
}

// check marks the cells that don't match the solution.
func (g *game) check() error {
	g.wrong = g.wrong[:0]
	for c, v := range g.Values {
		if v != 0 && v != g.Solution[c] {
			g.wrong = append(g.wrong, c)
		}
	}
	if len(g.wrong) == 0 {
		return appload.Info("No mistakes so far.")
	}
	if len(g.wrong) == 1 {
		return appload.Info("1 number is wrong.")
	}
	return appload.Info("%d numbers are wrong.", len(g.wrong))
}

func (g *game) restart() {
	g.Values, g.Notes, g.undo = g.Givens, [81]uint16{}, nil
	g.changed()
}

func (g *game) changed() {
	g.wrong = nil
	if g.Values == g.Solution && !g.Solved {
		g.Pause()
		g.Solved = true
	}
}

func (g *game) remaining() int {
	n := 0
	for _, v := range g.Values {
		if v == 0 {
			n++
		}
	}
	return n
}

func (g *game) state() gameState {
	conflicts := g.Values.Conflicts()
	given := make([]bool, 81)
	for i := range given {
		given[i] = g.given(i)
	}
	return gameState{
		Active:     true,
		Difficulty: g.Difficulty,
		Values:     g.Values,
		Given:      given,
		Notes:      g.Notes,
		Conflicts:  nonNil(conflicts),
		Wrong:      nonNil(g.wrong),
		Solved:     g.Solved,
		Seconds:    g.Now(),
		Hints:      g.Hints,
		CanUndo:    len(g.undo) > 0 && !g.Solved,
	}
}

func nonNil(s []int) []int {
	if s == nil {
		return []int{}
	}
	return s
}
