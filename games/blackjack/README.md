# Blackjack

You against the dealer, played with the tablet on its side.

- Casino rules: a 6-deck shoe (shuffled when three quarters are dealt), the dealer
  stands on all 17s and peeks for blackjack, blackjack pays 3:2.
- Hit, stand, double on any first two cards, split up to four hands (doubling after a
  split is fine; split aces get one card each), and late surrender.
- *Hint* tells you the basic strategy move for the hand.
- After your turn the dealer turns over its cards one at a time.
- Bets from 10 to 500. You start with 1,000 chips; when they're gone you can start over.
  Chips, the shoe and a hand in progress are saved, and *Stats* shows your record.
- *Flip* turns the table around if you hold the tablet the other way.

## Files

`~/.config/blackjack/save.json`

## Code

| File | |
|---|---|
| `backend/rules.go` | hand values and the basic strategy table |
| `backend/table.go` | a round: betting, dealing, the player's moves, the dealer, payouts |
| `backend/main.go` | wiring, saving, and the dealer's card-by-card reveal |
| `backend/protocol.go` | message types and JSON payloads, mirrored in `ui/Protocol.js` |
| `ui/Table.qml` | the table screen |

The dealer plays its whole hand as soon as your turn ends, and the result is saved
right away. Turning the cards over one by one only affects what the frontend sees, so
closing the app mid-reveal never changes the outcome.

The cards come from the framework (`framework/cards`, `framework/qml/PlayingCard.qml`).
