import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

Item {
    id: screen
    property var app
    readonly property real u: app.u
    readonly property var game: app.game
    readonly property bool stuck: !!game.active && !game.solved && game.moves === 0

    property int selected: -1

    // A new deal or shuffle invalidates the selection.
    Connections {
        target: screen.app
        function onGameChanged() {
            const g = screen.app.game;
            if (screen.selected >= 0 && (!g.present || !g.present[screen.selected] || !g.free[screen.selected]))
                screen.selected = -1;
        }
    }

    function tap(tile) {
        const g = game;
        if (!g.free[tile]) {
            app.showToast("That tile is blocked.", "info", 2000);
        } else if (selected < 0 || selected === tile) {
            selected = selected === tile ? -1 : tile;
        } else if (Protocol.matchClass(g.faces[selected]) === Protocol.matchClass(g.faces[tile])) {
            app.send(Protocol.REMOVE, { a: selected, b: tile });
            selected = -1;
        } else {
            selected = tile;
        }
    }

    Row {
        id: toolbar
        anchors.top: parent.top
        anchors.topMargin: 3 * screen.u
        anchors.horizontalCenter: parent.horizontalCenter
        spacing: 2.5 * screen.u

        MenuButton {
            fontSize: 4 * screen.u
            text: "Menu"
            onClicked: { screen.selected = -1; screen.app.send(Protocol.QUIT); }
        }
        MenuButton {
            fontSize: 4 * screen.u
            text: "Undo"
            enabled: !!screen.game.canUndo
            onClicked: { screen.selected = -1; screen.app.send(Protocol.UNDO); }
        }
        MenuButton {
            fontSize: 4 * screen.u
            text: "Hint"
            enabled: !screen.game.solved && !screen.stuck
            onClicked: screen.app.send(Protocol.HINT)
        }
        MenuButton {
            fontSize: 4 * screen.u
            text: "Shuffle"
            bold: screen.stuck
            enabled: !screen.game.solved
            onClicked: { screen.selected = -1; screen.app.send(Protocol.SHUFFLE); }
        }
    }

    Text {
        id: info
        anchors.top: toolbar.bottom
        anchors.topMargin: 1.5 * screen.u
        anchors.horizontalCenter: parent.horizontalCenter
        text: screen.game.solved ? "" : screen.stuck ? "No more moves. Shuffle or undo."
            : screen.game.remaining + " tiles left  ·  " + screen.game.moves + (screen.game.moves === 1 ? " move" : " moves")
        font.pixelSize: 3.5 * screen.u
        font.bold: screen.stuck
    }

    MahjongBoard {
        anchors.top: info.bottom
        anchors.topMargin: 2 * screen.u
        anchors.bottom: parent.bottom
        anchors.bottomMargin: screen.app.toastHeight + 2 * screen.u
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.margins: 1 * screen.u
        game: screen.game
        shadeBlocked: screen.app.shadeBlocked
        selected: screen.selected
        onTileTapped: (tile) => screen.tap(tile)
    }

    Column {
        anchors.centerIn: parent
        spacing: 3 * screen.u
        visible: !!screen.game.solved

        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            text: "Cleared in " + Protocol.duration(screen.game.seconds || 0) + "!"
            font.pixelSize: 8 * screen.u
            font.bold: true
        }

        MenuButton {
            anchors.horizontalCenter: parent.horizontalCenter
            width: 50 * screen.u
            fontSize: 4.5 * screen.u
            text: "Play again"
            onClicked: screen.app.send(Protocol.NEW_GAME, { layout: screen.game.layout })
        }
    }
}
