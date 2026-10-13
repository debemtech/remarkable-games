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
        spacing: 2.5 * menu.u

        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            text: "Mahjong"
            font.pixelSize: 11 * menu.u
            font.bold: true
        }

        Item { width: 1; height: 2 * menu.u }

        Column {
            anchors.horizontalCenter: parent.horizontalCenter
            visible: !!menu.app.menu.canContinue
            spacing: 1 * menu.u

            MenuButton {
                anchors.horizontalCenter: parent.horizontalCenter
                width: menu.buttonWidth
                fontSize: 5 * menu.u
                bold: true
                text: "Continue"
                onClicked: menu.app.send(Protocol.CONTINUE)
            }

            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                text: menu.app.menu.continue || ""
                font.pixelSize: 3.5 * menu.u
            }
        }

        Item { width: 1; height: 2 * menu.u }

        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            text: "New game"
            font.pixelSize: 5 * menu.u
        }

        Repeater {
            model: menu.app.menu.layouts

            Column {
                anchors.horizontalCenter: parent.horizontalCenter
                spacing: 0.8 * menu.u

                MenuButton {
                    anchors.horizontalCenter: parent.horizontalCenter
                    width: menu.buttonWidth
                    fontSize: 4.5 * menu.u
                    text: modelData.title + "  (" + modelData.tiles + " tiles)"
                    onClicked: menu.app.send(Protocol.NEW_GAME, { layout: modelData.name })
                }

                Text {
                    anchors.horizontalCenter: parent.horizontalCenter
                    visible: modelData.solved > 0
                    text: modelData.solved + " cleared" + (modelData.best > 0 ? "  ·  best " + Protocol.duration(modelData.best) : "")
                    font.pixelSize: 3 * menu.u
                    color: "#404040"
                }
            }
        }

        CheckRow {
            anchors.horizontalCenter: parent.horizontalCenter
            fontSize: 4 * menu.u
            text: "Shade blocked tiles"
            checked: menu.app.shadeBlocked
            onToggled: menu.app.shadeBlocked = !menu.app.shadeBlocked
        }

        Item { width: 1; height: 2 * menu.u }

        MenuButton {
            anchors.horizontalCenter: parent.horizontalCenter
            width: menu.buttonWidth
            fontSize: 4.5 * menu.u
            text: "Exit"
            onClicked: menu.app.close()
        }
    }
}
