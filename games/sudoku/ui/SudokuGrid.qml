import QtQuick 2.5

// The 9x9 grid. Cells are numbered 0-80 row by row, like the backend.
Rectangle {
    id: grid

    property var values: []     // 81 digits, 0 = empty
    property var given: []      // 81 bools
    property var notes: []      // 81 bitmasks, bit d-1 = pencil mark d
    property var mistakes: []   // cells to mark as wrong
    property int selected: -1
    property bool interactive: true
    signal cellTapped(int cell)

    readonly property real thick: Math.max(4, Math.round(width / 220))
    readonly property real thin: Math.max(2, Math.round(width / 600))
    readonly property real cellSize: (width - 4 * thick - 6 * thin) / 9
    readonly property int selectedValue: selected >= 0 && values.length === 81 ? values[selected] : 0

    // Colors for the Paper Pro's color screen. They also come out as distinct
    // grays on a black and white screen.
    readonly property color lineColor: "#1c2b4a"
    readonly property color givenColor: "black"
    readonly property color entryColor: "#1f4fbf"       // numbers you entered
    readonly property color noteColor: "#4a5a78"
    readonly property color selectedColor: "#1f4fbf"
    readonly property color peerColor: "#dce7fb"        // row, column and box of the selection
    readonly property color sameDigitColor: "#f5b83d"   // the selected number elsewhere
    readonly property color mistakeColor: "#d32020"

    color: lineColor

    function rowOf(c) { return Math.floor(c / 9); }
    function colOf(c) { return c % 9; }
    function boxOf(c) { return Math.floor(rowOf(c) / 3) * 3 + Math.floor(colOf(c) / 3); }

    Grid {
        x: grid.thick
        y: grid.thick
        columns: 3
        spacing: grid.thick

        Repeater {
            model: 9 // boxes

            Grid {
                readonly property int box: index
                columns: 3
                spacing: grid.thin

                Repeater {
                    model: 9 // cells of the box

                    Rectangle {
                        readonly property int cell: (Math.floor(box / 3) * 3 + Math.floor(index / 3)) * 9 + (box % 3) * 3 + index % 3
                        readonly property int value: grid.values.length === 81 ? grid.values[cell] : 0
                        readonly property bool isGiven: grid.given.length === 81 && grid.given[cell]
                        readonly property bool isSelected: grid.selected === cell
                        readonly property bool isPeer: grid.selected >= 0 && !isSelected
                            && (grid.rowOf(cell) === grid.rowOf(grid.selected)
                                || grid.colOf(cell) === grid.colOf(grid.selected)
                                || grid.boxOf(cell) === grid.boxOf(grid.selected))
                        readonly property bool sameDigit: value !== 0 && value === grid.selectedValue && !isSelected
                        readonly property int noteMask: value === 0 && grid.notes.length === 81 ? grid.notes[cell] : 0

                        width: grid.cellSize
                        height: grid.cellSize
                        readonly property bool isMistake: value !== 0 && grid.mistakes.indexOf(cell) >= 0

                        color: isSelected ? grid.selectedColor : sameDigit ? grid.sameDigitColor : isPeer ? grid.peerColor : "white"

                        // Mistake marker: a ring around the digit
                        Rectangle {
                            anchors.centerIn: parent
                            width: parent.width * 0.86
                            height: width
                            radius: width / 2
                            color: "transparent"
                            border.width: Math.max(3, parent.width * 0.06)
                            border.color: isSelected ? "white" : grid.mistakeColor
                            visible: isMistake
                        }

                        Text {
                            anchors.centerIn: parent
                            visible: value !== 0
                            text: value
                            font.pixelSize: grid.cellSize * 0.62
                            font.bold: isGiven
                            font.italic: !isGiven
                            color: isSelected ? "white" : isMistake ? grid.mistakeColor : isGiven ? grid.givenColor : grid.entryColor
                        }

                        Grid {
                            anchors.centerIn: parent
                            visible: noteMask !== 0
                            columns: 3

                            Repeater {
                                model: 9

                                Text {
                                    width: grid.cellSize / 3.3
                                    height: width
                                    horizontalAlignment: Text.AlignHCenter
                                    verticalAlignment: Text.AlignVCenter
                                    text: (noteMask >> index) & 1 ? index + 1 : ""
                                    font.pixelSize: grid.cellSize * 0.24
                                    color: isSelected ? "white" : grid.noteColor
                                }
                            }
                        }

                        MouseArea {
                            anchors.fill: parent
                            enabled: grid.interactive
                            onClicked: grid.cellTapped(cell)
                        }
                    }
                }
            }
        }
    }
}
