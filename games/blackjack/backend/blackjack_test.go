package main

import (
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"appload-games/framework/appload"
	"appload-games/framework/appload/apptest"
	"appload-games/framework/cards"
)

// c parses cards like "A", "10", "K".
func c(ranks ...string) []cards.Card {
	names := map[string]int8{"A": 1, "J": 11, "Q": 12, "K": 13}
	var out []cards.Card
	for _, r := range ranks {
		n, ok := names[r]
		if !ok {
			for _, d := range r {
				n = n*10 + int8(d-'0')
			}
		}
		out = append(out, cards.Card{Rank: n, Suit: cards.Hearts})
	}
	return out
}

// rig makes the shoe deal these cards next. Deal order: player, dealer, player, dealer.
func rig(t *table, ranks ...string) {
	draws := c(ranks...)
	t.Shoe = &cards.Shoe{Decks: 1}
	for i := len(draws) - 1; i >= 0; i-- {
		t.Shoe.Cards = append(t.Shoe.Cards, draws[i])
	}
}

func TestValue(t *testing.T) {
	for _, tc := range []struct {
		cards []string
		total int
		soft  bool
	}{
		{[]string{"A", "6"}, 17, true},
		{[]string{"A", "6", "10"}, 17, false},
		{[]string{"A", "A", "9"}, 21, true},
		{[]string{"K", "Q", "2"}, 22, false},
		{[]string{"A", "K"}, 21, true},
	} {
		total, soft := value(c(tc.cards...))
		if total != tc.total || soft != tc.soft {
			t.Errorf("%v: got %d soft=%v", tc.cards, total, soft)
		}
	}
}

func TestAdvice(t *testing.T) {
	for _, tc := range []struct {
		cards []string
		up    int
		want  string
	}{
		{[]string{"10", "6"}, 10, surrender},
		{[]string{"10", "6"}, 6, stand},
		{[]string{"10", "2"}, 3, hit},
		{[]string{"6", "5"}, 10, double},
		{[]string{"6", "5"}, 11, hit},
		{[]string{"A", "7"}, 9, hit},
		{[]string{"A", "7"}, 6, double},
		{[]string{"A", "7"}, 2, stand},
		{[]string{"8", "8"}, 10, split},
		{[]string{"9", "9"}, 7, stand},
		{[]string{"A", "A"}, 11, split},
		{[]string{"5", "5"}, 9, double},
		{[]string{"K", "Q"}, 6, stand},
	} {
		if got := advice(c(tc.cards...), tc.up, true, true, true); got != tc.want {
			t.Errorf("%v vs %d: got %s, want %s", tc.cards, tc.up, got, tc.want)
		}
	}
	// Without the option, fall back to the next best move.
	if got := advice(c("6", "5"), 6, false, false, false); got != hit {
		t.Errorf("11 without double: %s", got)
	}
	if got := advice(c("A", "7"), 4, false, false, false); got != stand {
		t.Errorf("soft 18 without double: %s", got)
	}
}

func play(t *testing.T, ranks []string, actions ...func(*table) error) *table {
	t.Helper()
	tb := newTable(rand.New(rand.NewPCG(1, 1)))
	tb.Bet = 100
	rig(tb, ranks...)
	if err := tb.deal(nil); err != nil {
		t.Fatal(err)
	}
	for _, a := range actions {
		if err := a(tb); err != nil {
			t.Fatal(err)
		}
	}
	if tb.Phase != "done" {
		t.Fatalf("round not over: %s", tb.Phase)
	}
	return tb
}

func TestRounds(t *testing.T) {
	for _, tc := range []struct {
		name     string
		cards    []string // player, dealer, player, dealer, then draws
		actions  []func(*table) error
		results  []string
		bankroll int
	}{
		{"player blackjack pays 3:2", []string{"A", "9", "K", "7"}, nil, []string{"blackjack"}, 1150},
		{"dealer blackjack", []string{"10", "A", "9", "K"}, nil, []string{"lose"}, 900},
		{"both blackjack", []string{"A", "A", "K", "Q"}, nil, []string{"push"}, 1000},
		{"stand and win", []string{"10", "10", "9", "7"}, []func(*table) error{(*table).stand}, []string{"win"}, 1100},
		{"dealer stands on soft 17", []string{"10", "A", "7", "6"}, []func(*table) error{(*table).stand}, []string{"push"}, 1000},
		{"dealer draws to 17", []string{"10", "6", "8", "5", "6"}, []func(*table) error{(*table).stand}, []string{"win"}, 1100},
		{"bust", []string{"10", "6", "6", "10", "K"}, []func(*table) error{(*table).hit}, []string{"bust"}, 900},
		{"double", []string{"6", "6", "5", "10", "10", "K"}, []func(*table) error{(*table).double}, []string{"win"}, 1200},
		{"surrender", []string{"10", "10", "6", "K"}, []func(*table) error{(*table).surrender}, []string{"surrender"}, 950},
		{"split", []string{"8", "10", "8", "7", "K", "9"},
			[]func(*table) error{(*table).split, (*table).stand, (*table).stand}, []string{"win", "push"}, 1100},
		{"split aces get one card", []string{"A", "6", "A", "10", "K", "9", "K"},
			[]func(*table) error{(*table).split}, []string{"win", "win"}, 1200},
	} {
		tb := play(t, tc.cards, tc.actions...)
		var results []string
		for _, h := range tb.Hands {
			results = append(results, h.Result)
		}
		if strings.Join(results, ",") != strings.Join(tc.results, ",") || tb.Bankroll != tc.bankroll {
			t.Errorf("%s: got %v and bankroll %d, want %v and %d (dealer %v)", tc.name, results, tb.Bankroll, tc.results, tc.bankroll, tb.Dealer)
		}
	}
}

