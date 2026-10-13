package main

import (
	"math/rand/v2"
	"strings"
	"testing"

	"appload-games/framework/appload"
	"appload-games/framework/appload/apptest"
	"appload-games/framework/cards"
	"appload-games/games/solitaire/backend/solitaire"
)

func start(t *testing.T, dir string) (*solitaireApp, *apptest.Host) {
	t.Helper()
	app, host := apptest.New(t)
	s := newSolitaireApp(app, dir, rand.New(rand.NewPCG(5, 5)))
	done := make(chan struct{})
	go func() { app.Run(); close(done) }()
	t.Cleanup(func() {
		host.Send(appload.MsgSystemTerminate, "close")
		<-done
	})
	host.Send(appload.MsgSystemNewCoordinator, "1")
	apptest.Expect[gameState](host, msgGameState, nil)
	return s, host
}

func active(s gameState) bool { return s.Active }

func TestKlondike(t *testing.T) {
	dir := t.TempDir()
	_, host := start(t, dir)
	host.Send(msgNewGame, newGameRequest{Variant: "klondike3"})
	s := apptest.Expect(host, msgGameState, active)
	if len(s.Piles) != 13 || s.Piles[0].Kind != solitaire.Stock || len(s.Piles[0].Cards) != 24 || s.Title != "Klondike · draw 3" {
		t.Fatalf("bad deal: %d piles, title %q", len(s.Piles), s.Title)
	}
	host.Send(msgStock, nil)
	s = apptest.Expect(host, msgGameState, func(s gameState) bool { return len(s.Piles[1].Cards) > 0 })
	if len(s.Piles[0].Cards) != 21 || !s.CanUndo {
		t.Fatalf("draw three failed: %d left", len(s.Piles[0].Cards))
	}
	// A face down card can't move.
	host.Send(msgMove, moveRequest{From: 12, Index: 0, To: 7})
	if toast := host.ExpectToast(); toast.Text != "That move isn't allowed." {
		t.Fatalf("unexpected toast %+v", toast)
	}
	host.Send(msgUndo, nil)
	s = apptest.Expect(host, msgGameState, func(s gameState) bool { return len(s.Piles[0].Cards) == 24 })

	host.Send(msgHint, nil)
	s = apptest.Expect(host, msgGameState, func(s gameState) bool { return s.Hint != nil })

	host.Send(msgQuit, nil)
	menu := apptest.Expect[menuState](host, msgMenu, nil)
	if !menu.CanContinue || menu.Variants[1].Played != 1 || !strings.HasPrefix(menu.Continue, "Klondike · draw 3") {
		t.Fatalf("bad menu %+v", menu)
	}
	_, host2 := start(t, dir)
	host2.Send(msgContinue, nil)
	if s2 := apptest.Expect(host2, msgGameState, active); len(s2.Piles[0].Cards) != 24 {
		t.Fatal("the saved game wasn't restored")
	}
}

func TestFreeCellWin(t *testing.T) {
	app, host := start(t, t.TempDir())
	host.Send(msgNewGame, newGameRequest{Variant: "freecell"})
	s := apptest.Expect(host, msgGameState, active)
	if len(s.Piles) != 16 || !strings.HasPrefix(s.Title, "FreeCell #") {
		t.Fatalf("bad deal %q", s.Title)
	}
	// An empty free cell takes any top card.
	host.Send(msgMove, moveRequest{From: 8, Index: len(s.Piles[8].Cards) - 1, To: 0})
	s = apptest.Expect(host, msgGameState, func(s gameState) bool { return len(s.Piles[0].Cards) == 1 })

	// Set up the end of a game: everything up but the kings.
	done := make(chan struct{})
	app.app.Post(func() {
		g := app.cur.Game
		for i := range g.Piles {
			g.Piles[i].Cards = nil
		}
		for i, suit := range []cards.Suit{cards.Spades, cards.Hearts, cards.Diamonds, cards.Clubs} {
			for r := int8(1); r <= 12; r++ {
				g.Piles[4+i].Cards = append(g.Piles[4+i].Cards, solitaire.Card{Rank: r, Suit: suit, Up: true})
			}
			g.Piles[8+i].Cards = []solitaire.Card{{Rank: 13, Suit: suit, Up: true}}
		}
		close(done)
	})
	<-done
	host.Send(msgAuto, moveRequest{From: 8, Index: 0})
	s = apptest.Expect(host, msgGameState, func(s gameState) bool { return s.Solved })
	host.Send(msgMove, moveRequest{From: 4, Index: 12, To: 8})
	if toast := host.ExpectToast(); toast.Text != "You already won this one." {
		t.Fatalf("unexpected toast %+v", toast)
	}
	host.Send(msgQuit, nil)
	menu := apptest.Expect[menuState](host, msgMenu, nil)
	if menu.CanContinue || menu.Variants[2].Won != 1 || menu.Variants[2].Played != 1 {
		t.Fatalf("win not recorded: %+v", menu)
	}
}

func TestInvalid(t *testing.T) {
	_, host := start(t, t.TempDir())
	host.Send(msgStock, nil)
	host.ExpectToast()
	host.Send(msgNewGame, newGameRequest{Variant: "spider"})
	host.ExpectToast()
	host.Send(msgNewGame, newGameRequest{Variant: "freecell"})
	apptest.Expect(host, msgGameState, active)
	for _, r := range []moveRequest{{-1, 0, 2}, {8, 99, 9}, {8, 0, 99}, {8, 0, 8}} {
		host.Send(msgMove, r)
		host.ExpectToast()
	}
	host.Send(msgStock, nil) // no stock in FreeCell
	host.ExpectToast()
}
