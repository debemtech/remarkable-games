package main

import (
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"

	"appload-games/framework/appload"
	"appload-games/framework/appload/apptest"
)

func start(t *testing.T, dir string) *apptest.Host {
	t.Helper()
	app, host := apptest.New(t)
	newSudokuApp(app, dir, rand.New(rand.NewPCG(7, 7)))
	done := make(chan struct{})
	go func() { app.Run(); close(done) }()
	t.Cleanup(func() {
		host.Send(appload.MsgSystemTerminate, "close")
		<-done
	})
	host.Send(appload.MsgSystemNewCoordinator, "1")
	apptest.Expect[gameState](host, msgGameState, nil)
	return host
}

func playing(s gameState) bool { return s.Active && !s.Generating }

func emptyCell(s gameState) int {
	for i, v := range s.Values {
		if v == 0 {
			return i
		}
	}
	return -1
}

func TestGame(t *testing.T) {
	dir := t.TempDir()
	host := start(t, dir)

	host.Send(msgNewGame, newGameRequest{Difficulty: "medium"})
	apptest.Expect(host, msgGameState, func(s gameState) bool { return s.Generating })
	s := apptest.Expect(host, msgGameState, playing)
	if s.Difficulty != "medium" || s.Solved || s.CanUndo {
		t.Fatalf("bad new game %+v", s)
	}

	// Givens can't be changed.
	given := 0
	for !s.Given[given] {
		given++
	}
	host.Send(msgSet, cellRequest{Cell: given, Digit: 1})
	if toast := host.ExpectToast(); toast.Text != "That number is part of the puzzle." {
		t.Fatalf("unexpected toast %+v", toast)
	}

	// A digit that's already in the row is a conflict.
	cell := emptyCell(s)
	row := cell / 9
	var dup int8
	for c := row * 9; c < row*9+9; c++ {
		if s.Values[c] != 0 {
			dup = s.Values[c]
			break
		}
	}
	host.Send(msgSet, cellRequest{Cell: cell, Digit: dup})
	s = apptest.Expect(host, msgGameState, func(s gameState) bool { return s.Values[cell] == dup })
	if len(s.Conflicts) < 2 || !s.CanUndo {
		t.Fatalf("expected a conflict: %+v", s.Conflicts)
	}
	host.Send(msgCheck, nil)
	s = apptest.Expect(host, msgGameState, func(s gameState) bool { return len(s.Wrong) == 1 })
	if toast := host.ExpectToast(); toast.Text != "1 number is wrong." {
		t.Fatalf("unexpected toast %+v", toast)
	}
	host.Send(msgUndo, nil)
	s = apptest.Expect(host, msgGameState, func(s gameState) bool { return s.Values[cell] == 0 })
	if len(s.Conflicts) != 0 || len(s.Wrong) != 0 {
		t.Fatalf("undo left marks behind: %+v", s)
	}

	// Notes in a peer go away when the digit is placed, and come back on undo.
	peer := row * 9
	for peer == cell || s.Values[peer] != 0 {
		peer++
	}
	host.Send(msgNote, cellRequest{Cell: peer, Digit: 5})
	host.Send(msgNote, cellRequest{Cell: peer, Digit: 6})
	s = apptest.Expect(host, msgGameState, func(s gameState) bool { return s.Notes[peer] == 1<<4|1<<5 })
	host.Send(msgSet, cellRequest{Cell: cell, Digit: 5})
	s = apptest.Expect(host, msgGameState, func(s gameState) bool { return s.Values[cell] == 5 })
	if s.Notes[peer] != 1<<5 {
		t.Fatalf("note 5 should be gone from the peer: %b", s.Notes[peer])
	}
	host.Send(msgUndo, nil)
	s = apptest.Expect(host, msgGameState, func(s gameState) bool { return s.Values[cell] == 0 })
	if s.Notes[peer] != 1<<4|1<<5 {
		t.Fatalf("undo should restore the peer's notes: %b", s.Notes[peer])
	}

	// Quit and continue later, from the save file.
	host.Send(msgQuit, nil)
	menu := apptest.Expect[menuState](host, msgMenu, nil)
	if !menu.CanContinue {
		t.Fatalf("expected a game to continue: %+v", menu)
	}
	if _, err := os.Stat(filepath.Join(dir, "save.json")); err != nil {
		t.Fatal("no save file")
	}
	host2 := start(t, dir)
	host2.Send(msgContinue, nil)
	s = apptest.Expect(host2, msgGameState, playing)
	if s.Notes[peer] != 1<<4|1<<5 {
		t.Fatal("the saved game lost its notes")
	}

	// Solve it with hints.
	for !s.Solved {
		host2.Send(msgHint, cellRequest{Cell: -1})
		s = apptest.Expect[gameState](host2, msgGameState, nil)
	}
	if s.CanUndo || len(s.Conflicts) != 0 || emptyCell(s) != -1 {
		t.Fatalf("bad solved state %+v", s)
	}
	host2.Send(msgSet, cellRequest{Cell: cell, Digit: 1})
	if toast := host2.ExpectToast(); toast.Text != "The puzzle is solved." {
		t.Fatalf("unexpected toast %+v", toast)
	}
	host2.Send(msgQuit, nil)
	menu = apptest.Expect[menuState](host2, msgMenu, nil)
	if menu.CanContinue || menu.Stats[1].Solved != 1 || menu.Stats[1].Best != 0 {
		t.Fatalf("expected a win without best time (hints were used): %+v", menu)
	}
}

func TestInvalid(t *testing.T) {
	host := start(t, t.TempDir())
	host.Send(msgSet, cellRequest{Cell: 0, Digit: 1})
	if toast := host.ExpectToast(); toast.Text != "No game is running." {
		t.Fatalf("unexpected toast %+v", toast)
	}
	host.Send(msgNewGame, newGameRequest{Difficulty: "impossible"})
	host.ExpectToast()
	host.Send(msgNewGame, newGameRequest{Difficulty: "easy"})
	apptest.Expect(host, msgGameState, playing)
	host.Send(msgSet, cellRequest{Cell: 81, Digit: 1})
	host.ExpectToast()
	host.Send(msgNote, cellRequest{Cell: emptyCellOf(host), Digit: 10})
	host.ExpectToast()
}

func emptyCellOf(host *apptest.Host) int {
	host.Send(appload.MsgHello, nil)
	return emptyCell(apptest.Expect(host, msgGameState, playing))
}
