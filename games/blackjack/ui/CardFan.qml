import QtQuick 2.5
import "framework"

// Cards fanned out to the right, like on a table.
Item {
    id: fan
    property var cards: []
    property int hidden: 0 // face down cards after the face up ones
    property real cardWidth: 200
    readonly property real step: cardWidth * 0.5
    readonly property int count: cards.length + hidden

    width: count > 0 ? cardWidth + (count - 1) * step : cardWidth
    height: Math.round(cardWidth * 1.4)

    Repeater {
        model: fan.count

        PlayingCard {
            x: index * fan.step
            width: fan.cardWidth
            faceDown: index >= fan.cards.length
            rank: faceDown ? 1 : fan.cards[index].r
            suit: faceDown ? 0 : fan.cards[index].s
        }
    }
}
