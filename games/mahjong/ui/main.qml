import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

GameApp {
    id: root
    appId: "mahjong"
    screen: "menu" // menu, game

    property var game: ({ active: false })
    property var menu: ({ canContinue: false, layouts: [] })
    property bool shadeBlocked: true

    onMessage: (type, data) => {
        switch (type) {
        case Protocol.GAME_STATE:
            game = data;
            if (data.active && screen !== "game") {
                hideToast();
                screen = "game";
            } else if (!data.active) {
                screen = "menu";
            }
            break;
        case Protocol.MENU:
            menu = data;
            break;
        }
    }

    Menu {
        anchors.fill: parent
        app: root
        visible: root.screen === "menu"
    }

    GameScreen {
        anchors.fill: parent
        app: root
        visible: root.screen === "game"
    }
}
