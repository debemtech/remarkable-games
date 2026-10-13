package appload_test

import (
	"errors"
	"strings"
	"testing"

	"appload-games/framework/appload"
	"appload-games/framework/appload/apptest"
)

type echo struct {
	Text string `json:"text"`
}

func TestDispatch(t *testing.T) {
	app, host := apptest.New(t)
	synced := 0
	app.OnSync(func() { synced++; app.Send(200, echo{"synced"}) })
	appload.On(app, 100, func(e echo) error {
		if e.Text == "fail" {
			return errors.New("plain error")
		}
		if e.Text == "toast" {
			return appload.Info("custom")
		}
		app.Send(201, e)
		return nil
	})
	app.Command(101, func() error { panic("boom") })
	app.Command(102, func() error {
		go app.Post(func() { app.Send(202, echo{"posted"}) })
		return nil
	})
	exited := make(chan struct{})
	app.OnExit(func() { close(exited) })
	done := make(chan error, 1)
	go func() { done <- app.Run() }()

	// AppLoad's system messages and an empty-bodied hello must both sync.
	host.Send(appload.MsgSystemNewCoordinator, "1")
	apptest.Expect(host, 200, func(e echo) bool { return e.Text == "synced" })
	host.Send(appload.MsgHello, nil)
	apptest.Expect(host, 200, func(e echo) bool { return e.Text == "synced" })

	big := strings.Repeat("x", 100_000)
	host.Send(100, echo{big})
	if got := apptest.Expect[echo](host, 201, nil); got.Text != big {
		t.Fatalf("big message got mangled (%d bytes)", len(got.Text))
	}

	host.Send(100, echo{"fail"})
	if toast := host.ExpectToast(); toast.Kind != "error" || toast.Text != "plain error" {
		t.Fatalf("unexpected toast %+v", toast)
	}
	host.Send(100, echo{"toast"})
	if toast := host.ExpectToast(); toast.Kind != "info" || toast.Text != "custom" {
		t.Fatalf("unexpected toast %+v", toast)
	}
	host.Send(100, "not json")
	if toast := host.ExpectToast(); !strings.Contains(toast.Text, "Invalid request") {
		t.Fatalf("unexpected toast %+v", toast)
	}

	// A panicking handler is reported, and the app keeps running.
	host.Send(101, nil)
	if toast := host.ExpectToast(); !strings.Contains(toast.Text, "boom") {
		t.Fatalf("unexpected toast %+v", toast)
	}
	host.Send(102, nil)
	apptest.Expect(host, 202, func(e echo) bool { return e.Text == "posted" })

	host.Send(appload.MsgSystemTerminate, "close")
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	<-exited
	if synced != 2 {
		t.Fatalf("synced %d times, want 2", synced)
	}
	if !host.Closed() {
		t.Fatal("connection still open after terminate")
	}
}

func TestPrefs(t *testing.T) {
	path := t.TempDir() + "/prefs.json"
	for round := 0; round < 2; round++ {
		app, host := apptest.New(t)
		app.UsePrefs(path)
		done := make(chan error, 1)
		go func() { done <- app.Run() }()
		host.Send(appload.MsgHello, nil)
		prefs := apptest.Expect[map[string]any](host, appload.MsgPrefs, nil)
		if round == 0 {
			if len(prefs) != 0 {
				t.Fatalf("unexpected prefs %v", prefs)
			}
			host.Send(appload.MsgPrefs, map[string]any{"flip": true})
			host.Send(appload.MsgHello, nil)
			apptest.Expect(host, appload.MsgPrefs, func(p map[string]any) bool { return p["flip"] == true })
		} else if prefs["flip"] != true {
			t.Fatalf("prefs not saved: %v", prefs)
		}
		host.Send(appload.MsgSystemTerminate, "close")
		<-done
	}
}
