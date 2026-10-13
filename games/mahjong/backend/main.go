// The backend of Mahjong solitaire for AppLoad: deals, rules and saving.
// The QML frontend only renders.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"

	"appload-games/framework/appload"
	"appload-games/framework/stats"
	"appload-games/framework/store"
	"appload-games/games/mahjong/backend/board"
)

type mahjongApp struct {
	app      *appload.App
	savePath string
	rng      *rand.Rand
	game     *game // the saved game, possibly cleared
	playing  bool
	records  *stats.Records
}

func newMahjongApp(app *appload.App, dataDir string, rng *rand.Rand) *mahjongApp {
	m := &mahjongApp{
		app:      app,
		savePath: filepath.Join(dataDir, "save.json"),
		rng:      rng,
		records:  stats.LoadRecords(filepath.Join(dataDir, "stats.json")),
	}
	var g game
	if err := store.Load(m.savePath, &g); err == nil && g.valid() {
		m.game = &g
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		app.Log.Error("loading the saved game", "err", err)
	}

	app.OnSync(m.sync)
	app.OnExit(m.pause)
	appload.On(app, msgNewGame, m.newGame)
	app.Command(msgContinue, m.continueGame)
	appload.On(app, msgRemove, func(r removeRequest) error {
		return m.update(func(g *game) error { return g.remove(r.A, r.B) })
	})
	app.Command(msgUndo, func() error { return m.update((*game).undoStep) })
	app.Command(msgHint, func() error { return m.update((*game).showHint) })
	app.Command(msgShuffle, func() error { return m.update(func(g *game) error { return g.shuffle(m.rng) }) })
	app.Command(msgQuit, m.quit)
	return m
}

func (m *mahjongApp) sync() {
	m.sendMenu()
	m.sendGame()
}

func (m *mahjongApp) sendGame() {
	if m.playing && m.game != nil {
		m.app.Send(msgGameState, m.game.state())
	} else {
		m.app.Send(msgGameState, gameState{})
	}
}

func (m *mahjongApp) sendMenu() {
	menu := menuState{}
	if g := m.game; g != nil && !g.Solved {
		menu.CanContinue = true
		menu.Continue = fmt.Sprintf("%s · %d tiles left · %d min", g.layout.Title, g.remaining(), g.Now()/60)
	}
	for _, l := range board.Layouts {
		rec := m.records.Get(l.Name)
		menu.Layouts = append(menu.Layouts, layoutInfo{
			Name: l.Name, Title: l.Title, Tiles: len(l.Tiles), Solved: rec.Solved, Best: rec.Best,
		})
	}
	m.app.Send(msgMenu, menu)
}

func (m *mahjongApp) save() {
	if m.game == nil {
		return
	}
	saved := *m.game
	saved.Seconds = m.game.Now() // without stopping the clock
	if err := store.Save(m.savePath, &saved); err != nil {
		m.app.Toast(appload.Errorf("Saving failed: %v", err))
	}
}

func (m *mahjongApp) pause() {
	if m.game != nil {
		m.game.Pause()
		m.save()
	}
}

func (m *mahjongApp) newGame(req newGameRequest) error {
	l := board.LayoutByName(req.Layout)
	if l == nil {
		return appload.Errorf("Unknown layout %q", req.Layout)
	}
	m.pause()
	m.game = newGame(l, m.rng)
	return m.continueGame()
}

func (m *mahjongApp) continueGame() error {
	if m.game == nil || m.game.Solved {
		return appload.Info("There's no game to continue.")
	}
	m.playing = true
	m.game.Resume()
	m.save()
	m.sendGame()
	return nil
}

// update runs an action on the current game, then saves and shows the result.
func (m *mahjongApp) update(action func(*game) error) error {
	g := m.game
	if !m.playing || g == nil {
		return appload.Info("No game is running.")
	}
	wasSolved := g.Solved
	err := action(g)
	if g.Solved && !wasSolved {
		if err := m.records.Win(g.Layout, g.Seconds, g.Hints == 0 && g.Shuffles == 0); err != nil {
			m.app.Toast(appload.Errorf("Saving stats failed: %v", err))
		}
	}
	m.save()
	m.sendGame()
	return err
}

func (m *mahjongApp) quit() error {
	m.pause()
	m.playing = false
	m.sendGame()
	m.sendMenu()
	return nil
}

func main() {
	app, err := appload.FromArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	newMahjongApp(app, store.Dir("mahjong"), rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())))
	if err := app.Run(); err != nil {
		app.Log.Error("connection to AppLoad failed", "err", err)
		os.Exit(1)
	}
}
