package pen

import (
	"testing"
	"time"
)

func TestReader(t *testing.T) {
	var got []Event
	clock := time.Unix(0, 0)
	d := &Device{maxX: 1000, maxY: 2000, Every: 20 * time.Millisecond}
	r := &reader{d: d, send: func(e Event) { got = append(got, e) }, now: func() time.Time { return clock }}
	sample := func(x, y int32) {
		r.event(evAbs, absX, x)
		r.event(evAbs, absY, y)
		r.event(evSyn, 0, 0)
		clock = clock.Add(5 * time.Millisecond)
	}
	// Hover, touch down, move, lift, leave; then the eraser end.
	r.event(evKey, btnToolPen, 1)
	sample(100, 200)
	r.event(evKey, btnTouch, 1)
	sample(100, 200)
	for i := int32(1); i <= 8; i++ {
		sample(100+10*i, 200)
	}
	r.event(evKey, btnTouch, 0)
	sample(180, 200)
	r.event(evKey, btnToolPen, 0)
	r.event(evSyn, 0, 0)
	r.event(evKey, btnToolRub, 1)
	r.event(evKey, btnTouch, 1)
	sample(500, 1000)

	kinds := ""
	for _, e := range got {
		kinds += e.Kind + ":" + e.Tool + " "
	}
	want := "near:pen down:pen move:pen move:pen up:pen far:pen near:eraser down:eraser "
	if kinds != want {
		t.Fatalf("events %q, want %q", kinds, want)
	}
	if p := got[1].Points; p[0] != 0.1 || p[1] != 0.1 {
		t.Fatalf("down at %v", p)
	}
	n := 0
	for _, e := range got[2:5] {
		n += len(e.Points) / 2
	}
	if n != 8 {
		t.Fatalf("%d points moved, want 8", n)
	}
	if p := got[7].Points; p[0] != 0.5 || p[1] != 0.5 {
		t.Fatalf("eraser at %v", p)
	}
}

func TestHasKey(t *testing.T) {
	// The reMarkable Paper Pro's marker and touch screen (64 bit words)
	if !hasKey("1c03 0 0 0 0 0", btnToolPen) || !hasKey("1c03 0 0 0 0 0", btnToolRub) || hasKey("400 0 0 0 0 0", btnToolPen) {
		t.Fatal("wrong capability bits")
	}
}
