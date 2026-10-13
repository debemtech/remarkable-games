import QtQuick 2.5

// The 8x8 board. Squares are indexed like the backend: 0 = A1, 7 = H1, 63 = H8.
//
// Pieces are moved like in chessMarkable, either by
//  1. tapping a piece, then the square it should go to (shows the possible moves), or
//  2. pressing a piece and releasing the finger on the destination square.
Item {
    id: board

    // 64 FEN piece letters ('.' = empty square)
    property string pieces: ""
    // source square -> list of destination squares the user may move to
    property var moves: ({})
    // [from, to] of the move to highlight, or null
    property var lastMove: null
    // Draw the pieces upside down (PvP with board rotation, black's turn)
    property bool rotated: false
    property bool interactive: true

    signal moveRequested(int from, int to, string promotion)
    signal illegalMove

    property int selected: -1
    property int pressedSquare: -1
    property int promotionFrom: -1
    property int promotionTo: -1
    readonly property var hints: selected >= 0 && moves && moves[selected] ? moves[selected] : []
    readonly property real squareSize: Math.floor(Math.min(width, height) / 8)

    readonly property color lightColor: "#d6d6d6"
    readonly property color darkColor: "#969696"

    readonly property var pieceFiles: ({
        "K": "king-white", "Q": "queen-white", "R": "rook-white",
        "B": "bishop-white", "N": "knight-white", "P": "pawn-white",
        "k": "king-black", "q": "queen-black", "r": "rook-black",
        "b": "bishop-black", "n": "knight-black", "p": "pawn-black"
    })
    // Relative size of the original piece images, so pawns stay smaller than kings.
    readonly property var pieceScale: ({
        "K": 91, "Q": 95, "R": 81, "B": 96, "N": 82, "P": 81,
        "k": 91, "q": 97, "r": 81, "b": 96, "n": 83, "p": 81
    })

    function pieceAt(square) {
        return pieces.length === 64 ? pieces.charAt(square) : ".";
    }

    function pieceSource(piece) {
        return "res/" + pieceFiles[piece] + ".svg";
    }

    function isPossible(from, to) {
        const destinations = moves ? moves[from] : undefined;
        return !!destinations && destinations.indexOf(to) >= 0;
    }

    function isPromotion(from, to) {
        const piece = pieceAt(from);
        return (piece === "P" && to >= 56) || (piece === "p" && to < 8);
    }

    function squareAt(x, y) {
        if (x < 0 || y < 0 || x >= squareSize * 8 || y >= squareSize * 8)
            return -1;
        const col = Math.floor(x / squareSize);
        const row = Math.floor(y / squareSize);
        return (7 - row) * 8 + col;
    }

    function resetSelection() {
        selected = -1;
        pressedSquare = -1;
        promotionFrom = -1;
        promotionTo = -1;
    }

    function request(from, to) {
        selected = -1;
        if (!isPossible(from, to)) {
            illegalMove();
            return;
        }
        if (isPromotion(from, to)) {
            promotionFrom = from;
            promotionTo = to;
            return;
        }
        moveRequested(from, to, "");
    }

    function handleTap(downSquare, square) {
        if (selected >= 0) {
            if (square === selected) {
                selected = -1; // Cancel
            } else if (isPossible(selected, square)) {
                request(selected, square);
            } else if (pieceAt(square) !== ".") {
                selected = square;
            } else {
                selected = -1;
            }
        } else if (downSquare >= 0 && downSquare !== square) {
            // Swiped from one square to another: move at once without highlighting
            if (pieceAt(downSquare) !== ".")
                request(downSquare, square);
        } else if (pieceAt(square) !== ".") {
            selected = square;
        }
    }

    onPiecesChanged: {
        promotionFrom = -1;
        if (selected >= 0 && pieceAt(selected) === ".")
            selected = -1;
    }
    onInteractiveChanged: if (!interactive) resetSelection()

    Grid {
        id: grid
        columns: 8
        width: board.squareSize * 8
        height: board.squareSize * 8
        anchors.centerIn: parent

        Repeater {
            model: 64

            Item {
                readonly property int row: Math.floor(index / 8)
                readonly property int col: index % 8
                readonly property int square: (7 - row) * 8 + col
                readonly property string piece: board.pieceAt(square)
                readonly property bool hasLastMove: board.lastMove !== null && board.lastMove !== undefined

                width: board.squareSize
                height: board.squareSize

                Rectangle {
                    anchors.fill: parent
                    color: (row + col) % 2 === 0 ? board.lightColor : board.darkColor
                }

                Image {
                    anchors.fill: parent
                    visible: hasLastMove && board.lastMove[0] === square
                    source: "res/piece-moved-from.png"
                    smooth: true
                    mipmap: true
                }

                Image {
                    anchors.fill: parent
                    visible: hasLastMove && board.lastMove[1] === square
                    source: "res/piece-moved-to.png"
                    smooth: true
                    mipmap: true
                }

                Image {
                    readonly property real size: Math.round(board.squareSize * 0.8 * (board.pieceScale[piece] || 97) / 97)
                    anchors.centerIn: parent
                    width: size
                    height: size
                    visible: piece !== "."
                    source: visible ? board.pieceSource(piece) : ""
                    sourceSize.width: size
                    sourceSize.height: size
                    rotation: board.rotated ? 180 : 0
                    smooth: true
                }

                Image {
                    anchors.fill: parent
                    anchors.margins: board.squareSize / 20
                    visible: board.selected === square && piece !== "."
                    source: "res/piece-selected.png"
                    smooth: true
                    mipmap: true
                }

                // Possible destination: a dot, or a ring around a piece that can be captured
                Rectangle {
                    readonly property bool capture: piece !== "."
                    anchors.centerIn: parent
                    visible: board.hints.indexOf(square) >= 0
                    width: board.squareSize * (capture ? 0.94 : 0.3)
                    height: width
                    radius: width / 2
                    color: capture ? "transparent" : "#303030"
                    border.width: capture ? Math.round(board.squareSize * 0.06) : Math.max(2, Math.round(board.squareSize * 0.02))
                    border.color: capture ? "#303030" : "white"
                }
            }
        }
    }

    Rectangle {
        anchors.fill: grid
        anchors.margins: -border.width
        color: "transparent"
        border.width: Math.max(2, Math.round(board.squareSize / 40))
        border.color: "black"
    }

    MouseArea {
        anchors.fill: grid
        enabled: board.interactive && board.promotionFrom < 0
        onPressed: (mouse) => {
            board.pressedSquare = board.squareAt(mouse.x, mouse.y);
        }
        onReleased: (mouse) => {
            const downSquare = board.pressedSquare;
            board.pressedSquare = -1;
            const square = board.squareAt(mouse.x, mouse.y);
            if (square >= 0)
                board.handleTap(downSquare, square);
        }
        onCanceled: board.pressedSquare = -1
    }

    // Promotion picker
    MouseArea {
        anchors.fill: grid
        visible: board.promotionFrom >= 0
        onClicked: board.resetSelection()

        Rectangle {
            id: picker
            readonly property bool white: board.pieceAt(board.promotionFrom) === "P"
            anchors.centerIn: parent
            width: board.squareSize * 4.8
            height: pickerColumn.height + board.squareSize * 0.6
            color: "white"
            border.width: Math.max(3, Math.round(board.squareSize / 25))
            border.color: "black"
            radius: board.squareSize / 8

            // Swallow taps on the picker background
            MouseArea { anchors.fill: parent }

            Column {
                id: pickerColumn
                anchors.centerIn: parent
                spacing: board.squareSize * 0.2

                Text {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: "Promote to"
                    font.pixelSize: board.squareSize * 0.32
                }

                Row {
                    anchors.horizontalCenter: parent.horizontalCenter
                    spacing: board.squareSize * 0.1

                    Repeater {
                        model: ["q", "r", "b", "n"]

                        Rectangle {
                            readonly property string piece: picker.white ? modelData.toUpperCase() : modelData
                            width: board.squareSize
                            height: board.squareSize
                            color: choice.pressed ? "black" : board.lightColor
                            border.width: 2
                            border.color: "black"

                            Image {
                                anchors.centerIn: parent
                                width: board.squareSize * 0.8
                                height: width
                                source: board.promotionFrom >= 0 ? board.pieceSource(parent.piece) : ""
                                sourceSize.width: width
                                sourceSize.height: height
                                rotation: board.rotated ? 180 : 0
                            }

                            MouseArea {
                                id: choice
                                anchors.fill: parent
                                onClicked: {
                                    const from = board.promotionFrom;
                                    const to = board.promotionTo;
                                    board.resetSelection();
                                    board.moveRequested(from, to, modelData);
                                }
                            }
                        }
                    }
                }

                Text {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: "Tap outside to cancel"
                    font.pixelSize: board.squareSize * 0.22
                    color: "#404040"
                }
            }
        }
    }
}
