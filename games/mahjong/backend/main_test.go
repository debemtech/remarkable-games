package main

import (
	"math/rand/v2"
	"testing"

	"appload-games/framework/appload"
	"appload-games/framework/appload/apptest"
)

func start(t *testing.T, dir string) *apptest.Host {
	t.Helper()
	app, host := apptest.New(t)
	newMahjongApp(app, dir, rand.New(rand.NewPCG(9, 9)))
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

func active(s gameState) bool { return s.Active }

// pairs finds a matching free pair, a blocked tile and two free tiles that don't match.
func analyze(s gameState) (blocked int, mismatch []int) {
	blocked = -1
	var free []int
	for i := range s.Faces {
		if !s.Present[i] {
			continue
		}
		if !s.Free[i] {
			blocked = i
		} else {
			free = append(free, i)
		}
	}
	for _, a := range free {
		for _, b := range free {
			if a < b && s.Faces[a] != s.Faces[b] && s.Faces[a] < 34 && s.Faces[b] < 34 && mismatch == nil {
				mismatch = []int{a, b}
			}
		}
	}
	return blocked, mismatch
}

func TestGame(t *testing.T) {
	dir := t.TempDir()
	host := start(t, dir)

	host.Send(msgNewGame, newGameRequest{Layout: "quick"})
	s := apptest.Expect(host, msgGameState, active)
	if s.Remaining != 56 || len(s.X) != 56 || s.Width != 12 || s.Levels != 3 || s.Moves == 0 {
		t.Fatalf("bad new game: remaining %d, width %d, levels %d, moves %d", s.Remaining, s.Width, s.Levels, s.Moves)
	}

	blocked, mismatch := analyze(s)
	free := mismatch[0]
	host.Send(msgRemove, removeRequest{A: free, B: blocked})
	if toast := host.ExpectToast(); toast.Text != "That tile is blocked." {
		t.Fatalf("unexpected toast %+v", toast)
	}
	host.Send(msgRemove, removeRequest{A: mismatch[0], B: mismatch[1]})
	if toast := host.ExpectToast(); toast.Text != "These tiles don't match." {
		t.Fatalf("unexpected toast %+v", toast)
	}

	host.Send(msgHint, nil)
	s = apptest.Expect(host, msgGameState, func(s gameState) bool { return len(s.Hint) == 2 })
	pair := s.Hint
	host.Send(msgRemove, removeRequest{A: pair[0], B: pair[1]})
	s = apptest.Expect(host, msgGameState, func(s gameState) bool { return s.Remaining == 54 })
	if len(s.Hint) != 0 || !s.CanUndo {
		t.Fatalf("hint should be gone after a move: %+v", s.Hint)
	}
	host.Send(msgUndo, nil)
	s = apptest.Expect(host, msgGameState, func(s gameState) bool { return s.Remaining == 56 })

	// Saved and continued
	host.Send(msgQuit, nil)
	if menu := apptest.Expect[menuState](host, msgMenu, nil); !menu.CanContinue || len(menu.Layouts) != 3 {
		t.Fatalf("bad menu %+v", menu)
	}
	host = start(t, dir)
	host.Send(msgContinue, nil)
	s = apptest.Expect(host, msgGameState, active)

	// Play it out with hints, shuffling when stuck.
	for steps := 0; !s.Solved; steps++ {
		if steps > 200 {
			t.Fatal("game doesn't end")
		}
		if s.Moves == 0 {
			host.Send(msgShuffle, nil)
		} else {
			host.Send(msgHint, nil)
			s = apptest.Expect(host, msgGameState, func(s gameState) bool { return len(s.Hint) == 2 })
			host.Send(msgRemove, removeRequest{A: s.Hint[0], B: s.Hint[1]})
		}
		s = apptest.Expect(host, msgGameState, func(s gameState) bool { return len(s.Hint) == 0 })
	}
	if s.Remaining != 0 || s.CanUndo {
		t.Fatalf("bad final state: %d left", s.Remaining)
	}
	host.Send(msgQuit, nil)
	menu := apptest.Expect[menuState](host, msgMenu, nil)
	if menu.CanContinue || menu.Layouts[2].Solved != 1 || menu.Layouts[2].Best != 0 {
		t.Fatalf("expected a win without best time: %+v", menu)
	}
}

func TestInvalid(t *testing.T) {
	host := start(t, t.TempDir())
	host.Send(msgRemove, removeRequest{A: 0, B: 1})
	if toast := host.ExpectToast(); toast.Text != "No game is running." {
		t.Fatalf("unexpected toast %+v", toast)
	}
	host.Send(msgNewGame, newGameRequest{Layout: "nope"})
	host.ExpectToast()
	host.Send(msgNewGame, newGameRequest{Layout: "turtle"})
	apptest.Expect(host, msgGameState, active)
	for _, r := range []removeRequest{{0, 0}, {-1, 3}, {0, 999}} {
		host.Send(msgRemove, r)
		if toast := host.ExpectToast(); toast.Kind != "error" {
			t.Fatalf("expected an error for %v, got %+v", r, toast)
		}
	}
}
