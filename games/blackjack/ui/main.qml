import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

// Blackjack, played with the tablet on its side.
GameApp {
    id: root
    appId: "blackjack"
    orientation: "landscape"

    property var table: ({ phase: "betting", hands: [], dealer: [], hidden: 0, chips: [], stats: {}, bankroll: 0, bet: 0 })
    property bool showStats: false

    onMessage: (type, data) => {
        if (type === Protocol.STATE)
            table = data;
    }

    Table {
        anchors.fill: parent
        app: root
    }
}
