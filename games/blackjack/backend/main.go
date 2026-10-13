// The backend of Blackjack for AppLoad: you against the dealer. The dealer
// follows casino rules; Hint gives the basic strategy move.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"

	"appload-games/framework/appload"
	"appload-games/framework/cards"
	"appload-games/framework/store"
)

type blackjackApp struct {
	app      *appload.App
	savePath string
	rng      *rand.Rand
	table    *table
	hint     string
	// The dealer's cards are turned over one by one after the player's turn.
	// shown counts the face up cards; revealing until the result is shown.
	shown     int
	revealing bool
	revealGen int
	delay     time.Duration
}

func newBlackjackApp(app *appload.App, dataDir string, rng *rand.Rand, delay time.Duration) *blackjackApp {
	b := &blackjackApp{app: app, savePath: filepath.Join(dataDir, "save.json"), rng: rng, delay: delay}
	app.UsePrefs(filepath.Join(dataDir, "prefs.json"))
	var t table
	if err := store.Load(b.savePath, &t); err == nil && t.Shoe != nil {
		b.table = &t
	} else {
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			app.Log.Error("loading the saved table", "err", err)
		}
		b.table = newTable(rng)
	}

	app.OnSync(func() { b.finishReveal(); b.send() })
	app.OnExit(b.save)
	appload.On(app, msgBet, func(r betRequest) error { return b.act(func(t *table) error { return t.addChip(r.Amount) }) })
	app.Command(msgDeal, func() error { return b.act(func(t *table) error { return t.deal(b.rng) }) })
	app.Command(msgHit, func() error { return b.act((*table).hit) })
	app.Command(msgStand, func() error { return b.act((*table).stand) })
	app.Command(msgDouble, func() error { return b.act((*table).double) })
	app.Command(msgSplit, func() error { return b.act((*table).split) })
	app.Command(msgSurrender, func() error { return b.act((*table).surrender) })
	app.Command(msgRestart, func() error { return b.act((*table).restart) })
	app.Command(msgHint, func() error {
		if b.hint = b.table.hint(); b.hint == "" {
			return appload.Info("There's no hand to play.")
		}
		b.send()
		return nil
	})
	return b
}

func (b *blackjackApp) save() {
	if err := store.Save(b.savePath, b.table); err != nil {
		b.app.Toast(appload.Errorf("Saving failed: %v", err))
	}
}

// act runs an action, saves, and starts the dealer's reveal if the round ended.
func (b *blackjackApp) act(action func(*table) error) error {
	if b.revealing {
		return nil // the dealer is still turning cards
	}
	rounds := b.table.Stats.Hands
	err := action(b.table)
	b.hint = ""
	b.save()
	switch {
	case b.table.Stats.Hands != rounds: // the round just ended
		b.startReveal()
	case b.table.Phase == "playing":
		b.shown = 1
	}
	b.send()
	return err
}

func (b *blackjackApp) startReveal() {
	if b.delay <= 0 {
		b.finishReveal()
		return
	}
	b.revealing, b.shown = true, 2 // the hole card turns over right away
	b.revealGen++
	gen := b.revealGen
	var tick func()
	tick = func() {
		time.AfterFunc(b.delay, func() {
			b.app.Post(func() {
				if gen != b.revealGen || !b.revealing {
					return
				}
				if b.shown < len(b.table.Dealer) {
					b.shown++
					tick()
				} else {
					b.revealing = false
				}
				b.send()
			})
		})
	}
	tick()
}

func (b *blackjackApp) finishReveal() {
	b.revealing = false
	b.revealGen++
	if b.table.Phase == "playing" {
		b.shown = 1
	} else {
		b.shown = len(b.table.Dealer)
	}
}

func (b *blackjackApp) send() {
	t := b.table
	s := tableState{
		Phase:    t.Phase,
		Bankroll: t.Bankroll,
		Bet:      t.Bet,
		MinBet:   minBet,
		MaxBet:   maxBet,
		Chips:    chips,
		Dealer:   t.Dealer[:min(b.shown, len(t.Dealer))],
		Hidden:   max(0, len(t.Dealer)-b.shown),
		Active:   -1,
		Broke:    t.broke() && !b.revealing,
		Hint:     b.hint,
		Message:  t.Message,
		Stats:    t.Stats,
	}
	if t.Phase == "playing" {
		s.Active = t.Active
		s.CanHit = true
		s.CanDouble, s.CanSplit, s.CanSurrend = t.canDouble(), t.canSplit(), t.canSurrender()
		s.Message = ""
	}
	if b.revealing {
		s.Phase, s.Message = "dealer", ""
		s.Hidden = 0
	}
	if s.Dealer == nil {
		s.Dealer = []cards.Card{}
	}
	s.DealerTotal = dealerTotal(s.Dealer)
	for _, h := range t.Hands {
		total, soft := value(h.Cards)
		v := handView{Cards: h.Cards, Bet: h.Bet, Total: total, Soft: soft}
		if t.Phase == "done" {
			if b.revealing {
				s.Bankroll -= h.Payout // paid when the result shows
			} else {
				v.Result, v.Net = h.Result, h.Payout-h.Bet
			}
		}
		s.Hands = append(s.Hands, v)
	}
	if s.Hands == nil {
		s.Hands = []handView{}
	}
	b.app.Send(msgState, s)
}

func main() {
	app, err := appload.FromArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	newBlackjackApp(app, store.Dir("blackjack"), rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())), 700*time.Millisecond)
	if err := app.Run(); err != nil {
		app.Log.Error("connection to AppLoad failed", "err", err)
		os.Exit(1)
	}
}
