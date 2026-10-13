# AppLoad games for reMarkable

Games for the **reMarkable Paper Pro** (and rM2 with AppLoad), running inside
xochitl through [AppLoad](https://github.com/asivery/rmpp-appload). Each game is
a QML frontend plus a Go backend, built on a small shared framework.

| Game | |
|---|---|
| [chess](games/chess) | Chess with [Stockfish](https://stockfishchess.org) as the bot, save slots and a PGN viewer |
| [sudoku](games/sudoku) | Generated puzzles in four difficulties, with notes, hints and stats |
| [mahjong](games/mahjong) | Mahjong solitaire on three layouts; every deal can be cleared |
| [blackjack](games/blackjack) | You against the dealer, with basic strategy hints; landscape |
| [solitaire](games/solitaire) | Klondike (draw 1 or 3) and FreeCell with Microsoft deal numbers; landscape |
| [crossword](games/crossword) | Crosswords with on-device handwriting recognition, free puzzles in English and Portuguese, `.puz`/`.ipuz` |

## Requirements on the tablet

- [xovi](https://github.com/asivery/xovi) with [qt-resource-rebuilder](https://github.com/asivery/rmpp-xovi-extensions/tree/master/qt-resource-rebuilder)
- [AppLoad](https://github.com/asivery/rmpp-appload)

## Building and installing

You need Go (see `go.mod`) and Qt 6's `rcc` (`qt6-base-dev-tools` on Debian/Ubuntu).
Backends are pure Go (`CGO_ENABLED=0`): static binaries with no cross compiler and
no dependence on the tablet's glibc.

```sh
./build.sh sudoku         # Paper Pro -> build/sudoku-rmpp
./build.sh sudoku rm2     # reMarkable 2 -> build/sudoku-rm2

ssh root@10.11.99.1 'rm -rf /home/root/xovi/exthome/appload/sudoku'
scp -r build/sudoku-rmpp root@10.11.99.1:/home/root/xovi/exthome/appload/sudoku
```

Then press reload in AppLoad. Remove the old copy first as shown, otherwise `scp -r`
puts the new one *inside* it, where AppLoad doesn't look.

Chess also needs a C++ cross compiler for Stockfish (`g++-aarch64-linux-gnu`, or
`g++-arm-linux-gnueabihf` for the rM2). `games/chess/stockfish.sh` builds it once and
caches it in `build/cache`; set `STOCKFISH_BIN` to use a binary you already have.

## Layout

```
build.sh                  builds any game: ./build.sh <game> [rmpp|rm2]
framework/
  appload/                Go: AppLoad socket, message dispatch, toasts
  appload/apptest/        Go: fake AppLoad for in-process tests
  store/                  Go: JSON files in ~/.config/<app>, written atomically
  stats/                  Go: play clock and per-variant records (solved, best time)
  cards/                  Go: playing cards and a shuffled multi-deck shoe
  qml/                    QML: GameApp, MenuButton, PagedList, CheckRow, PlayingCard
games/<game>/
  manifest.json icon.png
  ui/                     QML; the framework is available as ui/framework/
  backend/                Go main package
  build-extra.sh          optional build hook (chess uses it to add Stockfish)
tools/emulator.py         run a game on a PC with PySide6
```

## Writing a game

### Conventions

- Messages carry JSON in both directions.
- Types 1–99 belong to the framework: `1` hello (frontend → backend), `2` toast (backend → frontend),
  `3` preferences (both ways, see below).
- Games use 100–199 for requests and 200–299 for updates.
- **The backend owns the state; the frontend renders it.** After every change the
  backend sends the full state of what changed (for chess: the whole board).
  It's simple, cheap at these sizes, and a reconnecting frontend is never out of sync.

### Backend

```go
package main

import "appload-games/framework/appload"

const (
	msgNewPuzzle = 100
	msgSetCell   = 101
	msgState     = 200
)

type game struct {
	app   *appload.App
	cells [81]int
}

type setCell struct{ Index, Value int }

func main() {
	app, err := appload.FromArgs()
	if err != nil {
		panic(err)
	}
	g := &game{app: app}
	app.OnSync(g.send) // a frontend connected: send everything
	app.Command(msgNewPuzzle, g.newPuzzle)
	appload.On(app, msgSetCell, func(req setCell) error {
		if req.Index < 0 || req.Index >= 81 {
			return appload.Errorf("No such cell") // shown as a toast
		}
		g.cells[req.Index] = req.Value
		g.send()
		return nil
	})
	app.Run()
}

func (g *game) send() { g.app.Send(msgState, g.cells) }

func (g *game) newPuzzle() error {
	go func() { // slow work runs in its own goroutine...
		cells := generate()
		g.app.Post(func() { g.cells = cells; g.send() }) // ...and hands the result back
	}()
	return nil
}
```

Handlers, `OnSync`, `OnExit` and functions passed to `Post` all run on one goroutine,
so game state needs no locks. A handler's error becomes a toast (`appload.Info`,
`appload.Errorf` or any error); a panic is caught, logged and shown as an error.
Save with `store.Save(filepath.Join(store.Dir("sudoku"), "game.json"), v)`. For puzzle
games, embed `stats.Clock` in the saved game (time only counts while it's open) and
keep wins in `stats.Records`.

Test the backend in-process with `apptest`, which plays AppLoad's side of the socket
(see `games/sudoku/backend/main_test.go`):

```go
app, host := apptest.New(t)
setup(app)
go app.Run()
host.Send(msgSetCell, setCell{Index: 0, Value: 5})
state := apptest.Expect(host, msgState, func(cells [81]int) bool { return cells[0] == 5 })
```

### Frontend

`ui/main.qml` is a `GameApp`. It talks to the backend, shows toasts and provides
`u`, 1% of the shorter screen side, for all sizes:

```qml
import QtQuick 2.5
import "framework"

GameApp {
    id: app
    appId: "sudoku"           // same as in manifest.json
    property var cells: []
    onMessage: (type, data) => { if (type === 200) cells = data }

    MenuButton {
        anchors.centerIn: parent
        fontSize: 5 * app.u
        text: "New puzzle"
        onClicked: app.send(100)
    }
}
```

`app.send(type, payload)` sends JSON, `app.showToast(text, kind, ms)` shows a message
locally, `app.screen` is a free string for switching screens. For a game played with the
tablet on its side, set `orientation: "landscape"`: the content is turned by 90° and
laid out in landscape size (`app.toggleFlip()` turns it the other way).

Settings the user picks can be kept with `app.setPref(key, value)` and read from
`app.prefs` if the backend calls `app.UsePrefs(path)`. The landscape flip is stored there.

Languages: `app.tr("Check", "Verificar")` picks English or Portuguese by the `lang`
preference (`app.setPref("lang", "pt")`); the backend has the same as `app.Tr(en, pt)`. Keep e-ink in mind:
black on white, no animations, and pages instead of scrolling (`PagedList`).

`manifest.json` needs `"loadsBackend": true` and an `"id"` matching `appId`.

## Testing

```sh
go test ./...                                   # framework and backends
CHESS_STOCKFISH=$(games/chess/stockfish.sh host) go test ./...   # include games against Stockfish
```

With qemu-user and binfmt, `GOARCH=arm64 go test ./...` runs the tests as the binaries
that ship to the tablet.

`tools/emulator.py` runs a game on a PC: it mounts `resources.rcc`, provides the
`net.asivery.AppLoad` QML module and connects the UI to a real backend. It can also
play back taps, mouse and pen strokes (also the eraser end), finger touches and
two finger taps, and take screenshots, headless with `QT_QPA_PLATFORM=offscreen`.
See the header of the file for usage.

## Credits and licenses

- Crossword: the letter recognizer is trained on [EMNIST](https://www.nist.gov/itl/products-and-services/emnist-dataset) (NIST);
  [goja](https://github.com/dop251/goja) (MIT) runs Cruzadas Clube's puzzle scripts;
  the AmuseLabs decoding follows [kotwords](https://github.com/jpd236/kotwords) and [xword-dl](https://github.com/thisisparker/xword-dl) (Apache-2.0).
  Downloaded puzzles belong to their publishers.
- The card suits are outlines from [DejaVu Sans](https://dejavu-fonts.github.io) (`tools/make_suits.py`).
- The mahjong tiles' characters are outlines from [Noto Sans CJK](https://github.com/notofonts/noto-cjk) (SIL Open Font License).
- The chess pieces are [Cburnett's](https://commons.wikimedia.org/wiki/Category:SVG_chess_pieces) from Wikimedia Commons (GPL-2.0-or-later, also BSD, GFDL and CC BY-SA 3.0).
- [notnil/chess](https://github.com/notnil/chess) (MIT): chess rules and PGN parsing.
- [Stockfish](https://github.com/official-stockfish/Stockfish) (GPL-3.0): built from the official source by `games/chess/stockfish.sh` and shipped as a separate program.
- [AppLoad](https://github.com/asivery/rmpp-appload) and [appload-rmstream](https://github.com/asivery/appload-rmstream) by asivery.

This project is under the GPL-3.0 (`LICENSE`).
