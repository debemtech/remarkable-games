// The backend of Chess for AppLoad: chess rules, save slots and PGN
// replays, with Stockfish as the bot. The QML frontend only renders.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"appload-games/framework/appload"
	"appload-games/framework/store"

	"github.com/notnil/chess"
)

type config struct {
	dataDir   string // save slots
	pgnDir    string
	stockfish string
	botDelay  time.Duration
}

type chessApp struct {
	app   *appload.App
	cfg   config
	slots *slots
	bot   *bot

	match     *match
	cancelBot context.CancelFunc

	pgnGames []*chess.Game
	replay   *replay
}

func newChessApp(app *appload.App, cfg config) *chessApp {
	c := &chessApp{app: app, cfg: cfg, bot: &bot{path: cfg.stockfish, minDelay: cfg.botDelay}}
	var err error
	if c.slots, err = loadSlots(cfg.dataDir); err != nil {
		// Keep the broken file for the user; it gets replaced on the next save.
		app.Log.Error("loading save slots", "err", err)
		os.Rename(c.slots.path, c.slots.path+".broken")
	}

	app.OnSync(c.sync)
	app.OnExit(c.exit)
	appload.On(app, msgNewGame, c.newGame)
	appload.On(app, msgMove, c.move)
	app.Command(msgUndo, c.undo)
	app.Command(msgQuitGame, c.quitGame)
	app.Command(msgGetSlots, func() error { app.Send(msgSlots, c.slots.message()); return nil })
	app.Command(msgListPGNs, c.listPGNs)
	appload.On(app, msgOpenPGN, c.openPGN)
	appload.On(app, msgStartReplay, c.startReplay)
	app.Command(msgReplayNext, c.withReplay((*replay).next))
	app.Command(msgReplayUndo, c.withReplay((*replay).undo))
	app.Command(msgReplayReset, c.withReplay((*replay).reset))
	appload.On(app, msgReplayMove, c.replayMove)
	app.Command(msgReplayClose, func() error { c.replay = nil; c.sendReplay(); return nil })
	return c
}

func (c *chessApp) sync() {
	c.app.Send(msgSlots, c.slots.message())
	c.sendGame()
	c.sendReplay()
}

func (c *chessApp) exit() {
	c.stopBot()
	c.bot.close()
}

func (c *chessApp) sendGame() {
	if c.match == nil {
		c.app.Send(msgGameState, gameState{})
		return
	}
	c.app.Send(msgGameState, c.match.state())
}

func (c *chessApp) sendReplay() {
	if c.replay == nil {
		c.app.Send(msgReplayState, replayState{})
		return
	}
	c.app.Send(msgReplayState, c.replay.state())
}

// matchChanged autosaves the match, shows it and lets the bot move if it's its turn.
func (c *chessApp) matchChanged() {
	m := c.match
	save := savedGame{Start: m.save.Start, Moves: slices.Clone(m.save.Moves)}
	if err := c.slots.set(m.slot, &save); err != nil {
		c.app.Toast(appload.Errorf("Saving failed: %v", err))
	}
	c.startBot()
	c.sendGame()
}

