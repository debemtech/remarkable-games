# Mahjong

Mahjong solitaire: clear the board by removing pairs of matching tiles.

- A tile can be taken if nothing lies on it and its left or right side is open.
  Blocked tiles are shaded (you can turn that off in the menu).
- Tap a tile, then its partner. Any flower matches any flower, any season any season.
- Three layouts: the classic *Turtle* (144 tiles), *Pyramid* (120) and *Quick* (56).
- Every deal can be cleared. *Shuffle* rearranges the remaining tiles into a position
  that can be cleared too, for when you get stuck.
- *Hint* shows a matching pair, *Undo* takes back moves (and shuffles).
- The game is saved after every move. The menu shows how often you cleared each
  layout, and your best time without hints or shuffles.

## Files

`~/.config/mahjong/save.json` (the current game) and `~/.config/mahjong/stats.json`.

## Code

| File | |
|---|---|
| `backend/board/` | layouts, the free-tile rule, deals and shuffles, no I/O |
| `backend/game.go` | a deal in progress: moves, undo, hints, clock |
| `backend/main.go` | wiring: handlers, saving, stats |
| `backend/protocol.go` | message types and JSON payloads, mirrored in `ui/Protocol.js` |
| `ui/MahjongBoard.qml` | draws the tiles |
| `make_tiles.py` | draws the 42 tile faces in `ui/res/tiles/` |

Deals are built backwards: random free pairs are taken off the full layout until it's
empty, then matching faces are put on each of those pairs. Clearing them in that order
always works, so no deal is a dead end from the start.

A layout is a list of positions in half-tile units, so new ones are a few lines in
`board.go`: tiles can sit half a tile apart, like the turtle's ends and top.

The faces' Chinese characters are outlines copied from Noto Sans CJK (SIL Open Font
License), so the tablet doesn't need a CJK font. Regenerate them with
`pip install fonttools && python3 games/mahjong/make_tiles.py`.
