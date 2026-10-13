// Package sources downloads crosswords that publishers offer to play for free
// on the web: AmuseLabs (PuzzleMe) solvers and Crosshare.
package sources

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"

	"appload-games/games/crossword/backend/puzzle"
)

// Source is a place to get puzzles from.
type Source struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"` // "en", "pt"
	Kind     string `json:"kind"`     // "amuse", "crosshare", "valorfinal", "rachacuca", "clube"
	Base     string `json:"-"`        // AmuseLabs: e.g. https://cdn2.amuselabs.com/pmm
	Set      string `json:"-"`        // AmuseLabs puzzle set; ValorFinal kind
}

// Builtin are the sources the app knows. AmuseLabs sets as used by xword-dl.
var Builtin = []Source{
	{ID: "valorfinal", Name: "ValorFinal · Cruzada do dia", Language: "pt", Kind: "valorfinal", Set: "classica"},
	{ID: "valorfinal-mini", Name: "ValorFinal · Mini do dia", Language: "pt", Kind: "valorfinal", Set: "mini"},
	{ID: "valorfinal-grande", Name: "ValorFinal · Grande de domingo", Language: "pt", Kind: "valorfinal", Set: "grande"},
	{ID: "rachacuca", Name: "Racha Cuca", Language: "pt", Kind: "rachacuca"},
	{ID: "clube-diretas", Name: "Cruzadas Clube · Diretas", Language: "pt", Kind: "clube"},
	{ID: "freixinho", Name: "Paulo Freixinho · Palavras Cruzadas", Language: "pt", Kind: "amuse",
		Base: "https://puzzleme.amuselabs.com/pmm", Set: "4d074386cb-palavrascruzadasserie2"},
	{ID: "crosshare-mini", Name: "Crosshare Daily Mini", Language: "en", Kind: "crosshare"},
	{ID: "latimes", Name: "Los Angeles Times", Language: "en", Kind: "amuse", Base: "https://lat.amuselabs.com/lat", Set: "latimes"},
	{ID: "latimes-mini", Name: "Los Angeles Times Mini", Language: "en", Kind: "amuse", Base: "https://lat.amuselabs.com/lat", Set: "latimes-mini"},
	{ID: "newsday", Name: "Newsday", Language: "en", Kind: "amuse", Base: "https://cdn2.amuselabs.com/pmm", Set: "creatorsweb"},
	{ID: "atlantic", Name: "The Atlantic", Language: "en", Kind: "amuse", Base: "https://cdn3.amuselabs.com/atlantic", Set: "atlantic"},
	{ID: "vox", Name: "Vox", Language: "en", Kind: "amuse", Base: "https://cdn3.amuselabs.com/vox", Set: "vox"},
	{ID: "dailybeast", Name: "The Daily Beast", Language: "en", Kind: "amuse", Base: "https://cdn3.amuselabs.com/tdb", Set: "tdb"},
	{ID: "walrus", Name: "The Walrus", Language: "en", Kind: "amuse", Base: "https://cdn2.amuselabs.com/pmm", Set: "walrus-weekly-crossword"},
}

// ByID returns a builtin source.
func ByID(id string) (Source, bool) {
	for _, s := range Builtin {
		if s.ID == id {
			return s, true
		}
	}
	return Source{}, false
}

// Entry is a puzzle a source offers.
type Entry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Date  int64  `json:"date"` // ms since 1970, 0 if unknown
	Size  string `json:"size"` // e.g. "15×15"
}

// Client downloads puzzles. Its methods are safe for concurrent use.
type Client struct {
	http      *http.Client
	UserAgent string
}

func NewClient() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		http:      &http.Client{Jar: jar, Timeout: 30 * time.Second},
		UserAgent: "Mozilla/5.0 (reMarkable; appload-games crossword)",
	}
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %s", req.URL.Host, res.Status)
	}
	return io.ReadAll(io.LimitReader(res.Body, 20<<20))
}

// List returns the puzzles a source offers, newest first.
func (c *Client) List(ctx context.Context, s Source) ([]Entry, error) {
	switch s.Kind {
	case "amuse":
		return c.amuseList(ctx, s)
	case "crosshare":
		return c.crosshareList(ctx)
	case "valorfinal":
		return c.valorfinalList(ctx, s)
	case "rachacuca":
		return c.rachacucaList(ctx)
	case "clube":
		return c.clubeList(ctx)
	}
	return nil, errors.New("unknown source")
}

