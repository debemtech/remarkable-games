package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"appload-games/framework/appload"

	"github.com/notnil/chess"
)

// listPGNs returns the *.pgn files in dir, sorted by name.
func listPGNs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".pgn") {
			files = append(files, e.Name())
		}
	}
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i]) < strings.ToLower(files[j]) })
	return files
}

// readPGN parses all games of a PGN file in dir. Games that fail to parse are
// skipped, as long as at least one works.
func readPGN(dir, name string) ([]*chess.Game, error) {
	if name != filepath.Base(name) || strings.HasPrefix(name, ".") {
		return nil, errors.New("invalid file name")
	}
	f, err := os.Open(filepath.Join(dir, name))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, errors.New("the file doesn't exist (anymore)")
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var games []*chess.Game
	var firstErr error
	scanner := chess.NewScanner(f)
	// After a broken game the scanner can continue with the next one. Several
	// failures in a row mean it is stuck at the end of the file.
	for failures := 0; failures < 3; {
		if scanner.Scan() {
			failures = 0
			if g := scanner.Next(); len(g.Moves()) > 0 || len(g.TagPairs()) > 0 {
				games = append(games, g)
			}
			continue
		}
		err := scanner.Err()
		if err == nil || errors.Is(err, io.EOF) {
			break
		}
		failures++
		if firstErr == nil {
			firstErr = err
		}
	}
	if len(games) == 0 {
		if firstErr == nil {
			firstErr = errors.New("no games found")
		}
		return nil, fmt.Errorf("couldn't parse the PGN: %v", firstErr)
	}
	return games, nil
}

// describeGame is "White vs Black at Event Round".
func describeGame(g *chess.Game) string {
	tag := func(key string) string {
		if t := g.GetTagPair(key); t != nil && t.Value != "" {
			return t.Value
		}
		return "N/A"
	}
	return fmt.Sprintf("%s vs %s at %s %s", tag("White"), tag("Black"), tag("Event"), tag("Round"))
}

// replay steps through a game from a PGN. The user can try own moves on top,
// which have to be undone before the replay continues.
type replay struct {
	title     string
	positions []*chess.Position // positions[i] is the position after i moves
	moves     []*chess.Move
	comments  [][]string // comments[i] belongs to moves[i]
	result    string
	played    int
	own       []*chess.Move
	ownPos    []*chess.Position
	note      string
}

func newReplay(g *chess.Game) *replay {
	r := &replay{
		title:     describeGame(g),
		positions: g.Positions(),
		moves:     g.Moves(),
		comments:  g.Comments(),
	}
	switch g.Outcome() {
	case chess.WhiteWon:
		r.result = "Game over: White won"
	case chess.BlackWon:
		r.result = "Game over: Black won"
	case chess.Draw:
		r.result = "Game over: Draw"
	default:
		r.result = "Game over"
	}
	return r
}

func (r *replay) pos() *chess.Position {
	if n := len(r.ownPos); n > 0 {
		return r.ownPos[n-1]
	}
	return r.positions[r.played]
}

func (r *replay) lastMove() *chess.Move {
	if n := len(r.own); n > 0 {
		return r.own[n-1]
	}
	if r.played > 0 {
		return r.moves[r.played-1]
	}
	return nil
}

func (r *replay) next() {
	switch {
	case len(r.own) > 0:
		r.note = "Undo your own moves before continuing the replay."
	case r.played < len(r.moves):
		r.played++
		r.note = ""
		if i := r.played - 1; i < len(r.comments) {
			r.note = strings.TrimSpace(strings.Join(r.comments[i], " "))
		}
		if r.played == len(r.moves) {
			r.note = strings.TrimSpace(r.note + " " + r.result)
		}
	default:
		r.note = r.result
	}
}

func (r *replay) undo() {
	r.note = ""
	if n := len(r.own); n > 0 {
		r.own, r.ownPos = r.own[:n-1], r.ownPos[:n-1]
	} else if r.played > 0 {
		r.played--
	}
}

func (r *replay) reset() {
	r.played, r.own, r.ownPos, r.note = 0, nil, nil, ""
}

func (r *replay) move(req moveRequest) error {
	mv := findMove(r.pos(), req)
	if mv == nil {
		return appload.Info("That move isn't possible.")
	}
	r.own = append(r.own, mv)
	r.ownPos = append(r.ownPos, r.pos().Update(mv))
	r.note = ""
	return nil
}

func (r *replay) state() replayState {
	return replayState{
		Active:      true,
		Title:       r.title,
		Comment:     r.note,
		Played:      r.played,
		Total:       len(r.moves),
		ManualMoves: len(r.own),
		boardView:   newBoardView(r.pos(), r.lastMove(), true),
	}
}
