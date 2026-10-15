// The backend of Crossword for AppLoad: a library of .puz/.ipuz puzzles,
// downloads from free online sources, and handwriting recognition.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"appload-games/framework/appload"
	"appload-games/framework/pen"
	"appload-games/framework/store"
	"appload-games/games/crossword/backend/hwr"
	"appload-games/games/crossword/backend/puzzle"
	"appload-games/games/crossword/backend/sources"

	// The tablet's own certificates may be missing or old; fall back to a
	// built-in set for HTTPS.
	_ "golang.org/x/crypto/x509roots/fallback"
)

type crosswordApp struct {
	app    *appload.App
	lib    *library
	model  *hwr.Model
	client *sources.Client

	key      string
	puz      *puzzle.Puzzle
	pr       *progress
	wrong    []int
	guesses  *guessView
	fetching bool
	history  []step // for undo, while the puzzle is open
	samples  *sampleLog
}

// step is what one action changed, to undo it.
type step struct {
	cells    []int
	before   []string
	revealed int // len(Revealed) before
}

const maxHistory = 500

// remember saves the squares an action is about to change.
func (c *crosswordApp) remember(join bool, cells ...int) {
	if join && len(c.history) > 0 {
		last := &c.history[len(c.history)-1]
		for _, i := range cells {
			last.cells = append(last.cells, i)
			last.before = append(last.before, c.pr.Entries[i])
		}
		return
	}
	st := step{revealed: len(c.pr.Revealed)}
	for _, i := range cells {
		st.cells = append(st.cells, i)
		st.before = append(st.before, c.pr.Entries[i])
	}
	if len(c.history) >= maxHistory {
		c.history = c.history[1:]
	}
	c.history = append(c.history, st)
}

func (c *crosswordApp) undo() error {
	return c.playing(func() error {
		if c.pr.Solved {
			return c.info("The puzzle is solved.", "A cruzada está resolvida.")
		}
		if len(c.history) == 0 {
			return c.info("Nothing to undo.", "Nada para desfazer.")
		}
		st := c.history[len(c.history)-1]
		c.history = c.history[:len(c.history)-1]
		for k := len(st.cells) - 1; k >= 0; k-- {
			c.pr.Entries[st.cells[k]] = st.before[k]
		}
		if st.revealed < len(c.pr.Revealed) {
			c.pr.Revealed = c.pr.Revealed[:st.revealed]
		}
		c.wrong, c.guesses = nil, nil
		c.changed()
		return nil
	})
}

func newCrosswordApp(app *appload.App, dir string, client *sources.Client) (*crosswordApp, error) {
	model, err := hwr.Load()
	if err != nil {
		return nil, err
	}
	c := &crosswordApp{app: app, lib: openLibrary(dir), model: model, client: client, samples: openSamples(dir)}
	app.UsePrefs(filepath.Join(dir, "prefs.json"))
	pen.Serve(app)
	app.OnSync(c.sync)
	app.OnExit(c.pause)
	app.Command(msgLibrary, func() error { c.sendLibrary(); return nil })
	appload.On(app, msgOpen, func(r keyRequest) error { return c.open(r.Key) })
	app.Command(msgSources, func() error {
		app.Send(msgSrcList, sourcesView{Sources: sources.Builtin})
		return nil
	})
	appload.On(app, msgListFrom, c.listFrom)
	appload.On(app, msgDownload, c.download)
	appload.On(app, msgSet, func(r setRequest) error { return c.set(r.Cell, r.Letter, false, r.Join) })
	appload.On(app, msgStrokes, c.strokes)
	app.Command(msgCheck, c.check)
	app.Command(msgUndo, c.undo)
	app.Command(msgClearWrong, func() error {
		return c.playing(func() error {
			if len(c.wrong) > 0 {
				c.remember(false, c.wrong...)
			}
			for _, i := range c.wrong {
				c.pr.Entries[i] = ""
			}
			c.wrong = nil
			c.changed()
			return nil
		})
	})
	appload.On(app, msgReveal, func(r setRequest) error { return c.reveal(r.Cell) })
	app.Command(msgClose, c.close)
	appload.On(app, msgDelete, func(r keyRequest) error {
		if !strings.HasPrefix(r.Key, "downloads/") {
			return c.info("Only downloaded puzzles can be deleted here.", "Só dá para apagar aqui as cruzadas baixadas.")
		}
		if r.Key == c.key {
			c.close()
		}
		if err := c.lib.remove(r.Key); err != nil {
			return err
		}
		c.sendLibrary()
		return nil
	})
	return c, nil
}

// info is a toast in the user's language.
func (c *crosswordApp) info(en, pt string, args ...any) error {
	return appload.Info("%s", fmt.Sprintf(c.app.Tr(en, pt), args...))
}

func (c *crosswordApp) sync() {
	c.sendMenu()
	if c.puz != nil {
		c.sendPuzzle()
		c.sendProgress()
	} else {
		c.app.Send(msgPuzzle, puzzleView{})
	}
}

