import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

// Chess. This file holds the state sent by the backend and switches
// screens; the screens only render it.
GameApp {
    id: root
    appId: "chess"
    screen: "menu" // menu, slots, game, pgns, games, replay

    property string pendingMode: "pvp"
    property bool rotateBoard: false

    property var slots: []
    property var game: ({ active: false })
    property var replay: ({ active: false })
    property string pgnDir: ""
    property var pgnFiles: []
    property var pgnGames: []
    property string pgnError: ""

    function openSlots(mode) {
        pendingMode = mode;
        send(Protocol.GET_SLOTS);
        screen = "slots";
    }

    function startGame(slot, reset) {
        send(Protocol.NEW_GAME, {
            mode: pendingMode,
            slot: slot,
            reset: reset,
            rotate: pendingMode === "pvp" && rotateBoard
        });
    }

    function openPgnFiles() {
        send(Protocol.LIST_PGNS);
        screen = "pgns";
    }

    onMessage: (type, data) => {
        switch (type) {
        case Protocol.GAME_STATE:
            game = data;
            if (data.active && screen !== "game") {
                hideToast();
                screen = "game";
            } else if (!data.active && screen === "game") {
                screen = "menu";
            }
            break;
        case Protocol.REPLAY_STATE:
            replay = data;
            if (data.active && screen !== "replay" && !game.active) {
                hideToast();
                screen = "replay";
            } else if (!data.active && screen === "replay") {
                screen = "games";
            }
            break;
        case Protocol.SLOTS:
            slots = data.slots;
            break;
        case Protocol.PGN_LIST:
            pgnDir = data.dir;
            pgnFiles = data.files || [];
            break;
        case Protocol.GAME_LIST:
            pgnGames = data.games || [];
            pgnError = data.error || "";
            screen = "games";
            break;
        }
    }

    MainMenu {
        anchors.fill: parent
        app: root
        visible: root.screen === "menu"
    }

    SlotSelect {
        anchors.fill: parent
        app: root
        visible: root.screen === "slots"
    }

    GameScreen {
        anchors.fill: parent
        app: root
        visible: root.screen === "game"
    }

    PagedList {
        anchors.fill: parent
        app: root
        visible: root.screen === "pgns"
        title: "Chess"
        heading: "Choose PGN:"
        items: root.pgnFiles
        emptyText: "No PGNs found, please add them to:\n" + root.pgnDir
        backText: "Main Menu"
        onItemClicked: (index) => root.send(Protocol.OPEN_PGN, { file: root.pgnFiles[index] })
        onBack: root.screen = "menu"
    }

    PagedList {
        anchors.fill: parent
        app: root
        visible: root.screen === "games"
        title: "Chess"
        heading: "Choose Game:"
        items: root.pgnGames
        emptyText: root.pgnError !== "" ? root.pgnError : "Couldn't parse any games from PGN"
        backText: "PGNs"
        onItemClicked: (index) => root.send(Protocol.START_REPLAY, { index: index })
        onBack: root.openPgnFiles()
    }

    ReplayScreen {
        anchors.fill: parent
        app: root
        visible: root.screen === "replay"
    }
}
