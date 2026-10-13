// Package appload is the backend half of an AppLoad application.
//
// AppLoad starts the backend with the path of a SOCK_SEQPACKET unix socket as
// the first argument. Every message is a packet with an 8 byte header (type and
// length, native endian) followed by a packet with the body.
//
// App runs everything on one goroutine: message handlers, functions passed to
// Post and the sync/exit hooks. Game state therefore needs no locking. Slow
// work (an engine search, generating a puzzle) runs in its own goroutine and
// hands its result back with Post.
//
// Conventions shared with framework/qml:
//   - bodies are JSON in both directions
//   - types below 100 belong to the framework (MsgHello, MsgToast, MsgPrefs,
//     pen.Msg)
//   - games use 100-199 for requests and 200-299 for updates
package appload

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"
)

const (
	// MsgHello is sent by the frontend when it starts. Like a new coordinator,
	// it triggers the OnSync hook.
	MsgHello uint32 = 1
	// MsgToast carries a Toast to the frontend.
	MsgToast uint32 = 2
	// MsgPrefs carries the frontend's preferences (a JSON object) in both
	// directions, see UsePrefs.
	MsgPrefs uint32 = 3

	MsgSystemTerminate       uint32 = 0xFFFFFFFF
	MsgSystemNewCoordinator  uint32 = 0xFFFFFFFE
	MsgSystemLostCoordinator uint32 = 0xFFFFFFFD

	// MaxMessageSize is AppLoad's limit for a message body.
	MaxMessageSize = 10 << 20
)

// Conn is a connection to AppLoad. Writes are safe from any goroutine.
type Conn struct {
	c   *net.UnixConn
	wmu sync.Mutex
}

// Dial connects to the socket AppLoad passed on the command line.
func Dial(path string) (*Conn, error) {
	c, err := net.DialUnix("unixpacket", nil, &net.UnixAddr{Name: path, Net: "unixpacket"})
	if err != nil {
		return nil, err
	}
	return NewConn(c), nil
}

// NewConn wraps an already connected unixpacket socket.
func NewConn(c *net.UnixConn) *Conn { return &Conn{c: c} }

// Read returns the next message.
func (c *Conn) Read() (uint32, []byte, error) {
	var header [8]byte
	n, err := c.c.Read(header[:])
	if err != nil {
		return 0, nil, err
	}
	if n != len(header) {
		return 0, nil, fmt.Errorf("appload: short header (%d bytes)", n)
	}
	msgType := binary.NativeEndian.Uint32(header[0:])
	length := binary.NativeEndian.Uint32(header[4:])
	if length > MaxMessageSize {
		return 0, nil, fmt.Errorf("appload: message of %d bytes exceeds the limit", length)
	}
	// AppLoad sends a body packet even when it's empty. Reading it needs a
	// non-empty buffer (Go skips zero-length reads), and Go reports the empty
	// packet as io.EOF. That's fine: a real EOF shows up on the next header.
	if length == 0 {
		var scratch [1]byte
		c.c.Read(scratch[:])
		return msgType, nil, nil
	}
	body := make([]byte, length)
	n, err = c.c.Read(body)
	if err != nil {
		return 0, nil, err
	}
	return msgType, body[:n], nil
}

// Write sends a message. Empty bodies are not sent, like AppLoad's own clients do.
func (c *Conn) Write(msgType uint32, body []byte) error {
	if len(body) > MaxMessageSize {
		return fmt.Errorf("appload: message of %d bytes exceeds the limit", len(body))
	}
	var header [8]byte
	binary.NativeEndian.PutUint32(header[0:], msgType)
	binary.NativeEndian.PutUint32(header[4:], uint32(len(body)))
	c.wmu.Lock()
	defer c.wmu.Unlock()
	if _, err := c.c.Write(header[:]); err != nil {
		return err
	}
	if len(body) > 0 {
		if _, err := c.c.Write(body); err != nil {
			return err
		}
	}
	return nil
}

func (c *Conn) Close() error { return c.c.Close() }

// Toast is a short message for the user. It is also an error: returning a
// Toast from a handler shows it as is.
type Toast struct {
	Text string `json:"text"`
	Kind string `json:"kind"` // "info" or "error"
	Ms   int    `json:"ms"`   // how long to show it, 0 = until replaced
}

func (t Toast) Error() string { return t.Text }

// For returns the toast with a different display time.
func (t Toast) For(d time.Duration) Toast {
	t.Ms = int(d / time.Millisecond)
	return t
}

// Info is a neutral message, shown for 3 seconds.
func Info(format string, args ...any) Toast {
	return Toast{Text: fmt.Sprintf(format, args...), Kind: "info", Ms: 3000}
}

// Errorf is an error message, shown for 10 seconds.
func Errorf(format string, args ...any) Toast {
	return Toast{Text: fmt.Sprintf(format, args...), Kind: "error", Ms: 10000}
}

// Handler handles the body of one message type.
type Handler func(body []byte) error

// App dispatches messages from AppLoad.
type App struct {
	conn      *Conn
	handlers  map[uint32]Handler
	prefsPath string
	prefs     map[string]any
	onSync    func()
	onExit    func()
	posts     chan func()
	done      chan struct{}
	Log       *slog.Logger
}

