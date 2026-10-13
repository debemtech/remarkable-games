package solitaire

import (
	"math/rand/v2"
	"strings"
	"testing"

	"appload-games/framework/cards"
)

func name(c Card) string {
	ranks := "A23456789TJQK"
	suits := map[cards.Suit]string{cards.Spades: "S", cards.Hearts: "H", cards.Diamonds: "D", cards.Clubs: "C"}
	return string(ranks[c.Rank-1]) + suits[c.Suit]
}

func TestMicrosoftDeal(t *testing.T) {
	// Game #1 from Microsoft FreeCell, row by row.
	want := []string{
		"JD 2D 9H JC 5D 7H 7C 5H",
		"KD KC 9S 5S AD QC KH 3H",
		"2S KS 9D QD JS AS AH 3C",
		"4C 5C TS QH 4H AC 4D 7S",
		"3S TD 4S TH 8H 2C JH 7D",
		"6D 8S 8D QS 6C 3D 8C TC",
		"6S 9C 2H 6H",
	}
	g := NewFreeCell(1)
	for row, line := range want {
		var got []string
		for col := range 8 {
			if p := g.Piles[8+col]; row < len(p.Cards) {
				got = append(got, name(p.Cards[row]))
			}
		}
		if strings.Join(got, " ") != line {
			t.Fatalf("row %d: got %s, want %s", row, strings.Join(got, " "), line)
		}
	}
}

func TestKlondikeDeal(t *testing.T) {
	g, _ := New(Klondike3, rand.New(rand.NewPCG(1, 1)))
	if len(g.Piles[0].Cards) != 24 {
		t.Fatalf("stock has %d cards", len(g.Piles[0].Cards))
	}
	seen := map[string]bool{}
	for i, p := range g.Piles {
		for k, c := range p.Cards {
			seen[name(c)] = true
			if p.Kind == Tableau && c.Up != (k == len(p.Cards)-1) {
				t.Fatalf("pile %d card %d: wrong side up", i, k)
			}
		}
		if p.Kind == Tableau && len(p.Cards) != i-5 {
			t.Fatalf("column %d has %d cards", i-6, len(p.Cards))
		}
	}
	if len(seen) != 52 {
		t.Fatal("not a full deck")
	}
	// Draw three, then cycle the waste back.
	g.DrawStock()
	if len(g.Piles[1].Cards) != 3 || !g.Piles[1].Cards[2].Up {
		t.Fatal("draw three failed")
	}
	for len(g.Piles[0].Cards) > 0 {
		g.DrawStock()
	}
	waste := len(g.Piles[1].Cards)
	g.DrawStock()
	if len(g.Piles[0].Cards) != waste || len(g.Piles[1].Cards) != 0 || g.Piles[0].Cards[0].Up {
		t.Fatal("recycling the waste failed")
	}
	g.Undo()
	if len(g.Piles[1].Cards) != waste {
		t.Fatal("undo failed")
	}
}

// pile builds a pile from names like "KS QH".
func pile(kind, names string) Pile {
	p := Pile{Kind: kind}
	suits := map[byte]cards.Suit{'S': cards.Spades, 'H': cards.Hearts, 'D': cards.Diamonds, 'C': cards.Clubs}
	for _, n := range strings.Fields(names) {
		up := true
		if n[0] == '-' { // face down
			up, n = false, n[1:]
		}
		p.Cards = append(p.Cards, Card{int8(strings.IndexByte("A23456789TJQK", n[0]) + 1), suits[n[1]], up})
	}
	return p
}