func (c *chessApp) startBot() {
	m := c.match
	if m == nil || m.thinking || !m.botToMove() {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelBot = cancel
	m.thinking = true
	gen, fen, lvl, fresh := m.gen, m.pos().String(), levels[m.mode], !m.engineUsed
	m.engineUsed = true
	go func() {
		move, err := c.bot.think(ctx, fen, lvl, fresh)
		c.app.Post(func() { c.botDone(m, gen, move, err) })
	}()
}

func (c *chessApp) botDone(m *match, gen int, move string, err error) {
	if c.match != m || m.gen != gen {
		return // the match was quit or replaced meanwhile
	}
	m.thinking = false
	c.cancelBot = nil
	if err == nil {
		err = m.botMove(move)
	}
	if err != nil {
		c.app.Log.Error("bot", "err", err)
		c.app.Toast(appload.Errorf("The bot failed: %v. Undo to try again.", err))
		c.sendGame()
		return
	}
	c.matchChanged()
}

func (c *chessApp) stopBot() {
	if c.cancelBot != nil {
		c.cancelBot()
		c.cancelBot = nil
	}
}

func (c *chessApp) newGame(req newGameRequest) error {
	mode := mode(req.Mode)
	if !mode.valid() {
		return appload.Errorf("Unknown game mode %q", req.Mode)
	}
	if req.Slot < 1 || req.Slot > slotCount {
		return appload.Errorf("Invalid slot %d", req.Slot)
	}
	var save savedGame
	if s := c.slots.get(req.Slot); s != nil && !req.Reset {
		save = *s
	}
	m, err := newMatch(mode, req.Slot, req.Rotate, save)
	if err != nil {
		return appload.Errorf("Can't continue slot %d: %v", req.Slot, err)
	}
	c.stopBot()
	c.match = m
	c.matchChanged()
	return nil
}

func (c *chessApp) move(req moveRequest) error {
	if c.match == nil {
		return appload.Info("No game is running.")
	}
	if err := c.match.move(req); err != nil {
		c.sendGame()
		return err
	}
	c.matchChanged()
	return nil
}

func (c *chessApp) undo() error {
	if c.match == nil {
		return appload.Info("No game is running.")
	}
	if err := c.match.undo(); err != nil {
		return err
	}
	c.matchChanged()
	return nil
}

// quitGame ends the match. It has been saved after every move already.
func (c *chessApp) quitGame() error {
	c.stopBot()
	c.match = nil
	c.sendGame()
	c.app.Send(msgSlots, c.slots.message())
	return nil
}

func (c *chessApp) listPGNs() error {
	c.app.Send(msgPGNList, pgnList{Dir: c.cfg.pgnDir, Files: listPGNs(c.cfg.pgnDir)})
	return nil
}

func (c *chessApp) openPGN(req openPGNRequest) error {
	games, err := readPGN(c.cfg.pgnDir, req.File)
	c.pgnGames = games
	list := gameList{File: req.File, Games: []string{}}
	if err != nil {
		list.Error = err.Error()
	}
	for _, g := range games {
		list.Games = append(list.Games, describeGame(g))
	}
	c.app.Send(msgGameList, list)
	return nil
}

func (c *chessApp) startReplay(req startReplayRequest) error {
	if req.Index < 0 || req.Index >= len(c.pgnGames) {
		return appload.Errorf("That game doesn't exist.")
	}
	c.replay = newReplay(c.pgnGames[req.Index])
	c.sendReplay()
	return nil
}

func (c *chessApp) withReplay(action func(*replay)) func() error {
	return func() error {
		if c.replay == nil {
			return appload.Info("No replay is open.")
		}
		action(c.replay)
		c.sendReplay()
		return nil
	}
}

func (c *chessApp) replayMove(req moveRequest) error {
	if c.replay == nil {
		return appload.Info("No replay is open.")
	}
	err := c.replay.move(req)
	c.sendReplay()
	return err
}

func main() {
	app, err := appload.FromArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cfg := config{dataDir: store.Dir("chess"), botDelay: time.Second}
	cfg.pgnDir = filepath.Join(cfg.dataDir, "pgn")
	// Stockfish ships next to the backend; CHESS_STOCKFISH overrides it for testing on a PC.
	if cfg.stockfish = os.Getenv("CHESS_STOCKFISH"); cfg.stockfish == "" {
		exe, _ := os.Executable()
		cfg.stockfish = filepath.Join(filepath.Dir(exe), "stockfish")
	}
	newChessApp(app, cfg)
	if err := app.Run(); err != nil {
		app.Log.Error("connection to AppLoad failed", "err", err)
		os.Exit(1)
	}
}
