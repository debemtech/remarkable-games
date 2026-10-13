package sources

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/dop251/goja"

	"appload-games/games/crossword/backend/puzzle"
)

// Cruzadas Clube (cruzadasclube.com.br) shows Crossword Compiler puzzles. The
// puzzle comes as a script that builds the XML (obfuscated), so it's run in a
// JavaScript interpreter without access to anything. Only puzzles whose page
// is free to play are offered; older ones are for subscribers.

var (
	clubeLink   = regexp.MustCompile(`href="/jogo/i/id/(\d+)/cruzadas-diretas-(\d+)/"`)
	clubeScript = regexp.MustCompile(`<script src="/jogos/([A-Za-z0-9_]+)\.js"`)
)

func (c *Client) clubeList(ctx context.Context) ([]Entry, error) {
	page, err := c.get(ctx, "https://cruzadasclube.com.br/jogo/categoria/id/4/n/cruzadas-diretas")
	if err != nil {
		return nil, err
	}
	var out []Entry
	seen := map[string]bool{}
	for _, m := range clubeLink.FindAllSubmatch(page, -1) {
		id := string(m[1]) + "-" + string(m[2])
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, Entry{ID: id, Title: "Cruzadas diretas " + string(m[2]), Size: "14×9"})
		if len(out) == 40 { // the newest are free, older ones mostly for subscribers
			break
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no puzzles listed")
	}
	return out, nil
}

var clubeID = regexp.MustCompile(`^(\d+)-(\d+)$`)

func (c *Client) clubeFetch(ctx context.Context, id string) (*puzzle.Puzzle, error) {
	m := clubeID.FindStringSubmatch(id)
	if m == nil {
		return nil, errors.New("invalid puzzle id")
	}
	page, err := c.get(ctx, fmt.Sprintf("https://cruzadasclube.com.br/jogo/i/id/%s/cruzadas-diretas-%s/", m[1], m[2]))
	if err != nil {
		return nil, err
	}
	sm := clubeScript.FindSubmatch(page)
	if sm == nil {
		return nil, errors.New("this puzzle is for subscribers of the site")
	}
	script, err := c.get(ctx, "https://cruzadasclube.com.br/jogos/"+string(sm[1])+".js")
	if err != nil {
		return nil, err
	}
	xml, err := runScript(string(script), "CrosswordPuzzleData")
	if err != nil {
		return nil, err
	}
	return puzzle.FromCompiler([]byte(xml))
}

// runScript runs JavaScript in an empty interpreter, for at most a second,
// and returns the string in a global variable.
func runScript(src, variable string) (string, error) {
	vm := goja.New()
	timer := time.AfterFunc(time.Second, func() { vm.Interrupt("too slow") })
	defer timer.Stop()
	if _, err := vm.RunString(src); err != nil {
		return "", fmt.Errorf("couldn't read the puzzle: %v", err)
	}
	v := vm.Get(variable)
	if v == nil || goja.IsUndefined(v) {
		return "", errors.New("couldn't read the puzzle")
	}
	return v.String(), nil
}
