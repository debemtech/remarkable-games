package puzzle_test

import (
	"os"
	"strings"
	"testing"

	"appload-games/games/crossword/backend/puzzle"
)

func TestPuz(t *testing.T) {
	data, err := os.ReadFile("testdata/mini.puz")
	if err != nil {
		t.Fatal(err)
	}
	p, err := puzzle.ParsePuz(data)
	if err != nil {
		t.Fatal(err)
	}
	if p.Width != 4 || p.Height != 4 || !strings.HasPrefix(p.Title, "Mini 4") || !p.HasSolution {
		t.Fatalf("bad puzzle %q %dx%d", p.Title, p.Width, p.Height)
	}
	var row strings.Builder
	for c := range 4 {
		row.WriteString(p.Cells[c].Solution)
	}
	if row.String() != "CARD" || p.Across[0].Number != 1 || len(p.Across[0].Cells) != 4 || p.Down[2].Text != "Bring up, as children" || p.Across[1].Number != 5 {
		t.Fatalf("first row %q, first clue %+v", row.String(), p.Across[0])
	}
	if len(p.Across)+len(p.Down) != 8 {
		t.Fatalf("%d clues", len(p.Across)+len(p.Down))
	}
	// Round trip through .ipuz
	out, err := p.IPuz()
	if err != nil {
		t.Fatal(err)
	}
	q, err := puzzle.ParseIPuz(out)
	if err != nil {
		t.Fatal(err)
	}
	if q.Title != p.Title || len(q.Down) != len(p.Down) || q.Down[2].Text != p.Down[2].Text || q.Cells[5].Solution != p.Cells[5].Solution {
		t.Fatal(".ipuz round trip lost something")
	}
}

func TestIPuz(t *testing.T) {
	data, err := os.ReadFile("testdata/sample.ipuz")
	if err != nil {
		t.Fatal(err)
	}
	p, err := puzzle.ParseIPuz(data)
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Pequena cruzada" || len(p.Across) != 3 || len(p.Down) != 3 {
		t.Fatalf("bad puzzle %+v", p)
	}
	if got := p.Across[2]; got.Number != 5 || got.Text != "Ação sem o" || len(got.Cells) != 2 {
		t.Fatalf("clue 5 across: %+v", got)
	}
	if puzzle.Letter(p.Cells[1].Solution) != "O" || puzzle.Letter(p.Cells[8].Solution) != "C" {
		t.Fatal("accents should not count")
	}
	if !p.Cells[2].Block || p.Cells[4].Number != 0 || p.Cells[5].Number != 4 {
		t.Fatal("wrong grid")
	}
}

func TestLetter(t *testing.T) {
	for in, want := range map[string]string{"ç": "C", "Ã": "A", "STAR": "S", "é": "E", " ": "", "7": "7"} {
		if got := puzzle.Letter(in); got != want {
			t.Errorf("Letter(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFromRows(t *testing.T) {
	p, err := puzzle.FromRows("Mini", "Teste", []string{"SOL", "O#A", "LAC"}, []puzzle.Word{
		{Row: 0, Col: 0, Across: true, Clue: "Astro"},
		{Row: 2, Col: 0, Across: true, Clue: "Laço sem o"},
		{Row: 0, Col: 0, Clue: "Sol"},
		{Row: 0, Col: 2, Clue: "Lá + c"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !p.Cells[4].Block || p.Cells[2].Number != 2 || p.Across[1].Number != 3 || p.Down[1].Number != 2 || len(p.Down[1].Cells) != 3 {
		t.Fatalf("bad puzzle %+v", p)
	}
	if _, err := puzzle.FromRows("x", "", []string{"AB", "C"}, nil); err == nil {
		t.Fatal("accepted rows of different lengths")
	}
}

func TestCompilerArrowword(t *testing.T) {
	data, err := os.ReadFile("testdata/diretas.xml")
	if err != nil {
		t.Fatal(err)
	}
	p, err := puzzle.FromCompiler(data)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Arrows || p.Width != 4 || p.Height != 4 || len(p.Across) != 3 || len(p.Down) != 3 {
		t.Fatalf("bad puzzle %dx%d arrows=%v clues=%d+%d", p.Width, p.Height, p.Arrows, len(p.Across), len(p.Down))
	}
	// "Pó branco das paredes" is left of CAL, the first answer across.
	first := p.Across[0]
	word := ""
	for _, i := range first.Cells {
		word += p.Cells[i].Solution
	}
	if first.At != 4 || word != "CAL" || first.Text != "Pó branco\ndas paredes" || !p.Cells[4].HasClues || p.Cells[0].HasClues {
		t.Fatalf("first clue %+v %q", first, word)
	}
	if d := p.Down[2]; d.At != 3 || len(d.Cells) != 3 || d.Cells[0] != 7 {
		t.Fatalf("LAR down: %+v", d)
	}
	out, err := p.IPuz()
	if err != nil {
		t.Fatal(err)
	}
	q, err := puzzle.ParseIPuz(out)
	if err != nil {
		t.Fatal(err)
	}
	if !q.Arrows || len(q.Across) != len(p.Across) || q.Across[2].At != p.Across[2].At || len(q.Down[1].Cells) != len(p.Down[1].Cells) || !q.Cells[4].HasClues {
		t.Fatal(".ipuz round trip lost the arrows")
	}
}
