package puzzle

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ParseIPuz reads an .ipuz crossword (http://ipuz.org/crossword).
func ParseIPuz(data []byte) (*Puzzle, error) {
	var raw struct {
		Kind       []string                     `json:"kind"`
		Title      string                       `json:"title"`
		Author     string                       `json:"author"`
		Copyright  string                       `json:"copyright"`
		Notes      string                       `json:"notes"`
		Intro      string                       `json:"intro"`
		Block      *string                      `json:"block"`
		Dimensions struct{ Width, Height int }  `json:"dimensions"`
		Puzzle     [][]json.RawMessage          `json:"puzzle"`
		Solution   [][]json.RawMessage          `json:"solution"`
		Clues      map[string][]json.RawMessage `json:"clues"`
		Arrows     bool                         `json:"appload:arrows"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("not an .ipuz file: %v", err)
	}
	crossword := false
	for _, k := range raw.Kind {
		crossword = crossword || strings.Contains(k, "ipuz.org/crossword")
	}
	if !crossword {
		return nil, errors.New("the .ipuz file isn't a crossword")
	}
	block := "#"
	if raw.Block != nil {
		block = *raw.Block
	}
	w, h := raw.Dimensions.Width, raw.Dimensions.Height
	p := &Puzzle{
		Title: raw.Title, Author: raw.Author, Copyright: raw.Copyright,
		Notes: raw.Notes, Width: w, Height: h, Cells: make([]Cell, w*h), Arrows: raw.Arrows,
	}
	if p.Notes == "" {
		p.Notes = raw.Intro
	}
	if w <= 0 || h <= 0 || len(raw.Puzzle) < h {
		return nil, errors.New("the .ipuz grid has a strange size")
	}
	for r := range h {
		for c := range w {
			cell := &p.Cells[r*w+c]
			if c < len(raw.Puzzle[r]) {
				blocked, circled := ipuzCell(raw.Puzzle[r][c], block)
				cell.Block, cell.Circled = blocked, circled
			}
			if r < len(raw.Solution) && c < len(raw.Solution[r]) && !cell.Block {
				cell.Solution = ipuzValue(raw.Solution[r][c])
				p.HasSolution = p.HasSolution || cell.Solution != ""
			}
		}
	}
	for direction, list := range raw.Clues {
		var clues *[]Clue
		switch strings.ToLower(strings.SplitN(direction, ":", 2)[0]) {
		case "across":
			clues = &p.Across
		case "down":
			clues = &p.Down
		default:
			continue
		}
		for _, item := range list {
			if c, ok := ipuzClue(item, w, h); ok {
				*clues = append(*clues, c)
			}
		}
	}
	return p, p.validate()
}

// ipuzCell reads a LabeledCell: a number, "#", null, or {"cell": ..., "style": ...}.
func ipuzCell(raw json.RawMessage, block string) (blocked, circled bool) {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s == block, false
	}
	if string(raw) == "null" {
		return true, false // omitted cells are not part of the grid
	}
	var obj struct {
		Cell  json.RawMessage `json:"cell"`
		Style struct {
			Shapebg string `json:"shapebg"`
		} `json:"style"`
	}
	if json.Unmarshal(raw, &obj) == nil && obj.Cell != nil {
		b, _ := ipuzCell(obj.Cell, block)
		return b, obj.Style.Shapebg == "circle"
	}
	return false, false
}

func ipuzValue(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if s == "#" {
			return ""
		}
		return s
	}
	var obj struct {
		Value string `json:"value"`
	}
	if json.Unmarshal(raw, &obj) == nil {
		return obj.Value
	}
	return ""
}

// ipuzClue reads [number, "clue"], {"number": n, "clue": "...", "cells": ...} or "clue".
func ipuzClue(raw json.RawMessage, w, h int) (Clue, bool) {
	var pair []json.RawMessage
	if json.Unmarshal(raw, &pair) == nil && len(pair) >= 2 {
		var text string
		json.Unmarshal(pair[1], &text)
		return Clue{Number: ipuzNumber(pair[0]), Text: text}, true
	}
	var obj struct {
		Number json.RawMessage `json:"number"`
		Clue   string          `json:"clue"`
		Cells  [][2]int        `json:"cells"` // [column, row], from 1
		At     *[2]int         `json:"appload:at"`
	}
	if json.Unmarshal(raw, &obj) == nil && obj.Clue != "" {
		index := func(cr [2]int) int {
			if cr[0] < 1 || cr[1] < 1 || cr[0] > w || cr[1] > h {
				return -1
			}
			return (cr[1]-1)*w + cr[0] - 1
		}
		c := Clue{Number: ipuzNumber(obj.Number), Text: obj.Clue}
		for _, cr := range obj.Cells {
			c.Cells = append(c.Cells, index(cr))
		}
		if obj.At != nil {
			c.At = index(*obj.At)
		}
		return c, true
	}
	return Clue{}, false
}

func ipuzNumber(raw json.RawMessage) int {
	var n int
	if json.Unmarshal(raw, &n) == nil {
		return n
	}
	var s string
	json.Unmarshal(raw, &s)
	n, _ = strconv.Atoi(strings.TrimSpace(s))
	return n
}

// IPuz writes the puzzle as .ipuz, the open format other apps read too.
func (p *Puzzle) IPuz() ([]byte, error) {
	grid := make([][]any, p.Height)
	solution := make([][]any, p.Height)
	for r := range p.Height {
		grid[r] = make([]any, p.Width)
		solution[r] = make([]any, p.Width)
		for c := range p.Width {
			cell := p.Cells[r*p.Width+c]
			var label any = cell.Number
			if cell.Block {
				label, solution[r][c] = "#", "#"
			} else {
				solution[r][c] = cell.Solution
			}
			if cell.Circled {
				label = map[string]any{"cell": label, "style": map[string]string{"shapebg": "circle"}}
			}
			grid[r][c] = label
		}
	}
	coords := func(i int) [2]int { return [2]int{i%p.Width + 1, i/p.Width + 1} }
	clues := func(list []Clue) []any {
		out := []any{}
		for _, c := range list {
			cells := [][2]int{}
			for _, i := range c.Cells {
				cells = append(cells, coords(i))
			}
			clue := map[string]any{"number": c.Number, "clue": c.Text, "cells": cells}
			if p.Arrows {
				clue["appload:at"] = coords(c.At)
			}
			out = append(out, clue)
		}
		return out
	}
	doc := map[string]any{
		"version":    "http://ipuz.org/v2",
		"kind":       []string{"http://ipuz.org/crossword#1"},
		"title":      p.Title,
		"author":     p.Author,
		"copyright":  p.Copyright,
		"notes":      p.Notes,
		"dimensions": map[string]int{"width": p.Width, "height": p.Height},
		"puzzle":     grid,
		"clues":      map[string]any{"Across": clues(p.Across), "Down": clues(p.Down)},
	}
	if p.HasSolution {
		doc["solution"] = solution
	}
	if p.Arrows {
		// Clues in the grid: not standard ipuz, other apps show them as a list
		doc["appload:arrows"] = true
	}
	return json.MarshalIndent(doc, "", " ")
}
