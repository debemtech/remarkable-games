// Package cards has French playing cards and a shuffled shoe of several decks.
package cards

import "math/rand/v2"

type Suit int8

const (
	Spades Suit = iota
	Hearts
	Diamonds
	Clubs
)

// Card is a playing card. Rank is 1 (ace) to 13 (king). The JSON names are
// short because a shoe holds hundreds of cards.
type Card struct {
	Rank int8 `json:"r"`
	Suit Suit `json:"s"`
}

// Deck returns the 52 cards in order.
func Deck() []Card {
	cards := make([]Card, 0, 52)
	for s := Spades; s <= Clubs; s++ {
		for r := int8(1); r <= 13; r++ {
			cards = append(cards, Card{r, s})
		}
	}
	return cards
}

// Shoe is a stack of shuffled decks. Cards are drawn from the end.
type Shoe struct {
	Decks int    `json:"decks"`
	Cards []Card `json:"cards"`
	// Reshuffle once fewer cards than this are left (the cut card).
	Cut int `json:"cut"`
}

// NewShoe shuffles decks decks. penetration is the share dealt before the
// next reshuffle, e.g. 0.75.
func NewShoe(rng *rand.Rand, decks int, penetration float64) *Shoe {
	s := &Shoe{Decks: decks, Cut: int(float64(decks*52) * (1 - penetration))}
	s.Shuffle(rng)
	return s
}

// Shuffle puts all cards back and shuffles them.
func (s *Shoe) Shuffle(rng *rand.Rand) {
	s.Cards = s.Cards[:0]
	for range s.Decks {
		s.Cards = append(s.Cards, Deck()...)
	}
	rng.Shuffle(len(s.Cards), func(i, j int) { s.Cards[i], s.Cards[j] = s.Cards[j], s.Cards[i] })
}

// NeedsShuffle reports whether the cut card was reached.
func (s *Shoe) NeedsShuffle() bool { return len(s.Cards) < s.Cut }

// Draw takes the next card. An empty shoe is refilled with an unshuffled
// deck rather than failing, so callers should check NeedsShuffle between rounds.
func (s *Shoe) Draw() Card {
	if len(s.Cards) == 0 {
		s.Cards = Deck()
	}
	c := s.Cards[len(s.Cards)-1]
	s.Cards = s.Cards[:len(s.Cards)-1]
	return c
}
