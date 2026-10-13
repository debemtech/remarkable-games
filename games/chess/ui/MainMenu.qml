import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

Item {
    id: menu
    property var app
    readonly property real u: app.u
    readonly property real buttonWidth: Math.min(width * 0.6, 70 * u)

    Column {
        anchors.centerIn: parent
        spacing: 2 * menu.u

        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            text: "Chess"
            font.pixelSize: 9 * menu.u
            font.bold: true
        }

        Item { width: 1; height: 3 * menu.u }

        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            text: "Player vs Player"
            font.pixelSize: 5 * menu.u
        }

        MenuButton {
            anchors.horizontalCenter: parent.horizontalCenter
            width: menu.buttonWidth
            fontSize: 4.5 * menu.u
            text: "Play!"
            onClicked: menu.app.openSlots("pvp")
        }

        CheckRow {
            anchors.horizontalCenter: parent.horizontalCenter
            fontSize: 4 * menu.u
            text: "Rotate pieces for black"
            checked: menu.app.rotateBoard
            onToggled: menu.app.rotateBoard = !menu.app.rotateBoard
        }

        Item { width: 1; height: 2 * menu.u }

        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            text: "Player vs Bot"
            font.pixelSize: 5 * menu.u
        }

        Repeater {
            model: [
                { label: "Easy", mode: "easy" },
                { label: "Normal", mode: "normal" },
                { label: "Hard", mode: "hard" }
            ]

            MenuButton {
                anchors.horizontalCenter: parent.horizontalCenter
                width: menu.buttonWidth
                fontSize: 4.5 * menu.u
                text: modelData.label
                onClicked: menu.app.openSlots(modelData.mode)
            }
        }

        Item { width: 1; height: 3 * menu.u }

        MenuButton {
            anchors.horizontalCenter: parent.horizontalCenter
            width: menu.buttonWidth
            fontSize: 4.5 * menu.u
            text: "PGN Viewer"
            onClicked: menu.app.openPgnFiles()
        }

        MenuButton {
            anchors.horizontalCenter: parent.horizontalCenter
            width: menu.buttonWidth
            fontSize: 4.5 * menu.u
            text: "Exit"
            onClicked: menu.app.close()
        }
    }
}
