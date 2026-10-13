package puzzle

import (
	"encoding/xml"
	"errors"
	"strconv"
	"strings"
)

// FromCompiler reads a Crossword Compiler XML puzzle (the format of its web
// applet): classic crosswords with clue lists, and arrowwords ("cruzadas
// diretas") with the clues in cells.
func FromCompiler(data []byte) (*Puzzle, error) {
	var doc struct {
		Puzzle struct {
			Metadata struct {
				Title     string `xml:"title"`
				Creator   string `xml:"creator"`
				Copyright string `xml:"copyright"`
			} `xml:"metadata"`
			Crossword struct {
				Grid struct {
					Width  int `xml:"width,attr"`
					Height int `xml:"height,attr"`
					Cells  []struct {
						X        int    `xml:"x,attr"`
						Y        int    `xml:"y,attr"`
						Type     string `xml:"type,attr"`
						Solution string `xml:"solution,attr"`
						Shape    string `xml:"background-shape,attr"`
						TopBar   bool   `xml:"top-bar,attr"`
						LeftBar  bool   `xml:"left-bar,attr"`
						Clues    []struct {
							Word string `xml:"word,attr"`
							Text string `xml:",chardata"`
						} `xml:"clue"`
					} `xml:"cell"`
				} `xml:"grid"`
				Words []struct {
					ID    string `xml:"id,attr"`
					X     string `xml:"x,attr"`
					Y     string `xml:"y,attr"`
					Cells []struct {
						X int `xml:"x,attr"`
						Y int `xml:"y,attr"`
					} `xml:"cells"`
				} `xml:"word"`
				Clues []struct {
					Clues []struct {
						Word   string `xml:"word,attr"`
						Number string `xml:"number,attr"`
						Text   string `xml:",innerxml"`
					} `xml:"clue"`
				} `xml:"clues"`
			} `xml:"crossword"`
		} `xml:"rectangular-puzzle"`
	}
	data = []byte(strings.Replace(string(data), "<crossword-compiler ", "<crossword-compiler-applet ", 1))
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	cw := doc.Puzzle.Crossword
	w, h := cw.Grid.Width, cw.Grid.Height
	if w <= 0 || h <= 0 || w > 50 || h > 50 {
		return nil, errors.New("the grid has a strange size")
	}
	p := &Puzzle{Title: doc.Puzzle.Metadata.Title, Author: doc.Puzzle.Metadata.Creator,
		Copyright: doc.Puzzle.Metadata.Copyright, Width: w, Height: h, Cells: make([]Cell, w*h), HasSolution: true}
	at := func(x, y int) int { // 1-based coordinates
		if x < 1 || y < 1 || x > w || y > h {
			return -1
		}
		return (y-1)*w + x - 1
	}
	open := make([]bool, w*h)
	clueAt := map[string]int{} // word id -> cell its clue is in
	clueText := map[string]string{}
	for _, c := range cw.Grid.Cells {
		i := at(c.X, c.Y)
		if i < 0 {
			continue
		}
		if c.TopBar || c.LeftBar {
			return nil, errors.New("crosswords with bars between squares aren't supported yet")
		}
		if c.Type == "" && c.Solution != "" {
			open[i] = true
			p.Cells[i].Solution = c.Solution
			p.Cells[i].Circled = c.Shape == "circle"
		}
		for _, cl := range c.Clues {
			// Line breaks are written as \n
			clueAt[cl.Word], clueText[cl.Word] = i, strings.ReplaceAll(cl.Text, `\n`, "\n")
		}
	}
	for i := range p.Cells {
		p.Cells[i].Block = !open[i]
	}
	p.number()

	// The cells of every word
	words := map[string][]int{}
	across := map[string]bool{}
	for _, wd := range cw.Words {
		var cells []int
		if len(wd.Cells) > 0 {
			for _, c := range wd.Cells {
				cells = append(cells, at(c.X, c.Y))
			}
		} else {
			x0, x1, err1 := span(wd.X)
			y0, y1, err2 := span(wd.Y)
			if err1 != nil || err2 != nil {
				return nil, errors.New("a word has strange coordinates")
			}
			dx, dy := sign(x1-x0), sign(y1-y0)
			for x, y := x0, y0; ; x, y = x+dx, y+dy {
				cells = append(cells, at(x, y))
				if x == x1 && y == y1 {
					break
				}
			}
			across[wd.ID] = dy == 0
		}
		if len(cells) > 1 && !across[wd.ID] && cells[1]-cells[0] == 1 {
			across[wd.ID] = true
		}
		for _, c := range cells {
			if c < 0 || !open[c] {
				return nil, errors.New("a word doesn't fit the grid")
			}
		}
		words[wd.ID] = cells
	}
	add := func(id, text string, number int, where int) error {
		cells, ok := words[id]
		if !ok || len(cells) == 0 {
			return errors.New("a clue has no word")
		}
		if number == 0 {
			number = p.Cells[cells[0]].Number
		}
		c := Clue{Number: number, Text: text, Cells: cells, At: where}
		if across[id] {
			p.Across = append(p.Across, c)
		} else {
			p.Down = append(p.Down, c)
		}
		return nil
	}
	if len(clueAt) > 0 {
		p.Arrows = true
		for _, wd := range cw.Words {
			if where, ok := clueAt[wd.ID]; ok {
				if err := add(wd.ID, clueText[wd.ID], 0, where); err != nil {
					return nil, err
				}
			}
		}
	} else {
		for _, list := range cw.Clues {
			for _, cl := range list.Clues {
				n, _ := strconv.Atoi(cl.Number)
				if err := add(cl.Word, cl.Text, n, 0); err != nil {
					return nil, err
				}
			}
		}
	}
	sortClues(p.Across)
	sortClues(p.Down)
	return p, p.validate()
}

// span reads "3" or "3-7".
func span(s string) (int, int, error) {
	a, b, found := strings.Cut(s, "-")
	x0, err := strconv.Atoi(strings.TrimSpace(a))
	if err != nil || !found {
		return x0, x0, err
	}
	x1, err := strconv.Atoi(strings.TrimSpace(b))
	return x0, x1, err
}

func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	}
	return 0
}
