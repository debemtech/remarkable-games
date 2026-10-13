// Package solitaire has the rules of Klondike and FreeCell. Both share piles,
// moves, undo, hints and moving cards to the foundations automatically.
package solitaire

import (
	"errors"
	"math/rand/v2"

	"appload-games/framework/cards"
)

// Card is a playing card that can lie face down.
type Card struct {
	Rank int8       `json:"r"`
	Suit cards.Suit `json:"s"`
	Up   bool       `json:"u"`
}

func (c Card) red() bool { return c.Suit == cards.Hearts || c.Suit == cards.Diamonds }

// Kinds of piles
const (
	Stock      = "stock"      // Klondike: face down, tap to draw
	Waste      = "waste"      // Klondike: drawn cards, the top one can be played
	Foundation = "foundation" // build each suit up from the ace
	Tableau    = "tableau"    // build down in alternating colors
	Cell       = "cell"       // FreeCell: holds one card
)

type Pile struct {
	Kind  string `json:"kind"`
	Cards []Card `json:"cards"`
}

func (p *Pile) top() (Card, bool) {
	if len(p.Cards) == 0 {
		return Card{}, false
	}
	return p.Cards[len(p.Cards)-1], true
}

// Variants
const (
	Klondike1 = "klondike1" // draw one card at a time
	Klondike3 = "klondike3" // draw three
	FreeCell  = "freecell"
)

// Game is a deal in progress. All fields are saved.
type Game struct {
	Variant string `json:"variant"`
	Deal    int    `json:"deal"` // FreeCell deal number (Microsoft numbering)
	Piles   []Pile `json:"piles"`
	Moves   int    `json:"moves"`
	// Shuffled order of the original deal, to restart it (Klondike).
	Start []Pile `json:"start"`
	undo  [][]Pile
}

var ErrIllegal = errors.New("that move isn't allowed")

func (g *Game) klondike() bool { return g.Variant != FreeCell }

func (g *Game) draw() int {
	if g.Variant == Klondike3 {
		return 3
	}
	return 1
}

// New deals a game. rng picks the Klondike shuffle and the FreeCell deal
// number. Cards that can safely go to the foundations right away do.
func New(variant string, rng *rand.Rand) (*Game, error) {
	var g *Game
	switch variant {
	case Klondike1, Klondike3:
		g = newKlondike(variant, rng)
	case FreeCell:
		n := 1 + rng.IntN(32000)
		if n == 11982 { // the one deal that can't be won
			n++
		}
		g = NewFreeCell(n)
	default:
		return nil, errors.New("unknown game " + variant)
	}
	g.autoplay()
	return g, nil
}

func newKlondike(variant string, rng *rand.Rand) *Game {
	deck := cards.Deck()
	rng.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	g := &Game{Variant: variant}
	g.Piles = append(g.Piles, Pile{Kind: Stock}, Pile{Kind: Waste})
	for range 4 {
		g.Piles = append(g.Piles, Pile{Kind: Foundation})
	}
	next := 0
	for col := range 7 {
		p := Pile{Kind: Tableau}
		for i := 0; i <= col; i++ {
			c := deck[next]
			next++
			p.Cards = append(p.Cards, Card{c.Rank, c.Suit, i == col})
		}
		g.Piles = append(g.Piles, p)
	}
	for _, c := range deck[next:] {
		g.Piles[0].Cards = append(g.Piles[0].Cards, Card{c.Rank, c.Suit, false})
	}
	g.Start = clonePiles(g.Piles)
	return g
}

// NewFreeCell deals FreeCell game n the way Microsoft's FreeCell did, so deal
// numbers match the ones known from there (1 to 32000).
func NewFreeCell(n int) *Game {
	g := &Game{Variant: FreeCell, Deal: n}
	for range 4 {
		g.Piles = append(g.Piles, Pile{Kind: Cell})
	}
	for range 4 {
		g.Piles = append(g.Piles, Pile{Kind: Foundation})
	}
	for range 8 {
		g.Piles = append(g.Piles, Pile{Kind: Tableau})
	}
	// Microsoft's deck order: card i is rank i/4+1 in clubs, diamonds, hearts, spades.
	msSuits := []cards.Suit{cards.Clubs, cards.Diamonds, cards.Hearts, cards.Spades}
	deck := make([]Card, 52)
	for i := range deck {
		deck[i] = Card{int8(i/4 + 1), msSuits[i%4], true}
	}
	seed := uint32(n)
	for i := 0; i < 52; i++ {
		seed = (seed*214013 + 2531011) & 0x7fffffff
		left := 52 - i
		j := int(seed>>16) % left
		g.Piles[8+i%8].Cards = append(g.Piles[8+i%8].Cards, deck[j])
		deck[j] = deck[left-1]
	}
	g.Start = clonePiles(g.Piles)
	return g
}

