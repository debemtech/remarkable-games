// Package pen reads the tablet's pen straight from its input device. xochitl
// only hands apps a late, mouse-like copy of the pen, without telling the tip
// from the eraser; the device itself reports everything at full speed.
package pen

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"appload-games/framework/appload"
)

// Msg carries pen messages in both directions: the frontend sends
// {"on": true} to get Events while it wants the pen (GameApp.penActive) and
// {"on": false} to stop; the backend answers with a "ready" or "none" Event.
const Msg uint32 = 4

// Event is something the pen did. Points are x, y pairs in screen
// proportions: 0..1 from the left and from the top of the portrait screen.
type Event struct {
	Kind   string    `json:"e"`             // "near", "far" (hovering or not), "down", "move", "up"; "ready", "none"
	Tool   string    `json:"t,omitempty"`   // "pen" or "eraser"; the device for "ready"
	Points []float64 `json:"pts,omitempty"` // for down, move and up
	// Time is when the pen was there (the last point), in ms since 1970.
	Time int64 `json:"ms,omitempty"`
}

// Serve makes the pen available to the frontend (see Msg). The device is
// opened the first time the frontend asks for it.
func Serve(app *appload.App) {
	var on atomic.Bool
	var once sync.Once
	var dev *Device
	var err error
	appload.On(app, Msg, func(r struct {
		On bool `json:"on"`
	}) error {
		once.Do(func() {
			if dev, err = Open(); err != nil {
				app.Log.Warn("no pen, the frontend gets it from Qt", "err", err)
				return
			}
			go func() {
				err := dev.Run(func(e Event) {
					if on.Load() {
						app.Send(Msg, e)
					}
				})
				app.Log.Error("reading the pen stopped", "err", err)
			}()
		})
		on.Store(r.On)
		if dev == nil {
			app.Send(Msg, Event{Kind: "none"})
		} else {
			app.Send(Msg, Event{Kind: "ready", Tool: dev.Name()})
		}
		return nil
	})
}

// Device is an opened pen digitizer.
type Device struct {
	f          *os.File
	name       string
	maxX, maxY float64
	// rotated: the digitizer's axes are turned against the screen (reMarkable 2)
	rotated bool
	// Every is how often moves are sent while drawing.
	Every time.Duration
}

const (
	evSyn = 0
	evKey = 1
	evAbs = 3

	absX        = 0x00
	absY        = 0x01
	btnToolPen  = 0x140
	btnToolRub  = 0x141
	btnTouch    = 0x14a
	eventsGlob  = "/sys/class/input/event*"
	devicesRoot = "/dev/input"
)

// Open finds the pen digitizer (an input device with BTN_TOOL_PEN).
func Open() (*Device, error) {
	dirs, _ := filepath.Glob(eventsGlob)
	for _, dir := range dirs {
		name := strings.TrimSpace(read(filepath.Join(dir, "device/name")))
		if !hasKey(read(filepath.Join(dir, "device/capabilities/key")), btnToolPen) {
			continue
		}
		f, err := os.Open(filepath.Join(devicesRoot, filepath.Base(dir)))
		if err != nil {
			return nil, err
		}
		d := &Device{f: f, name: name, Every: 15 * time.Millisecond, rotated: strings.Contains(name, "Wacom")}
		d.maxX, d.maxY = absMax(f, absX), absMax(f, absY)
		if d.maxX <= 0 || d.maxY <= 0 {
			f.Close()
			return nil, fmt.Errorf("pen %q: no size", name)
		}
		return d, nil
	}
	return nil, errors.New("no pen found")
}

// Name is the device's name, e.g. "Elan marker input".
func (d *Device) Name() string { return d.name }

func read(path string) string {
	b, _ := os.ReadFile(path)
	return string(b)
}

// hasKey reads a capability bitmap from sysfs: hex words of the kernel's
// long size, most significant first.
func hasKey(bitmap string, key int) bool {
	words := strings.Fields(bitmap)
	size := bits.UintSize
	i := len(words) - 1 - key/size
	if i < 0 || i >= len(words) {
		return false
	}
	w, err := strconv.ParseUint(words[i], 16, 64)
	return err == nil && w&(1<<(key%size)) != 0
}

