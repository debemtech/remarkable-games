package main

import "github.com/notnil/chess"

// Message types, see ui/Protocol.js. Framework types (hello, toast) are below 100.
const (
	// Frontend -> backend
	msgNewGame     = 100 // newGameRequest
	msgMove        = 101 // moveRequest
	msgUndo        = 102
	msgQuitGame    = 103
	msgGetSlots    = 104
	msgListPGNs    = 110
	msgOpenPGN     = 111 // openPGNRequest
	msgStartReplay = 112 // startReplayRequest
	msgReplayNext  = 113
	msgReplayUndo  = 114
	msgReplayReset = 115
	msgReplayMove  = 116 // moveRequest
	msgReplayClose = 117

	// Backend -> frontend
	msgGameState   = 200 // gameState
	msgSlots       = 201 // slotsMessage
	msgPGNList     = 202 // pgnList
	msgGameList    = 203 // gameList
	msgReplayState = 204 // replayState
)

type newGameRequest struct {
	Mode   string `json:"mode"` // pvp, easy, normal, hard
	Slot   int    `json:"slot"`
	Reset  bool   `json:"reset"`  // start over instead of continuing the slot
	Rotate bool   `json:"rotate"` // PvP: turn the pieces around on black's turn
}

// Squares are numbered 0 (A1) to 63 (H8), like chess.Square.
type moveRequest struct {
	From      int    `json:"from"`
	To        int    `json:"to"`
	Promotion string `json:"promotion"` // q, r, b, n; empty means queen
}

type openPGNRequest struct {
	File string `json:"file"`
}

type startReplayRequest struct {
	Index int `json:"index"`
}

// boardView is everything the board component draws.
type boardView struct {
	Board    string        `json:"board"` // 64 FEN letters indexed by square, '.' = empty
	Turn     string        `json:"turn"`  // white, black
	InCheck  bool          `json:"inCheck"`
	Moves    map[int][]int `json:"moves"`    // legal moves the user may play: from -> to
	LastMove []int         `json:"lastMove"` // [from, to] or null
}

type gameState struct {
	Active   bool   `json:"active"`
	Mode     string `json:"mode"`
	Slot     int    `json:"slot"`
	Rotated  bool   `json:"rotated"`
	Status   string `json:"status"`
	GameOver bool   `json:"gameOver"`
	Busy     bool   `json:"busy"` // the bot is thinking
	boardView
}

type slotInfo struct {
	Slot        int    `json:"slot"`
	Saved       bool   `json:"saved"`
	Description string `json:"description"`
}

type slotsMessage struct {
	Slots []slotInfo `json:"slots"`
}

type pgnList struct {
	Dir   string   `json:"dir"`
	Files []string `json:"files"`
}

type gameList struct {
	File  string   `json:"file"`
	Games []string `json:"games"`
	Error string   `json:"error,omitempty"`
}

type replayState struct {
	Active      bool   `json:"active"`
	Title       string `json:"title"`
	Comment     string `json:"comment"`
	Played      int    `json:"played"`
	Total       int    `json:"total"`
	ManualMoves int    `json:"manualMoves"`
	boardView
}

var pieceLetters = map[chess.Piece]byte{
	chess.WhiteKing: 'K', chess.WhiteQueen: 'Q', chess.WhiteRook: 'R',
	chess.WhiteBishop: 'B', chess.WhiteKnight: 'N', chess.WhitePawn: 'P',
	chess.BlackKing: 'k', chess.BlackQueen: 'q', chess.BlackRook: 'r',
	chess.BlackBishop: 'b', chess.BlackKnight: 'n', chess.BlackPawn: 'p',
}

// newBoardView describes pos. The legal moves are only included if movable.
func newBoardView(pos *chess.Position, last *chess.Move, movable bool) boardView {
	board := pos.Board()
	letters := make([]byte, 64)
	for sq := range 64 {
		if l, ok := pieceLetters[board.Piece(chess.Square(sq))]; ok {
			letters[sq] = l
		} else {
			letters[sq] = '.'
		}
	}
	v := boardView{
		Board:   string(letters),
		Turn:    colorName(pos.Turn()),
		InCheck: last != nil && last.HasTag(chess.Check),
		Moves:   map[int][]int{},
	}
	if last != nil {
		v.LastMove = []int{int(last.S1()), int(last.S2())}
	}
	if movable {
		for _, m := range pos.ValidMoves() {
			from, to := int(m.S1()), int(m.S2())
			if m.Promo() != chess.NoPieceType && m.Promo() != chess.Queen {
				continue // one entry per promotion square is enough
			}
			v.Moves[from] = append(v.Moves[from], to)
		}
	}
	return v
}

func colorName(c chess.Color) string {
	if c == chess.Black {
		return "black"
	}
	return "white"
}

// findMove returns the legal move matching a request.
func findMove(pos *chess.Position, req moveRequest) *chess.Move {
	promo := map[string]chess.PieceType{"r": chess.Rook, "b": chess.Bishop, "n": chess.Knight}[req.Promotion]
	if promo == chess.NoPieceType {
		promo = chess.Queen
	}
	for _, m := range pos.ValidMoves() {
		if int(m.S1()) == req.From && int(m.S2()) == req.To &&
			(m.Promo() == chess.NoPieceType || m.Promo() == promo) {
			return m
		}
	}
	return nil
}