func (c *crosswordApp) sendMenu() {
	m := menuView{}
	var best int64
	for key, pr := range c.lib.progress {
		if !pr.Solved && pr.Played > best && c.lib.has(key) {
			best, m.ContinueKey, m.ContinueTitle = pr.Played, key, pr.Title
		}
	}
	c.app.Send(msgMenu, m)
}

func (c *crosswordApp) origin(key string) string {
	folder, name, _ := strings.Cut(key, "/")
	if folder == "puzzles" {
		return c.app.Tr("Your files", "Seus arquivos")
	}
	for _, s := range sources.Builtin {
		if strings.HasPrefix(name, s.ID+"-") {
			return s.Name
		}
	}
	return c.app.Tr("Downloaded", "Baixada")
}

func (c *crosswordApp) sendLibrary() {
	c.app.Send(msgLibList, libraryView{Items: c.lib.list(c.origin), Dir: filepath.Join(c.lib.dir, "puzzles")})
}

func (c *crosswordApp) sendPuzzle() {
	p := c.puz
	v := puzzleView{Active: true, Key: c.key, Title: p.Title, Author: p.Author, Copyright: p.Copyright,
		Notes: p.Notes, Width: p.Width, Height: p.Height, Across: p.Across, Down: p.Down, HasSolution: p.HasSolution, Arrows: p.Arrows}
	for _, cell := range p.Cells {
		v.Cells = append(v.Cells, cellView{Block: cell.Block, Number: cell.Number, Circled: cell.Circled, HasClues: cell.HasClues})
	}
	c.app.Send(msgPuzzle, v)
}

func (c *crosswordApp) sendProgress() {
	pr := c.pr
	v := progressView{Entries: pr.Entries, Wrong: nonNil(c.wrong), Revealed: nonNil(pr.Revealed),
		Total: pr.Open, Solved: pr.Solved, Seconds: pr.Now(), Guesses: c.guesses}
	for _, e := range pr.Entries {
		if e != "" {
			v.Filled++
		}
	}
	c.app.Send(msgProgress, v)
}

func nonNil(s []int) []int {
	if s == nil {
		return []int{}
	}
	return s
}

func (c *crosswordApp) pause() {
	if c.pr != nil {
		c.pr.Pause()
		c.lib.saveProgress()
	}
}

func (c *crosswordApp) open(key string) error {
	p, err := c.lib.load(key)
	if err != nil {
		return appload.Errorf("%s: %v", c.app.Tr("Can't open the puzzle", "Não deu para abrir a cruzada"), err)
	}
	c.pause()
	c.key, c.puz, c.wrong, c.guesses, c.history = key, p, nil, nil, nil
	c.samples.reset()
	c.pr = c.lib.open(key, p)
	if !c.pr.Solved {
		c.pr.Resume()
	}
	c.lib.saveProgress()
	c.sendPuzzle()
	c.sendProgress()
	return nil
}

func (c *crosswordApp) close() error {
	c.pause()
	c.key, c.puz, c.pr, c.wrong, c.guesses, c.history = "", nil, nil, nil, nil, nil
	c.app.Send(msgPuzzle, puzzleView{})
	c.sendMenu()
	c.sendLibrary()
	return nil
}

func (c *crosswordApp) playing(action func() error) error {
	if c.puz == nil {
		return c.info("No puzzle is open.", "Nenhuma cruzada aberta.")
	}
	return action()
}

func (c *crosswordApp) cellOK(i int) bool {
	return i >= 0 && i < len(c.puz.Cells) && !c.puz.Cells[i].Block
}

func (c *crosswordApp) set(cell int, letter string, recognized, join bool) error {
	return c.playing(func() error {
		if !c.cellOK(cell) {
			return appload.Errorf("No such square %d", cell)
		}
		if c.pr.Solved {
			return c.info("The puzzle is solved.", "A cruzada está resolvida.")
		}
		if l := puzzle.Letter(letter); l != c.pr.Entries[cell] {
			c.remember(join, cell)
			c.pr.Entries[cell] = l
		}
		if !recognized {
			c.guesses = nil
		}
		c.wrong = nil
		c.changed()
		return nil
	})
}

func (c *crosswordApp) strokes(r strokesRequest) error {
	return c.playing(func() error {
		var strokes []hwr.Stroke
		for _, flat := range r.Strokes {
			var s hwr.Stroke
			for i := 0; i+1 < len(flat); i += 2 {
				s = append(s, hwr.Point{X: flat[i], Y: flat[i+1]})
			}
			if len(s) > 0 {
				strokes = append(strokes, s)
			}
		}
		guesses := c.model.Recognize(strokes, 4)
		if len(guesses) == 0 || !c.cellOK(r.Cell) {
			return nil
		}
		truth := ""
		if c.puz.HasSolution {
			truth = puzzle.Letter(c.puz.Cells[r.Cell].Solution)
		}
		if _, err := c.samples.written(c.key, r.Cell, r.Strokes, guesses, truth, time.Now()); err != nil {
			c.app.Log.Warn("saving a handwriting sample", "err", err)
		}
		c.guesses = &guessView{Cell: r.Cell, Letters: guesses}
		return c.set(r.Cell, guesses[0].Letter, true, false)
	})
}

