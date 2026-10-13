// Package sudoku generates and solves 9x9 sudokus.
//
// Cells are numbered 0-80, row by row. A value of 0 is an empty cell.
package sudoku

import (
	"math/bits"
	"math/rand/v2"
)

// Grid holds the 81 cell values.
type Grid [81]int8

const all = 0x1FF // candidate mask with all digits 1-9

// Row, Col and Box return the row, column and 3x3 box (0-8) of a cell.
func Row(cell int) int { return cell / 9 }
func Col(cell int) int { return cell % 9 }
func Box(cell int) int { return Row(cell)/3*3 + Col(cell)/3 }

var (
	// units are the 27 rows, columns and boxes.
	units [27][9]int
	// peers are the 20 cells sharing a unit with a cell.
	peers [81][]int
)

func init() {
	for cell := range 81 {
		units[Row(cell)][Col(cell)] = cell
		units[9+Col(cell)][Row(cell)] = cell
		units[18+Box(cell)][Row(cell)%3*3+Col(cell)%3] = cell
	}
	for cell := range 81 {
		for other := range 81 {
			if other != cell && (Row(other) == Row(cell) || Col(other) == Col(cell) || Box(other) == Box(cell)) {
				peers[cell] = append(peers[cell], other)
			}
		}
	}
}

// Peers returns the cells sharing a row, column or box with cell.
func Peers(cell int) []int { return peers[cell] }

// Conflicts returns the filled cells whose value appears again among their peers.
func (g *Grid) Conflicts() []int {
	var out []int
	for cell, v := range g {
		if v == 0 {
			continue
		}
		for _, p := range peers[cell] {
			if g[p] == v {
				out = append(out, cell)
				break
			}
		}
	}
	return out
}

// solver tracks the digits used per row, column and box as bitmasks.
type solver struct {
	g                 Grid
	rows, cols, boxes [9]uint16
	rng               *rand.Rand // nil: try digits in order
}

func newSolver(g Grid) (*solver, bool) {
	s := &solver{g: g}
	for cell, v := range g {
		if v == 0 {
			continue
		}
		bit := uint16(1) << (v - 1)
		r, c, b := Row(cell), Col(cell), Box(cell)
		if s.rows[r]&bit != 0 || s.cols[c]&bit != 0 || s.boxes[b]&bit != 0 {
			return nil, false
		}
		s.rows[r] |= bit
		s.cols[c] |= bit
		s.boxes[b] |= bit
	}
	return s, true
}

func (s *solver) candidates(cell int) uint16 {
	return ^(s.rows[Row(cell)] | s.cols[Col(cell)] | s.boxes[Box(cell)]) & all
}

func (s *solver) set(cell int, d int8) {
	bit := uint16(1) << (d - 1)
	s.g[cell] = d
	s.rows[Row(cell)] ^= bit
	s.cols[Col(cell)] ^= bit
	s.boxes[Box(cell)] ^= bit
}

// search counts solutions up to limit. With limit 1 the solution stays in s.g.
func (s *solver) search(limit int) int {
	// Branch on the empty cell with the fewest candidates.
	best, bestMask, bestCount := -1, uint16(0), 10
	for cell := range 81 {
		if s.g[cell] != 0 {
			continue
		}
		m := s.candidates(cell)
		n := bits.OnesCount16(m)
		if n == 0 {
			return 0
		}
		if n < bestCount {
			best, bestMask, bestCount = cell, m, n
			if n == 1 {
				break
			}
		}
	}
	if best < 0 {
		return 1
	}

	digits := make([]int8, 0, 9)
	for m := bestMask; m != 0; m &= m - 1 {
		digits = append(digits, int8(bits.TrailingZeros16(m)+1))
	}
	if s.rng != nil {
		s.rng.Shuffle(len(digits), func(i, j int) { digits[i], digits[j] = digits[j], digits[i] })
	}
	count := 0
	for _, d := range digits {
		s.set(best, d)
		count += s.search(limit - count)
		if count >= limit {
			return count // leave the found solution in place
		}
		s.set(best, d) // xor undoes the bits
		s.g[best] = 0
	}
	return count
}

