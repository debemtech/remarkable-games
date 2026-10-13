// Package stats has the bookkeeping most puzzle games need: a clock that only
// runs while the game is open, and per-variant records.
package stats

import (
	"time"

	"appload-games/framework/store"
)

// Clock measures play time. Embed it in a saved game; only Seconds is saved.
type Clock struct {
	Seconds int `json:"seconds"` // time played before the current session
	resumed time.Time
}

// Elapsed is the total time played, including the running session.
func (c *Clock) Elapsed() time.Duration {
	d := time.Duration(c.Seconds) * time.Second
	if !c.resumed.IsZero() {
		d += time.Since(c.resumed)
	}
	return d
}

// Now is Elapsed in whole seconds.
func (c *Clock) Now() int { return int(c.Elapsed() / time.Second) }

// Resume starts the clock. It does nothing if it's running already.
func (c *Clock) Resume() {
	if c.resumed.IsZero() {
		c.resumed = time.Now()
	}
}

// Pause stops the clock and adds the session to Seconds.
func (c *Clock) Pause() {
	if !c.resumed.IsZero() {
		c.Seconds = c.Now()
		c.resumed = time.Time{}
	}
}

// Record is the history of one variant (difficulty, layout, ...).
type Record struct {
	Played int `json:"played,omitempty"` // only counted by games that call Start
	Solved int `json:"solved"`
	Best   int `json:"best"` // seconds, 0 = none yet
}

// Records are kept per variant in a JSON file.
type Records struct {
	path string
	byID map[string]*Record
}

// LoadRecords reads the records at path. A missing or broken file starts empty.
func LoadRecords(path string) *Records {
	r := &Records{path: path, byID: map[string]*Record{}}
	store.Load(path, &r.byID)
	return r
}

// Get returns the record of a variant (zero if there is none).
func (r *Records) Get(id string) Record {
	if rec := r.byID[id]; rec != nil {
		return *rec
	}
	return Record{}
}

func (r *Records) get(id string) *Record {
	rec := r.byID[id]
	if rec == nil {
		rec = &Record{}
		r.byID[id] = rec
	}
	return rec
}

// Start counts a game as played, for a win rate.
func (r *Records) Start(id string) error {
	r.get(id).Played++
	return store.Save(r.path, r.byID)
}

// Win records a finished game. Only fair games (e.g. without hints) can set a best time.
func (r *Records) Win(id string, seconds int, fair bool) error {
	rec := r.get(id)
	rec.Solved++
	if fair && (rec.Best == 0 || seconds < rec.Best) {
		rec.Best = seconds
	}
	return store.Save(r.path, r.byID)
}
