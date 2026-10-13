import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

// Step through a game of a PGN file. Own moves can be tried out on the board
// and have to be undone before the replay continues.
Item {
    id: replayScreen
    property var app
    readonly property real u: app.u
    readonly property var replay: app.replay

    Row {
        id: toolbar
        anchors.top: parent.top
        anchors.topMargin: 3 * replayScreen.u
        anchors.horizontalCenter: parent.horizontalCenter
        spacing: 2.5 * replayScreen.u

        MenuButton {
            fontSize: 4 * replayScreen.u
            text: "Back"
            onClicked: replayScreen.app.send(Protocol.REPLAY_CLOSE)
        }

        MenuButton {
            fontSize: 4 * replayScreen.u
            text: "Reset"
            onClicked: replayScreen.app.send(Protocol.REPLAY_RESET)
        }

        MenuButton {
            fontSize: 4 * replayScreen.u
            text: "Undo"
            onClicked: replayScreen.app.send(Protocol.REPLAY_UNDO)
        }

        MenuButton {
            fontSize: 4 * replayScreen.u
            text: "Next"
            bold: true
            onClicked: replayScreen.app.send(Protocol.REPLAY_NEXT)
        }
    }

    Text {
        id: title
        anchors.top: toolbar.bottom
        anchors.topMargin: 2 * replayScreen.u
        anchors.horizontalCenter: parent.horizontalCenter
        width: parent.width - 6 * replayScreen.u
        horizontalAlignment: Text.AlignHCenter
        elide: Text.ElideRight
        text: replayScreen.replay.title || ""
        font.pixelSize: 3.5 * replayScreen.u
    }

    ChessBoard {
        id: chessBoard
        readonly property real available: Math.min(replayScreen.width - 4 * replayScreen.u,
                                                   info.y - title.y - title.height - 5 * replayScreen.u)
        anchors.top: title.bottom
        anchors.topMargin: 2 * replayScreen.u
        anchors.horizontalCenter: parent.horizontalCenter
        width: Math.floor(available / 8) * 8
        height: width

        pieces: replayScreen.replay.board || ""
        moves: replayScreen.replay.moves || ({})
        lastMove: replayScreen.replay.lastMove === undefined ? null : replayScreen.replay.lastMove
        interactive: !!replayScreen.replay.active

        onMoveRequested: (from, to, promotion) => {
            replayScreen.app.send(Protocol.REPLAY_MOVE, { from: from, to: to, promotion: promotion || null });
        }
        onIllegalMove: replayScreen.app.showToast("That move isn't possible.", "info", 3000)
    }

    Column {
        id: info
        anchors.bottom: parent.bottom
        anchors.bottomMargin: replayScreen.app.toastHeight + 2 * replayScreen.u
        anchors.horizontalCenter: parent.horizontalCenter
        width: parent.width - 6 * replayScreen.u
        spacing: 1 * replayScreen.u

        Text {
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
            text: {
                const r = replayScreen.replay;
                let progress = "Move " + (r.played || 0) + " / " + (r.total || 0);
                if (r.manualMoves > 0)
                    progress += "  ·  " + r.manualMoves + " own move" + (r.manualMoves === 1 ? "" : "s");
                return progress;
            }
            font.pixelSize: 3.5 * replayScreen.u
            font.bold: true
        }

        Text {
            width: parent.width
            height: 3 * 4 * replayScreen.u * 1.25
            horizontalAlignment: Text.AlignHCenter
            verticalAlignment: Text.AlignTop
            wrapMode: Text.Wrap
            maximumLineCount: 3
            elide: Text.ElideRight
            text: replayScreen.replay.comment || ""
            font.pixelSize: 3.5 * replayScreen.u
        }
    }
}