func absMax(f *os.File, axis uintptr) float64 {
	var info [6]int32            // value, minimum, maximum, fuzz, flat, resolution
	const eviocgabs = 0x80184540 // _IOR('E', 0x40 + axis, struct input_absinfo)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), eviocgabs+axis, uintptr(unsafe.Pointer(&info)))
	if errno != 0 {
		return 0
	}
	return float64(info[2])
}

// Run reads the pen until it fails, calling send for every event (from the
// calling goroutine).
func (d *Device) Run(send func(Event)) error {
	r := &reader{d: d, send: send}
	size := 8 + 2*bits.UintSize/8 // struct input_event: timeval, type, code, value
	buf := make([]byte, size*64)
	for {
		n, err := d.f.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		for i := 0; i+size <= n; i += size {
			ev := buf[i+size-8 : i+size]
			if bits.UintSize == 64 {
				r.stamp = int64(binary.LittleEndian.Uint64(buf[i:]))*1000 + int64(binary.LittleEndian.Uint64(buf[i+8:]))/1000
			} else {
				r.stamp = int64(binary.LittleEndian.Uint32(buf[i:]))*1000 + int64(binary.LittleEndian.Uint32(buf[i+4:]))/1000
			}
			r.event(binary.LittleEndian.Uint16(ev), binary.LittleEndian.Uint16(ev[2:]), int32(binary.LittleEndian.Uint32(ev[4:])))
		}
	}
}

// reader turns raw input events into Events.
type reader struct {
	d    *Device
	send func(Event)
	now  func() time.Time

	stamp      int64 // of the event being read, ms
	x, y       int32
	tool       string // in reach: "pen", "eraser" or ""
	sentTool   string
	touching   bool
	down       bool // a stroke is being sent
	pending    []float64
	lastSent   time.Time
	moved      bool
	lastX      float64
	lastY      float64
	keyPen     bool
	keyEraser  bool
	keyTouched bool
}

func (r *reader) event(typ, code uint16, value int32) {
	switch typ {
	case evAbs:
		switch code {
		case absX:
			r.x, r.moved = value, true
		case absY:
			r.y, r.moved = value, true
		}
	case evKey:
		switch code {
		case btnToolPen:
			r.keyPen = value != 0
		case btnToolRub:
			r.keyEraser = value != 0
		case btnTouch:
			r.keyTouched = value != 0
		}
	case evSyn:
		if code == 0 {
			r.report()
		}
	}
}

func (r *reader) point() (float64, float64) {
	x, y := float64(r.x)/r.d.maxX, float64(r.y)/r.d.maxY
	if r.d.rotated {
		x, y = y, 1-x
	}
	return min(max(x, 0), 1), min(max(y, 0), 1)
}

func (r *reader) clock() time.Time {
	if r.now != nil {
		return r.now()
	}
	return time.Now()
}

// report handles one complete reading of the pen (at SYN_REPORT).
func (r *reader) report() {
	tool := ""
	switch {
	case r.keyEraser:
		tool = "eraser"
	case r.keyPen:
		tool = "pen"
	}
	if r.down && (!r.keyTouched || tool != r.sentTool) {
		r.send(Event{Kind: "up", Tool: r.sentTool, Points: r.take(), Time: r.stamp})
		r.down = false
	}
	if tool != r.tool {
		if r.tool != "" {
			r.send(Event{Kind: "far", Tool: r.tool})
		}
		if tool != "" {
			r.send(Event{Kind: "near", Tool: tool})
		}
		r.tool = tool
	}
	if tool == "" {
		return
	}
	x, y := r.point()
	switch {
	case r.keyTouched && !r.down:
		r.down, r.sentTool = true, tool
		r.lastX, r.lastY = x, y
		r.pending = nil
		r.lastSent = r.clock()
		r.send(Event{Kind: "down", Tool: tool, Points: []float64{x, y}, Time: r.stamp})
	case r.down && r.moved && (x != r.lastX || y != r.lastY):
		r.pending = append(r.pending, x, y)
		r.lastX, r.lastY = x, y
		if now := r.clock(); now.Sub(r.lastSent) >= r.d.Every {
			r.lastSent = now
			r.send(Event{Kind: "move", Tool: tool, Points: r.take(), Time: r.stamp})
		}
	}
	r.moved = false
}

func (r *reader) take() []float64 {
	p := r.pending
	r.pending = nil
	return p
}