func clonePiles(piles []Pile) []Pile {
	out := make([]Pile, len(piles))
	for i, p := range piles {
		out[i] = Pile{Kind: p.Kind, Cards: append([]Card(nil), p.Cards...)}
	}
	return out
}

// Restart deals the same cards again.
func (g *Game) Restart() {
	g.Piles, g.Moves, g.undo = clonePiles(g.Start), 0, nil
	g.autoplay()
}

// Won reports whether all cards are on the foundations.
func (g *Game) Won() bool {
	n := 0
	for _, p := range g.Piles {
		if p.Kind == Foundation {
			n += len(p.Cards)
		}
	}
	return n == 52
}

func (g *Game) CanUndo() bool { return len(g.undo) > 0 }

func (g *Game) Undo() bool {
	n := len(g.undo)
	if n == 0 {
		return false
	}
	g.Piles, g.undo = g.undo[n-1], g.undo[:n-1]
	return true
}

func (g *Game) snapshot() { g.undo = append(g.undo, clonePiles(g.Piles)) }

// sequence reports whether cards[i:] can be moved together: face up, going
// down by one in alternating colors.
func sequence(cs []Card, i int) bool {
	if i < 0 || i >= len(cs) || !cs[i].Up {
		return false
	}
	for k := i + 1; k < len(cs); k++ {
		if !cs[k].Up || cs[k].Rank != cs[k-1].Rank-1 || cs[k].red() == cs[k-1].red() {
			return false
		}
	}
	return true
}

func (g *Game) count(kind string, empty bool) int {
	n := 0
	for _, p := range g.Piles {
		if p.Kind == kind && (len(p.Cards) == 0) == empty {
			n++
		}
	}
	return n
}

// check validates moving the cards from index on in pile from onto pile to.
func (g *Game) check(from, index, to int) error {
	if from < 0 || to < 0 || from >= len(g.Piles) || to >= len(g.Piles) || from == to {
		return ErrIllegal
	}
	src, dst := &g.Piles[from], &g.Piles[to]
	n := len(src.Cards) - index
	if n <= 0 || index < 0 {
		return ErrIllegal
	}
	switch src.Kind {
	case Stock:
		return ErrIllegal
	case Tableau:
		if !sequence(src.Cards, index) {
			return ErrIllegal
		}
	default:
		if n != 1 {
			return ErrIllegal
		}
	}
	moving := src.Cards[index]
	top, hasTop := dst.top()

	switch dst.Kind {
	case Foundation:
		if n != 1 {
			return ErrIllegal
		}
		if !hasTop {
			if moving.Rank != 1 {
				return ErrIllegal
			}
		} else if top.Suit != moving.Suit || top.Rank != moving.Rank-1 {
			return ErrIllegal
		}
	case Tableau:
		if !hasTop {
			if g.klondike() && moving.Rank != 13 {
				return errors.New("only a king can go into an empty column")
			}
		} else if !top.Up || top.Rank != moving.Rank+1 || top.red() == moving.red() {
			return ErrIllegal
		}
		// FreeCell moves several cards one at a time, through free cells and
		// empty columns. An empty target column doesn't count as free.
		if !g.klondike() && n > 1 {
			empty := g.count(Tableau, true)
			if !hasTop {
				empty--
			}
			if limit := (g.count(Cell, true) + 1) << empty; n > limit {
				return errors.New("not enough free cells to move that many cards")
			}
		}
	case Cell:
		if hasTop || n != 1 {
			return ErrIllegal
		}
	default:
		return ErrIllegal
	}
	return nil
}

// Move moves the cards from index on in pile from to pile to, then moves
// cards to the foundations where that's safe.
func (g *Game) Move(from, index, to int) error {
	if err := g.check(from, index, to); err != nil {
		return err
	}
	g.snapshot()
	g.move(from, index, to)
	g.Moves++
	g.autoplay()
	return nil
}

func (g *Game) move(from, index, to int) {
	src, dst := &g.Piles[from], &g.Piles[to]
	dst.Cards = append(dst.Cards, src.Cards[index:]...)
	src.Cards = src.Cards[:index]
	// Klondike: the card underneath turns over.
	if src.Kind == Tableau && len(src.Cards) > 0 {
		src.Cards[len(src.Cards)-1].Up = true
	}
}

// Auto moves the cards from index on to the best place: a foundation, else a
// column, else a free cell.
func (g *Game) Auto(from, index int) error {
	if to, ok := g.bestTarget(from, index); ok {
		return g.Move(from, index, to)
	}
	return ErrIllegal
}

