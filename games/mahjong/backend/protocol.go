package main

// Message types, see ui/Protocol.js. Framework types (hello, toast) are below 100.
const (
	// Frontend -> backend
	msgNewGame  = 100 // newGameRequest
	msgContinue = 101
	msgRemove   = 102 // removeRequest
	msgUndo     = 103
	msgHint     = 104
	msgShuffle  = 105
	msgQuit     = 106

	// Backend -> frontend
	msgGameState = 200 // gameState
	msgMenu      = 201 // menuState
)

type newGameRequest struct {
	Layout string `json:"layout"`
}

type removeRequest struct {
	A int `json:"a"`
	B int `json:"b"`
}

// gameState describes the board. Tiles are indexed in drawing order; their
// positions are in half-tile units, see board.Pos.
type gameState struct {
	Active    bool   `json:"active"`
	Layout    string `json:"layout"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Levels    int    `json:"levels"`
	X         []int  `json:"x"`
	Y         []int  `json:"y"`
	Z         []int  `json:"z"`
	Faces     []int8 `json:"faces"`
	Present   []bool `json:"present"`
	Free      []bool `json:"free"`
	Hint      []int  `json:"hint"`
	Remaining int    `json:"remaining"`
	Moves     int    `json:"moves"` // matching free pairs
	Solved    bool   `json:"solved"`
	Seconds   int    `json:"seconds"`
	Hints     int    `json:"hints"`
	Shuffles  int    `json:"shuffles"`
	CanUndo   bool   `json:"canUndo"`
}

type menuState struct {
	CanContinue bool         `json:"canContinue"`
	Continue    string       `json:"continue"`
	Layouts     []layoutInfo `json:"layouts"`
}

type layoutInfo struct {
	Name   string `json:"name"`
	Title  string `json:"title"`
	Tiles  int    `json:"tiles"`
	Solved int    `json:"solved"`
	Best   int    `json:"best"`
}
