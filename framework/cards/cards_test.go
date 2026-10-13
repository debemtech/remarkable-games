package cards

import (
	"math/rand/v2"
	"testing"
)

func TestShoe(t *testing.T) {
	if d := Deck(); len(d) != 52 || d[0] != (Card{1, Spades}) || d[51] != (Card{13, Clubs}) {
		t.Fatal("bad deck")
	}
	s := NewShoe(rand.New(rand.NewPCG(1, 1)), 6, 0.75)
	if len(s.Cards) != 312 || s.Cut != 78 {
		t.Fatalf("%d cards, cut at %d", len(s.Cards), s.Cut)
	}
	counts := map[Card]int{}
	for _, c := range s.Cards {
		counts[c]++
	}
	if len(counts) != 52 || counts[Card{1, Hearts}] != 6 {
		t.Fatal("shoe doesn't hold 6 full decks")
	}
	for range 312 - 78 {
		s.Draw()
	}
	if s.NeedsShuffle() {
		t.Fatal("shuffle before the cut card")
	}
	s.Draw()
	if !s.NeedsShuffle() {
		t.Fatal("no shuffle after the cut card")
	}
	s.Shuffle(rand.New(rand.NewPCG(2, 2)))
	if len(s.Cards) != 312 {
		t.Fatal("shuffle didn't refill the shoe")
	}
	empty := &Shoe{Decks: 1}
	empty.Draw() // must not panic
}
