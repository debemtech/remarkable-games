import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

Item {
    id: screen
    property var app
    readonly property real u: app.u
    readonly property var game: app.game
    readonly property bool ready: !!game.active && !game.generating && game.values !== undefined

    property int selected: -1
    property bool notesMode: false

    onReadyChanged: if (!ready) selected = -1

    function digitCount(d) {
        let n = 0;
        if (ready)
            for (let i = 0; i < 81; i++)
                if (game.values[i] === d) n++;
        return n;
    }

    function enter(d) {
        if (selected < 0) {
            app.showToast("Tap a cell first.", "info", 2000);
            return;
        }
        if (game.given[selected]) {
            app.showToast("That number is part of the puzzle.", "info", 2000);
            return;
        }
        if (notesMode)
            app.send(Protocol.NOTE, { cell: selected, digit: d });
        else
            // Tapping the digit that's already there erases it
            app.send(Protocol.SET, { cell: selected, digit: game.values[selected] === d ? 0 : d });
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
            onClicked: screen.app.send(Protocol.QUIT)
        }
        MenuButton {
            fontSize: 4 * screen.u
            text: "Undo"
            enabled: screen.ready && !!screen.game.canUndo
            onClicked: screen.app.send(Protocol.UNDO)
        }
        MenuButton {
            fontSize: 4 * screen.u
            text: "Hint"
            enabled: screen.ready && !screen.game.solved
            onClicked: screen.app.send(Protocol.HINT, { cell: screen.selected, digit: 0 })
        }
        MenuButton {
            fontSize: 4 * screen.u
            text: "Check"
            enabled: screen.ready && !screen.game.solved
            onClicked: screen.app.send(Protocol.CHECK)
        }
    }

    Text {
        id: info
        anchors.top: toolbar.bottom
        anchors.topMargin: 1.5 * screen.u
        anchors.horizontalCenter: parent.horizontalCenter
        text: Protocol.title(screen.game.difficulty) + (screen.game.hints > 0 ? "  ·  " + screen.game.hints + " hint" + (screen.game.hints === 1 ? "" : "s") : "")
        font.pixelSize: 3.5 * screen.u
    }

    SudokuGrid {
        id: sudokuGrid
        readonly property real available: Math.min(screen.width - 4 * screen.u,
                                                   controls.y - info.y - info.height - 4 * screen.u)
        anchors.top: info.bottom
        anchors.topMargin: 1.5 * screen.u
        anchors.horizontalCenter: parent.horizontalCenter
        width: Math.floor(available)
        height: width
        visible: screen.ready

        values: screen.game.values || []
        given: screen.game.given || []
        notes: screen.game.notes || []
        mistakes: (screen.game.conflicts || []).concat(screen.game.wrong || [])
        selected: screen.selected
        interactive: screen.ready && !screen.game.solved
        onCellTapped: (cell) => screen.selected = cell
    }

    Text {
        anchors.centerIn: sudokuGrid
        visible: !!screen.game.generating
        text: "Generating a puzzle..."
        font.pixelSize: 5 * screen.u
    }

    // Number pad, or the result once solved
    Item {
        id: controls
        anchors.bottom: parent.bottom
        anchors.bottomMargin: screen.app.toastHeight + 2 * screen.u
        anchors.horizontalCenter: parent.horizontalCenter
        width: parent.width - 4 * screen.u
        height: 20 * screen.u

        Column {
            anchors.fill: parent
            spacing: 1.5 * screen.u
            visible: screen.ready && !screen.game.solved

            Row {
                id: digits
                spacing: 1.2 * screen.u
                readonly property real buttonWidth: (parent.width - 8 * spacing) / 9

                Repeater {
                    model: 9

                    MenuButton {
                        readonly property int digit: index + 1
                        width: digits.buttonWidth
                        height: 10.5 * screen.u
                        fontSize: 6 * screen.u
                        bold: !screen.notesMode
                        text: digit
                        textColor: screen.notesMode ? sudokuGrid.noteColor : sudokuGrid.entryColor
                        accent: sudokuGrid.selectedColor
                        // All nine placed: nothing left to enter
                        enabled: screen.notesMode || screen.digitCount(digit) < 9
                        onClicked: screen.enter(digit)
                    }
                }
            }

            Row {
                anchors.horizontalCenter: parent.horizontalCenter
                spacing: 3 * screen.u

                MenuButton {
                    width: 30 * screen.u
                    height: 8 * screen.u
                    fontSize: 4 * screen.u
                    text: screen.notesMode ? "Notes: on" : "Notes: off"
                    highlighted: screen.notesMode
                    accent: sudokuGrid.selectedColor
                    onClicked: screen.notesMode = !screen.notesMode
                }
                MenuButton {
                    width: 30 * screen.u
                    height: 8 * screen.u
                    fontSize: 4 * screen.u
                    text: "Erase"
                    onClicked: {
                        if (screen.selected >= 0 && !screen.game.given[screen.selected])
                            screen.app.send(Protocol.SET, { cell: screen.selected, digit: 0 });
                    }
                }
            }
        }

        Column {
            anchors.centerIn: parent
            spacing: 2 * screen.u
            visible: screen.ready && !!screen.game.solved

            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                text: "Solved in " + Protocol.duration(screen.game.seconds || 0) + "!"
                font.pixelSize: 6.5 * screen.u
                font.bold: true
                color: "#1b7a34"
            }

            MenuButton {
                anchors.horizontalCenter: parent.horizontalCenter
                width: 50 * screen.u
                fontSize: 4.5 * screen.u
                text: "New " + Protocol.title(screen.game.difficulty) + " game"
                onClicked: screen.app.send(Protocol.NEW_GAME, { difficulty: screen.game.difficulty })
            }
        }
    }
}
