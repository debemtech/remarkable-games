package main

import "appload-games/framework/cards"

// cardValue counts an ace as 1; value() makes it 11 when that doesn't bust.
func cardValue(c cards.Card) int { return min(int(c.Rank), 10) }

// value returns the best total of a hand and whether an ace counts as 11 (a soft hand).
func value(cs []cards.Card) (total int, soft bool) {
	aces := false
	for _, c := range cs {
		total += cardValue(c)
		aces = aces || c.Rank == 1
	}
	if aces && total+10 <= 21 {
		return total + 10, true
	}
	return total, false
}

func isBlackjack(cs []cards.Card) bool {
	total, _ := value(cs)
	return len(cs) == 2 && total == 21
}

// Basic strategy for a 6-deck shoe where the dealer stands on soft 17, doubling
// after a split is allowed and late surrender is offered.
const (
	hit       = "Hit"
	stand     = "Stand"
	double    = "Double"
	split     = "Split"
	surrender = "Surrender"
)

// advice is what basic strategy recommends. up is the dealer's up card value (2-11).
func advice(cs []cards.Card, up int, canDouble, canSplit, canSurrender bool) string {
	total, soft := value(cs)
	between := func(lo, hi int) bool { return up >= lo && up <= hi }
	doubleOr := func(otherwise string) string {
		if canDouble {
			return double
		}
		return otherwise
	}

	if canSplit && len(cs) == 2 && cardValue(cs[0]) == cardValue(cs[1]) {
		pair := cardValue(cs[0])
		if cs[0].Rank == 1 {
			pair = 11
		}
		switch {
		case pair == 11, pair == 8:
			return split
		case pair == 9 && (between(2, 6) || between(8, 9)):
			return split
		case (pair == 7 || pair == 3 || pair == 2) && between(2, 7):
			return split
		case pair == 6 && between(2, 6):
			return split
		case pair == 4 && between(5, 6):
			return split
		}
	}

	if canSurrender && !soft && len(cs) == 2 {
		if (total == 16 && between(9, 11)) || (total == 15 && up == 10) {
			return surrender
		}
	}

	if soft {
		switch {
		case total >= 19:
			return stand
		case total == 18:
			if between(3, 6) {
				return doubleOr(stand)
			}
			if between(2, 8) {
				return stand
			}
			return hit
		case total == 17 && between(3, 6),
			(total == 15 || total == 16) && between(4, 6),
			(total == 13 || total == 14) && between(5, 6):
			return doubleOr(hit)
		}
		return hit
	}

	switch {
	case total >= 17:
		return stand
	case total >= 13:
		if between(2, 6) {
			return stand
		}
		return hit
	case total == 12:
		if between(4, 6) {
			return stand
		}
		return hit
	case total == 11 && between(2, 10),
		total == 10 && between(2, 9),
		total == 9 && between(3, 6):
		return doubleOr(hit)
	}
	return hit
}