// Fetch downloads one puzzle.
func (c *Client) Fetch(ctx context.Context, s Source, id string) (*puzzle.Puzzle, error) {
	switch s.Kind {
	case "amuse":
		return c.amuseFetch(ctx, s, id)
	case "crosshare":
		if !safeID.MatchString(id) {
			return nil, errors.New("invalid puzzle id")
		}
		data, err := c.get(ctx, "https://crosshare.org/api/puz/"+id)
		if err != nil {
			return nil, err
		}
		return puzzle.ParsePuz(data)
	case "valorfinal":
		return c.valorfinalFetch(ctx, id)
	case "rachacuca":
		return c.rachacucaFetch(ctx, id)
	case "clube":
		return c.clubeFetch(ctx, id)
	}
	return nil, errors.New("unknown source")
}

var (
	safeID      = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	paramsTag   = regexp.MustCompile(`(?s)<script[^>]*id="params"[^>]*>(.*?)</script>`)
	nextDataTag = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__"[^>]*>(.*?)</script>`)
)

// AmuseLabs pages carry their data as JSON in <script id="params">.
func amuseParams(page []byte) map[string]json.RawMessage {
	m := paramsTag.FindSubmatch(page)
	if m == nil {
		return nil
	}
	var p map[string]json.RawMessage
	json.Unmarshal(m[1], &p)
	return p
}

func (c *Client) amusePicker(ctx context.Context, s Source) (map[string]json.RawMessage, error) {
	page, err := c.get(ctx, s.Base+"/date-picker?set="+s.Set)
	if err != nil {
		return nil, err
	}
	p := amuseParams(page)
	if p == nil {
		return nil, errors.New("couldn't read the list of puzzles")
	}
	return p, nil
}

func (c *Client) amuseList(ctx context.Context, s Source) ([]Entry, error) {
	p, err := c.amusePicker(ctx, s)
	if err != nil {
		return nil, err
	}
	var streak []struct {
		PuzzleDetails struct {
			PuzzleID        string `json:"puzzleId"`
			Title           string `json:"title"`
			GridWidth       int    `json:"gridWidth"`
			GridHeight      int    `json:"gridHeight"`
			PublicationTime int64  `json:"publicationTime"`
		} `json:"puzzleDetails"`
	}
	json.Unmarshal(p["streakInfo"], &streak)
	var out []Entry
	for _, e := range streak {
		d := e.PuzzleDetails
		if d.PuzzleID == "" {
			continue
		}
		out = append(out, Entry{ID: d.PuzzleID, Title: puzzle.CleanText(d.Title), Date: d.PublicationTime,
			Size: fmt.Sprintf("%d×%d", d.GridWidth, d.GridHeight)})
	}
	if len(out) == 0 {
		return nil, errors.New("no puzzles listed")
	}
	return out, nil
}

func charsum(s string) uint32 {
	var t uint32
	for _, r := range s {
		t += uint32(r)
	}
	return t
}

func (c *Client) amuseFetch(ctx context.Context, s Source, id string) (*puzzle.Puzzle, error) {
	if !safeID.MatchString(id) {
		return nil, errors.New("invalid puzzle id")
	}
	url := s.Base + "/crossword?id=" + id + "&set=" + s.Set
	rawc, err := c.amuseRawc(ctx, url)
	if err != nil {
		return nil, err
	}
	if rawc == "" {
		// Most sets want the load token from the picker, plus a hash that
		// includes the session cookie (computed like their picker script does).
		p, err := c.amusePicker(ctx, s)
		if err != nil {
			return nil, err
		}
		var sps struct {
			LoadToken string `json:"loadToken"`
		}
		var encoded string
		json.Unmarshal(p["rawsps"], &encoded)
		if raw, err := base64.StdEncoding.DecodeString(encoded); err == nil {
			json.Unmarshal(raw, &sps)
		}
		if sps.LoadToken != "" {
			url += "&loadToken=" + sps.LoadToken
			if u, err := http.NewRequest(http.MethodGet, s.Base, nil); err == nil {
				for _, ck := range c.http.Jar.Cookies(u.URL) {
					if ck.Name == "uid" {
						url += fmt.Sprintf("&fvlt=%x", charsum(s.Set)^charsum(id)^charsum(ck.Value))
					}
				}
			}
			if rawc, err = c.amuseRawc(ctx, url); err != nil {
				return nil, err
			}
		}
	}
	if rawc == "" {
		return nil, errors.New("the puzzle isn't available (maybe for subscribers only)")
	}
	data, err := Deobfuscate(rawc)
	if err != nil {
		return nil, err
	}
	return puzzle.FromAmuse(data)
}

