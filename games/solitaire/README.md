# Solitaire

Klondike and FreeCell in one app, played with the tablet on its side.

- **Klondike** (draw 1 or draw 3): build the four suits up from the ace. Tap the stock to
  turn cards over; when it's empty, tap it again to turn the waste back. Only kings go into
  empty columns.
- **FreeCell**: all cards face up, four free cells to park cards, any card can go into an
  empty column. Deals use Microsoft's numbering (`FreeCell #1` to `#32000`), and every one
  of them can be won except #11982, which is never dealt.

Tap a card to pick it up (in a column, together with the cards on it), then tap where it
should go. Tap it again to send it to the best place: its foundation, a column, or a free
cell. Cards that can't be needed anymore go to the foundations by themselves.

*Hint* marks a move, *Undo* takes moves back, *Restart* deals the same cards again.
The game is saved after every move, and the menu shows how many games of each kind you won
and your best time without hints. *Flip* in the menu turns the table around; it's remembered.

## Files

`~/.config/solitaire/save.json`, `stats.json` and `prefs.json`.

## Code

| File | |
|---|---|
| `backend/solitaire/` | the rules of both games: deals, moves, autoplay, hints, no I/O |
| `backend/main.go` | wiring: handlers, saving, stats |
| `backend/protocol.go` | message types and JSON payloads, mirrored in `ui/Protocol.js` |
| `ui/GameScreen.qml` | the table: lays out the piles for both games, tap to move |

Both games are one set of piles (stock, waste, foundations, columns, free cells) with a
few rules switched on or off, so a new variant (Spider, Yukon, ...) mostly needs its deal
and its rules for what may go where.
