import QtQuick 2.5

// A playing card. Set the width; the height follows (standard 5:7 card).
// rank: 1 (ace) to 13 (king); suit: 0 spades, 1 hearts, 2 diamonds, 3 clubs
// (like framework/cards). compactIndex puts rank and suit side by side in the
// corner, so stacked cards can overlap more and stay readable.
Item {
    id: card
    property int rank: 1
    property int suit: 0
    property bool faceDown: false
    property bool compactIndex: false

    readonly property var suitNames: ["spades", "hearts", "diamonds", "clubs"]
    readonly property var rankNames: ["", "A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"]
    readonly property bool red: suit === 1 || suit === 2
    readonly property string suitSource: "res/suits/" + suitNames[suit] + ".svg"
    readonly property color backColor: "#1f4fbf"

    height: Math.round(width * 1.4)

    Rectangle {
        anchors.fill: parent
        radius: card.width * 0.08
        color: card.faceDown ? card.backColor : "white"
        border.width: Math.max(2, card.width * 0.02)
        border.color: "black"

        // Back: a framed pattern
        Rectangle {
            visible: card.faceDown
            anchors.fill: parent
            anchors.margins: card.width * 0.08
            radius: card.width * 0.05
            color: "transparent"
            border.width: Math.max(2, card.width * 0.025)
            border.color: "white"

            Grid {
                anchors.centerIn: parent
                columns: 3
                spacing: card.width * 0.06
                Repeater {
                    model: 12
                    Rectangle {
                        width: card.width * 0.14
                        height: width
                        rotation: 45
                        color: "white"
                    }
                }
            }
        }
    }

    // Corner index, top left and (upside down) bottom right
    Repeater {
        model: card.faceDown || card.compactIndex ? 0 : 2

        Column {
            x: index === 0 ? card.width * 0.07 : card.width - width - card.width * 0.07
            y: index === 0 ? card.width * 0.06 : card.height - height - card.width * 0.06
            rotation: index === 0 ? 0 : 180
            spacing: 0

            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                text: card.rankNames[card.rank]
                font.pixelSize: card.width * 0.24
                font.bold: true
                color: card.red ? "#c62828" : "black"
            }
            Image {
                anchors.horizontalCenter: parent.horizontalCenter
                width: card.width * 0.17
                height: width
                source: card.suitSource
                sourceSize.width: width
                sourceSize.height: height
            }
        }
    }

    Row {
        visible: card.compactIndex && !card.faceDown
        x: card.width * 0.07
        y: card.width * 0.03
        spacing: card.width * 0.03

        Text {
            anchors.verticalCenter: parent.verticalCenter
            text: card.rankNames[card.rank]
            font.pixelSize: card.width * 0.25
            font.bold: true
            color: card.red ? "#c62828" : "black"
        }
        Image {
            anchors.verticalCenter: parent.verticalCenter
            width: card.width * 0.2
            height: width
            source: card.suitSource
            sourceSize.width: width
            sourceSize.height: height
        }
    }

    // Center: the suit for number cards and aces, the letter for court cards
    Image {
        visible: !card.faceDown && card.rank <= 10
        anchors.centerIn: parent
        width: card.width * (card.rank === 1 ? 0.6 : 0.45)
        height: width
        source: card.suitSource
        sourceSize.width: width
        sourceSize.height: height
    }

    Rectangle {
        visible: !card.faceDown && card.rank > 10
        anchors.centerIn: parent
        width: card.width * 0.5
        height: card.height * 0.5
        radius: card.width * 0.04
        color: "transparent"
        border.width: Math.max(2, card.width * 0.02)
        border.color: card.red ? "#c62828" : "black"

        Text {
            anchors.centerIn: parent
            text: card.rankNames[card.rank]
            font.pixelSize: card.width * 0.38
            font.bold: true
            color: card.red ? "#c62828" : "black"
        }
    }
}
