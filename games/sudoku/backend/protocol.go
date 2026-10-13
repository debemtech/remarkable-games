package main

import "appload-games/games/sudoku/backend/sudoku"

// Message types, see ui/Protocol.js. Framework types (hello, toast) are below 100.
const (
	// Frontend -> backend
	msgNewGame  = 100 // newGameRequest
	msgContinue = 101
	msgSet      = 102 // cellRequest: put a digit (0 = erase)
	msgNote     = 103 // cellRequest: toggle a pencil mark
	msgUndo     = 104
	msgHint     = 105 // cellRequest, digit unused; cell -1 = any
	msgCheck    = 106
	msgRestart  = 107
	msgQuit     = 108

	// Backend -> frontend
	msgGameState = 200 // gameState
	msgMenu      = 201 // menuState
)

type newGameRequest struct {
	Difficulty string `json:"difficulty"`
}

type cellRequest struct {
	Cell  int  `json:"cell"`
	Digit int8 `json:"digit"`
}

type gameState struct {
	Active     bool        `json:"active"`
	Generating bool        `json:"generating"`
	Difficulty string      `json:"difficulty"`
	Values     sudoku.Grid `json:"values"`
	Given      []bool      `json:"given"`
	Notes      [81]uint16  `json:"notes"`
	Conflicts  []int       `json:"conflicts"`
	Wrong      []int       `json:"wrong"`
	Solved     bool        `json:"solved"`
	Seconds    int         `json:"seconds"`
	Hints      int         `json:"hints"`
	CanUndo    bool        `json:"canUndo"`
}

type menuState struct {
	CanContinue bool         `json:"canContinue"`
	Continue    string       `json:"continue"` // e.g. "Medium · 34 cells left · 12 min"
	Stats       []statsEntry `json:"stats"`
}

type statsEntry struct {
	Difficulty string `json:"difficulty"`
	Solved     int    `json:"solved"`
	Best       int    `json:"best"` // seconds, 0 = none yet
}
