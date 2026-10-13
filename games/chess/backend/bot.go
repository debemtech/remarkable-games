package main

import (
	"context"
	"strconv"
	"sync"
	"time"

	"appload-games/games/chess/backend/uci"
)

// level is how a bot difficulty maps to Stockfish settings.
type level struct {
	skill  int // Stockfish "Skill Level", 0-20
	limits uci.Limits
}

var levels = map[mode]level{
	easy:   {skill: 0, limits: uci.Limits{Depth: 3, MoveTime: 200 * time.Millisecond}},
	normal: {skill: 8, limits: uci.Limits{MoveTime: 800 * time.Millisecond}},
	hard:   {skill: 20, limits: uci.Limits{MoveTime: 2 * time.Second}},
}

// bot owns the Stockfish process. It starts on the first search and restarts
// after a crash. Safe for concurrent use.
type bot struct {
	path string
	// The bot waits at least this long before answering. Instant answers make
	// it hard to see on e-ink which move was played.
	minDelay time.Duration

	mu     sync.Mutex
	engine *uci.Engine
}

func (b *bot) start() (*uci.Engine, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.engine != nil {
		return b.engine, nil
	}
	e, err := uci.Start(b.path)
	if err != nil {
		return nil, err
	}
	// Leave CPU for xochitl, keep memory use small.
	for name, value := range map[string]string{"Threads": "2", "Hash": "32"} {
		if err := e.SetOption(name, value); err != nil {
			e.Close()
			return nil, err
		}
	}
	b.engine = e
	return e, nil
}

func (b *bot) drop(e *uci.Engine) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.engine == e {
		b.engine = nil
		go e.Close()
	}
}

// think returns the bot's move for the position in UCI notation. newGame
// tells the engine to forget what it learned in an earlier game.
func (b *bot) think(ctx context.Context, fen string, lvl level, newGame bool) (string, error) {
	started := time.Now()
	e, err := b.start()
	if err != nil {
		return "", err
	}
	if newGame {
		if err := e.NewGame(); err != nil {
			b.drop(e)
			return "", err
		}
	}
	if err := e.SetOption("Skill Level", strconv.Itoa(lvl.skill)); err != nil {
		b.drop(e)
		return "", err
	}
	move, err := e.BestMove(ctx, fen, lvl.limits)
	if err != nil {
		b.drop(e)
		return "", err
	}
	select {
	case <-time.After(b.minDelay - time.Since(started)):
	case <-ctx.Done():
	}
	return move, nil
}

func (b *bot) close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.engine != nil {
		b.engine.Close()
		b.engine = nil
	}
}
