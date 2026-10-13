import QtQuick 2.5

// A checkbox with a label. The whole row is tappable. Bind `checked` to your
// state and change that state in onToggled.
Item {
    id: row
    property alias text: label.text
    property bool checked: false
    property real fontSize: 40
    signal toggled

    implicitWidth: content.width
    implicitHeight: content.height + fontSize * 0.5

    Row {
        id: content
        anchors.centerIn: parent
        spacing: row.fontSize * 0.5

        Rectangle {
            width: row.fontSize * 1.1
            height: width
            anchors.verticalCenter: parent.verticalCenter
            border.width: Math.max(3, row.fontSize / 16)
            border.color: "black"

            Rectangle {
                anchors.fill: parent
                anchors.margins: parent.width * 0.2
                color: "black"
                visible: row.checked
            }
        }

        Text {
            id: label
            anchors.verticalCenter: parent.verticalCenter
            font.pixelSize: row.fontSize
        }
    }

    MouseArea {
        anchors.fill: parent
        onClicked: row.toggled()
    }
}