// changed checks for a finished puzzle, saves and shows the progress.
func (c *crosswordApp) changed() {
	if c.puz.HasSolution && !c.pr.Solved {
		done := true
		for i, cell := range c.puz.Cells {
			if !cell.Block && c.pr.Entries[i] != puzzle.Letter(cell.Solution) {
				done = false
				break
			}
		}
		if done {
			c.pr.Pause()
			c.pr.Solved = true
		}
	}
	if err := c.lib.saveProgress(); err != nil {
		c.app.Toast(appload.Errorf("%s: %v", c.app.Tr("Saving failed", "Falha ao salvar"), err))
	}
	c.sendProgress()
}

func (c *crosswordApp) check() error {
	return c.playing(func() error {
		if !c.puz.HasSolution {
			return c.info("This puzzle came without its answers.", "Esta cruzada veio sem as respostas.")
		}
		c.wrong = nil
		for i, cell := range c.puz.Cells {
			if e := c.pr.Entries[i]; !cell.Block && e != "" && e != puzzle.Letter(cell.Solution) {
				c.wrong = append(c.wrong, i)
			}
		}
		c.sendProgress()
		switch n := len(c.wrong); n {
		case 0:
			return c.info("No mistakes so far.", "Nenhum erro até agora.")
		case 1:
			return c.info("1 letter is wrong.", "1 letra está errada.")
		default:
			return c.info("%d letters are wrong.", "%d letras estão erradas.", n)
		}
	})
}

func (c *crosswordApp) reveal(cell int) error {
	return c.playing(func() error {
		if !c.puz.HasSolution {
			return c.info("This puzzle came without its answers.", "Esta cruzada veio sem as respostas.")
		}
		if !c.cellOK(cell) {
			return appload.Errorf("No such square %d", cell)
		}
		if c.pr.Solved {
			return c.info("The puzzle is solved.", "A cruzada está resolvida.")
		}
		c.remember(false, cell)
		c.pr.Revealed = append(c.pr.Revealed, cell)
		c.pr.Entries[cell] = puzzle.Letter(c.puz.Cells[cell].Solution)
		c.wrong, c.guesses = nil, nil
		c.changed()
		return nil
	})
}

func (c *crosswordApp) listFrom(r sourceRequest) error {
	src, ok := sources.ByID(r.Source)
	if !ok {
		return appload.Errorf("Unknown source %q", r.Source)
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		list, err := c.client.List(ctx, src)
		c.app.Post(func() {
			v := entriesView{Source: src.ID, Name: src.Name, Entries: []entryView{}}
			if err != nil {
				v.Error = c.netError(err)
			}
			for _, e := range list {
				v.Entries = append(v.Entries, entryView{Entry: e, Downloaded: c.lib.has(downloadKey(src, e.ID))})
			}
			c.app.Send(msgEntries, v)
		})
	}()
	return nil
}

func downloadKey(src sources.Source, id string) string {
	return "downloads/" + src.ID + "-" + id + ".ipuz"
}

func (c *crosswordApp) download(r sourceRequest) error {
	src, ok := sources.ByID(r.Source)
	if !ok {
		return appload.Errorf("Unknown source %q", r.Source)
	}
	if key := downloadKey(src, r.ID); c.lib.has(key) {
		return c.open(key)
	}
	if c.fetching {
		return nil
	}
	c.fetching = true
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		p, err := c.client.Fetch(ctx, src, r.ID)
		c.app.Post(func() {
			c.fetching = false
			if err != nil {
				c.app.Toast(appload.Errorf("%s", c.netError(err)))
				return
			}
			key, err := c.lib.store(src.ID+"-"+r.ID, p)
			if err != nil {
				c.app.Toast(appload.Errorf("%s: %v", c.app.Tr("Saving failed", "Falha ao salvar"), err))
				return
			}
			if err := c.open(key); err != nil {
				c.app.Toast(appload.Errorf("%v", err))
			}
		})
	}()
	return nil
}

func (c *crosswordApp) netError(err error) string {
	var dns interface{ Temporary() bool }
	if errors.As(err, &dns) || strings.Contains(err.Error(), "dial tcp") || strings.Contains(err.Error(), "no such host") {
		return c.app.Tr("No internet connection. Turn on Wi-Fi and try again.", "Sem conexão com a internet. Ligue o Wi-Fi e tente de novo.")
	}
	return fmt.Sprintf("%s: %v", c.app.Tr("Download failed", "Falha no download"), err)
}

func main() {
	app, err := appload.FromArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := newCrosswordApp(app, store.Dir("crossword"), sources.NewClient()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		app.Log.Error("connection to AppLoad failed", "err", err)
		os.Exit(1)
	}
}
