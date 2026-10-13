import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

// The table. Tap a card to pick it up (in a column: it and the cards on it),
// then tap where it should go. Tap a picked up card again to send it to the
// best place, e.g. its foundation.
Item {
    id: screen
    property var app
    readonly property real u: app.u
    readonly property var game: app.game
    readonly property bool klondike: game.variant !== "freecell"
    readonly property int cols: klondike ? 7 : 8

    readonly property color accent: "#1f4fbf"  // picked up cards
    readonly property color hintColor: "#f5a623"

    // Selection: {pile, index} or null
    property var sel: null

    readonly property real gap: 2 * u
    readonly property real cardW: Math.floor(Math.min((board.width - (cols - 1) * gap) / cols, 12.5 * u))
    readonly property real cardH: Math.round(cardW * 1.4)
    readonly property real boardLeft: (board.width - (cols * cardW + (cols - 1) * gap)) / 2

    Connections {
        target: screen.app
        function onGameChanged() { screen.sel = null; }
    }

    function pileKind(p) { return game.piles && game.piles[p] ? game.piles[p].kind : ""; }

    // Column and row (0 = top row, 1 = columns) of a pile
    function slot(p) {
        if (klondike) {
            if (p === 0) return [0, 0];       // stock
            if (p === 1) return [1, 0];       // waste
            if (p < 6) return [p + 1, 0];     // foundations
            return [p - 6, 1];                // columns
        }
        if (p < 8) return [p, 0];             // free cells, foundations
        return [p - 8, 1];
    }

    function red(c) { return c.s === 1 || c.s === 2; }

    // Whether the cards from i on can be picked up together
    function isRun(cards, i) {
        for (let k = i; k < cards.length; k++) {
            if (!cards[k].u)
                return false;
            if (k > i && (cards[k].r !== cards[k - 1].r - 1 || red(cards[k]) === red(cards[k - 1])))
                return false;
        }
        return true;
    }

    function drawStock() {
        sel = null;
        app.send(Protocol.STOCK);
    }

    function tapCard(p, i) {
        const pile = game.piles[p];
        if (pile.kind === "stock") {
            drawStock();
            return;
        }
        if (sel !== null) {
            if (sel.pile !== p) {
                app.send(Protocol.MOVE, { from: sel.pile, index: sel.index, to: p });
                sel = null;
                return;
            }
            if (sel.index === i || pile.kind !== "tableau") {
                app.send(Protocol.AUTO, { from: p, index: sel.index });
                sel = null;
                return;
            }
        }
        if (!pile.cards[i].u)
            return;
        if (pile.kind !== "tableau")
            i = pile.cards.length - 1;
        if (!isRun(pile.cards, i)) {
            app.showToast("Those cards can't move together.", "info", 2500);
            return;
        }
        sel = { pile: p, index: i };
    }

    function tapPile(p) {
        if (pileKind(p) === "stock") {
            drawStock();
        } else if (sel !== null && sel.pile !== p) {
            app.send(Protocol.MOVE, { from: sel.pile, index: sel.index, to: p });
            sel = null;
        } else {
            sel = null;
        }
    }

    // Top bar
    Row {
        id: toolbar
        anchors.top: parent.top
        anchors.left: parent.left
        anchors.margins: 2 * screen.u
        spacing: 2 * screen.u

        MenuButton {
            fontSize: 3.5 * screen.u
            text: "Menu"
            onClicked: screen.app.send(Protocol.QUIT)
        }
        MenuButton {
            fontSize: 3.5 * screen.u
            text: "Undo"
            enabled: !!screen.game.canUndo
            onClicked: screen.app.send(Protocol.UNDO)
        }
        MenuButton {
            fontSize: 3.5 * screen.u
            text: "Hint"
            textColor: screen.accent
            enabled: !screen.game.solved
            onClicked: screen.app.send(Protocol.HINT)
        }
        MenuButton {
            fontSize: 3.5 * screen.u
            text: "Restart"
            enabled: !screen.game.solved
            onClicked: screen.app.send(Protocol.RESTART)
        }
    }

    Text {
        anchors.verticalCenter: toolbar.verticalCenter
        anchors.right: parent.right
        anchors.rightMargin: 3 * screen.u
        text: (screen.game.title || "") + "  ·  " + (screen.game.moves || 0) + (screen.game.moves === 1 ? " move" : " moves")
        font.pixelSize: 3.5 * screen.u
    }

    Item {
        id: board
        anchors.top: toolbar.bottom
        anchors.topMargin: 2 * screen.u
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        anchors.leftMargin: 2 * screen.u
        anchors.rightMargin: 2 * screen.u
        anchors.bottomMargin: screen.app.toastHeight

        // Tapping the felt drops the selection
        MouseArea {
            anchors.fill: parent
            onClicked: screen.sel = null
        }

        Repeater {
            model: screen.game.piles ? screen.game.piles.length : 0

            Item {
                id: pileItem
                readonly property int p: index
                readonly property var pile: screen.game.piles[p]
                readonly property var cards: pile ? pile.cards : []
                readonly property string kind: pile ? pile.kind : ""
                readonly property var pos: screen.slot(p)
                readonly property var hint: screen.game.hint
                readonly property bool hintFrom: !!hint && (hint.from === p || (hint.from === -1 && kind === "stock"))
                readonly property bool hintTo: !!hint && hint.from !== -1 && hint.to === p

                // Which cards are drawn: columns show all, the waste its top three
                // (draw 3), the other piles only the top card.
                readonly property int first: kind === "tableau" ? 0
                    : kind === "waste" ? Math.max(0, cards.length - (screen.game.variant === "klondike3" ? 3 : 1))
                    : Math.max(0, cards.length - 1)

                // Columns: spacing between cards, squeezed to fit the height.
                readonly property real downStep: 0.12 * screen.cardH
                readonly property real upStep: 0.3 * screen.cardH
                readonly property real squeeze: {
                    let total = 0;
                    for (let i = 1; i < cards.length; i++)
                        total += cards[i - 1].u ? upStep : downStep;
                    const room = board.height - screen.cardH - screen.gap - screen.cardH;
                    return total > room ? room / total : 1;
                }
                function offset(i) {
                    if (kind === "waste")
                        return Qt.point((i - first) * 0.32 * screen.cardW, 0);
                    if (kind !== "tableau")
                        return Qt.point(0, 0);
                    let y = 0;
                    for (let k = 1; k <= i; k++)
                        y += (cards[k - 1].u ? upStep : downStep) * squeeze;
                    return Qt.point(0, y);
                }

                x: screen.boardLeft + pos[0] * (screen.cardW + screen.gap)
                y: pos[1] === 0 ? 0 : screen.cardH + screen.gap * 1.5
                width: screen.cardW
                height: screen.cardH

                // Empty place
                Rectangle {
                    anchors.fill: parent
                    radius: screen.cardW * 0.08
                    color: "transparent"
                    readonly property bool marked: pileItem.hintTo || (pileItem.hintFrom && pileItem.kind === "stock")
                    border.width: marked ? Math.max(5, screen.cardW * 0.05) : Math.max(2, screen.cardW * 0.015)
                    border.color: marked ? screen.hintColor : "#9a9a9a"

                    Text {
                        anchors.centerIn: parent
                        font.pixelSize: screen.cardW * (pileItem.kind === "stock" ? 0.16 : 0.3)
                        color: "#9a9a9a"
                        text: pileItem.kind === "foundation" ? "A"
                            : pileItem.kind === "tableau" && screen.klondike ? "K"
                            : pileItem.kind === "stock" ? "Redeal" : ""
                    }

                    MouseArea {
                        anchors.fill: parent
                        onClicked: screen.tapPile(pileItem.p)
                    }
                }

                // Cards left in the stock
                Text {
                    visible: pileItem.kind === "stock" && pileItem.cards.length > 0
                    anchors.top: parent.bottom
                    anchors.topMargin: 0.3 * screen.u
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: pileItem.cards.length + " left"
                    font.pixelSize: 2.6 * screen.u
                    color: "#505050"
                }

                Repeater {
                    model: pileItem.cards.length - pileItem.first

                    Item {
                        readonly property int i: pileItem.first + index
                        readonly property var card: pileItem.cards[i]
                        readonly property bool picked: screen.sel !== null && screen.sel.pile === pileItem.p && i >= screen.sel.index
                        readonly property bool hinted: (pileItem.hintFrom && (pileItem.kind === "stock" || i >= screen.game.hint.index))
                            || (pileItem.hintTo && i === pileItem.cards.length - 1)
                        readonly property point at: pileItem.offset(i)

                        x: at.x
                        y: at.y
                        width: screen.cardW
                        height: screen.cardH

                        PlayingCard {
                            width: screen.cardW
                            rank: card ? card.r : 1
                            suit: card ? card.s : 0
                            faceDown: !card || !card.u
                            compactIndex: true
                        }

                        Rectangle {
                            anchors.fill: parent
                            visible: picked || hinted
                            radius: screen.cardW * 0.08
                            color: "transparent"
                            border.width: Math.max(5, screen.cardW * 0.05)
                            border.color: picked ? screen.accent : screen.hintColor
                        }


                        MouseArea {
                            anchors.fill: parent
                            onClicked: screen.tapCard(pileItem.p, i)
                        }
                    }
                }
            }
        }
    }

    // Won
    Rectangle {
        anchors.centerIn: parent
        visible: !!screen.game.solved
        width: 80 * screen.u
        height: wonColumn.height + 8 * screen.u
        radius: 2 * screen.u
        color: "white"
        border.width: Math.max(3, screen.u / 3)
        border.color: "black"

        MouseArea { anchors.fill: parent }

        Column {
            id: wonColumn
            anchors.centerIn: parent
            spacing: 2.5 * screen.u

            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                text: "You won!"
                font.pixelSize: 8 * screen.u
                font.bold: true
                color: "#1b7a34"
            }
            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                text: Protocol.duration(screen.game.seconds || 0) + "  ·  " + (screen.game.moves || 0) + (screen.game.moves === 1 ? " move" : " moves")
                font.pixelSize: 4.5 * screen.u
            }
            Row {
                anchors.horizontalCenter: parent.horizontalCenter
                spacing: 3 * screen.u
                MenuButton {
                    width: 30 * screen.u
                    fontSize: 4 * screen.u
                    text: "New game"
                    bold: true
                    onClicked: screen.app.send(Protocol.NEW_GAME, { variant: screen.game.variant })
                }
                MenuButton {
                    width: 30 * screen.u
                    fontSize: 4 * screen.u
                    text: "Menu"
                    onClicked: screen.app.send(Protocol.QUIT)
                }
            }
        }
    }
}