// CountSolutions returns the number of solutions of g, but at most limit.
func CountSolutions(g Grid, limit int) int {
	s, ok := newSolver(g)
	if !ok {
		return 0
	}
	return s.search(limit)
}

// Solve returns a solution of g.
func Solve(g Grid) (Grid, bool) {
	s, ok := newSolver(g)
	if !ok || s.search(1) == 0 {
		return g, false
	}
	return s.g, true
}

// SolvableWithSingles reports whether g can be solved by only ever filling in
// a cell with a single candidate, or a digit with a single place in a unit.
// Those are the techniques easy puzzles need.
func SolvableWithSingles(g Grid) bool {
	s, ok := newSolver(g)
	if !ok {
		return false
	}
	for {
		progress, empty := false, false
		for cell := range 81 {
			if s.g[cell] != 0 {
				continue
			}
			empty = true
			m := s.candidates(cell)
			if m == 0 {
				return false
			}
			if bits.OnesCount16(m) == 1 {
				s.set(cell, int8(bits.TrailingZeros16(m)+1))
				progress = true
			}
		}
		if !empty {
			return true
		}
		for _, unit := range units {
			for d := int8(1); d <= 9; d++ {
				bit := uint16(1) << (d - 1)
				place, count := -1, 0
				for _, cell := range unit {
					if s.g[cell] == d {
						count = -1
						break
					}
					if s.g[cell] == 0 && s.candidates(cell)&bit != 0 {
						place = cell
						count++
					}
				}
				if count == 1 {
					s.set(place, d)
					progress = true
				}
			}
		}
		if !progress {
			return false
		}
	}
}

// Difficulty describes how a puzzle is generated.
type Difficulty struct {
	Name string
	// Clues to aim for. Removing stops here, or earlier when no cell can go
	// without losing a unique solution.
	Clues int
	// Singles: must be solvable with singles. !Singles: must not be.
	Singles bool
	// Generate this many puzzles and keep the one with the fewest clues.
	Tries int
}

var Difficulties = []Difficulty{
	{Name: "easy", Clues: 40, Singles: true, Tries: 1},
	{Name: "medium", Clues: 32, Singles: true, Tries: 1},
	{Name: "hard", Clues: 28, Singles: false, Tries: 1},
	{Name: "expert", Clues: 17, Singles: false, Tries: 8},
}

// DifficultyByName returns the difficulty with that name.
func DifficultyByName(name string) (Difficulty, bool) {
	for _, d := range Difficulties {
		if d.Name == name {
			return d, true
		}
	}
	return Difficulty{}, false
}

// Generate returns a puzzle with a unique solution, and that solution. Clues
// are removed in pairs mirrored around the center, like printed sudokus.
func Generate(rng *rand.Rand, d Difficulty) (puzzle, solution Grid) {
	best := 82
	for range max(1, d.Tries) {
		p, s := generate(rng, d)
		if n := clueCount(p); n < best {
			best, puzzle, solution = n, p, s
		}
	}
	return puzzle, solution
}

func clueCount(g Grid) int {
	n := 0
	for _, v := range g {
		if v != 0 {
			n++
		}
	}
	return n
}

func generate(rng *rand.Rand, d Difficulty) (puzzle, solution Grid) {
	for attempt := 0; ; attempt++ {
		s, _ := newSolver(Grid{})
		s.rng = rng
		s.search(1)
		solution = s.g
		puzzle = solution
		clues := 81

		for _, i := range rng.Perm(41) {
			if clues <= d.Clues {
				break
			}
			a, b := i, 80-i
			removed := puzzle
			removed[a], removed[b] = 0, 0
			n := 2
			if a == b {
				n = 1
			}
			if CountSolutions(removed, 2) != 1 {
				continue
			}
			if d.Singles && !SolvableWithSingles(removed) {
				continue
			}
			puzzle = removed
			clues -= n
		}
		// Hard puzzles need more than singles. Give up on that after a few
		// tries rather than keep the user waiting.
		if d.Singles || !SolvableWithSingles(puzzle) || attempt >= 20 {
			return puzzle, solution
		}
	}
}
