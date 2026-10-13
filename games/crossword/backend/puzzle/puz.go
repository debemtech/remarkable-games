package puzzle

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
	"unicode/utf8"
)

// ParsePuz reads an Across Lite .puz file.
func ParsePuz(data []byte) (*Puzzle, error) {
	if len(data) < 0x34 || string(data[2:14]) != "ACROSS&DOWN\x00" {
		return nil, errors.New("not a .puz file")
	}
	version := string(bytes.TrimRight(data[0x18:0x1C], "\x00"))
	w, h := int(data[0x2C]), int(data[0x2D])
	nclues := int(binary.LittleEndian.Uint16(data[0x2E:]))
	scrambled := binary.LittleEndian.Uint16(data[0x32:]) != 0
	n := w * h
	if len(data) < 0x34+2*n {
		return nil, errors.New("the .puz file is cut off")
	}
	solution := data[0x34 : 0x34+n]
	rest := data[0x34+2*n:]

	// Strings are NUL-terminated: title, author, copyright, clues, notes.
	// Version 2 files use UTF-8, older ones ISO-8859-1.
	utf := strings.HasPrefix(version, "2")
	next := func() string {
		i := bytes.IndexByte(rest, 0)
		if i < 0 {
			i = len(rest)
		}
		s := rest[:i]
		rest = rest[min(i+1, len(rest)):]
		if utf && utf8.Valid(s) {
			return string(s)
		}
		r := make([]rune, len(s))
		for k, b := range s {
			r[k] = rune(b)
		}
		return string(r)
	}
	p := &Puzzle{Width: w, Height: h, Cells: make([]Cell, n), HasSolution: !scrambled}
	p.Title, p.Author, p.Copyright = next(), next(), next()
	clues := make([]string, nclues)
	for i := range clues {
		clues[i] = next()
	}
	p.Notes = next()

	for i, b := range solution {
		if b == '.' || b == ':' {
			p.Cells[i].Block = true
		} else {
			p.Cells[i].Solution = string(rune(b))
		}
	}
	readExtensions(p, rest)

	// Clues come sorted by number, across before down for the same number.
	across, down := p.number()
	k := 0
	for i, c := range p.Cells {
		if c.Number == 0 {
			continue
		}
		if word, ok := across[i]; ok && k < len(clues) {
			p.Across = append(p.Across, Clue{Number: c.Number, Text: clues[k], Cells: word})
			k++
		}
		if word, ok := down[i]; ok && k < len(clues) {
			p.Down = append(p.Down, Clue{Number: c.Number, Text: clues[k], Cells: word})
			k++
		}
	}
	return p, p.validate()
}

// readExtensions applies circles (GEXT) and rebus answers (GRBS + RTBL).
func readExtensions(p *Puzzle, data []byte) {
	sections := map[string][]byte{}
	for len(data) >= 8 {
		name := string(data[:4])
		length := int(binary.LittleEndian.Uint16(data[4:]))
		if len(data) < 8+length {
			break
		}
		sections[name] = data[8 : 8+length]
		data = data[8+length:]
		if len(data) > 0 && data[0] == 0 {
			data = data[1:]
		}
	}
	if gext := sections["GEXT"]; len(gext) == len(p.Cells) {
		for i, b := range gext {
			p.Cells[i].Circled = b&0x80 != 0
		}
	}
	grbs, rtbl := sections["GRBS"], sections["RTBL"]
	if len(grbs) != len(p.Cells) || rtbl == nil {
		return
	}
	table := map[int]string{}
	for _, entry := range strings.Split(string(rtbl), ";") {
		key, value, ok := strings.Cut(entry, ":")
		if !ok {
			continue
		}
		var k int
		for _, d := range strings.TrimSpace(key) {
			k = k*10 + int(d-'0')
		}
		table[k] = value
	}
	for i, b := range grbs {
		if b > 0 {
			if v, ok := table[int(b)-1]; ok {
				p.Cells[i].Solution = v
			}
		}
	}
}
