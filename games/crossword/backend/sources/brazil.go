package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"appload-games/games/crossword/backend/puzzle"
)

// ValorFinal (valorfinal.com.br) publishes its daily crosswords as JSON:
// an index of all puzzles and one file per kind and month.

var brasilia = time.FixedZone("BRT", -3*3600)

func (c *Client) valorfinalList(ctx context.Context, s Source) ([]Entry, error) {
	data, err := c.get(ctx, "https://valorfinal.com.br/data/cruzadas/indice.json")
	if err != nil {
		return nil, err
	}
	var index []struct {
		Date  string `json:"d"`
		Kind  string `json:"t"`
		ID    string `json:"id"`
		Title string `json:"ti"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}
	today := time.Now().In(brasilia).Format("2006-01-02")
	var out []Entry
	for _, e := range index {
		if e.Kind != s.Set || e.Date > today || !safeID.MatchString(e.ID) {
			continue // the index has the puzzles of the coming months too
		}
		date, _ := time.ParseInLocation("2006-01-02", e.Date, brasilia)
		out = append(out, Entry{ID: e.ID, Title: e.Title + " · " + date.Format("02/01/2006"), Date: date.UnixMilli(), Size: valorfinalSize[s.Set]})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	if len(out) > 60 {
		out = out[:60]
	}
	if len(out) == 0 {
		return nil, errors.New("no puzzles listed")
	}
	return out, nil
}

var valorfinalSize = map[string]string{"mini": "5×5", "classica": "11×11", "grande": "13×13"}

var valorfinalID = regexp.MustCompile(`^([a-z]+)-(\d{4}-\d{2})-\d{2}$`)

func (c *Client) valorfinalFetch(ctx context.Context, id string) (*puzzle.Puzzle, error) {
	m := valorfinalID.FindStringSubmatch(id)
	if m == nil {
		return nil, errors.New("invalid puzzle id")
	}
	data, err := c.get(ctx, "https://valorfinal.com.br/data/cruzadas/"+m[1]+"/"+m[2]+".json")
	if err != nil {
		return nil, err
	}
	var month map[string]struct {
		ID       string   `json:"id"`
		Title    string   `json:"titulo"`
		Rows     []string `json:"linhas"`
		Palavras []struct {
			Direction string `json:"direcao"`
			Row       int    `json:"linha"`
			Col       int    `json:"coluna"`
			Clue      string `json:"dica"`
		} `json:"palavras"`
	}
	if err := json.Unmarshal(data, &month); err != nil {
		return nil, err
	}
	for _, pz := range month {
		if pz.ID != id {
			continue
		}
		var words []puzzle.Word
		for _, w := range pz.Palavras {
			words = append(words, puzzle.Word{Row: w.Row, Col: w.Col, Across: w.Direction == "h", Clue: w.Clue})
		}
		return puzzle.FromRows(pz.Title, "ValorFinal", pz.Rows, words)
	}
	return nil, errors.New("the puzzle isn't there anymore")
}

// Racha Cuca (rachacuca.com.br) has an archive of numbered crosswords, each
// page with the grid and clues as JavaScript variables.

var (
	rachacucaLink  = regexp.MustCompile(`href="/palavras/palavras-cruzadas/(\d+)/"`)
	rachacucaBoard = regexp.MustCompile(`(?s)var BOARD = (\{.*?\});\s*\n`)
	rachacucaClues = regexp.MustCompile(`(?s)var CLUES = (\{.*?\});\s*\n`)
)

func (c *Client) rachacucaList(ctx context.Context) ([]Entry, error) {
	page, err := c.get(ctx, "https://rachacuca.com.br/palavras/palavras-cruzadas/")
	if err != nil {
		return nil, err
	}
	seen := map[int]bool{}
	for _, m := range rachacucaLink.FindAllSubmatch(page, -1) {
		n, _ := strconv.Atoi(string(m[1]))
		seen[n] = true
	}
	var numbers []int
	for n := range seen {
		numbers = append(numbers, n)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(numbers)))
	var out []Entry
	for _, n := range numbers {
		out = append(out, Entry{ID: strconv.Itoa(n), Title: fmt.Sprintf("Palavra Cruzada #%d", n), Size: "13×13"})
	}
	if len(out) == 0 {
		return nil, errors.New("no puzzles listed")
	}
	return out, nil
}

func (c *Client) rachacucaFetch(ctx context.Context, id string) (*puzzle.Puzzle, error) {
	if _, err := strconv.Atoi(id); err != nil {
		return nil, errors.New("invalid puzzle id")
	}
	page, err := c.get(ctx, "https://rachacuca.com.br/palavras/palavras-cruzadas/"+id+"/")
	if err != nil {
		return nil, err
	}
	bm, cm := rachacucaBoard.FindSubmatch(page), rachacucaClues.FindSubmatch(page)
	if bm == nil || cm == nil {
		return nil, errors.New("couldn't read the puzzle")
	}
	var board struct {
		Grid [][]struct {
			IsBlack bool `json:"is_black"`
		} `json:"grid"`
		Groups map[string]map[string][][2]int `json:"groups"` // "a"/"d" -> number -> [x, y] cells
	}
	var clues map[string]map[string]struct {
		Text string `json:"text"`
		Word string `json:"word"`
	}
	if err := json.Unmarshal(bm[1], &board); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(cm[1], &clues); err != nil {
		return nil, err
	}
	rows := make([][]rune, len(board.Grid))
	for y, row := range board.Grid {
		rows[y] = make([]rune, len(row))
		for x, cell := range row {
			rows[y][x] = '?'
			if cell.IsBlack {
				rows[y][x] = '#'
			}
		}
	}
	var words []puzzle.Word
	for dir, name := range map[string]string{"a": "across", "d": "down"} {
		for num, cells := range board.Groups[dir] {
			clue, ok := clues[name][num]
			letters := []rune(strings.ToUpper(clue.Word))
			if !ok || len(cells) == 0 || len(letters) != len(cells) {
				return nil, errors.New("the puzzle's answers don't fit its grid")
			}
			for i, xy := range cells {
				x, y := xy[0], xy[1]
				if y < 0 || y >= len(rows) || x < 0 || x >= len(rows[y]) {
					return nil, errors.New("the puzzle's answers don't fit its grid")
				}
				rows[y][x] = letters[i]
			}
			words = append(words, puzzle.Word{Row: cells[0][1], Col: cells[0][0], Across: dir == "a", Clue: clue.Text})
		}
	}
	lines := make([]string, len(rows))
	for i, r := range rows {
		lines[i] = string(r)
	}
	return puzzle.FromRows("Palavra Cruzada #"+id, "Racha Cuca", lines, words)
}
