import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

Item {
    id: menu
    property var app
    readonly property real u: app.u
    readonly property real buttonWidth: 62 * u

    Column {
        anchors.centerIn: parent
        spacing: 2.5 * menu.u

        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            width: menu.width - 8 * menu.u
            horizontalAlignment: Text.AlignHCenter
            fontSizeMode: Text.HorizontalFit
            minimumPixelSize: 5 * menu.u
            text: menu.app.tr("Crossword", "Palavras Cruzadas")
            font.pixelSize: 10 * menu.u
            font.bold: true
        }

        Item { width: 1; height: 1 * menu.u }

        Column {
            anchors.horizontalCenter: parent.horizontalCenter
            visible: menu.app.menu.continueKey !== ""
            spacing: 0.8 * menu.u

            MenuButton {
                anchors.horizontalCenter: parent.horizontalCenter
                width: menu.buttonWidth
                fontSize: 4.5 * menu.u
                bold: true
                text: menu.app.tr("Continue", "Continuar")
                onClicked: menu.app.send(Protocol.OPEN, { key: menu.app.menu.continueKey })
            }
            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                width: menu.buttonWidth
                horizontalAlignment: Text.AlignHCenter
                elide: Text.ElideRight
                text: menu.app.menu.continueTitle || ""
                font.pixelSize: 3.2 * menu.u
            }
        }

        MenuButton {
            anchors.horizontalCenter: parent.horizontalCenter
            width: menu.buttonWidth
            fontSize: 4.5 * menu.u
            text: menu.app.tr("My puzzles", "Minhas cruzadas")
            onClicked: menu.app.openLibrary()
        }

        MenuButton {
            anchors.horizontalCenter: parent.horizontalCenter
            width: menu.buttonWidth
            fontSize: 4.5 * menu.u
            text: menu.app.tr("Get puzzles", "Baixar cruzadas")
            onClicked: menu.app.openSources()
        }

        Item { width: 1; height: 1 * menu.u }

        Row {
            anchors.horizontalCenter: parent.horizontalCenter
            spacing: 2 * menu.u

            MenuButton {
                width: 30 * menu.u
                fontSize: 4 * menu.u
                text: "English"
                highlighted: menu.app.lang === "en"
                onClicked: menu.app.setPref("lang", "en")
            }
            MenuButton {
                width: 30 * menu.u
                fontSize: 4 * menu.u
                text: "Português"
                highlighted: menu.app.lang === "pt"
                onClicked: menu.app.setPref("lang", "pt")
            }
        }

        Row {
            anchors.horizontalCenter: parent.horizontalCenter
            spacing: 2 * menu.u

            MenuButton {
                width: 30 * menu.u
                fontSize: 4 * menu.u
                text: menu.app.tr("Exit", "Sair")
                onClicked: menu.app.close()
            }
            MenuButton {
                width: 30 * menu.u
                fontSize: 4 * menu.u
                text: menu.app.tr("Flip", "Girar")
                visible: menu.app.rotated
                onClicked: menu.app.toggleFlip()
            }
        }
    }
}
