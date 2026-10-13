// Package uci runs a chess engine such as Stockfish as a child process and
// talks to it with the Universal Chess Interface.
package uci

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Engine is a running engine process. Its methods are safe for concurrent use;
// searches run one at a time.
type Engine struct {
	mu    sync.Mutex
	cmd   *exec.Cmd
	in    io.WriteCloser
	lines chan string // engine output without "info" lines; closed when the engine exits
	Name  string
}

// ErrExited is returned when the engine process is gone.
var ErrExited = errors.New("uci: engine exited")

const handshakeTimeout = 10 * time.Second

// Start launches the engine at path and waits until it's ready.
func Start(path string) (*Engine, error) {
	cmd := exec.Command(path)
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	e := &Engine{cmd: cmd, in: in, lines: make(chan string, 64)}
	go e.read(out)

	if err := e.send("uci"); err != nil {
		e.Close()
		return nil, err
	}
	for {
		line, err := e.next(handshakeTimeout)
		if err != nil {
			e.Close()
			return nil, err
		}
		if name, ok := strings.CutPrefix(line, "id name "); ok {
			e.Name = name
		}
		if line == "uciok" {
			break
		}
	}
	if err := e.sync(); err != nil {
		e.Close()
		return nil, err
	}
	return e, nil
}

func (e *Engine) read(out io.Reader) {
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "info ") {
			continue // search progress, not needed
		}
		e.lines <- line
	}
	close(e.lines)
}

func (e *Engine) send(format string, args ...any) error {
	_, err := fmt.Fprintf(e.in, format+"\n", args...)
	return err
}

func (e *Engine) next(timeout time.Duration) (string, error) {
	select {
	case line, ok := <-e.lines:
		if !ok {
			return "", ErrExited
		}
		return line, nil
	case <-time.After(timeout):
		return "", errors.New("uci: engine is not responding")
	}
}

// sync waits until the engine processed everything sent so far.
func (e *Engine) sync() error {
	if err := e.send("isready"); err != nil {
		return err
	}
	for {
		line, err := e.next(handshakeTimeout)
		if err != nil {
			return err
		}
		if line == "readyok" {
			return nil
		}
	}
}

// SetOption sets an engine option, e.g. ("Skill Level", "5").
func (e *Engine) SetOption(name, value string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.send("setoption name %s value %s", name, value); err != nil {
		return err
	}
	return e.sync()
}

// NewGame tells the engine that the next search belongs to a different game.
func (e *Engine) NewGame() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.send("ucinewgame"); err != nil {
		return err
	}
	return e.sync()
}

// Limits bound a search. Zero values are unlimited, but at least one should be set.
type Limits struct {
	Depth    int
	MoveTime time.Duration
}

// BestMove searches the position and returns the best move in UCI notation
// (e.g. "e2e4", "e7e8q"), or "(none)" if there is no legal move. Cancelling
// ctx stops the search early; the engine's answer so far is returned.
func (e *Engine) BestMove(ctx context.Context, fen string, limits Limits) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	goCmd := "go"
	if limits.Depth > 0 {
		goCmd += fmt.Sprintf(" depth %d", limits.Depth)
	}
	if limits.MoveTime > 0 {
		goCmd += fmt.Sprintf(" movetime %d", limits.MoveTime.Milliseconds())
	}
	if err := e.send("position fen %s", fen); err != nil {
		return "", err
	}
	if err := e.send("%s", goCmd); err != nil {
		return "", err
	}

	// A search may run past its movetime a little, but not by this much.
	deadline := time.After(limits.MoveTime + 30*time.Second)
	stopped := false
	for {
		select {
		case line, ok := <-e.lines:
			if !ok {
				return "", ErrExited
			}
			if rest, found := strings.CutPrefix(line, "bestmove "); found {
				move, _, _ := strings.Cut(rest, " ")
				return move, nil
			}
		case <-ctx.Done():
			if !stopped {
				stopped = true
				if err := e.send("stop"); err != nil {
					return "", err
				}
				deadline = time.After(5 * time.Second)
			}
		case <-deadline:
			e.kill()
			return "", errors.New("uci: search did not finish, engine killed")
		}
	}
}

// Close asks the engine to quit and kills it if it doesn't.
func (e *Engine) Close() error {
	e.send("quit")
	e.in.Close()
	done := make(chan error, 1)
	go func() { done <- e.cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		e.kill()
		return <-done
	}
}

func (e *Engine) kill() {
	if e.cmd.Process != nil {
		e.cmd.Process.Kill()
	}
}
