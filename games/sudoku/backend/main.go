// The backend of Sudoku for AppLoad: generates puzzles and keeps the game.
// The QML frontend only renders.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"

	"appload-games/framework/appload"
	"appload-games/framework/stats"
	"appload-games/framework/store"
	"appload-games/games/sudoku/backend/sudoku"
)

type sudokuApp struct {
	app        *appload.App
	savePath   string
	rng        *rand.Rand
	game       *game // the saved game, possibly solved
	playing    bool
	generating bool
	records    *stats.Records
}

func newSudokuApp(app *appload.App, dataDir string, rng *rand.Rand) *sudokuApp {
	s := &sudokuApp{
		app:      app,
		savePath: filepath.Join(dataDir, "save.json"),
		rng:      rng,
		records:  stats.LoadRecords(filepath.Join(dataDir, "stats.json")),
	}
	var g game
	if err := store.Load(s.savePath, &g); err == nil {
		s.game = &g
	} else if !errors.Is(err, fs.ErrNotExist) {
		app.Log.Error("loading the saved game", "err", err)
	}

	app.OnSync(s.sync)
	app.OnExit(func() { s.pause() })
	appload.On(app, msgNewGame, s.newGame)
	app.Command(msgContinue, s.continueGame)
	appload.On(app, msgSet, func(r cellRequest) error { return s.update(func(g *game) error { return g.set(r.Cell, r.Digit) }) })
	appload.On(app, msgNote, func(r cellRequest) error { return s.update(func(g *game) error { return g.note(r.Cell, r.Digit) }) })
	appload.On(app, msgHint, func(r cellRequest) error { return s.update(func(g *game) error { return g.hint(r.Cell) }) })
	app.Command(msgUndo, func() error { return s.update((*game).undoStep) })
	app.Command(msgCheck, func() error { return s.update((*game).check) })
	app.Command(msgRestart, func() error { return s.update(func(g *game) error { g.restart(); return nil }) })
	app.Command(msgQuit, s.quit)
	return s
}

func (s *sudokuApp) sync() {
	s.sendMenu()
	s.sendGame()
}

func (s *sudokuApp) sendGame() {
	switch {
	case s.generating:
		s.app.Send(msgGameState, gameState{Active: true, Generating: true})
	case s.playing && s.game != nil:
		s.app.Send(msgGameState, s.game.state())
	default:
		s.app.Send(msgGameState, gameState{})
	}
}

func (s *sudokuApp) sendMenu() {
	m := menuState{}
	if g := s.game; g != nil && !g.Solved {
		m.CanContinue = true
		m.Continue = fmt.Sprintf("%s · %d left · %d min", title(g.Difficulty), g.remaining(), g.Now()/60)
	}
	for _, d := range sudoku.Difficulties {
		rec := s.records.Get(d.Name)
		m.Stats = append(m.Stats, statsEntry{Difficulty: d.Name, Solved: rec.Solved, Best: rec.Best})
	}
	s.app.Send(msgMenu, m)
}

func (s *sudokuApp) save() {
	if s.game == nil {
		return
	}
	// Bring Seconds up to date without stopping the clock.
	saved := *s.game
	saved.Seconds = s.game.Now()
	if err := store.Save(s.savePath, &saved); err != nil {
		s.app.Toast(appload.Errorf("Saving failed: %v", err))
	}
}

func (s *sudokuApp) pause() {
	if s.game != nil {
		s.game.Pause()
		s.save()
	}
}

func (s *sudokuApp) newGame(req newGameRequest) error {
	d, ok := sudoku.DifficultyByName(req.Difficulty)
	if !ok {
		return appload.Errorf("Unknown difficulty %q", req.Difficulty)
	}
	if s.generating {
		return nil
	}
	s.pause()
	s.generating = true
	s.sendGame()
	go func() {
		puzzle, solution := sudoku.Generate(s.rng, d)
		s.app.Post(func() {
			s.generating = false
			s.game = newGame(d.Name, puzzle, solution)
			s.continueGame()
		})
	}()
	return nil
}

func (s *sudokuApp) continueGame() error {
	if s.game == nil || s.game.Solved {
		return appload.Info("There's no game to continue.")
	}
	s.playing = true
	s.game.Resume()
	s.save()
	s.sendGame()
	return nil
}

// update runs an action on the current game, then saves and shows the result.
func (s *sudokuApp) update(action func(*game) error) error {
	g := s.game
	if !s.playing || g == nil {
		return appload.Info("No game is running.")
	}
	wasSolved := g.Solved
	err := action(g)
	if g.Solved && !wasSolved {
		s.recordWin(g)
	}
	s.save()
	s.sendGame()
	return err
}

func (s *sudokuApp) recordWin(g *game) {
	if err := s.records.Win(g.Difficulty, g.Seconds, g.Hints == 0); err != nil {
		s.app.Toast(appload.Errorf("Saving stats failed: %v", err))
	}
}

func (s *sudokuApp) quit() error {
	s.pause()
	s.playing = false
	s.sendGame()
	s.sendMenu()
	return nil
}

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func main() {
	app, err := appload.FromArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	newSudokuApp(app, store.Dir("sudoku"), rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())))
	if err := app.Run(); err != nil {
		app.Log.Error("connection to AppLoad failed", "err", err)
		os.Exit(1)
	}
}
