import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

Item {
    id: menu
    property var app
    readonly property real u: app.u
    readonly property real buttonWidth: 60 * u

    Column {
        anchors.centerIn: parent
        spacing: 2 * menu.u

        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            text: "Solitaire"
            font.pixelSize: 10 * menu.u
            font.bold: true
        }

        Item { width: 1; height: 1 * menu.u }

        Column {
            anchors.horizontalCenter: parent.horizontalCenter
            visible: !!menu.app.menu.canContinue
            spacing: 0.8 * menu.u

            MenuButton {
                anchors.horizontalCenter: parent.horizontalCenter
                width: menu.buttonWidth
                fontSize: 4.5 * menu.u
                bold: true
                text: "Continue"
                onClicked: menu.app.send(Protocol.CONTINUE)
            }
            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                text: menu.app.menu.continue || ""
                font.pixelSize: 3.2 * menu.u
            }
        }

        Item { width: 1; height: 1 * menu.u }

        Repeater {
            model: menu.app.menu.variants

            Column {
                anchors.horizontalCenter: parent.horizontalCenter
                spacing: 0.6 * menu.u

                MenuButton {
                    anchors.horizontalCenter: parent.horizontalCenter
                    width: menu.buttonWidth
                    fontSize: 4.5 * menu.u
                    text: modelData.title
                    onClicked: menu.app.send(Protocol.NEW_GAME, { variant: modelData.id })
                }
                Text {
                    anchors.horizontalCenter: parent.horizontalCenter
                    visible: modelData.played > 0
                    text: "Won " + modelData.won + " of " + modelData.played
                          + (modelData.best > 0 ? "  ·  best " + Protocol.duration(modelData.best) : "")
                    font.pixelSize: 3 * menu.u
                    color: "#404040"
                }
            }
        }

        Item { width: 1; height: 1 * menu.u }

        Row {
            anchors.horizontalCenter: parent.horizontalCenter
            spacing: 3 * menu.u

            MenuButton {
                width: 28 * menu.u
                fontSize: 4 * menu.u
                text: "Exit"
                onClicked: menu.app.close()
            }
            MenuButton {
                width: 28 * menu.u
                fontSize: 4 * menu.u
                text: "Flip"
                visible: menu.app.rotated
                onClicked: menu.app.toggleFlip()
            }
        }
    }
}
