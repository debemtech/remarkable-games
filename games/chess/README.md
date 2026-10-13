# Chess

A port of [chessMarkable](https://github.com/LinusCDE/chessmarkable) by LinusCDE.

- Player vs Player, optionally with the pieces turned around on black's turn for two players sitting opposite each other
- Player vs Bot (Easy / Normal / Hard), played by Stockfish 19
- 3 save slots, with every move saved right away
- PGN viewer: step through games and try your own moves in between
- Undo, also after reopening a saved game

Move a piece by tapping it and then its destination, or by pressing it and lifting the
finger on the destination. Close the app with *Exit* or AppLoad's swipe from the top center.

## Files

- `~/.config/chess/slots.json`: the save slots (starting position and moves).
  chessMarkable's `~/.config/chessmarkable/savestates.yml` is imported on the first start.
- `~/.config/chess/pgn/*.pgn`: games for the PGN viewer.

## Backend

| File | |
|---|---|
| `main.go` | wiring: handlers, autosave, running the bot |
| `match.go` | a game in progress: rules, undo, status |
| `bot.go` | Stockfish process and difficulty levels |
| `uci/` | UCI client for any engine |
| `slots.go` | save slots |
| `replay.go` | PGN files and replays |
| `protocol.go` | message types and JSON payloads, mirrored in `ui/Protocol.js` |

Difficulty is Stockfish's *Skill Level* plus a time limit per move (`levels` in `bot.go`).
The bot waits at least a second before answering, so its move is easy to follow on e-ink.

`stockfish.sh` builds a static Stockfish from the official source (version pinned in
the script); `build-extra.sh` puts it next to the backend.
