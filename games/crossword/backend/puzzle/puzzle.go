// Package puzzle reads crosswords (.puz, .ipuz, AmuseLabs JSON) into one model.
package puzzle

import (
	"errors"
	"html"
	"regexp"
	"strings"
)

// Cell is one square of the grid.
type Cell struct {
	Block    bool   `json:"block,omitempty"`
	Solution string `json:"solution,omitempty"` // as published, e.g. "Ç" or a rebus "STAR"
	Number   int    `json:"number,omitempty"`
	Circled  bool   `json:"circled,omitempty"`
	// HasClues: a block with clues written in it (cruzadas diretas)
	HasClues bool `json:"hasClues,omitempty"`
}

// Clue is a numbered clue and the cells of its answer.
type Clue struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
	Cells  []int  `json:"cells"`
	// At is the block the clue is written in, if the puzzle has Arrows.
	At int `json:"at"`
}

// Puzzle is a crossword. Cells are numbered row by row.
type Puzzle struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	Copyright string `json:"copyright"`
	Notes     string `json:"notes"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Cells     []Cell `json:"cells"`
	Across    []Clue `json:"across"`
	Down      []Clue `json:"down"`
	// HasSolution is false when the answers are missing or scrambled; the
	// puzzle can be played, but not checked.
	HasSolution bool `json:"hasSolution"`
	// Arrows: the clues are in blocks next to their answers (an arrowword,
	// "cruzadas diretas" in Brazil), not in lists.
	Arrows bool `json:"arrows,omitempty"`
}

// Words lists the cells of every answer in a direction: runs of two or more
// open cells, in reading order.
func (p *Puzzle) words(across bool) [][]int {
	var out [][]int
	open := func(r, c int) bool {
		return r >= 0 && c >= 0 && r < p.Height && c < p.Width && !p.Cells[r*p.Width+c].Block
	}
	for r := range p.Height {
		for c := range p.Width {
			dr, dc := 0, 1
			if !across {
				dr, dc = 1, 0
			}
			if !open(r, c) || open(r-dr, c-dc) || !open(r+dr, c+dc) {
				continue
			}
			var word []int
			for rr, cc := r, c; open(rr, cc); rr, cc = rr+dr, cc+dc {
				word = append(word, rr*p.Width+cc)
			}
			out = append(out, word)
		}
	}
	return out
}

// number assigns standard crossword numbers to the cells that start a word
// and returns the words in each direction, keyed by their start cell.
func (p *Puzzle) number() (across, down map[int][]int) {
	across, down = map[int][]int{}, map[int][]int{}
	for _, w := range p.words(true) {
		across[w[0]] = w
	}
	for _, w := range p.words(false) {
		down[w[0]] = w
	}
	n := 0
	for i := range p.Cells {
		p.Cells[i].Number = 0
		_, a := across[i]
		_, d := down[i]
		if a || d {
			n++
			p.Cells[i].Number = n
		}
	}
	return across, down
}

// attach fills in the cells of clues that only have a number.
func (p *Puzzle) attach() error {
	across, down := p.number()
	byNumber := map[int]int{}
	for i, c := range p.Cells {
		if c.Number > 0 {
			byNumber[c.Number] = i
		}
	}
	for _, list := range []struct {
		clues []Clue
		words map[int][]int
	}{{p.Across, across}, {p.Down, down}} {
		for i := range list.clues {
			cl := &list.clues[i]
			cl.Text = CleanText(cl.Text)
			if p.Arrows {
				if cl.At < 0 || cl.At >= len(p.Cells) || !p.Cells[cl.At].Block || len(cl.Cells) == 0 {
					return errors.New("a clue isn't next to its answer")
				}
				p.Cells[cl.At].HasClues = true
			}
			if len(cl.Cells) > 0 {
				for _, c := range cl.Cells {
					if c < 0 || c >= len(p.Cells) || p.Cells[c].Block {
						return errors.New("a clue doesn't match the grid")
					}
				}
				continue
			}
			start, ok := byNumber[cl.Number]
			if !ok || list.words[start] == nil {
				return errors.New("a clue doesn't match the grid")
			}
			cl.Cells = list.words[start]
		}
	}
	return nil
}

var (
	tags   = regexp.MustCompile(`<[^>]*>`)
	spaces = regexp.MustCompile(`[ \t]+`)
)

// CleanText removes HTML from clues and titles. Line breaks stay: clues
// written in a cell are broken to fit.
func CleanText(s string) string {
	s = html.UnescapeString(tags.ReplaceAllString(s, ""))
	lines := strings.Split(strings.ReplaceAll(s, "\r", ""), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(spaces.ReplaceAllString(l, " "))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

var plain = strings.NewReplacer(
	"Á", "A", "À", "A", "Â", "A", "Ã", "A", "Ä", "A", "Å", "A",
	"Ç", "C", "É", "E", "È", "E", "Ê", "E", "Ë", "E",
	"Í", "I", "Ì", "I", "Î", "I", "Ï", "I", "Ñ", "N",
	"Ó", "O", "Ò", "O", "Ô", "O", "Õ", "O", "Ö", "O",
	"Ú", "U", "Ù", "U", "Û", "U", "Ü", "U", "Ý", "Y",
)

// Letter is what a cell's answer is written as: one plain capital letter.
// Accents don't count (Portuguese crosswords are written without them), and a
// rebus counts by its first letter.
func Letter(solution string) string {
	s := plain.Replace(strings.ToUpper(strings.TrimSpace(solution)))
	for _, r := range s {
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return string(r)
		}
	}
	return ""
}

func (p *Puzzle) validate() error {
	if p.Width <= 0 || p.Height <= 0 || p.Width > 50 || p.Height > 50 || len(p.Cells) != p.Width*p.Height {
		return errors.New("the grid has a strange size")
	}
	if len(p.Across)+len(p.Down) == 0 {
		return errors.New("the puzzle has no clues")
	}
	p.Title = CleanText(p.Title)
	p.Author = CleanText(p.Author)
	p.Copyright = CleanText(p.Copyright)
	return p.attach()
}
