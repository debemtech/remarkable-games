package main

import "appload-games/games/solitaire/backend/solitaire"

// Message types, see ui/Protocol.js. Framework types (hello, toast, prefs) are below 100.
const (
	// Frontend -> backend
	msgNewGame  = 100 // newGameRequest
	msgContinue = 101
	msgMove     = 102 // moveRequest
	msgAuto     = 103 // moveRequest without "to": move to the best place
	msgStock    = 104 // draw from the stock (Klondike)
	msgUndo     = 105
	msgHint     = 106
	msgRestart  = 107 // the same deal again
	msgQuit     = 108

	// Backend -> frontend
	msgGameState = 200 // gameState
	msgMenu      = 201 // menuState
)

type newGameRequest struct {
	Variant string `json:"variant"`
}

// Piles are indexed as in gameState.Piles; index is the first card that moves.
type moveRequest struct {
	From  int `json:"from"`
	Index int `json:"index"`
	To    int `json:"to"`
}

type gameState struct {
	Active  bool             `json:"active"`
	Variant string           `json:"variant"`
	Title   string           `json:"title"`
	Piles   []solitaire.Pile `json:"piles"`
	Solved  bool             `json:"solved"`
	Seconds int              `json:"seconds"`
	Moves   int              `json:"moves"`
	CanUndo bool             `json:"canUndo"`
	Hint    *solitaire.Hint  `json:"hint"` // from -1: draw from the stock
}

type menuState struct {
	CanContinue bool          `json:"canContinue"`
	Continue    string        `json:"continue"`
	Variants    []variantInfo `json:"variants"`
}

type variantInfo struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Played int    `json:"played"`
	Won    int    `json:"won"`
	Best   int    `json:"best"`
}
