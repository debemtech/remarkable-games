package sources_test

import (
	"context"
	"os"
	"testing"
	"time"

	"appload-games/games/crossword/backend/puzzle"
	"appload-games/games/crossword/backend/sources"
)

func TestDeobfuscate(t *testing.T) {
	rawc, err := os.ReadFile("../puzzle/testdata/amuse_rawc.txt")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	data, err := sources.Deobfuscate(string(rawc))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("decoded in %v", time.Since(start))
	p, err := puzzle.FromAmuse(data)
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Cruzadinha de teste" || p.Width != 4 || p.Height != 4 || !p.Cells[15].Block {
		t.Fatalf("bad puzzle %+v", p)
	}
	word := ""
	for _, c := range p.Across[1].Cells {
		word += p.Cells[c].Solution
	}
	if word != "AMOR" || p.Across[1].Number != 5 || p.Across[1].Text != "Sentimento de afeição" || len(p.Down[3].Cells) != 3 {
		t.Fatalf("5 across: %q %+v", word, p.Across[1])
	}
	if _, err := sources.Deobfuscate(""); err == nil {
		t.Fatal("decoded nothing")
	}
}

// Downloads from every source. Needs the internet: CROSSWORD_ONLINE=1 go test
func TestOnline(t *testing.T) {
	if os.Getenv("CROSSWORD_ONLINE") == "" {
		t.Skip("set CROSSWORD_ONLINE=1 to test the downloads")
	}
	c := sources.NewClient()
	for _, s := range sources.Builtin {
		t.Run(s.ID, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			list, err := c.List(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			p, err := c.Fetch(ctx, s, list[0].ID)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%d puzzles, latest %q %dx%d, %d clues", len(list), p.Title, p.Width, p.Height, len(p.Across)+len(p.Down))
		})
	}
}
