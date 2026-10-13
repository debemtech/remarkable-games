package sudoku

import (
	"math/rand/v2"
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for _, d := range Difficulties {
		for i := 0; i < 5; i++ {
			start := time.Now()
			puzzle, solution := Generate(rng, d)
			took := time.Since(start)

			if CountSolutions(puzzle, 2) != 1 {
				t.Fatalf("%s: puzzle has no unique solution", d.Name)
			}
			if solved, _ := Solve(puzzle); solved != solution {
				t.Fatalf("%s: wrong solution", d.Name)
			}
			for cell, v := range puzzle {
				if v != 0 && v != solution[cell] {
					t.Fatalf("%s: clue doesn't match the solution", d.Name)
				}
				if puzzle[80-cell] == 0 != (v == 0) {
					t.Fatalf("%s: clues are not symmetric", d.Name)
				}
			}
			if len(solution.Conflicts()) != 0 || clueCount(solution) != 81 {
				t.Fatalf("%s: invalid solution", d.Name)
			}
			if d.Singles && !SolvableWithSingles(puzzle) {
				t.Fatalf("%s: needs more than singles", d.Name)
			}
			if d.Singles && clueCount(puzzle) > d.Clues+6 {
				t.Errorf("%s: %d clues, aiming for %d", d.Name, clueCount(puzzle), d.Clues)
			}
			if d.Name == "expert" && clueCount(puzzle) > 27 {
				t.Errorf("expert puzzle with %d clues", clueCount(puzzle))
			}
			t.Logf("%s: %d clues, singles=%v, %v", d.Name, clueCount(puzzle), SolvableWithSingles(puzzle), took)
		}
	}
}

func TestSolve(t *testing.T) {
	// A well-known hard puzzle
	var g Grid
	for i, c := range "8..........36......7..9.2...5...7.......457.....1...3...1....68..85...1..9....4.." {
		if c != '.' {
			g[i] = int8(c - '0')
		}
	}
	solution, ok := Solve(g)
	if !ok || len(solution.Conflicts()) != 0 || clueCount(solution) != 81 {
		t.Fatal("failed to solve")
	}
	if CountSolutions(g, 2) != 1 || SolvableWithSingles(g) {
		t.Fatal("misjudged the puzzle")
	}
	g[1], g[2] = 1, 1
	if _, ok := Solve(g); ok || len(g.Conflicts()) < 2 {
		t.Fatal("solved an invalid grid")
	}
	if CountSolutions(Grid{}, 2) != 2 {
		t.Fatal("an empty grid has many solutions")
	}
}

func BenchmarkGenerateExpert(b *testing.B) {
	rng := rand.New(rand.NewPCG(1, 2))
	for range b.N {
		Generate(rng, Difficulties[3])
	}
}
