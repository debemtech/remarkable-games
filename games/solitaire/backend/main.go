// The backend of Solitaire for AppLoad: Klondike and FreeCell.
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
	"appload-games/games/solitaire/backend/solitaire"
)

var variants = []struct{ id, title string }{
	{solitaire.Klondike1, "Klondike · draw 1"},
	{solitaire.Klondike3, "Klondike · draw 3"},
	{solitaire.FreeCell, "FreeCell"},
}

func variantTitle(id string) string {
	for _, v := range variants {
		if v.id == id {
			return v.title
		}
	}
	return id
}

// saved is the game in progress as stored in save.json.
type saved struct {
	Game   *solitaire.Game `json:"game"`
	Hints  int             `json:"hints"`
	Solved bool            `json:"solved"`
	stats.Clock
}

type solitaireApp struct {
	app      *appload.App
	savePath string
	rng      *rand.Rand
	records  *stats.Records
	cur      *saved
	playing  bool
	hint     *solitaire.Hint
}

func newSolitaireApp(app *appload.App, dataDir string, rng *rand.Rand) *solitaireApp {
	s := &solitaireApp{
		app:      app,
		savePath: filepath.Join(dataDir, "save.json"),
		rng:      rng,
		records:  stats.LoadRecords(filepath.Join(dataDir, "stats.json")),
	}
	app.UsePrefs(filepath.Join(dataDir, "prefs.json"))
	var sv saved
	if err := store.Load(s.savePath, &sv); err == nil && sv.Game != nil {
		s.cur = &sv
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		app.Log.Error("loading the saved game", "err", err)
	}

	app.OnSync(s.sync)
	app.OnExit(s.pause)
	appload.On(app, msgNewGame, s.newGame)
	app.Command(msgContinue, s.continueGame)
	appload.On(app, msgMove, func(r moveRequest) error {
		return s.update(func(g *solitaire.Game) error { return g.Move(r.From, r.Index, r.To) })
	})
	appload.On(app, msgAuto, func(r moveRequest) error {
		return s.update(func(g *solitaire.Game) error { return g.Auto(r.From, r.Index) })
	})
	app.Command(msgStock, func() error { return s.update((*solitaire.Game).DrawStock) })
	app.Command(msgUndo, func() error {
		return s.update(func(g *solitaire.Game) error {
			if !g.Undo() {
				return appload.Info("There's nothing to undo.")
			}
			return nil
		})
	})
	app.Command(msgHint, s.showHint)
	app.Command(msgRestart, func() error {
		return s.update(func(g *solitaire.Game) error { g.Restart(); return nil })
	})
	app.Command(msgQuit, s.quit)
	return s
}

func (s *solitaireApp) title() string {
	g := s.cur.Game
	if g.Variant == solitaire.FreeCell {
		return fmt.Sprintf("FreeCell #%d", g.Deal)
	}
	return variantTitle(g.Variant)
}

func (s *solitaireApp) sync() {
	s.sendMenu()
	s.sendGame()
}

func (s *solitaireApp) sendGame() {
	if !s.playing || s.cur == nil {
		s.app.Send(msgGameState, gameState{})
		return
	}
	g := s.cur.Game
	piles := make([]solitaire.Pile, len(g.Piles))
	for i, p := range g.Piles {
		piles[i] = p
		if p.Cards == nil {
			piles[i].Cards = []solitaire.Card{} // a list, not null, in the JSON
		}
	}
	s.app.Send(msgGameState, gameState{
		Active:  true,
		Variant: g.Variant,
		Title:   s.title(),
		Piles:   piles,
		Solved:  s.cur.Solved,
		Seconds: s.cur.Now(),
		Moves:   g.Moves,
		CanUndo: g.CanUndo() && !s.cur.Solved,
		Hint:    s.hint,
	})
}

func (s *solitaireApp) sendMenu() {
	m := menuState{}
	if s.cur != nil && !s.cur.Solved {
		m.CanContinue = true
		m.Continue = fmt.Sprintf("%s · %d moves · %d min", s.title(), s.cur.Game.Moves, s.cur.Now()/60)
	}
	for _, v := range variants {
		rec := s.records.Get(v.id)
		m.Variants = append(m.Variants, variantInfo{ID: v.id, Title: v.title, Played: rec.Played, Won: rec.Solved, Best: rec.Best})
	}
	s.app.Send(msgMenu, m)
}

func (s *solitaireApp) save() {
	if s.cur == nil {
		return
	}
	sv := *s.cur
	sv.Seconds = s.cur.Now() // without stopping the clock
	if err := store.Save(s.savePath, &sv); err != nil {
		s.app.Toast(appload.Errorf("Saving failed: %v", err))
	}
}

func (s *solitaireApp) pause() {
	if s.cur != nil {
		s.cur.Pause()
		s.save()
	}
}

func (s *solitaireApp) newGame(req newGameRequest) error {
	g, err := solitaire.New(req.Variant, s.rng)
	if err != nil {
		return appload.Errorf("%v", err)
	}
	s.pause()
	s.cur = &saved{Game: g}
	if err := s.records.Start(req.Variant); err != nil {
		s.app.Log.Error("saving stats", "err", err)
	}
	return s.continueGame()
}

func (s *solitaireApp) continueGame() error {
	if s.cur == nil || s.cur.Solved {
		return appload.Info("There's no game to continue.")
	}
	s.playing, s.hint = true, nil
	s.cur.Resume()
	s.save()
	s.sendGame()
	return nil
}

// update runs an action on the current game, then saves and shows the result.
func (s *solitaireApp) update(action func(*solitaire.Game) error) error {
	if !s.playing || s.cur == nil {
		return appload.Info("No game is running.")
	}
	if s.cur.Solved {
		return appload.Info("You already won this one.")
	}
	err := action(s.cur.Game)
	s.hint = nil
	var toast appload.Toast
	if err != nil && !errors.As(err, &toast) {
		// The rules' errors are lower case phrases; show them as a sentence.
		msg := err.Error()
		err = appload.Info("%s%s.", strings.ToUpper(msg[:1]), msg[1:])
	}
	if s.cur.Game.Won() {
		s.cur.Pause()
		s.cur.Solved = true
		if err := s.records.Win(s.cur.Game.Variant, s.cur.Seconds, s.cur.Hints == 0); err != nil {
			s.app.Log.Error("saving stats", "err", err)
		}
	}
	s.save()
	s.sendGame()
	return err
}

func (s *solitaireApp) showHint() error {
	if !s.playing || s.cur == nil || s.cur.Solved {
		return nil
	}
	h, ok := s.cur.Game.Hint()
	if !ok {
		return appload.Info("No moves found. Undo, or start a new game.")
	}
	s.hint = &h
	s.cur.Hints++
	s.sendGame()
	return nil
}

func (s *solitaireApp) quit() error {
	s.pause()
	s.playing, s.hint = false, nil
	s.sendGame()
	s.sendMenu()
	return nil
}

func main() {
	app, err := appload.FromArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	newSolitaireApp(app, store.Dir("solitaire"), rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())))
	if err := app.Run(); err != nil {
		app.Log.Error("connection to AppLoad failed", "err", err)
		os.Exit(1)
	}
}
