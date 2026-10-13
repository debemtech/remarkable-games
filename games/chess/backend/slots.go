package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"appload-games/framework/store"

	"github.com/notnil/chess"
)

const slotCount = 3

// savedGame is what a slot holds. Keeping the moves (not just the position)
// means undo still works after the app was closed.
type savedGame struct {
	Start string   `json:"start,omitempty"` // FEN, empty = standard starting position
	Moves []string `json:"moves,omitempty"` // UCI notation
}

func (s savedGame) replay() (*chess.Game, error) {
	var opts []func(*chess.Game)
	if s.Start != "" {
		fen, err := chess.FEN(s.Start)
		if err != nil {
			return nil, fmt.Errorf("the saved game is invalid: %v", err)
		}
		opts = append(opts, fen)
	}
	game := chess.NewGame(opts...)
	for _, move := range s.Moves {
		mv, err := chess.UCINotation{}.Decode(game.Position(), move)
		if err == nil {
			err = game.Move(mv)
		}
		if err != nil {
			return nil, fmt.Errorf("the saved game is invalid: move %s: %v", move, err)
		}
	}
	return game, nil
}

func (s savedGame) describe() string {
	game, err := s.replay()
	if err != nil {
		return "Invalid save"
	}
	pos := game.Position()
	// The FEN's last field is the move number.
	fields := strings.Fields(pos.String())
	number := fields[len(fields)-1]
	switch game.Method() {
	case chess.Checkmate:
		return fmt.Sprintf("Move %s, checkmate", number)
	case chess.NoMethod:
		return fmt.Sprintf("Move %s, %s to play", number, title(pos.Turn()))
	default:
		return fmt.Sprintf("Move %s, draw", number)
	}
}

// slots are the save slots, stored in slots.json.
type slots struct {
	path  string
	Saves [slotCount]*savedGame `json:"saves"`
}

func loadSlots(dir string) (*slots, error) {
	s := &slots{path: filepath.Join(dir, "slots.json")}
	err := store.Load(s.path, s)
	if errors.Is(err, fs.ErrNotExist) {
		// First start: take over the savestates of chessMarkable on the rM1/rM2.
		return s, s.importLegacy(filepath.Join(dir, "savestates.yml"))
	}
	if err != nil {
		return s, fmt.Errorf("reading %s: %w", s.path, err)
	}
	return s, nil
}

func (s *slots) get(slot int) *savedGame {
	if slot < 1 || slot > slotCount {
		return nil
	}
	return s.Saves[slot-1]
}

func (s *slots) set(slot int, save *savedGame) error {
	if slot < 1 || slot > slotCount {
		return fmt.Errorf("invalid slot %d", slot)
	}
	s.Saves[slot-1] = save
	return store.Save(s.path, s)
}

func (s *slots) message() slotsMessage {
	var msg slotsMessage
	for i, save := range s.Saves {
		info := slotInfo{Slot: i + 1, Description: "Empty"}
		if save != nil {
			info.Saved = true
			info.Description = save.describe()
		}
		msg.Slots = append(msg.Slots, info)
	}
	return msg
}

// importLegacy reads chessMarkable's savestates.yml, which looks like
//
//	slot_1: rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1
//	slot_2: ~
func (s *slots) importLegacy(path string) error {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			continue
		}
		var slot int
		if _, err := fmt.Sscanf(strings.TrimSpace(key), "slot_%d", &slot); err != nil || slot < 1 || slot > slotCount {
			continue
		}
		fen := strings.Trim(strings.TrimSpace(value), `"'`)
		if fen == "" || fen == "~" || fen == "null" {
			continue
		}
		if _, err := chess.FEN(fen); err == nil {
			s.Saves[slot-1] = &savedGame{Start: fen}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return store.Save(s.path, s)
}
