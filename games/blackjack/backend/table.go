package main

import (
	"fmt"
	"math/rand/v2"

	"appload-games/framework/appload"
	"appload-games/framework/cards"
)

const (
	startBankroll = 1000
	minBet        = 10
	maxBet        = 500
	maxHands      = 4 // after splitting
	decks         = 6
	penetration   = 0.75
)

// Chips the bet is made of. All multiples of 10, so 3:2 always pays whole chips.
var chips = []int{10, 50, 100}

type hand struct {
	Cards       []cards.Card `json:"cards"`
	Bet         int          `json:"bet"`
	Done        bool         `json:"done"`
	FromSplit   bool         `json:"fromSplit"`
	Doubled     bool         `json:"doubled"`
	Surrendered bool         `json:"surrendered"`
	Result      string       `json:"result"` // win, lose, push, blackjack, bust, surrender
	Payout      int          `json:"payout"` // returned to the bankroll, stake included
}

func (h *hand) total() int { t, _ := value(h.Cards); return t }

type tableStats struct {
	Hands      int `json:"hands"`
	Wins       int `json:"wins"`
	Losses     int `json:"losses"`
	Pushes     int `json:"pushes"`
	Blackjacks int `json:"blackjacks"`
	Peak       int `json:"peak"` // highest bankroll
	Restarts   int `json:"restarts"`
}

// table is the saved state: bankroll, the shoe and the round in progress.
type table struct {
	Bankroll int          `json:"bankroll"`
	Bet      int          `json:"bet"`   // for the next round
	Phase    string       `json:"phase"` // betting, playing, done
	Dealer   []cards.Card `json:"dealer"`
	Hands    []*hand      `json:"hands"`
	Active   int          `json:"active"`
	Shoe     *cards.Shoe  `json:"shoe"`
	Stats    tableStats   `json:"stats"`
	Message  string       `json:"message"`
}

func newTable(rng *rand.Rand) *table {
	return &table{
		Bankroll: startBankroll,
		Bet:      minBet,
		Phase:    "betting",
		Shoe:     cards.NewShoe(rng, decks, penetration),
		Stats:    tableStats{Peak: startBankroll},
	}
}

func (t *table) betting() bool { return t.Phase == "betting" || t.Phase == "done" }

// broke means the bankroll can't cover the smallest bet.
func (t *table) broke() bool { return t.betting() && t.Bankroll < minBet }

func (t *table) addChip(amount int) error {
	if !t.betting() {
		return appload.Info("Finish the hand first.")
	}
	if amount == 0 {
		t.Bet = 0
		return nil
	}
	valid := false
	for _, c := range chips {
		valid = valid || c == amount
	}
	if !valid {
		return appload.Errorf("No such chip %d", amount)
	}
	t.Bet = min(t.Bet+amount, maxBet, t.Bankroll/minBet*minBet)
	return nil
}

func (t *table) deal(rng *rand.Rand) error {
	switch {
	case !t.betting():
		return appload.Info("Finish the hand first.")
	case t.Bet < minBet:
		return appload.Info("Place a bet of at least %d.", minBet)
	case t.Bet > t.Bankroll:
		return appload.Info("You don't have enough chips for that bet.")
	}
	t.Message = ""
	if t.Shoe.NeedsShuffle() {
		t.Shoe.Shuffle(rng)
		t.Message = "The shoe was shuffled."
	}
	t.Bankroll -= t.Bet
	h := &hand{Bet: t.Bet}
	t.Hands, t.Active, t.Dealer = []*hand{h}, 0, nil
	h.Cards = append(h.Cards, t.Shoe.Draw())
	t.Dealer = append(t.Dealer, t.Shoe.Draw())
	h.Cards = append(h.Cards, t.Shoe.Draw())
	t.Dealer = append(t.Dealer, t.Shoe.Draw())
	t.Phase = "playing"

	// The dealer peeks for blackjack; a natural on either side ends the round.
	if isBlackjack(t.Dealer) || isBlackjack(h.Cards) {
		h.Done = true
		t.finish()
	}
	return nil
}

func (t *table) current() *hand {
	if t.Phase != "playing" {
		return nil
	}
	return t.Hands[t.Active]
}

func (t *table) canDouble() bool {
	h := t.current()
	return h != nil && len(h.Cards) == 2 && t.Bankroll >= h.Bet
}

func (t *table) canSplit() bool {
	h := t.current()
	return h != nil && len(h.Cards) == 2 && cardValue(h.Cards[0]) == cardValue(h.Cards[1]) &&
		len(t.Hands) < maxHands && t.Bankroll >= h.Bet && !(h.FromSplit && h.Cards[0].Rank == 1)
}

func (t *table) canSurrender() bool {
	h := t.current()
	return h != nil && len(t.Hands) == 1 && len(h.Cards) == 2
}

func (t *table) playing() (*hand, error) {
	h := t.current()
	if h == nil {
		return nil, appload.Info("Deal a new hand first.")
	}
	return h, nil
}

func (t *table) hit() error {
	h, err := t.playing()
	if err != nil {
		return err
	}
	h.Cards = append(h.Cards, t.Shoe.Draw())
	if h.total() >= 21 {
		h.Done = true
	}
	t.advance()
	return nil
}

