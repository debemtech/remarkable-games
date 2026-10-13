import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

Item {
    id: gameScreen
    property var app
    readonly property real u: app.u
    readonly property var game: app.game

    Row {
        id: toolbar
        anchors.top: parent.top
        anchors.topMargin: 3 * gameScreen.u
        anchors.horizontalCenter: parent.horizontalCenter
        spacing: 3 * gameScreen.u

        MenuButton {
            fontSize: 4 * gameScreen.u
            text: "Save & Quit"
            onClicked: gameScreen.app.send(Protocol.QUIT_GAME)
        }

        MenuButton {
            fontSize: 4 * gameScreen.u
            text: "Undo"
            onClicked: gameScreen.app.send(Protocol.UNDO)
        }
    }

    Text {
        id: title
        anchors.top: toolbar.bottom
        anchors.topMargin: 2 * gameScreen.u
        anchors.horizontalCenter: parent.horizontalCenter
        text: (Protocol.MODE_NAMES[gameScreen.game.mode] || "") + "  ·  Slot " + gameScreen.game.slot
        font.pixelSize: 3.5 * gameScreen.u
    }

    ChessBoard {
        id: chessBoard
        readonly property real available: Math.min(gameScreen.width - 4 * gameScreen.u,
                                                   status.y - title.y - title.height - 6 * gameScreen.u)
        anchors.top: title.bottom
        anchors.topMargin: 2.5 * gameScreen.u
        anchors.horizontalCenter: parent.horizontalCenter
        width: Math.floor(available / 8) * 8
        height: width

        pieces: gameScreen.game.board || ""
        moves: gameScreen.game.moves || ({})
        lastMove: gameScreen.game.lastMove === undefined ? null : gameScreen.game.lastMove
        rotated: !!gameScreen.game.rotated
        interactive: !!gameScreen.game.active && !gameScreen.game.gameOver

        onMoveRequested: (from, to, promotion) => {
            gameScreen.app.send(Protocol.MOVE, { from: from, to: to, promotion: promotion || null });
        }
        onIllegalMove: {
            gameScreen.app.showToast(gameScreen.game.busy ? "Please wait for your turn." : "That move isn't possible.",
                                     "info", 3000);
        }
    }

    Text {
        id: status
        anchors.bottom: parent.bottom
        anchors.bottomMargin: gameScreen.app.toastHeight + 2 * gameScreen.u
        anchors.horizontalCenter: parent.horizontalCenter
        width: parent.width - 4 * gameScreen.u
        height: 8 * gameScreen.u
        horizontalAlignment: Text.AlignHCenter
        verticalAlignment: Text.AlignVCenter
        fontSizeMode: Text.HorizontalFit
        minimumPixelSize: 3 * gameScreen.u
        text: gameScreen.game.status || ""
        font.pixelSize: (gameScreen.game.gameOver ? 6.5 : 4.5) * gameScreen.u
        font.bold: !!gameScreen.game.gameOver
    }
}
