package main

import "appload-games/framework/cards"

// Message types, see ui/Protocol.js. Framework types (hello, toast) are below 100.
const (
	// Frontend -> backend
	msgBet       = 100 // betRequest: add a chip, 0 clears the bet
	msgDeal      = 101
	msgHit       = 102
	msgStand     = 103
	msgDouble    = 104
	msgSplit     = 105
	msgSurrender = 106
	msgHint      = 107
	msgRestart   = 108 // new bankroll when broke

	// Backend -> frontend
	msgState = 200 // tableState
)

type betRequest struct {
	Amount int `json:"amount"`
}

type handView struct {
	Cards  []cards.Card `json:"cards"`
	Bet    int          `json:"bet"`
	Total  int          `json:"total"`
	Soft   bool         `json:"soft"`
	Result string       `json:"result"` // empty until the round is over
	Net    int          `json:"net"`
}

type tableState struct {
	Phase       string       `json:"phase"` // betting, playing, dealer (revealing its cards), done
	Bankroll    int          `json:"bankroll"`
	Bet         int          `json:"bet"`
	MinBet      int          `json:"minBet"`
	MaxBet      int          `json:"maxBet"`
	Chips       []int        `json:"chips"`
	Dealer      []cards.Card `json:"dealer"` // face up cards
	Hidden      int          `json:"hidden"` // face down cards after those
	DealerTotal int          `json:"dealerTotal"`
	Hands       []handView   `json:"hands"`
	Active      int          `json:"active"` // hand being played, -1 if none
	CanHit      bool         `json:"canHit"`
	CanDouble   bool         `json:"canDouble"`
	CanSplit    bool         `json:"canSplit"`
	CanSurrend  bool         `json:"canSurrender"`
	Broke       bool         `json:"broke"`
	Hint        string       `json:"hint"`
	Message     string       `json:"message"`
	Stats       tableStats   `json:"stats"`
}
