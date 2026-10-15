# Crossword

Crosswords with the pen, in English or Portuguese.

- **Write with the pen** straight into the squares, one capital letter each. A letter is read
  on the tablet itself (no internet needed) once the pen rests for a moment or moves on to
  another square. If it was misread, write over it. Fingers don't write. The backend reads the
  pen from the tablet itself (`framework/pen`): xochitl only passes apps a late copy of it.
- **Erase** with the other end of the pen: it clears every letter it passes over.
- **Undo** with a two finger tap (letters, erasing, reveals; while writing a letter, it drops
  its ink instead).
- Everything is black and white, which e-ink redraws fastest; the selection is gray and
  wrong letters (after *Check*) red. A quick tap on a square selects it (again: the other
  direction) and its clue in the list under the grid; tap a clue to select its answer.
  *Reveal letter* fills in the selected square.
- Tap the top edge for the buttons: *Check* marks wrong letters, *Clear wrong* erases them,
  *Reveal letter* fills in the selected square, *Menu* goes back.
- **Cruzadas diretas** (arrowwords): the clues are written in the grid, in the squares next
  to their answers; a small arrow shows where each answer starts and which way it goes. There's
  no clue list, so the grid sits in the middle of the screen with bigger squares. Tapping a
  clue square selects its answer (again: the other clue in that square); holding it, with the
  pen or a finger, shows its clues big until you let go.
- Other puzzles are shown the same way when all their clues fit in the squares (laid out by Qt) with letters
  at least 1.5 mm tall (mostly minis): a row of clue squares is added above the grid and a
  column on its left, so every answer has a square right before it for its clue. Otherwise the
  clues are listed under the grid. *Clues in grid* / *Clue list* in the top bar switches.
- Accents don't count: Portuguese answers are written without them (Ç is C), as usual in
  Portuguese crosswords.
- Progress and time are saved after every letter. *Continue* opens the last puzzle.
- English or Portuguese for the whole interface (chosen in the menu, remembered).

## Puzzles

*Get puzzles* downloads free puzzles (needs Wi-Fi). The sources are crosswords their
publishers offer to play for free on the web:

| Source | Language | |
|---|---|---|
| ValorFinal · Cruzada do dia, Mini do dia, Grande de domingo | Brazilian Portuguese | [valorfinal.com.br](https://valorfinal.com.br/jogos/palavras-cruzadas-do-dia): daily 11×11 and 5×5, weekly 13×13 (JSON files) |
| Cruzadas Clube · Diretas | Brazilian Portuguese | [cruzadasclube.com.br](https://cruzadasclube.com.br/jogo/categoria/id/4/n/cruzadas-diretas): cruzadas diretas, the 40 newest (older ones are for the site's subscribers and not offered) |
| Racha Cuca | Brazilian Portuguese | [rachacuca.com.br](https://rachacuca.com.br/palavras/palavras-cruzadas/): an archive of 55 13×13 puzzles |
| Paulo Freixinho · Palavras Cruzadas | European Portuguese | [palavrascruzadas.pt](https://www.palavrascruzadas.pt) |
| Crosshare Daily Mini | English | [crosshare.org](https://crosshare.org) (open `.puz` API) |
| Los Angeles Times, LA Times Mini | English | daily |
| Newsday | English | daily |
| The Atlantic | English | |
| Vox | English | |
| The Daily Beast | English | |
| The Walrus | English | weekly |

ValorFinal publishes its puzzles as plain JSON, Racha Cuca puts them in each puzzle's page.
Cruzadas Clube's are Crossword Compiler XML built by an obfuscated script, which the app runs
in an empty JavaScript interpreter ([goja](https://github.com/dop251/goja), with a time limit).
Freixinho and the English ones except Crosshare use the AmuseLabs (PuzzleMe) web solver, read the same way as
[xword-dl](https://github.com/thisisparker/xword-dl) does. Publishers can change their
sites at any time, so a source may stop working; `CROSSWORD_ONLINE=1 go test
./games/crossword/backend/sources` checks all of them. They're for your own solving.

Your own puzzles: copy `.puz` (Across Lite) or `.ipuz` files to
`~/.config/crossword/puzzles/`. Downloads are kept as `.ipuz` in `~/.config/crossword/downloads/`
(hold one in *My puzzles* to delete it). Progress is in `~/.config/crossword/progress.json`.

## Handwriting recognition

`backend/hwr` turns the pen strokes of one square into a 28×28 image the way the
[EMNIST](https://www.nist.gov/itl/products-and-services/emnist-dataset) letters were
made, then runs a small convolutional network (two convolutions, two dense layers,
830 KB of int8 weights in `model.bin`; the layer sizes are read from the file). It is
trained on EMNIST's letters, the capitals of its byclass split (what people write in a
crossword) and synthetic pen letters (`hwrtrain/letters`), which include the quick
forms: open O's, plain I's, A's with the bar anywhere or missing. It scores 98% on
EMNIST's capitals, and takes about 10 ms per letter on a PC.

Real handwriting it gets wrong is kept: when a letter is read as something other than
the square's answer and then written again in the same square and read right, both
writings go to `~/.config/crossword/samples.jsonl` (at most 2000). Train on them with:

```sh
scp remarkable:.config/crossword/samples.jsonl build/hwr/
go run ./games/crossword/hwrtrain/samples -o build/hwr/real.bin build/hwr/samples.jsonl
```

To retrain (needs PyTorch, the EMNIST letters and byclass splits and a GPU for speed):

```sh
go run ./games/crossword/hwrtrain/synth -n 104000 -o build/hwr/synth.bin   # synthetic pen letters
python3 games/crossword/hwrtrain/train.py --emnist path/to/emnist --synth build/hwr/synth.bin \
    --size l [--real build/hwr/real.bin]
```

That writes `model.bin` and `testdata/reference.bin`; `go test ./games/crossword/backend/hwr`
checks that the Go inference matches PyTorch.

## Code

| Path | |
|---|---|
| `backend/puzzle/` | `.puz`, `.ipuz`, AmuseLabs and Crossword Compiler data into one model; writes `.ipuz` (clues in the grid as `appload:` extensions) |
| `backend/sources/` | listing and downloading from the sources above |
| `backend/hwr/` | the letter recognizer |
| `backend/library.go` | puzzles on the tablet and their progress |
| `backend/main.go` | wiring |
| `hwrtrain/` | training: synthetic letters and logged samples (Go), the network (PyTorch) |
| `ui/GameScreen.qml` | grid with pen input, clues |