// New creates an App on an existing connection.
func New(conn *Conn) *App {
	return &App{
		conn:     conn,
		handlers: map[uint32]Handler{},
		posts:    make(chan func(), 64),
		done:     make(chan struct{}),
		Log:      slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}
}

// FromArgs connects to the socket given by AppLoad (os.Args[1]).
func FromArgs() (*App, error) {
	if len(os.Args) < 2 {
		return nil, errors.New("this program is started by AppLoad with the socket path as its argument")
	}
	conn, err := Dial(os.Args[1])
	if err != nil {
		return nil, err
	}
	return New(conn), nil
}

// Handle registers the handler for a message type.
func (a *App) Handle(msgType uint32, h Handler) { a.handlers[msgType] = h }

// Command registers a handler for a message without payload.
func (a *App) Command(msgType uint32, h func() error) {
	a.Handle(msgType, func([]byte) error { return h() })
}

// On registers a handler that receives the body decoded from JSON.
func On[T any](a *App, msgType uint32, h func(T) error) {
	a.Handle(msgType, func(body []byte) error {
		var v T
		if len(body) > 0 {
			if err := json.Unmarshal(body, &v); err != nil {
				return Errorf("Invalid request: %v", err)
			}
		}
		return h(v)
	})
}

// OnSync sets the hook that runs whenever a frontend (re)connects. It should
// send the complete state, since the frontend starts out empty.
func (a *App) OnSync(f func()) { a.onSync = f }

// UsePrefs keeps the frontend's preferences (GameApp.setPref) in a JSON file
// at path and sends them to every frontend that connects.
func (a *App) UsePrefs(path string) {
	a.prefsPath = path
	a.prefs = map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &a.prefs)
	}
	a.Handle(MsgPrefs, func(body []byte) error {
		var changed map[string]any
		if err := json.Unmarshal(body, &changed); err != nil {
			return Errorf("Invalid preferences: %v", err)
		}
		for k, v := range changed {
			a.prefs[k] = v
		}
		data, _ := json.MarshalIndent(a.prefs, "", "  ")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, data, 0o644)
	})
}

// Pref returns a preference set by the frontend (nil if unset or UsePrefs
// wasn't called), e.g. to answer in the user's language.
func (a *App) Pref(key string) any { return a.prefs[key] }

// Tr picks the text for the user's language (the "lang" preference, see
// GameApp.tr): English by default, Portuguese for "pt".
func (a *App) Tr(en, pt string) string {
	if a.Pref("lang") == "pt" {
		return pt
	}
	return en
}

// OnExit sets the hook that runs when AppLoad terminates the backend.
func (a *App) OnExit(f func()) { a.onExit = f }

// Send sends v as JSON. Failures are logged: there's nobody else to tell.
func (a *App) Send(msgType uint32, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		a.Log.Error("encoding message", "type", msgType, "err", err)
		return
	}
	if err := a.conn.Write(msgType, body); err != nil {
		a.Log.Warn("sending message", "type", msgType, "err", err)
	}
}

// Toast shows a message on the frontend.
func (a *App) Toast(t Toast) { a.Send(MsgToast, t) }

// Post runs f on the App's goroutine. It's safe to call from anywhere and
// does nothing once the App has stopped.
func (a *App) Post(f func()) {
	select {
	case a.posts <- f:
	case <-a.done:
	}
}

// Run handles messages until AppLoad terminates the backend or the socket closes.
func (a *App) Run() error {
	type message struct {
		msgType uint32
		body    []byte
	}
	msgs := make(chan message)
	readErr := make(chan error, 1)
	go func() {
		for {
			msgType, body, err := a.conn.Read()
			if err != nil {
				readErr <- err
				return
			}
			select {
			case msgs <- message{msgType, body}:
			case <-a.done:
				return
			}
		}
	}()

	defer func() {
		close(a.done)
		if a.onExit != nil {
			a.safely("exit", a.onExit)
		}
		a.conn.Close()
	}()

	for {
		select {
		case m := <-msgs:
			if m.msgType == MsgSystemTerminate {
				return nil
			}
			a.dispatch(m.msgType, m.body)
		case f := <-a.posts:
			a.safely("posted function", f)
		case err := <-readErr:
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
	}
}

func (a *App) dispatch(msgType uint32, body []byte) {
	switch msgType {
	case MsgHello, MsgSystemNewCoordinator:
		if a.prefs != nil {
			a.Send(MsgPrefs, a.prefs)
		}
		if a.onSync != nil {
			a.safely("sync", a.onSync)
		}
		return
	case MsgSystemLostCoordinator:
		return
	}
	h, ok := a.handlers[msgType]
	if !ok {
		a.Log.Warn("unhandled message", "type", msgType)
		return
	}
	a.safely(fmt.Sprintf("handler %d", msgType), func() {
		if err := h(body); err != nil {
			var toast Toast
			if !errors.As(err, &toast) {
				toast = Errorf("%v", err)
			}
			a.Toast(toast)
		}
	})
}

// safely runs f and turns a panic into a log entry and an error toast, so one
// bug doesn't take the whole backend down.
func (a *App) safely(what string, f func()) {
	defer func() {
		if r := recover(); r != nil {
			a.Log.Error("panic", "in", what, "err", r, "stack", string(debug.Stack()))
			a.Toast(Errorf("Internal error: %v", r))
		}
	}()
	f()
}
