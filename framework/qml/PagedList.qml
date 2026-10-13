import QtQuick 2.5

// A list of buttons split into pages, with "<" and ">" to flip through them.
// E-ink doesn't scroll well, so lists page instead. On a wide screen the
// list has two columns.
Item {
    id: list
    property var app

    property string title: ""
    property string heading: ""
    property var items: []
    property string emptyText: ""
    property string backText: "Back"
    property int page: 0
    readonly property real u: app.u
    readonly property bool wide: width > height
    readonly property int columns: wide ? 2 : 1
    readonly property real itemHeight: 3.8 * u * 2.1
    readonly property real spacing: 2.5 * u
    readonly property int rows: Math.max(1, Math.floor((buttons.y - itemsArea.y + spacing) / (itemHeight + spacing)))
    readonly property int perPage: rows * columns
    readonly property int pageCount: Math.max(1, Math.ceil(items.length / perPage))

    signal itemClicked(int index)
    signal itemHeld(int index) // long press
    signal back

    onItemsChanged: page = 0

    Text {
        id: titleText
        anchors.top: parent.top
        anchors.topMargin: (list.wide ? 3 : 6) * list.u
        anchors.horizontalCenter: parent.horizontalCenter
        width: parent.width - 8 * list.u
        horizontalAlignment: Text.AlignHCenter
        fontSizeMode: Text.HorizontalFit
        minimumPixelSize: 4 * list.u
        text: list.title
        font.pixelSize: (list.wide ? 7 : 9) * list.u
        font.bold: true
    }

    Text {
        id: headingText
        anchors.top: titleText.bottom
        anchors.topMargin: (list.wide ? 1.5 : 3) * list.u
        anchors.horizontalCenter: parent.horizontalCenter
        width: parent.width - 8 * list.u
        horizontalAlignment: Text.AlignHCenter
        elide: Text.ElideMiddle
        fontSizeMode: Text.HorizontalFit
        minimumPixelSize: 3 * list.u
        text: list.items.length > 0 ? list.heading : ""
        font.pixelSize: (list.wide ? 4 : 5) * list.u
    }

    Text {
        anchors.centerIn: parent
        width: parent.width - 10 * list.u
        visible: list.items.length === 0
        horizontalAlignment: Text.AlignHCenter
        wrapMode: Text.Wrap
        text: list.emptyText
        font.pixelSize: 4.5 * list.u
    }

    Item {
        id: itemsArea
        anchors.top: headingText.bottom
        anchors.topMargin: (list.wide ? 2.5 : 4) * list.u
        anchors.horizontalCenter: parent.horizontalCenter
        width: parent.width - 8 * list.u
    }

    Grid {
        anchors.top: itemsArea.top
        anchors.horizontalCenter: parent.horizontalCenter
        columns: list.columns
        spacing: list.spacing

        Repeater {
            model: list.items.slice(list.page * list.perPage, (list.page + 1) * list.perPage)

            MenuButton {
                width: (itemsArea.width - (list.columns - 1) * list.spacing) / list.columns
                height: list.itemHeight
                fontSize: 3.8 * list.u
                text: modelData
                onClicked: list.itemClicked(list.page * list.perPage + index)
                onHeld: list.itemHeld(list.page * list.perPage + index)
            }
        }
    }

    Row {
        id: buttons
        anchors.bottom: parent.bottom
        anchors.bottomMargin: list.app.toastHeight + (list.wide ? 1.5 : 3) * list.u
        anchors.horizontalCenter: parent.horizontalCenter
        spacing: 4 * list.u

        MenuButton {
            width: 14 * list.u
            fontSize: 5 * list.u
            text: "<"
            enabled: list.page > 0
            opacity: enabled ? 1 : 0
            onClicked: list.page--
        }

        Column {
            spacing: 1 * list.u

            MenuButton {
                width: 40 * list.u
                fontSize: 4.5 * list.u
                text: list.backText
                onClicked: list.back()
            }

            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                visible: list.pageCount > 1
                text: list.app.tr("Page ", "Página ") + (list.page + 1) + " / " + list.pageCount
                font.pixelSize: 3 * list.u
            }
        }

        MenuButton {
            width: 14 * list.u
            fontSize: 5 * list.u
            text: ">"
            enabled: list.page + 1 < list.pageCount
            opacity: enabled ? 1 : 0
            onClicked: list.page++
        }
    }
}
