// Package apptest plays AppLoad's side of the socket, so a game backend can be
// tested in-process:
//
//	app, host := apptest.New(t)
//	setup(app)            // register the game's handlers
//	go app.Run()
//	host.Send(appload.MsgHello, nil)
//	state := apptest.Expect(host, MsgState, func(s State) bool { return s.Ready })
package apptest

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	"appload-games/framework/appload"
)

// Message is one message received from the backend.
type Message struct {
	Type uint32
	Body []byte
}

// Host is the fake AppLoad end of the connection.
type Host struct {
	t       testing.TB
	conn    *net.UnixConn
	inbox   chan Message
	Timeout time.Duration
	// Seen collects everything received, for failure messages.
	Seen []Message
}

// New returns an App connected to a Host. The connection is closed when the
// test ends, which stops a running App.
func New(t testing.TB) (*appload.App, *Host) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app.sock")
	ln, err := net.ListenUnix("unixpacket", &net.UnixAddr{Name: path, Net: "unixpacket"})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	accepted := make(chan *net.UnixConn, 1)
	go func() {
		c, err := ln.AcceptUnix()
		if err != nil {
			t.Error(err)
		}
		accepted <- c
	}()
	conn, err := appload.Dial(path)
	if err != nil {
		t.Fatal(err)
	}
	host := &Host{t: t, conn: <-accepted, inbox: make(chan Message, 1024), Timeout: 10 * time.Second}
	go host.read()
	t.Cleanup(func() { host.conn.Close() })

	app := appload.New(conn)
	return app, host
}

func (h *Host) read() {
	buf := make([]byte, appload.MaxMessageSize)
	for {
		n, err := h.conn.Read(buf[:8])
		if err != nil || n != 8 {
			close(h.inbox)
			return
		}
		msgType := binary.NativeEndian.Uint32(buf[0:])
		length := binary.NativeEndian.Uint32(buf[4:])
		var body []byte
		if length > 0 { // Backends don't send empty bodies
			n, err = h.conn.Read(buf[:length])
			if err != nil {
				close(h.inbox)
				return
			}
			body = append([]byte(nil), buf[:n]...)
		}
		h.inbox <- Message{msgType, body}
	}
}

// Send sends v as JSON (nil sends an empty body), framed like AppLoad does.
func (h *Host) Send(msgType uint32, v any) {
	h.t.Helper()
	var body []byte
	switch v := v.(type) {
	case nil:
	case string:
		body = []byte(v)
	default:
		var err error
		if body, err = json.Marshal(v); err != nil {
			h.t.Fatal(err)
		}
	}
	h.SendRaw(msgType, body)
}

// SendRaw sends a body as is. Like AppLoad, it always sends the body packet.
func (h *Host) SendRaw(msgType uint32, body []byte) {
	h.t.Helper()
	var header [8]byte
	binary.NativeEndian.PutUint32(header[0:], msgType)
	binary.NativeEndian.PutUint32(header[4:], uint32(len(body)))
	if _, err := h.conn.Write(header[:]); err != nil {
		h.t.Fatal(err)
	}
	if body == nil {
		body = []byte{}
	}
	if _, err := h.conn.Write(body); err != nil {
		h.t.Fatal(err)
	}
}

// Next returns the next message, failing the test after Timeout.
func (h *Host) Next() Message {
	h.t.Helper()
	select {
	case m, ok := <-h.inbox:
		if !ok {
			h.t.Fatalf("backend closed the connection. Seen: %s", h.seen())
		}
		h.Seen = append(h.Seen, m)
		return m
	case <-time.After(h.Timeout):
		h.t.Fatalf("timed out waiting for a message. Seen: %s", h.seen())
	}
	return Message{}
}

// Closed reports whether the backend closed the connection within Timeout.
func (h *Host) Closed() bool {
	deadline := time.After(h.Timeout)
	for {
		select {
		case _, ok := <-h.inbox:
			if !ok {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

// Expect skips messages until one of msgType decodes into a T for which
// match returns true (match may be nil).
func Expect[T any](h *Host, msgType uint32, match func(T) bool) T {
	h.t.Helper()
	for {
		m := h.Next()
		if m.Type != msgType {
			continue
		}
		var v T
		if err := json.Unmarshal(m.Body, &v); err != nil {
			h.t.Fatalf("decoding message %d: %v (%s)", msgType, err, m.Body)
		}
		if match == nil || match(v) {
			return v
		}
	}
}

// ExpectToast waits for a toast and returns it.
func (h *Host) ExpectToast() appload.Toast {
	h.t.Helper()
	return Expect[appload.Toast](h, appload.MsgToast, nil)
}

func (h *Host) seen() string {
	s := ""
	for _, m := range h.Seen[max(0, len(h.Seen)-10):] {
		s += fmt.Sprintf("\n  %d %.200s", m.Type, m.Body)
	}
	return s
}