func TestRules(t *testing.T) {
	g := &Game{Variant: Klondike1, Piles: []Pile{
		pile(Stock, ""), pile(Waste, "9C"),
		pile(Foundation, "AS 2S"), pile(Foundation, ""), pile(Foundation, ""), pile(Foundation, ""),
		pile(Tableau, "-4H KS QH JC"), pile(Tableau, "TD"), pile(Tableau, ""), pile(Tableau, "3S"),
	}}
	for _, tc := range []struct {
		from, index, to int
		ok              bool
	}{
		{1, 0, 7, true},  // 9C on TD
		{7, 0, 6, true},  // TD on JC
		{6, 1, 8, true},  // K run into the empty column
		{6, 2, 8, false}, // only kings go into empty columns
		{9, 0, 2, true},  // 3S on 2S
		{9, 0, 3, false}, // 3S on an empty foundation
		{6, 0, 7, false}, // face down card
	} {
		if err := g.check(tc.from, tc.index, tc.to); (err == nil) != tc.ok {
			t.Errorf("move %v: got %v", tc, err)
		}
	}
	if err := g.Move(6, 1, 8); err != nil {
		t.Fatal(err)
	}
	if !g.Piles[6].Cards[0].Up {
		t.Fatal("the card underneath didn't turn over")
	}
	// 3S would need both red twos up to be safe, so autoplay leaves it.
	if len(g.Piles[2].Cards) != 2 {
		t.Fatalf("autoplay moved an unsafe card: %v", g.Piles[2].Cards)
	}
}

func TestFreeCellSupermove(t *testing.T) {
	g := NewFreeCell(1)
	for i := range g.Piles {
		g.Piles[i].Cards = nil
	}
	for i, names := range []string{"8C 7H 6S 5D 4C 3H", "9D", "8S", "7D", "KC", "KH", "KD", "KS"} {
		g.Piles[8+i] = pile(Tableau, names)
	}
	// 4 free cells, no empty column: 5 cards at most.
	if g.check(8, 0, 9) == nil {
		t.Fatal("moved 6 cards with room for 5")
	}
	if err := g.check(8, 1, 10); err != nil {
		t.Fatal(err)
	}
	// One free cell: 2 cards; an empty column doubles that.
	g.Piles[0], g.Piles[1], g.Piles[2] = pile(Cell, "QD"), pile(Cell, "QC"), pile(Cell, "JS")
	if g.check(8, 2, 11) == nil {
		t.Fatal("moved 4 cards with one free cell")
	}
	g.Piles[15].Cards = nil
	if err := g.check(8, 2, 11); err != nil {
		t.Fatal(err)
	}
	if g.check(8, 1, 10) == nil {
		t.Fatal("moved 5 cards with room for 4")
	}
	// Into the empty column itself, it doesn't count: 2 cards.
	if g.check(8, 3, 15) == nil || g.check(8, 4, 15) != nil {
		t.Fatal("wrong limit for moving into an empty column")
	}
}

func TestAutoplayAndWin(t *testing.T) {
	g := NewFreeCell(1)
	for i := range g.Piles {
		g.Piles[i].Cards = nil
	}
	suits := []cards.Suit{cards.Spades, cards.Hearts, cards.Diamonds, cards.Clubs}
	for i, s := range suits {
		for r := int8(1); r <= 12; r++ {
			g.Piles[4+i].Cards = append(g.Piles[4+i].Cards, Card{r, s, true})
		}
	}
	for i, s := range suits {
		g.Piles[8+i].Cards = []Card{{13, s, true}}
	}
	g.Piles[12] = pile(Tableau, "")
	// Moving any king triggers the others: all are safe now.
	if err := g.Auto(8, 0); err != nil {
		t.Fatal(err)
	}
	if !g.Won() {
		t.Fatal("not won")
	}
}

func TestHintsPlayKlondike(t *testing.T) {
	rng := rand.New(rand.NewPCG(4, 4))
	wins := 0
	for game := 0; game < 30; game++ {
		g, _ := New(Klondike1, rng)
		cycles := 0
		for step := 0; step < 2000 && !g.Won(); step++ {
			h, ok := g.Hint()
			if !ok {
				break
			}
			if h.From == -1 {
				if len(g.Piles[0].Cards) == 0 {
					if cycles++; cycles > 3 {
						break
					}
				}
				g.DrawStock()
				continue
			}
			if err := g.Move(h.From, h.Index, h.To); err != nil {
				t.Fatalf("hint %v is illegal: %v", h, err)
			}
		}
		if g.Won() {
			wins++
		}
	}
	t.Logf("hints alone won %d of 30 Klondike games", wins)
}
