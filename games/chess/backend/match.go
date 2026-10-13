package main

import (
	"fmt"

	"appload-games/framework/appload"

	"github.com/notnil/chess"
)

// mode is how a match is played. Bots always play black.
type mode string

const (
	pvp    mode = "pvp"
	easy   mode = "easy"
	normal mode = "normal"
	hard   mode = "hard"
)

func (m mode) valid() bool { return m == pvp || m == easy || m == normal || m == hard }

// match is a game in progress. It only holds rules and state; talking to the
// frontend and the engine is done by chessApp.
type match struct {
	mode   mode
	slot   int
	rotate bool
	save   savedGame // starting position and moves, as stored in the slot
	game   *chess.Game
	// The bot is searching. gen changes whenever the position changes, so a
	// result for an outdated position can be recognised and dropped.
	thinking bool
	gen      int
	// The engine was already used in this match (else it gets "ucinewgame").
	engineUsed bool
}

func newMatch(mode mode, slot int, rotate bool, save savedGame) (*match, error) {
	game, err := save.replay()
	if err != nil {
		return nil, err
	}
	return &match{mode: mode, slot: slot, rotate: rotate, save: save, game: game}, nil
}

func (m *match) pos() *chess.Position { return m.game.Position() }
func (m *match) over() bool           { return m.game.Outcome() != chess.NoOutcome }
func (m *match) human(c chess.Color) bool {
	return m.mode == pvp || c == chess.White
}
func (m *match) botToMove() bool { return !m.over() && !m.human(m.pos().Turn()) }

func (m *match) lastMove() *chess.Move {
	moves := m.game.Moves()
	if len(moves) == 0 {
		return nil
	}
	return moves[len(moves)-1]
}

// play applies a legal move and records it in the savegame.
func (m *match) play(mv *chess.Move) error {
	if err := m.game.Move(mv); err != nil {
		return err
	}
	m.save.Moves = append(m.save.Moves, chess.UCINotation{}.Encode(nil, mv))
	m.gen++
	return nil
}

// move plays a move requested by the user.
func (m *match) move(req moveRequest) error {
	switch {
	case m.over():
		return appload.Info("The game is over.")
	case !m.human(m.pos().Turn()):
		return appload.Info("Please wait for your turn.")
	}
	mv := findMove(m.pos(), req)
	if mv == nil {
		return appload.Info("That move isn't possible.")
	}
	return m.play(mv)
}

// botMove plays the engine's answer, given in UCI notation.
func (m *match) botMove(uciMove string) error {
	mv, err := chess.UCINotation{}.Decode(m.pos(), uciMove)
	if err != nil {
		return fmt.Errorf("the engine played an invalid move %q", uciMove)
	}
	return m.play(mv)
}

// undo takes back the last move of each local player:
// one move in PvP; against the bot your move and the bot's answer.
func (m *match) undo() error {
	count := 1
	if m.mode != pvp {
		if m.thinking {
			return appload.Info("You can't undo while the bot is thinking.")
		}
		if m.pos().Turn() == chess.White {
			count = 2 // the bot's answer and your move
		}
	}
	if count > len(m.save.Moves) {
		count = len(m.save.Moves)
	}
	if count == 0 {
		return appload.Info("There's nothing to undo.")
	}
	save := savedGame{Start: m.save.Start, Moves: m.save.Moves[:len(m.save.Moves)-count]}
	game, err := save.replay()
	if err != nil {
		return err
	}
	m.save, m.game = save, game
	m.gen++
	return nil
}

func (m *match) status() string {
	turn := m.pos().Turn()
	switch m.game.Method() {
	case chess.Checkmate:
		return fmt.Sprintf("%s is checkmated!", title(turn))
	case chess.Stalemate:
		return "Stalemate!"
	case chess.InsufficientMaterial:
		return "Draw: insufficient material."
	case chess.FivefoldRepetition:
		return "Draw: fivefold repetition."
	case chess.SeventyFiveMoveRule:
		return "Draw: 75-move rule."
	}
	var s string
	switch {
	case !m.human(turn):
		s = "Waiting on your opponent..."
	case m.mode == pvp:
		s = fmt.Sprintf("It's %s's turn.", title(turn))
	default:
		s = "It's your turn."
	}
	if last := m.lastMove(); last != nil && last.HasTag(chess.Check) {
		s += " Check!"
	}
	return s
}

func (m *match) state() gameState {
	turn := m.pos().Turn()
	return gameState{
		Active:    true,
		Mode:      string(m.mode),
		Slot:      m.slot,
		Rotated:   m.rotate && m.mode == pvp && turn == chess.Black,
		Status:    m.status(),
		GameOver:  m.over(),
		Busy:      m.botToMove(),
		boardView: newBoardView(m.pos(), m.lastMove(), !m.over() && m.human(turn)),
	}
}

func title(c chess.Color) string {
	if c == chess.Black {
		return "Black"
	}
	return "White"
}
