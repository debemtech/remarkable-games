package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"appload-games/framework/stats"
	"appload-games/framework/store"
	"appload-games/games/crossword/backend/puzzle"
)

// library is the puzzles on the tablet: files the user copied to puzzles/,
// and downloads (stored as .ipuz in downloads/), with progress in progress.json.
type library struct {
	dir      string
	progress map[string]*progress
}

type progress struct {
	Entries  []string `json:"entries"`
	Revealed []int    `json:"revealed,omitempty"`
	Solved   bool     `json:"solved"`
	Played   int64    `json:"played"` // last opened, unix ms
	Title    string   `json:"title"`
	Open     int      `json:"open"` // cells to fill
	Size     string   `json:"size"`
	stats.Clock
}

func openLibrary(dir string) *library {
	l := &library{dir: dir, progress: map[string]*progress{}}
	os.MkdirAll(filepath.Join(dir, "puzzles"), 0o755)
	store.Load(filepath.Join(dir, "progress.json"), &l.progress)
	return l
}

func (l *library) saveProgress() error {
	// Clocks are saved with their current time without being stopped.
	snapshot := map[string]progress{}
	for k, p := range l.progress {
		cp := *p
		cp.Seconds = p.Now()
		snapshot[k] = cp
	}
	return store.Save(filepath.Join(l.dir, "progress.json"), snapshot)
}

// path checks a key ("puzzles/name.puz" or "downloads/name.ipuz") and returns its file.
func (l *library) path(key string) (string, error) {
	folder, name, ok := strings.Cut(key, "/")
	if !ok || (folder != "puzzles" && folder != "downloads") || name == "" ||
		name != filepath.Base(name) || strings.HasPrefix(name, ".") {
		return "", errors.New("invalid puzzle")
	}
	return filepath.Join(l.dir, folder, name), nil
}

func (l *library) load(key string) (*puzzle.Puzzle, error) {
	path, err := l.path(key)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, errors.New("the puzzle file is gone")
		}
		return nil, err
	}
	if strings.EqualFold(filepath.Ext(path), ".puz") {
		return puzzle.ParsePuz(data)
	}
	return puzzle.ParseIPuz(data)
}

// store saves a downloaded puzzle and returns its key.
func (l *library) store(name string, p *puzzle.Puzzle) (string, error) {
	data, err := p.IPuz()
	if err != nil {
		return "", err
	}
	key := "downloads/" + name + ".ipuz"
	path, err := l.path(key)
	if err != nil {
		return "", err
	}
	os.MkdirAll(filepath.Dir(path), 0o755)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	return key, os.Rename(tmp, path)
}

func (l *library) has(key string) bool {
	path, err := l.path(key)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func (l *library) remove(key string) error {
	path, err := l.path(key)
	if err != nil {
		return err
	}
	delete(l.progress, key)
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return l.saveProgress()
}

// list returns every puzzle, most recently played first, then newest files.
func (l *library) list(originName func(key string) string) []libraryItem {
	type found struct {
		item     libraryItem
		played   int64
		modified time.Time
	}
	var all []found
	for _, folder := range []string{"puzzles", "downloads"} {
		entries, _ := os.ReadDir(filepath.Join(l.dir, folder))
		for _, e := range entries {
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if e.IsDir() || (ext != ".puz" && ext != ".ipuz") {
				continue
			}
			key := folder + "/" + e.Name()
			info, _ := e.Info()
			f := found{item: libraryItem{Key: key, Origin: originName(key), Title: strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))}}
			if info != nil {
				f.modified = info.ModTime()
			}
			if pr := l.progress[key]; pr == nil {
				// Not played yet: the title is in the file
				if p, err := l.load(key); err == nil {
					if p.Title != "" {
						f.item.Title = p.Title
					}
					f.item.Size = fmt.Sprintf("%d×%d", p.Width, p.Height)
				}
			} else {
				f.played = pr.Played
				f.item.Solved = pr.Solved
				if pr.Title != "" {
					f.item.Title = pr.Title
				}
				filled := 0
				for _, e := range pr.Entries {
					if e != "" {
						filled++
					}
				}
				if pr.Open > 0 {
					f.item.Percent = 100 * filled / pr.Open
				}
				f.item.Size = pr.Size
			}
			all = append(all, f)
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].played != all[j].played {
			return all[i].played > all[j].played
		}
		return all[i].modified.After(all[j].modified)
	})
	items := make([]libraryItem, len(all))
	for i, f := range all {
		items[i] = f.item
	}
	return items
}

// open returns the progress of a puzzle, starting it if needed.
func (l *library) open(key string, p *puzzle.Puzzle) *progress {
	pr := l.progress[key]
	if pr == nil || len(pr.Entries) != len(p.Cells) {
		pr = &progress{Entries: make([]string, len(p.Cells))}
		l.progress[key] = pr
	}
	pr.Title = p.Title
	pr.Size = fmt.Sprintf("%d×%d", p.Width, p.Height)
	pr.Open = 0
	for _, c := range p.Cells {
		if !c.Block {
			pr.Open++
		}
	}
	pr.Played = time.Now().UnixMilli()
	return pr
}