func TestBetsAndBankruptcy(t *testing.T) {
	tb := newTable(rand.New(rand.NewPCG(1, 1)))
	tb.addChip(100)
	tb.addChip(100)
	if tb.Bet != 210 {
		t.Fatalf("bet %d", tb.Bet)
	}
	for range 10 {
		tb.addChip(100)
	}
	if tb.Bet != maxBet {
		t.Fatalf("bet above the limit: %d", tb.Bet)
	}
	if err := tb.addChip(7); err == nil {
		t.Fatal("accepted a 7 chip")
	}
	tb.Bankroll = 45
	tb.Bet = 40
	rig(tb, "10", "10", "6", "K", "K")
	tb.deal(nil)
	tb.hit()
	if !tb.broke() || tb.Bet != 0 {
		t.Fatalf("expected to be broke: bankroll %d bet %d", tb.Bankroll, tb.Bet)
	}
	if err := tb.restart(); err != nil || tb.Bankroll != startBankroll || tb.Stats.Restarts != 1 {
		t.Fatal("restart failed")
	}
}

func TestApp(t *testing.T) {
	app, host := apptest.New(t)
	dir := t.TempDir()
	b := newBlackjackApp(app, dir, rand.New(rand.NewPCG(2, 2)), 20*time.Millisecond)
	done := make(chan error, 1)
	go func() { done <- app.Run() }()
	host.Send(appload.MsgSystemNewCoordinator, "1")
	s := apptest.Expect[tableState](host, msgState, nil)
	if s.Phase != "betting" || s.Bankroll != 1000 || s.Bet != 10 {
		t.Fatalf("bad start %+v", s)
	}
	host.Send(msgHit, nil)
	if toast := host.ExpectToast(); toast.Text != "Deal a new hand first." {
		t.Fatalf("unexpected toast %+v", toast)
	}

	host.Send(msgBet, betRequest{Amount: 50})
	apptest.Expect(host, msgState, func(s tableState) bool { return s.Bet == 60 })

	// Rig a hand: player 10+6 against a dealer 10 with a 7 in the hole.
	b.table.Shoe.Cards = append(b.table.Shoe.Cards, c("9", "7", "6", "10", "10")...)
	host.Send(msgDeal, nil)
	s = apptest.Expect(host, msgState, func(s tableState) bool { return s.Phase == "playing" })
	if len(s.Dealer) != 1 || s.Hidden != 1 || s.Hands[0].Total != 16 || !s.CanSurrend || s.Bankroll != 940 {
		t.Fatalf("bad deal %+v", s)
	}
	host.Send(msgHint, nil)
	if s = apptest.Expect(host, msgState, func(s tableState) bool { return s.Hint != "" }); s.Hint != surrender {
		t.Fatalf("hint %q", s.Hint)
	}

	// Hitting draws the 9: bust. The dealer reveals its hole card, then the result shows.
	host.Send(msgHit, nil)
	s = apptest.Expect(host, msgState, func(s tableState) bool { return s.Phase == "dealer" })
	if s.Hands[0].Result != "" || s.Bankroll != 940 || len(s.Dealer) != 2 || s.Hidden != 0 {
		t.Fatalf("result shown too early %+v", s)
	}
	s = apptest.Expect(host, msgState, func(s tableState) bool { return s.Phase == "done" })
	if s.Hands[0].Result != "bust" || s.Hands[0].Net != -60 || s.Message != "You lose 60." {
		t.Fatalf("bad result %+v", s)
	}

	// The table survives a restart of the backend.
	host.Send(appload.MsgSystemTerminate, "close")
	<-done
	app2, host2 := apptest.New(t)
	newBlackjackApp(app2, dir, rand.New(rand.NewPCG(3, 3)), 0)
	go func() { done <- app2.Run() }()
	host2.Send(appload.MsgHello, nil)
	s = apptest.Expect[tableState](host2, msgState, nil)
	if s.Bankroll != 940 || s.Stats.Hands != 1 || s.Stats.Losses != 1 {
		t.Fatalf("not restored %+v", s)
	}
	host2.Send(appload.MsgSystemTerminate, "close")
	<-done
}
