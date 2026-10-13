package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"appload-games/framework/appload"
	"appload-games/framework/appload/apptest"
)

func sq(name string) int {
	return int(name[0]-'a') + int(name[1]-'1')*8
}

type harness struct {
	*apptest.Host
	t *testing.T
}

func start(t *testing.T, cfg config) *harness {
	t.Helper()
	if cfg.dataDir == "" {
		cfg.dataDir = t.TempDir()
	}
	cfg.pgnDir = filepath.Join(cfg.dataDir, "pgn")
	app, host := apptest.New(t)
	c := newChessApp(app, cfg)
	done := make(chan struct{})
	go func() { app.Run(); close(done) }()
	t.Cleanup(func() {
		host.Send(appload.MsgSystemTerminate, "close")
		<-done
		c.bot.close()
	})
	h := &harness{host, t}
	h.Send(appload.MsgSystemNewCoordinator, "1")
	apptest.Expect[replayState](h.Host, msgReplayState, nil)
	return h
}

// play makes moves like "e2e4" and waits until the board shows each of them.
func (h *harness) play(moves ...string) gameState {
	h.t.Helper()
	var state gameState
	for _, m := range moves {
		req := moveRequest{From: sq(m[0:2]), To: sq(m[2:4])}
		if len(m) == 5 {
			req.Promotion = m[4:]
		}
		h.Send(msgMove, req)
		state = apptest.Expect(h.Host, msgGameState, func(s gameState) bool {
			return len(s.LastMove) == 2 && s.LastMove[0] == req.From && s.LastMove[1] == req.To
		})
	}
	return state
}

func TestPvP(t *testing.T) {
	h := start(t, config{})

	h.Send(msgNewGame, newGameRequest{Mode: "pvp", Slot: 1, Reset: true, Rotate: true})
	s := apptest.Expect(h.Host, msgGameState, func(s gameState) bool { return s.Active })
	if s.Turn != "white" || len(s.Moves[sq("e2")]) != 2 || s.Board[sq("e1")] != 'K' {
		t.Fatalf("bad starting state %+v", s)
	}

	s = h.play("f2f3", "e7e5")
	if s.Rotated {
		t.Fatal("pieces rotated on white's turn")
	}
	s = h.play("g2g4", "d8h4")
	if !s.GameOver || s.Status != "White is checkmated!" || len(s.Moves) != 0 || !s.InCheck {
		t.Fatalf("expected checkmate, got %+v", s)
	}

	h.Send(msgMove, moveRequest{From: sq("a2"), To: sq("a3")})
	if toast := h.ExpectToast(); toast.Text != "The game is over." {
		t.Fatalf("unexpected toast %+v", toast)
	}

	h.Send(msgUndo, nil)
	s = apptest.Expect(h.Host, msgGameState, func(s gameState) bool { return !s.GameOver })
	if s.Turn != "black" || !s.Rotated || s.Status != "It's Black's turn." {
		t.Fatalf("bad state after undo %+v", s)
	}

	h.Send(msgQuitGame, nil)
	slots := apptest.Expect[slotsMessage](h.Host, msgSlots, nil)
	if !slots.Slots[0].Saved || slots.Slots[0].Description != "Move 2, Black to play" {
		t.Fatalf("slot not saved: %+v", slots)
	}

	// Continuing keeps the move history, so undo still works.
	h.Send(msgNewGame, newGameRequest{Mode: "pvp", Slot: 1})
	apptest.Expect(h.Host, msgGameState, func(s gameState) bool { return s.Active })
	for i := 0; i < 3; i++ {
		h.Send(msgUndo, nil)
		apptest.Expect[gameState](h.Host, msgGameState, nil)
	}
	h.Send(msgUndo, nil)
	if toast := h.ExpectToast(); toast.Text != "There's nothing to undo." {
		t.Fatalf("unexpected toast %+v", toast)
	}
}

func TestPromotion(t *testing.T) {
	h := start(t, config{})
	h.Send(msgNewGame, newGameRequest{Mode: "pvp", Slot: 2, Reset: true})
	apptest.Expect(h.Host, msgGameState, func(s gameState) bool { return s.Active })
	s := h.play("a2a4", "b7b5", "a4b5", "a7a6", "b5a6", "c8b7", "a6b7", "b8c6")
	if moves := s.Moves[sq("b7")]; len(moves) != 2 {
		t.Fatalf("promotions should be listed once per square: %v", moves)
	}
	s = h.play("b7a8n")
	if s.Board[sq("a8")] != 'N' {
		t.Fatalf("expected a knight on a8: %s", s.Board)
	}
	s = h.play("c6b8", "h2h4", "b8c6", "h4h5", "c6b8", "h5h6", "b8c6", "h6g7", "c6b8", "g7h8")
	if s.Board[sq("h8")] != 'Q' {
		t.Fatalf("expected a queen on h8 by default: %s", s.Board)
	}
}

const samplePGN = `[Event "Test"]
[Round "1"]
[White "Alice"]
[Black "Bob"]
[Result "1-0"]

1. e4 {King's pawn} e5 2. Bc4 (2. Nf3 Nc6) Nc6 3. Qh5 Nf6?? 4. Qxf7# 1-0

[Event "Broken"]
[White "X"]
[Black "Y"]

1. e4 e4 *

[Event "Endgame"]
[White "C"]
[Black "D"]
[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"]
[SetUp "1"]

1. e4 Kd7 *
`