func (t *table) stand() error {
	h, err := t.playing()
	if err != nil {
		return err
	}
	h.Done = true
	t.advance()
	return nil
}

func (t *table) double() error {
	h, err := t.playing()
	if err != nil {
		return err
	}
	if !t.canDouble() {
		return appload.Info("You can only double on your first two cards, with enough chips.")
	}
	t.Bankroll -= h.Bet
	h.Bet *= 2
	h.Doubled = true
	h.Cards = append(h.Cards, t.Shoe.Draw())
	h.Done = true
	t.advance()
	return nil
}

func (t *table) split() error {
	h, err := t.playing()
	if err != nil {
		return err
	}
	if !t.canSplit() {
		return appload.Info("You can only split two cards of the same value, with enough chips.")
	}
	t.Bankroll -= h.Bet
	second := &hand{Cards: []cards.Card{h.Cards[1]}, Bet: h.Bet, FromSplit: true}
	h.Cards, h.FromSplit = h.Cards[:1], true
	t.Hands = append(t.Hands[:t.Active+1], append([]*hand{second}, t.Hands[t.Active+1:]...)...)
	aces := h.Cards[0].Rank == 1
	for _, x := range []*hand{h, second} {
		x.Cards = append(x.Cards, t.Shoe.Draw())
		// Split aces get one card each.
		x.Done = aces || x.total() == 21
	}
	t.advance()
	return nil
}

func (t *table) surrender() error {
	h, err := t.playing()
	if err != nil {
		return err
	}
	if !t.canSurrender() {
		return appload.Info("You can only surrender on your first two cards.")
	}
	h.Surrendered, h.Done = true, true
	t.advance()
	return nil
}

// advance moves to the next unfinished hand, or lets the dealer play.
func (t *table) advance() {
	for t.Active < len(t.Hands) && t.Hands[t.Active].Done {
		t.Active++
	}
	if t.Active < len(t.Hands) {
		return
	}
	t.Active = len(t.Hands) - 1
	// The dealer only draws if a hand is still in the game.
	live := false
	for _, h := range t.Hands {
		live = live || (!h.Surrendered && h.total() <= 21)
	}
	for live && dealerTotal(t.Dealer) < 17 {
		t.Dealer = append(t.Dealer, t.Shoe.Draw())
	}
	t.finish()
}

// dealerTotal: the dealer stands on all 17s, soft ones included.
func dealerTotal(cs []cards.Card) int { total, _ := value(cs); return total }

// finish pays out every hand and ends the round.
func (t *table) finish() {
	dealerBJ := isBlackjack(t.Dealer)
	dealer := dealerTotal(t.Dealer)
	net := 0
	for _, h := range t.Hands {
		natural := !h.FromSplit && isBlackjack(h.Cards)
		total := h.total()
		switch {
		case h.Surrendered:
			h.Result, h.Payout = "surrender", h.Bet/2
		case dealerBJ && natural:
			h.Result, h.Payout = "push", h.Bet
		case dealerBJ:
			h.Result, h.Payout = "lose", 0
		case natural:
			h.Result, h.Payout = "blackjack", h.Bet+h.Bet*3/2
		case total > 21:
			h.Result, h.Payout = "bust", 0
		case dealer > 21 || total > dealer:
			h.Result, h.Payout = "win", 2*h.Bet
		case total == dealer:
			h.Result, h.Payout = "push", h.Bet
		default:
			h.Result, h.Payout = "lose", 0
		}
		t.Bankroll += h.Payout
		net += h.Payout - h.Bet

		t.Stats.Hands++
		switch h.Result {
		case "win":
			t.Stats.Wins++
		case "blackjack":
			t.Stats.Wins++
			t.Stats.Blackjacks++
		case "push":
			t.Stats.Pushes++
		default:
			t.Stats.Losses++
		}
	}
	t.Stats.Peak = max(t.Stats.Peak, t.Bankroll)
	t.Phase = "done"
	t.Bet = min(t.Bet, t.Bankroll/minBet*minBet)

	var msg string
	switch {
	case dealerBJ:
		msg = "Dealer has blackjack. "
	case len(t.Hands) == 1 && t.Hands[0].Result == "blackjack":
		msg = "Blackjack! "
	case dealer > 21:
		msg = "Dealer busts. "
	}
	switch {
	case net > 0:
		msg += fmt.Sprintf("You win %d.", net)
	case net < 0:
		msg += fmt.Sprintf("You lose %d.", -net)
	default:
		msg += "Push."
	}
	t.Message = msg
}

// restart gives a broke player a new bankroll.
func (t *table) restart() error {
	if !t.broke() {
		return appload.Info("You still have chips.")
	}
	t.Bankroll, t.Bet = startBankroll, minBet
	t.Stats.Restarts++
	t.Message = ""
	return nil
}

func (t *table) hint() string {
	h := t.current()
	if h == nil {
		return ""
	}
	up := cardValue(t.Dealer[0])
	if up == 1 {
		up = 11
	}
	return advice(h.Cards, up, t.canDouble(), t.canSplit(), t.canSurrender())
}
