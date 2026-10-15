package main

import (
	"encoding/json"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"appload-games/framework/appload"
	"appload-games/framework/appload/apptest"
	"appload-games/games/crossword/backend/puzzle"
	"appload-games/games/crossword/backend/sources"
	"appload-games/games/crossword/hwrtrain/letters"
)

func start(t *testing.T, dir string) (*crosswordApp, *apptest.Host) {
	t.Helper()
	app, host := apptest.New(t)
	c, err := newCrosswordApp(app, dir, sources.NewClient())
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { app.Run(); close(done) }()
	t.Cleanup(func() {
		host.Send(appload.MsgSystemTerminate, "close")
		<-done
	})
	host.Send(appload.MsgSystemNewCoordinator, "1")
	apptest.Expect[menuView](host, msgMenu, nil)
	return c, host
}

func TestPlay(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "puzzles"), 0o755)
	data, _ := os.ReadFile("puzzle/testdata/mini.puz")
	os.WriteFile(filepath.Join(dir, "puzzles", "mini.puz"), data, 0o644)
	os.WriteFile(filepath.Join(dir, "puzzles", "notes.txt"), nil, 0o644)
	sol, _ := puzzle.ParsePuz(data)

	_, host := start(t, dir)
	host.Send(msgLibrary, nil)
	lib := apptest.Expect[libraryView](host, msgLibList, nil)
	if len(lib.Items) != 1 || lib.Items[0].Key != "puzzles/mini.puz" {
		t.Fatalf("bad library %+v", lib)
	}

	host.Send(msgOpen, keyRequest{Key: "puzzles/mini.puz"})
	pv := apptest.Expect(host, msgPuzzle, func(p puzzleView) bool { return p.Active })
	if pv.Width != 4 || len(pv.Cells) != 16 || !pv.HasSolution || len(pv.Across) != 4 {
		t.Fatalf("bad puzzle %+v", pv)
	}
	apptest.Expect[progressView](host, msgProgress, nil)

	// Type the first answer, one letter wrong.
	for i, l := range "CARX" {
		host.Send(msgSet, setRequest{Cell: i, Letter: string(l)})
	}
	pr := apptest.Expect(host, msgProgress, func(p progressView) bool { return p.Entries[3] == "X" })
	if pr.Filled != 4 || pr.Total != 16 {
		t.Fatalf("filled %d of %d", pr.Filled, pr.Total)
	}
	host.Send(msgCheck, nil)
	pr = apptest.Expect(host, msgProgress, func(p progressView) bool { return len(p.Wrong) == 1 })
	if pr.Wrong[0] != 3 {
		t.Fatalf("wrong cells %v", pr.Wrong)
	}
	if toast := host.ExpectToast(); toast.Text != "1 letter is wrong." {
		t.Fatalf("unexpected toast %+v", toast)
	}

	// Write the D by hand.
	rng := rand.New(rand.NewPCG(3, 3))
	var strokes [][]float64
	for _, s := range letters.Variant(rng, 'D') {
		var flat []float64
		for _, p := range s {
			flat = append(flat, p.X*20+500, p.Y*20+300)
		}
		strokes = append(strokes, flat)
	}
	host.Send(msgStrokes, strokesRequest{Cell: 3, Strokes: strokes})
	pr = apptest.Expect(host, msgProgress, func(p progressView) bool { return p.Guesses != nil })
	if pr.Entries[3] != "D" || pr.Guesses.Letters[0].Letter != "D" || len(pr.Guesses.Letters) != 4 || len(pr.Wrong) != 0 {
		t.Fatalf("handwriting: entry %q, guesses %+v", pr.Entries[3], pr.Guesses)
	}

	// Undo the handwritten D (back to X), then redo it by typing.
	host.Send(msgUndo, nil)
	apptest.Expect(host, msgProgress, func(p progressView) bool { return p.Entries[3] == "X" })
	// An eraser swipe over two squares is one step.
	host.Send(msgSet, setRequest{Cell: 2, Letter: ""})
	host.Send(msgSet, setRequest{Cell: 3, Letter: "", Join: true})
	apptest.Expect(host, msgProgress, func(p progressView) bool { return p.Entries[2] == "" && p.Entries[3] == "" })
	host.Send(msgUndo, nil)
	apptest.Expect(host, msgProgress, func(p progressView) bool { return p.Entries[2] == "R" && p.Entries[3] == "X" })
	host.Send(msgSet, setRequest{Cell: 3, Letter: "D"})

	// Reveal one square, fill in the rest: solved.
	host.Send(msgReveal, setRequest{Cell: 4})
	pr = apptest.Expect(host, msgProgress, func(p progressView) bool { return len(p.Revealed) == 1 })
	host.Send(msgUndo, nil)
	apptest.Expect(host, msgProgress, func(p progressView) bool { return len(p.Revealed) == 0 && p.Entries[4] == "" })
	host.Send(msgReveal, setRequest{Cell: 4})
	apptest.Expect(host, msgProgress, func(p progressView) bool { return len(p.Revealed) == 1 })
	for i, cell := range sol.Cells {
		if i > 4 && !cell.Block {
			host.Send(msgSet, setRequest{Cell: i, Letter: cell.Solution})
		}
	}
	pr = apptest.Expect(host, msgProgress, func(p progressView) bool { return p.Solved })
	host.Send(msgSet, setRequest{Cell: 0, Letter: "Z"})
	if toast := host.ExpectToast(); toast.Text != "The puzzle is solved." {
		t.Fatalf("unexpected toast %+v", toast)
	}

	host.Send(msgClose, nil)
	lib = apptest.Expect[libraryView](host, msgLibList, nil)
	if !lib.Items[0].Solved || lib.Items[0].Percent != 100 || lib.Items[0].Size != "4×4" {
		t.Fatalf("library didn't keep the progress: %+v", lib.Items[0])
	}
	if _, err := os.Stat(filepath.Join(dir, "progress.json")); err != nil {
		t.Fatal("progress not saved")
	}
}

