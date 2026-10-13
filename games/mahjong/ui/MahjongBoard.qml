import QtQuick 2.5
import "Protocol.js" as Protocol

// The tiles. Positions are in half-tile units (see the backend's board.Pos);
// higher levels are drawn shifted up-left, with the tile's side showing below
// and to the right.
Item {
    id: board

    property var game: ({})
    property bool shadeBlocked: true
    property int selected: -1
    signal tileTapped(int tile)

    readonly property int count: game.faces ? game.faces.length : 0
    readonly property int cols: game.width || 2
    readonly property int rows: game.height || 2
    readonly property int levels: game.levels || 1
    // Tile face width; faces are 3:4
    readonly property real tileW: Math.floor(Math.min(width / (cols / 2 + 0.1 * (levels + 1)),
                                                      height / (rows / 2 * 4 / 3 + 0.1 * (levels + 1))))
    readonly property real tileH: Math.round(tileW * 4 / 3)
    readonly property real depth: Math.max(3, Math.round(tileW * 0.1))
    readonly property real boardW: cols / 2 * tileW + (levels + 1) * depth
    readonly property real boardH: rows / 2 * tileH + (levels + 1) * depth
    readonly property var hint: game.hint || []

    Item {
        id: area
        width: board.boardW
        height: board.boardH
        anchors.centerIn: parent

        Repeater {
            // Tiles never move during a game; bindings update them in place.
            model: board.count

            Item {
                readonly property bool present: !!board.game.present && board.game.present[index]
                readonly property bool free: !!board.game.free && board.game.free[index]
                readonly property int face: board.game.faces ? board.game.faces[index] : 0
                readonly property bool isSelected: board.selected === index
                readonly property bool isHint: board.hint.indexOf(index) >= 0
                readonly property int level: board.game.z ? board.game.z[index] : 0

                visible: present
                z: index
                x: (board.game.x ? board.game.x[index] : 0) / 2 * board.tileW + (board.levels - level) * board.depth
                y: (board.game.y ? board.game.y[index] : 0) / 2 * board.tileH + (board.levels - level) * board.depth
                width: board.tileW
                height: board.tileH

                // Side of the tile
                Rectangle {
                    x: board.depth
                    y: board.depth
                    width: parent.width
                    height: parent.height
                    radius: board.tileW * 0.08
                    color: "#707070"
                }

                Rectangle {
                    anchors.fill: parent
                    radius: board.tileW * 0.08
                    color: isSelected ? "#a0a0a0" : (board.shadeBlocked && !free) ? "#d6d6d6" : "white"
                    border.width: isSelected || isHint ? Math.max(4, board.tileW * 0.07) : Math.max(1, board.tileW * 0.015)
                    border.color: "black"

                    Image {
                        anchors.fill: parent
                        anchors.margins: board.tileW * 0.07
                        source: present ? "res/tiles/" + face + ".svg" : ""
                        sourceSize.width: width
                        sourceSize.height: height
                        smooth: true
                    }
                }

                MouseArea {
                    anchors.fill: parent
                    enabled: present
                    onClicked: board.tileTapped(index)
                }
            }
        }
    }
}
