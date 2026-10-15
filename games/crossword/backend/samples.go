package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"appload-games/games/crossword/backend/hwr"
)

// Handwriting the recognizer got wrong, kept to train it on: a letter read
// as something other than the square's answer, then written again in the same
// square and read right. The first writing is very likely that letter too,
// just written in a way the model doesn't know (see hwrtrain).
type sampleLog struct {
	path  string
	count int
	last  *attempt
}

// At most this many samples are kept (each is a few KB).
const maxSamples = 2000

// A rewrite this soon after counts as correcting the first try.
const rewriteWindow = 2 * time.Minute

type attempt struct {
	cell    int
	strokes [][]float64
	guesses []hwr.Guess
	at      time.Time
}

type writing struct {
	Strokes [][]float64 `json:"strokes"`
	Guesses []hwr.Guess `json:"guesses"`
}

type sample struct {
	Letter  string  `json:"letter"`  // the square's answer
	Misread writing `json:"misread"` // read as something else
	Fixed   writing `json:"fixed"`   // written again, read right
	Puzzle  string  `json:"puzzle"`
	Cell    int     `json:"cell"`
	Time    int64   `json:"time"` // unix ms
}

func openSamples(dir string) *sampleLog {
	s := &sampleLog{path: filepath.Join(dir, "samples.jsonl")}
	if f, err := os.Open(s.path); err == nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for sc.Scan() {
			s.count++
		}
		f.Close()
	}
	return s
}

// written notes a recognized letter; truth is the square's answer ("" if the
// puzzle has none). It returns whether a sample was saved.
func (s *sampleLog) written(puzzleKey string, cell int, strokes [][]float64, guesses []hwr.Guess, truth string, now time.Time) (bool, error) {
	prev := s.last
	s.last = &attempt{cell: cell, strokes: strokes, guesses: guesses, at: now}
	if prev == nil || prev.cell != cell || now.Sub(prev.at) > rewriteWindow || truth == "" ||
		prev.guesses[0].Letter == truth || guesses[0].Letter != truth || s.count >= maxSamples {
		return false, nil
	}
	line, err := json.Marshal(sample{
		Letter:  truth,
		Misread: writing{prev.strokes, prev.guesses},
		Fixed:   writing{strokes, guesses},
		Puzzle:  puzzleKey,
		Cell:    cell,
		Time:    now.UnixMilli(),
	})
	if err != nil {
		return false, err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return false, err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return false, err
	}
	s.count++
	return true, nil
}

// reset forgets the last try (another puzzle was opened).
func (s *sampleLog) reset() { s.last = nil }
