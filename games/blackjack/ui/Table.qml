import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

Item {
    id: tableScreen
    property var app
    readonly property real u: app.u
    readonly property var t: app.table
    readonly property bool betting: t.phase === "betting" || t.phase === "done"
    readonly property color winColor: "#1b7a34"
    readonly property color loseColor: "#c62828"
    readonly property color accent: "#1f4fbf"

    // Cards shrink when split hands need the room.
    readonly property real cardWidth: {
        let units = 0;
        for (let i = 0; i < t.hands.length; i++)
            units += 1 + 0.5 * Math.max(0, t.hands[i].cards.length - 1);
        const gaps = Math.max(0, t.hands.length - 1) * 6 * u;
        return Math.min(13 * u, (width - 10 * u - gaps) / Math.max(1, units));
    }

    // Top bar
    Row {
        id: topLeft
        anchors.top: parent.top
        anchors.left: parent.left
        anchors.margins: 2.5 * tableScreen.u
        spacing: 2 * tableScreen.u

        MenuButton {
            fontSize: 3.5 * tableScreen.u
            text: "Exit"
            onClicked: tableScreen.app.close()
        }
        MenuButton {
            fontSize: 3.5 * tableScreen.u
            text: "Stats"
            onClicked: tableScreen.app.showStats = true
        }
        // Turn the table around, for holding the tablet the other way
        MenuButton {
            fontSize: 3.5 * tableScreen.u
            text: "Flip"
            visible: tableScreen.app.rotated
            onClicked: tableScreen.app.toggleFlip()
        }
    }

    Text {
        anchors.verticalCenter: topLeft.verticalCenter
        anchors.right: parent.right
        anchors.rightMargin: 3 * tableScreen.u
        text: "Chips: " + Protocol.money(tableScreen.t.bankroll)
        font.pixelSize: 5 * tableScreen.u
        font.bold: true
    }

    // Dealer
    Column {
        id: dealerArea
        anchors.top: topLeft.bottom
        anchors.topMargin: 1.5 * tableScreen.u
        anchors.horizontalCenter: parent.horizontalCenter
        spacing: 1 * tableScreen.u

        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            text: tableScreen.t.dealer.length === 0 ? "Dealer"
                : "Dealer  ·  " + (tableScreen.t.hidden > 0 ? "shows " : "") + tableScreen.t.dealerTotal
                  + (tableScreen.t.dealerTotal > 21 ? "  bust" : "")
            font.pixelSize: 4 * tableScreen.u
        }

        Item {
            anchors.horizontalCenter: parent.horizontalCenter
            width: dealerFan.width
            height: dealerFan.height

            CardFan {
                id: dealerFan
                cards: tableScreen.t.dealer
                hidden: tableScreen.t.hidden
                cardWidth: 12 * tableScreen.u
                visible: count > 0
            }
        }
    }

    // What's going on
    Text {
        id: message
        anchors.top: dealerArea.bottom
        anchors.topMargin: 2 * tableScreen.u
        anchors.horizontalCenter: parent.horizontalCenter
        width: parent.width - 10 * tableScreen.u
        horizontalAlignment: Text.AlignHCenter
        font.pixelSize: 5 * tableScreen.u
        font.bold: true
        color: tableScreen.t.hint ? tableScreen.accent : "black"
        text: {
            const t = tableScreen.t;
            if (t.phase === "playing")
                return t.hint ? "Basic strategy says: " + t.hint
                    : t.hands.length > 1 ? "Your turn, hand " + (t.active + 1) + " of " + t.hands.length : "Your turn";
            if (t.phase === "dealer")
                return "Dealer's turn...";
            if (t.message)
                return t.message;
            return t.hands.length === 0 ? "Place your bet" : "";
        }
    }

    Text {
        anchors.top: message.bottom
        anchors.topMargin: 1.5 * tableScreen.u
        anchors.horizontalCenter: parent.horizontalCenter
        visible: tableScreen.t.hands.length === 0
        text: "Blackjack pays 3:2  ·  Dealer stands on all 17s  ·  Bets " + tableScreen.t.minBet + " to " + tableScreen.t.maxBet
        font.pixelSize: 3.5 * tableScreen.u
        color: "#505050"
    }

    // Player hands
    Row {
        id: hands
        anchors.bottom: controls.top
        anchors.bottomMargin: 3 * tableScreen.u
        anchors.horizontalCenter: parent.horizontalCenter
        spacing: 6 * tableScreen.u

        Repeater {
            model: tableScreen.t.hands

            Column {
                readonly property bool active: index === tableScreen.t.active
                spacing: 1 * tableScreen.u

                CardFan {
                    anchors.horizontalCenter: parent.horizontalCenter
                    cards: modelData.cards
                    cardWidth: tableScreen.cardWidth
                }

                // Marks the hand being played
                Rectangle {
                    anchors.horizontalCenter: parent.horizontalCenter
                    width: tableScreen.cardWidth
                    height: Math.max(4, 0.6 * tableScreen.u)
                    color: active && tableScreen.t.hands.length > 1 ? tableScreen.accent : "transparent"
                }

                Text {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: (modelData.soft && modelData.total < 21 ? "soft " : "") + modelData.total
                          + "  ·  bet " + Protocol.money(modelData.bet)
                    font.pixelSize: 3.5 * tableScreen.u
                    font.bold: active
                }

                Text {
                    anchors.horizontalCenter: parent.horizontalCenter
                    visible: modelData.result !== ""
                    text: (Protocol.RESULTS[modelData.result] || "") + "  " + Protocol.signed(modelData.net)
                    font.pixelSize: 4 * tableScreen.u
                    font.bold: true
                    color: modelData.net > 0 ? tableScreen.winColor : modelData.net < 0 ? tableScreen.loseColor : "black"
                }
            }
        }
    }

    // Buttons
    Item {
        id: controls
        anchors.bottom: parent.bottom
        anchors.bottomMargin: tableScreen.app.toastHeight + 2 * tableScreen.u
        anchors.left: parent.left
        anchors.right: parent.right
        height: 10 * tableScreen.u

        Row {
            anchors.centerIn: parent
            spacing: 2 * tableScreen.u
            visible: tableScreen.betting && !tableScreen.t.broke

            Text {
                anchors.verticalCenter: parent.verticalCenter
                width: 18 * tableScreen.u
                text: "Bet " + Protocol.money(tableScreen.t.bet)
                font.pixelSize: 4.5 * tableScreen.u
                font.bold: true
            }
            Repeater {
                model: tableScreen.t.chips
                MenuButton {
                    fontSize: 4 * tableScreen.u
                    text: "+" + modelData
                    enabled: tableScreen.t.bet + modelData <= Math.min(tableScreen.t.maxBet, tableScreen.t.bankroll)
                    onClicked: tableScreen.app.send(Protocol.BET, { amount: modelData })
                }
            }
            MenuButton {
                fontSize: 4 * tableScreen.u
                text: "Clear"
                enabled: tableScreen.t.bet > 0
                onClicked: tableScreen.app.send(Protocol.BET, { amount: 0 })
            }
            Item { width: 2 * tableScreen.u; height: 1 }
            MenuButton {
                width: 20 * tableScreen.u
                fontSize: 4.5 * tableScreen.u
                text: "Deal"
                bold: true
                highlighted: true
                accent: tableScreen.accent
                enabled: tableScreen.t.bet >= tableScreen.t.minBet
                onClicked: tableScreen.app.send(Protocol.DEAL)
            }
        }

        Row {
            anchors.centerIn: parent
            spacing: 2.5 * tableScreen.u
            visible: tableScreen.t.phase === "playing" || tableScreen.t.phase === "dealer"

            // Room for the six buttons on one row
            readonly property real small: 3.6 * tableScreen.u

            MenuButton {
                width: Math.max(implicitWidth, 16 * tableScreen.u)
                fontSize: 4.5 * tableScreen.u
                text: "Hit"
                bold: true
                enabled: !!tableScreen.t.canHit
                onClicked: tableScreen.app.send(Protocol.HIT)
            }
            MenuButton {
                width: Math.max(implicitWidth, 16 * tableScreen.u)
                fontSize: 4.5 * tableScreen.u
                text: "Stand"
                bold: true
                enabled: !!tableScreen.t.canHit
                onClicked: tableScreen.app.send(Protocol.STAND)
            }
            MenuButton {
                fontSize: parent.small
                text: "Double"
                enabled: !!tableScreen.t.canDouble
                onClicked: tableScreen.app.send(Protocol.DOUBLE)
            }
            MenuButton {
                fontSize: parent.small
                text: "Split"
                enabled: !!tableScreen.t.canSplit
                onClicked: tableScreen.app.send(Protocol.SPLIT)
            }
            MenuButton {
                fontSize: parent.small
                text: "Surrender"
                enabled: !!tableScreen.t.canSurrender
                onClicked: tableScreen.app.send(Protocol.SURRENDER)
            }
            MenuButton {
                fontSize: parent.small
                text: "Hint"
                textColor: tableScreen.accent
                enabled: !!tableScreen.t.canHit
                onClicked: tableScreen.app.send(Protocol.HINT)
            }
        }

        Row {
            anchors.centerIn: parent
            spacing: 4 * tableScreen.u
            visible: !!tableScreen.t.broke

            Text {
                anchors.verticalCenter: parent.verticalCenter
                text: "You're out of chips."
                font.pixelSize: 4.5 * tableScreen.u
            }
            MenuButton {
                fontSize: 4.5 * tableScreen.u
                text: "Start over with 1,000"
                bold: true
                onClicked: tableScreen.app.send(Protocol.RESTART)
            }
        }
    }

    // Stats
    MouseArea {
        anchors.fill: parent
        visible: tableScreen.app.showStats
        onClicked: tableScreen.app.showStats = false

        Rectangle {
            anchors.centerIn: parent
            width: 70 * tableScreen.u
            height: statsColumn.height + 8 * tableScreen.u
            radius: 2 * tableScreen.u
            color: "white"
            border.width: Math.max(3, tableScreen.u / 3)
            border.color: "black"

            MouseArea { anchors.fill: parent } // keep taps inside from closing it

            Column {
                id: statsColumn
                anchors.centerIn: parent
                spacing: 1.5 * tableScreen.u

                Text {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: "Stats"
                    font.pixelSize: 6 * tableScreen.u
                    font.bold: true
                }
                Repeater {
                    model: {
                        const s = tableScreen.t.stats || {};
                        const hands = s.hands || 0;
                        const pct = n => hands > 0 ? "  (" + Math.round(100 * n / hands) + "%)" : "";
                        return [
                            ["Hands played", hands],
                            ["Won", (s.wins || 0) + pct(s.wins || 0)],
                            ["Lost", (s.losses || 0) + pct(s.losses || 0)],
                            ["Pushed", (s.pushes || 0) + pct(s.pushes || 0)],
                            ["Blackjacks", s.blackjacks || 0],
                            ["Most chips", Protocol.money(s.peak || 0)],
                            ["Fresh starts", s.restarts || 0]
                        ];
                    }
                    Row {
                        spacing: 4 * tableScreen.u
                        Text {
                            width: 32 * tableScreen.u
                            text: modelData[0]
                            font.pixelSize: 4 * tableScreen.u
                        }
                        Text {
                            width: 26 * tableScreen.u
                            horizontalAlignment: Text.AlignRight
                            text: modelData[1]
                            font.pixelSize: 4 * tableScreen.u
                            font.bold: true
                        }
                    }
                }
                Item { width: 1; height: 1 * tableScreen.u }
                MenuButton {
                    anchors.horizontalCenter: parent.horizontalCenter
                    width: 30 * tableScreen.u
                    fontSize: 4 * tableScreen.u
                    text: "Close"
                    onClicked: tableScreen.app.showStats = false
                }
            }
        }
    }
}
