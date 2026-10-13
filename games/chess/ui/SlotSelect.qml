import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

// Pick a savestate slot to continue on or to start over on.
Item {
    id: slotScreen
    property var app
    readonly property real u: app.u
    // Slot waiting for a second tap before its saved game gets overwritten
    property int confirmSlot: -1

    onVisibleChanged: confirmSlot = -1

    Column {
        anchors.centerIn: parent
        width: Math.min(parent.width - 8 * slotScreen.u, 90 * slotScreen.u)
        spacing: 3 * slotScreen.u

        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            text: "Chess"
            font.pixelSize: 9 * slotScreen.u
            font.bold: true
        }

        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            text: Protocol.MODE_NAMES[slotScreen.app.pendingMode] || ""
            font.pixelSize: 5 * slotScreen.u
        }

        Item { width: 1; height: 2 * slotScreen.u }

        Repeater {
            model: slotScreen.app.slots

            Rectangle {
                width: parent.width
                height: slotColumn.height + 5 * slotScreen.u
                border.width: Math.max(3, slotScreen.u / 4)
                border.color: "black"
                radius: slotScreen.u

                Column {
                    id: slotColumn
                    anchors.centerIn: parent
                    width: parent.width - 5 * slotScreen.u
                    spacing: 2 * slotScreen.u

                    Text {
                        width: parent.width
                        text: "Slot " + modelData.slot + "  ·  " + modelData.description
                        font.pixelSize: 4 * slotScreen.u
                        elide: Text.ElideRight
                    }

                    Row {
                        spacing: 3 * slotScreen.u

                        MenuButton {
                            width: (slotColumn.width - 3 * slotScreen.u) / 2
                            fontSize: 4 * slotScreen.u
                            text: "Continue"
                            enabled: modelData.saved
                            onClicked: slotScreen.app.startGame(modelData.slot, false)
                        }

                        MenuButton {
                            width: (slotColumn.width - 3 * slotScreen.u) / 2
                            fontSize: 4 * slotScreen.u
                            text: slotScreen.confirmSlot === modelData.slot ? "Overwrite save?" : "New game"
                            bold: slotScreen.confirmSlot === modelData.slot
                            onClicked: {
                                if (modelData.saved && slotScreen.confirmSlot !== modelData.slot) {
                                    slotScreen.confirmSlot = modelData.slot;
                                    return;
                                }
                                slotScreen.confirmSlot = -1;
                                slotScreen.app.startGame(modelData.slot, true);
                            }
                        }
                    }
                }
            }
        }

        Item { width: 1; height: 2 * slotScreen.u }

        MenuButton {
            anchors.horizontalCenter: parent.horizontalCenter
            width: parent.width * 0.6
            fontSize: 4.5 * slotScreen.u
            text: "Back"
            onClicked: slotScreen.app.screen = "menu"
        }
    }
}