func TestDownloadsAndLanguage(t *testing.T) {
	dir := t.TempDir()
	c, host := start(t, dir)
	host.Send(appload.MsgPrefs, map[string]any{"lang": "pt"})

	// A stored download shows up under its source and can be deleted.
	data, _ := os.ReadFile("puzzle/testdata/sample.ipuz")
	p, err := puzzle.ParseIPuz(data)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan string, 1)
	c.app.Post(func() {
		key, _ := c.lib.store("freixinho-abc", p)
		done <- key
	})
	key := <-done
	host.Send(msgLibrary, nil)
	lib := apptest.Expect[libraryView](host, msgLibList, nil)
	if len(lib.Items) != 1 || lib.Items[0].Origin != "Paulo Freixinho · Palavras Cruzadas" {
		t.Fatalf("bad library %+v", lib)
	}
	host.Send(msgOpen, keyRequest{Key: key})
	apptest.Expect(host, msgPuzzle, func(p puzzleView) bool { return p.Active && p.Title == "Pequena cruzada" })
	// Accents don't count: Ó is entered as O.
	host.Send(msgSet, setRequest{Cell: 1, Letter: "ó"})
	pr := apptest.Expect(host, msgProgress, func(p progressView) bool { return p.Entries[1] != "" })
	if pr.Entries[1] != "O" {
		t.Fatalf("entry %q", pr.Entries[1])
	}
	host.Send(msgCheck, nil)
	if toast := host.ExpectToast(); toast.Text != "Nenhum erro até agora." {
		t.Fatalf("not in Portuguese: %+v", toast)
	}

	host.Send(msgDelete, keyRequest{Key: "puzzles/../../etc/passwd"})
	host.ExpectToast()
	host.Send(msgDelete, keyRequest{Key: key})
	apptest.Expect(host, msgLibList, func(l libraryView) bool { return len(l.Items) == 0 })

	host.Send(msgSources, nil)
	src := apptest.Expect[sourcesView](host, msgSrcList, nil)
	if len(src.Sources) < 5 || src.Sources[0].Language != "pt" {
		t.Fatalf("bad sources %+v", src)
	}
}

// A letter misread and then written again in the same square, read right this
// time, is kept as a sample for training.
func TestSamples(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "puzzles"), 0o755)
	data, _ := os.ReadFile("puzzle/testdata/mini.puz")
	os.WriteFile(filepath.Join(dir, "puzzles", "mini.puz"), data, 0o644)
	_, host := start(t, dir)
	host.Send(msgOpen, keyRequest{Key: "puzzles/mini.puz"})
	apptest.Expect(host, msgPuzzle, func(p puzzleView) bool { return p.Active })

	rng := rand.New(rand.NewPCG(5, 5))
	write := func(cell int, letter byte) string {
		var strokes [][]float64
		for _, s := range letters.Variant(rng, letter) {
			var flat []float64
			for _, p := range s {
				flat = append(flat, p.X*20, p.Y*20)
			}
			strokes = append(strokes, flat)
		}
		host.Send(msgStrokes, strokesRequest{Cell: cell, Strokes: strokes})
		pr := apptest.Expect(host, msgProgress, func(p progressView) bool { return p.Guesses != nil && p.Guesses.Cell == cell })
		return pr.Entries[cell]
	}
	// Square 0 is the C of CARD.
	if got := write(0, 'X'); got != "X" {
		t.Fatalf("read %q", got)
	}
	write(0, 'C')
	write(1, 'A') // right the first time: nothing to learn
	write(2, 'X')
	write(3, 'D') // another square: not a fix of the X

	lines, _ := os.ReadFile(filepath.Join(dir, "samples.jsonl"))
	var got []sample
	for _, line := range strings.Split(strings.TrimSpace(string(lines)), "\n") {
		var s sample
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			t.Fatal(err)
		}
		got = append(got, s)
	}
	if len(got) != 1 || got[0].Letter != "C" || got[0].Cell != 0 || got[0].Misread.Guesses[0].Letter != "X" ||
		got[0].Fixed.Guesses[0].Letter != "C" || len(got[0].Misread.Strokes) != 2 {
		t.Fatalf("samples: %+v", got)
	}
}