func (c *Client) amuseRawc(ctx context.Context, url string) (string, error) {
	page, err := c.get(ctx, url)
	if err != nil {
		return "", err
	}
	var rawc string
	if p := amuseParams(page); p != nil {
		json.Unmarshal(p["rawc"], &rawc)
	}
	return rawc, nil
}

func (c *Client) crosshareList(ctx context.Context) ([]Entry, error) {
	page, err := c.get(ctx, "https://crosshare.org/dailyminis")
	if err != nil {
		return nil, err
	}
	m := nextDataTag.FindSubmatch(page)
	if m == nil {
		return nil, errors.New("couldn't read the list of puzzles")
	}
	var data struct {
		Props struct {
			PageProps struct {
				Puzzles [][]json.RawMessage `json:"puzzles"` // [day, puzzle]
			} `json:"pageProps"`
		} `json:"props"`
	}
	if err := json.Unmarshal(m[1], &data); err != nil {
		return nil, err
	}
	var out []Entry
	for _, item := range data.Props.PageProps.Puzzles {
		if len(item) < 2 {
			continue
		}
		var p struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			AuthorName  string `json:"authorName"`
			PublishTime int64  `json:"publishTime"`
			Size        struct{ Rows, Cols int }
		}
		if json.Unmarshal(item[1], &p) == nil && p.ID != "" {
			title := p.Title
			if p.AuthorName != "" {
				title += " · " + p.AuthorName
			}
			out = append(out, Entry{ID: p.ID, Title: title, Date: p.PublishTime, Size: fmt.Sprintf("%d×%d", p.Size.Cols, p.Size.Rows)})
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no puzzles listed")
	}
	return out, nil
}

// Deobfuscate decodes AmuseLabs' "rawc" puzzle data: base64, with chunks of
// the string reversed according to a 7 digit key that isn't sent along. The
// key is found by search, pruned by checking that every chunk decodes to
// plausible UTF-8. Adapted from kotwords and xword-dl (Apache-2.0).
func Deobfuscate(rawc string) ([]byte, error) {
	rawc = strings.TrimSpace(rawc)
	if rawc == "" {
		return nil, errors.New("no puzzle data")
	}
	if data, err := base64.StdEncoding.DecodeString(rawc); err == nil && json.Valid(data) {
		return data, nil // not obfuscated
	}
	// "ye" and "we" are the base64 of `{"` and `{\n` reversed, at the start.
	first := len(rawc)
	if i := strings.Index(rawc, "ye"); i >= 0 {
		first = i
	}
	if i := strings.Index(rawc, "we"); i >= 0 && i < first {
		first = i
	}
	queue := [][]int{{}}
	if first+2 <= 20 {
		queue = [][]int{{first + 2}}
	}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if len(key) == 7 {
			if data := decodeWithKey(rawc, key); data != nil && json.Valid(data) {
				return data, nil
			}
			continue
		}
		for d := 2; d <= 20; d++ {
			next := append(append([]int(nil), key...), d)
			rest := 7 - len(next)
			for spacing := 2 * rest; spacing <= 20*rest; spacing++ {
				if validPrefix(rawc, next, spacing) {
					queue = append(queue, next)
					break
				}
			}
		}
	}
	return nil, errors.New("couldn't decode the puzzle data")
}

func reverse(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

func validPrefix(rawc string, key []int, spacing int) bool {
	pos := 0
	for pos < len(rawc) {
		start := pos
		var chunk strings.Builder
		for k := 0; k < len(key) && pos < len(rawc); k++ {
			n := min(key[k], len(rawc)-pos)
			chunk.WriteString(reverse(rawc[pos : pos+n]))
			pos += n
		}
		s := chunk.String()
		from := (start+3)/4*4 - start
		to := pos/4*4 - start
		if from >= len(s) || to <= from {
			pos += spacing
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(s[from:to])
		if err != nil {
			return false
		}
		for _, b := range decoded {
			if (b < 32 && b != '\t' && b != '\n' && b != '\r') || b == 0xC0 || b == 0xC1 || b >= 0xF5 {
				return false
			}
		}
		pos += spacing
	}
	return true
}

func decodeWithKey(rawc string, key []int) []byte {
	b := []byte(rawc)
	for i, seg := 0, 0; i < len(b)-1; seg++ {
		n := min(key[seg%len(key)], len(b)-i)
		for l, r := i, i+n-1; l < r; l, r = l+1, r-1 {
			b[l], b[r] = b[r], b[l]
		}
		i += n
	}
	data, err := base64.StdEncoding.DecodeString(string(b))
	if err != nil {
		return nil
	}
	return data
}
