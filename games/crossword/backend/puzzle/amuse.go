package puzzle

import (
	"encoding/json"
	"errors"
)

// FromAmuse reads the puzzle data of an AmuseLabs (PuzzleMe) solver, after
// deobfuscation (see package sources).
func FromAmuse(data []byte) (*Puzzle, error) {
	var raw struct {
		Title     string     `json:"title"`
		Author    string     `json:"author"`
		Copyright string     `json:"copyright"`
		W         int        `json:"w"`
		H         int        `json:"h"`
		Box       [][]string `json:"box"` // columns, then rows; "\u0000" is a block
		CellInfos []struct {
			X, Y      int
			IsCircled bool `json:"isCircled"`
		} `json:"cellInfos"`
		PlacedWords []struct {
			X, Y          int
			AcrossNotDown bool `json:"acrossNotDown"`
			Clue          struct {
				Clue string `json:"clue"`
			} `json:"clue"`
		} `json:"placedWords"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if raw.W <= 0 || raw.H <= 0 || len(raw.Box) < raw.W {
		return nil, errors.New("unexpected puzzle data")
	}
	p := &Puzzle{Title: raw.Title, Author: raw.Author, Copyright: raw.Copyright,
		Width: raw.W, Height: raw.H, Cells: make([]Cell, raw.W*raw.H)}
	for c := range raw.W {
		for r := range raw.H {
			cell := &p.Cells[r*raw.W+c]
			v := ""
			if r < len(raw.Box[c]) {
				v = raw.Box[c][r]
			}
			if v == "\x00" {
				cell.Block = true
			} else {
				cell.Solution = v
				p.HasSolution = p.HasSolution || v != ""
			}
		}
	}
	for _, ci := range raw.CellInfos {
		if ci.X >= 0 && ci.Y >= 0 && ci.X < raw.W && ci.Y < raw.H {
			p.Cells[ci.Y*raw.W+ci.X].Circled = ci.IsCircled
		}
	}
	across, down := p.number()
	for _, w := range raw.PlacedWords {
		if w.X < 0 || w.Y < 0 || w.X >= raw.W || w.Y >= raw.H {
			continue
		}
		start := w.Y*raw.W + w.X
		words, list := down, &p.Down
		if w.AcrossNotDown {
			words, list = across, &p.Across
		}
		if cells, ok := words[start]; ok {
			*list = append(*list, Clue{Number: p.Cells[start].Number, Text: w.Clue.Clue, Cells: cells})
		}
	}
	sortClues(p.Across)
	sortClues(p.Down)
	return p, p.validate()
}

func sortClues(cs []Clue) {
	for i := 1; i < len(cs); i++ {
		for k := i; k > 0 && cs[k].Number < cs[k-1].Number; k-- {
			cs[k], cs[k-1] = cs[k-1], cs[k]
		}
	}
}

// Word is an answer placed in a grid: where it starts, its direction, its clue.
type Word struct {
	Row, Col int
	Across   bool
	Clue     string
}

// FromRows builds a puzzle from its solution, one string per row with '#'
// for blocks, and the clues of its answers. Numbers are the standard ones.
func FromRows(title, author string, rows []string, words []Word) (*Puzzle, error) {
	if len(rows) == 0 {
		return nil, errors.New("empty grid")
	}
	grid := make([][]rune, len(rows))
	for r, row := range rows {
		grid[r] = []rune(row)
	}
	w, h := len(grid[0]), len(grid)
	p := &Puzzle{Title: title, Author: author, Width: w, Height: h, Cells: make([]Cell, w*h), HasSolution: true}
	for r, row := range grid {
		if len(row) != w {
			return nil, errors.New("the grid's rows differ in length")
		}
		for c, ch := range row {
			if ch == '#' {
				p.Cells[r*w+c].Block = true
			} else {
				p.Cells[r*w+c].Solution = string(ch)
			}
		}
	}
	across, down := p.number()
	for _, wd := range words {
		if wd.Row < 0 || wd.Col < 0 || wd.Row >= h || wd.Col >= w {
			continue
		}
		start := wd.Row*w + wd.Col
		cells, list := down[start], &p.Down
		if wd.Across {
			cells, list = across[start], &p.Across
		}
		if cells != nil {
			*list = append(*list, Clue{Number: p.Cells[start].Number, Text: wd.Clue, Cells: cells})
		}
	}
	sortClues(p.Across)
	sortClues(p.Down)
	return p, p.validate()
}
