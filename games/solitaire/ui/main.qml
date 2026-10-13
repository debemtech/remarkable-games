import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

// Solitaire, played with the tablet on its side.
GameApp {
    id: root
    appId: "solitaire"
    orientation: "landscape"
    screen: "menu" // menu, game

    property var game: ({ active: false, piles: [] })
    property var menu: ({ canContinue: false, variants: [] })

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