func TestReplay(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "pgn"), 0o755)
	os.WriteFile(filepath.Join(dir, "pgn", "games.pgn"), []byte(samplePGN), 0o644)
	os.WriteFile(filepath.Join(dir, "pgn", "notes.txt"), nil, 0o644)
	h := start(t, config{dataDir: dir})

	h.Send(msgListPGNs, nil)
	list := apptest.Expect[pgnList](h.Host, msgPGNList, nil)
	if len(list.Files) != 1 || list.Files[0] != "games.pgn" {
		t.Fatalf("unexpected files %+v", list)
	}

	h.Send(msgOpenPGN, openPGNRequest{File: "games.pgn"})
	games := apptest.Expect[gameList](h.Host, msgGameList, nil)
	if len(games.Games) != 2 || games.Games[0] != "Alice vs Bob at Test 1" {
		t.Fatalf("unexpected games %+v", games)
	}

	h.Send(msgStartReplay, startReplayRequest{Index: 0})
	r := apptest.Expect(h.Host, msgReplayState, func(r replayState) bool { return r.Active })
	if r.Total != 7 || r.Played != 0 {
		t.Fatalf("bad replay %+v", r)
	}
	h.Send(msgReplayNext, nil)
	r = apptest.Expect[replayState](h.Host, msgReplayState, nil)
	if r.Comment != "King's pawn" {
		t.Fatalf("comment missing: %+v", r)
	}
	for i := 0; i < 6; i++ {
		h.Send(msgReplayNext, nil)
		r = apptest.Expect[replayState](h.Host, msgReplayState, nil)
	}
	if r.Played != 7 || !strings.Contains(r.Comment, "White won") || !r.InCheck {
		t.Fatalf("bad final state %+v", r)
	}

	h.Send(msgReplayReset, nil)
	apptest.Expect(h.Host, msgReplayState, func(r replayState) bool { return r.Played == 0 })
	h.Send(msgReplayMove, moveRequest{From: sq("d2"), To: sq("d4")})
	r = apptest.Expect(h.Host, msgReplayState, func(r replayState) bool { return r.ManualMoves == 1 })
	h.Send(msgReplayNext, nil)
	r = apptest.Expect[replayState](h.Host, msgReplayState, nil)
	if !strings.Contains(r.Comment, "Undo your own moves") {
		t.Fatalf("expected a hint, got %+v", r)
	}

	// A PGN with its own starting position
	h.Send(msgStartReplay, startReplayRequest{Index: 1})
	r = apptest.Expect(h.Host, msgReplayState, func(r replayState) bool { return r.Title == "C vs D at Endgame N/A" })
	if r.Board != "....K..."+"....P..."+strings.Repeat(".", 40)+"....k..." {
		t.Fatalf("FEN tag ignored: %s", r.Board)
	}

	h.Send(msgOpenPGN, openPGNRequest{File: "../../etc/passwd"})
	if g := apptest.Expect[gameList](h.Host, msgGameList, nil); g.Error == "" {
		t.Fatal("path traversal not rejected")
	}
	h.Send(msgReplayClose, nil)
	apptest.Expect(h.Host, msgReplayState, func(r replayState) bool { return !r.Active })
}

func stockfishPath(t *testing.T) string {
	if p := os.Getenv("CHESS_STOCKFISH"); p != "" {
		return p
	}
	if p, err := exec.LookPath("stockfish"); err == nil {
		return p
	}
	t.Skip("Stockfish not found, set CHESS_STOCKFISH")
	return ""
}

func TestBot(t *testing.T) {
	h := start(t, config{stockfish: stockfishPath(t), botDelay: 300 * time.Millisecond})
	h.Send(msgNewGame, newGameRequest{Mode: "hard", Slot: 3, Reset: true})
	apptest.Expect(h.Host, msgGameState, func(s gameState) bool { return s.Active })

	s := h.play("e2e4")
	if !s.Busy || s.Status != "Waiting on your opponent..." || len(s.Moves) != 0 {
		t.Fatalf("bot should be thinking: %+v", s)
	}
	h.Send(msgUndo, nil)
	if toast := h.ExpectToast(); !strings.Contains(toast.Text, "thinking") {
		t.Fatalf("unexpected toast %+v", toast)
	}
	s = apptest.Expect(h.Host, msgGameState, func(s gameState) bool { return s.Turn == "white" })
	if s.Busy || s.Status != "It's your turn." || s.LastMove == nil {
		t.Fatalf("bad state after the bot moved: %+v", s)
	}

	h.Send(msgUndo, nil)
	s = apptest.Expect[gameState](h.Host, msgGameState, nil)
	if s.Board != "RNBQKBNRPPPPPPPP"+strings.Repeat(".", 32)+"pppppppprnbqkbnr" {
		t.Fatalf("undo should take back both moves: %s", s.Board)
	}

	// Quitting while the bot thinks must not leave it hanging for the next game.
	h.play("d2d4")
	h.Send(msgQuitGame, nil)
	apptest.Expect(h.Host, msgGameState, func(s gameState) bool { return !s.Active })
	h.Send(msgNewGame, newGameRequest{Mode: "easy", Slot: 3})
	apptest.Expect(h.Host, msgGameState, func(s gameState) bool { return s.Active && s.Turn == "white" && !s.Busy })
}

func TestBotMissing(t *testing.T) {
	h := start(t, config{stockfish: "/nonexistent/stockfish"})
	h.Send(msgNewGame, newGameRequest{Mode: "easy", Slot: 1, Reset: true})
	h.play("e2e4")
	if toast := h.ExpectToast(); !strings.Contains(toast.Text, "The bot failed") {
		t.Fatalf("unexpected toast %+v", toast)
	}
	h.Send(msgUndo, nil)
	apptest.Expect(h.Host, msgGameState, func(s gameState) bool { return s.Turn == "white" && len(s.Moves) > 0 })
}
