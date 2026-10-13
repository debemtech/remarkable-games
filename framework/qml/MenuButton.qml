import QtQuick 2.5

// Black-on-white button that inverts while pressed (like xochitl's buttons).
// Set `highlighted` to show it inverted, e.g. for a toggle that is on.
// `accent` colors the inverted state, `textColor` the label.
Rectangle {
    id: button

    property alias text: label.text
    property real fontSize: 40
    property bool bold: false
    property bool highlighted: false
    property color accent: "black"
    property color textColor: "black"
    signal clicked
    signal held // long press

    readonly property bool down: (mouse.pressed && mouse.containsMouse) !== highlighted

    implicitWidth: label.implicitWidth + fontSize * 2
    implicitHeight: fontSize * 2.1
    radius: fontSize * 0.25
    color: down ? accent : "white"
    border.width: Math.max(3, Math.round(fontSize / 12))
    border.color: !enabled ? "#a0a0a0" : highlighted ? accent : "black"

    Text {
        id: label
        anchors.centerIn: parent
        width: parent.width - button.fontSize
        horizontalAlignment: Text.AlignHCenter
        elide: Text.ElideRight
        font.pixelSize: button.fontSize
        font.bold: button.bold
        color: button.down ? "white" : (button.enabled ? button.textColor : "#a0a0a0")
    }

    MouseArea {
        id: mouse
        anchors.fill: parent
        onClicked: button.clicked()
        onPressAndHold: button.held()
    }
}