func (g *Game) bestTarget(from, index int) (int, bool) {
	order := []string{Foundation, Tableau, Cell}
	for _, kind := range order {
		// Non-empty columns before empty ones
		for _, wantEmpty := range []bool{false, true} {
			for to := range g.Piles {
				p := &g.Piles[to]
				if p.Kind != kind || (len(p.Cards) == 0) != wantEmpty {
					continue
				}
				if g.check(from, index, to) == nil {
					return to, true
				}
			}
		}
	}
	return 0, false
}

// DrawStock turns cards from the stock onto the waste, or the waste back over
// when the stock is empty.
func (g *Game) DrawStock() error {
	if !g.klondike() {
		return ErrIllegal
	}
	stock, waste := &g.Piles[0], &g.Piles[1]
	if len(stock.Cards) == 0 && len(waste.Cards) == 0 {
		return errors.New("the stock is empty")
	}
	g.snapshot()
	if len(stock.Cards) == 0 {
		for i := len(waste.Cards) - 1; i >= 0; i-- {
			c := waste.Cards[i]
			c.Up = false
			stock.Cards = append(stock.Cards, c)
		}
		waste.Cards = waste.Cards[:0]
	} else {
		for i := 0; i < g.draw() && len(stock.Cards) > 0; i++ {
			c := stock.Cards[len(stock.Cards)-1]
			stock.Cards = stock.Cards[:len(stock.Cards)-1]
			c.Up = true
			waste.Cards = append(waste.Cards, c)
		}
	}
	g.Moves++
	g.autoplay()
	return nil
}

// foundationRank is the highest rank of a suit on the foundations.
func (g *Game) foundationRank(s cards.Suit) int8 {
	for _, p := range g.Piles {
		if top, ok := p.top(); ok && p.Kind == Foundation && top.Suit == s {
			return top.Rank
		}
	}
	return 0
}

// safe reports whether a card can go to its foundation without ever being
// needed in the columns again: both cards of the other color one rank lower
// are already up there.
func (g *Game) safe(c Card) bool {
	if c.Rank <= 2 {
		return true
	}
	for s := cards.Spades; s <= cards.Clubs; s++ {
		if (Card{Suit: s}).red() != c.red() && g.foundationRank(s) < c.Rank-1 {
			return false
		}
	}
	return true
}

func (g *Game) autoplay() {
	for moved := true; moved; {
		moved = false
		for from := range g.Piles {
			p := &g.Piles[from]
			top, ok := p.top()
			if !ok || !top.Up || p.Kind == Foundation || p.Kind == Stock || !g.safe(top) {
				continue
			}
			for to := range g.Piles {
				if g.Piles[to].Kind == Foundation && g.check(from, len(p.Cards)-1, to) == nil {
					g.move(from, len(p.Cards)-1, to)
					moved = true
					break
				}
			}
		}
	}
}

// Hint is a suggested move. From -1 means "draw from the stock".
type Hint struct {
	From  int `json:"from"`
	Index int `json:"index"`
	To    int `json:"to"`
}

// Hint suggests a useful move, or returns false if there is none.
func (g *Game) Hint() (Hint, bool) {
	type candidate struct {
		h     Hint
		score int
	}
	var best *candidate
	consider := func(from, index, to, score int) {
		if g.check(from, index, to) == nil && (best == nil || score > best.score) {
			best = &candidate{Hint{from, index, to}, score}
		}
	}
	for from := range g.Piles {
		src := &g.Piles[from]
		if len(src.Cards) == 0 || src.Kind == Stock || src.Kind == Foundation {
			continue
		}
		top := len(src.Cards) - 1
		// Longest movable run in a column
		first := top
		if src.Kind == Tableau {
			for first > 0 && sequence(src.Cards, first-1) {
				first--
			}
		}
		for to := range g.Piles {
			dst := &g.Piles[to]
			switch {
			case dst.Kind == Foundation:
				consider(from, top, to, 100)
			case dst.Kind == Tableau && src.Kind == Tableau:
				// Only worth it if it reveals or frees something.
				if first == 0 && len(dst.Cards) == 0 {
					continue
				}
				score := 50
				if first > 0 && !src.Cards[first-1].Up {
					score = 80 // turns a card over
				} else if first == 0 {
					score = 70 // empties a column
				}
				if len(dst.Cards) == 0 {
					score -= 30
				}
				consider(from, first, to, score)
			case dst.Kind == Tableau:
				score := 60
				if len(dst.Cards) == 0 {
					score = 30
				}
				consider(from, top, to, score)
			case dst.Kind == Cell && src.Kind == Tableau && len(src.Cards) > 1:
				consider(from, top, to, 10)
			}
		}
	}
	if best != nil {
		return best.h, true
	}
	if g.klondike() && (len(g.Piles[0].Cards) > 0 || len(g.Piles[1].Cards) > 0) {
		return Hint{From: -1}, true
	}
	return Hint{}, false
}
