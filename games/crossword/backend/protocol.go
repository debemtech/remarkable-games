package main

import (
	"appload-games/games/crossword/backend/hwr"
	"appload-games/games/crossword/backend/puzzle"
	"appload-games/games/crossword/backend/sources"
)

// Message types, see ui/Protocol.js. Framework types (hello, toast, prefs) are below 100.
const (
	// Frontend -> backend
	msgLibrary    = 100
	msgOpen       = 101 // keyRequest
	msgSources    = 102
	msgListFrom   = 103 // sourceRequest
	msgDownload   = 104 // sourceRequest with id
	msgSet        = 105 // setRequest
	msgStrokes    = 106 // strokesRequest
	msgCheck      = 107
	msgReveal     = 108 // setRequest (cell)
	msgClose      = 109
	msgDelete     = 110 // keyRequest
	msgClearWrong = 111
	msgUndo       = 112

	// Backend -> frontend
	msgPuzzle   = 200 // puzzleView: sent once per opened puzzle
	msgProgress = 201 // progressView: after every change
	msgLibList  = 202 // libraryView
	msgSrcList  = 203 // sourcesView
	msgEntries  = 204 // entriesView
	msgMenu     = 205 // menuView
)

type keyRequest struct {
	Key string `json:"key"`
}

type sourceRequest struct {
	Source string `json:"source"`
	ID     string `json:"id"`
}

type setRequest struct {
	Cell   int    `json:"cell"`
	Letter string `json:"letter"` // "" clears the cell
	// Join makes this change part of the previous undo step (one eraser swipe)
	Join bool `json:"join,omitempty"`
}

// Strokes in any coordinates, each a flat list x0, y0, x1, y1, ...
type strokesRequest struct {
	Cell    int         `json:"cell"`
	Strokes [][]float64 `json:"strokes"`
}

type cellView struct {
	Block    bool `json:"b,omitempty"`
	Number   int  `json:"n,omitempty"`
	Circled  bool `json:"c,omitempty"`
	HasClues bool `json:"k,omitempty"` // clues are written in it (Arrows)
}

type puzzleView struct {
	Active      bool          `json:"active"`
	Key         string        `json:"key"`
	Title       string        `json:"title"`
	Author      string        `json:"author"`
	Copyright   string        `json:"copyright"`
	Notes       string        `json:"notes"`
	Width       int           `json:"width"`
	Height      int           `json:"height"`
	Cells       []cellView    `json:"cells"`
	Across      []puzzle.Clue `json:"across"`
	Down        []puzzle.Clue `json:"down"`
	HasSolution bool          `json:"hasSolution"`
	Arrows      bool          `json:"arrows"` // clues in the grid, see puzzle.Puzzle
}

type progressView struct {
	Entries  []string   `json:"entries"`
	Wrong    []int      `json:"wrong"`
	Revealed []int      `json:"revealed"`
	Filled   int        `json:"filled"`
	Total    int        `json:"total"`
	Solved   bool       `json:"solved"`
	Seconds  int        `json:"seconds"`
	Guesses  *guessView `json:"guesses"` // the last recognized letter
}

type guessView struct {
	Cell    int         `json:"cell"`
	Letters []hwr.Guess `json:"letters"`
}

type libraryItem struct {
	Key     string `json:"key"`
	Title   string `json:"title"`
	Origin  string `json:"origin"`
	Size    string `json:"size"`
	Percent int    `json:"percent"`
	Solved  bool   `json:"solved"`
}

type libraryView struct {
	Items []libraryItem `json:"items"`
	Dir   string        `json:"dir"`
}

type sourcesView struct {
	Sources []sources.Source `json:"sources"`
}

type entryView struct {
	sources.Entry
	Downloaded bool `json:"downloaded"`
}

type entriesView struct {
	Source  string      `json:"source"`
	Name    string      `json:"name"`
	Entries []entryView `json:"entries"`
	Error   string      `json:"error,omitempty"`
}

type menuView struct {
	ContinueKey   string `json:"continueKey"`
	ContinueTitle string `json:"continueTitle"`
}
