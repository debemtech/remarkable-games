# Sudoku

- Four difficulties. Easy and Medium can be solved by only filling in cells with a
  single possible number; Hard and Expert need more than that. Expert has the fewest clues
  (usually 23–26). Every puzzle has exactly one solution.
- Tap a cell, then a number. Tapping the number that's already there erases it.
- *Notes* switches the number pad to pencil marks. Placing a number removes it from the
  notes in its row, column and box.
- Numbers that clash with another in the same row, column or box get a ring. *Check*
  also rings the numbers that don't match the solution.
- *Hint* fills in the selected cell (or the first empty or wrong one).
- The game is saved after every change; *Continue* picks it up again. The clock only runs
  while the game is open.
- The menu shows how many puzzles you solved per difficulty, and your best time without hints.

## Files

`~/.config/sudoku/save.json` (the current game) and `~/.config/sudoku/stats.json`.

## Backend

| File | |
|---|---|
| `sudoku/` | solver, generator and difficulty rating, no I/O |
| `game.go` | a puzzle in progress: entries, notes, undo, hints, clock |
| `main.go` | wiring: handlers, saving, stats |
| `protocol.go` | message types and JSON payloads, mirrored in `ui/Protocol.js` |

Puzzles are made by filling a grid at random and removing clues in pairs mirrored around
the center, as long as the solution stays unique (and, for Easy/Medium, solvable with
singles). That takes well under a millisecond for Easy and about 12 ms for Expert on a PC.
