import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

GameApp {
    id: root
    appId: "crossword"
    screen: "menu" // menu, library, sources, entries, game

    property var menu: ({ continueKey: "" })
    property var library: ({ items: [], dir: "" })
    property var sources: []
    property var entries: ({ entries: [], name: "", loading: false })
    property var puzzle: ({ active: false, cells: [], across: [], down: [] })
    property var progress: ({ entries: [], wrong: [], revealed: [] })
    property bool downloading: false

    onMessage: (type, data) => {
        switch (type) {
        case Protocol.MENU:
            menu = data;
            break;
        case Protocol.LIBRARY_LIST:
            library = data;
            break;
        case Protocol.SOURCE_LIST:
            sources = data.sources;
            break;
        case Protocol.ENTRIES:
            data.loading = false;
            entries = data;
            break;
        case Protocol.PUZZLE:
            puzzle = data;
            downloading = false;
            if (data.active) {
                hideToast();
                screen = "game";
            } else if (screen === "game") {
                screen = "menu";
            }
            break;
        case Protocol.PROGRESS:
            progress = data;
            break;
        }
    }

    // A failed download ends with a toast; stop showing "Downloading..."
    onToastTextChanged: if (toastText !== "") downloading = false

    function openLibrary() {
        send(Protocol.LIBRARY);
        screen = "library";
    }
    function openSources() {
        send(Protocol.SOURCES);
        screen = "sources";
    }
    function openSource(src) {
        entries = { source: src.id, name: src.name, entries: [], loading: true };
        send(Protocol.LIST_FROM, { source: src.id });
        screen = "entries";
    }

    Menu { anchors.fill: parent; app: root; visible: root.screen === "menu" }
    Library { anchors.fill: parent; app: root; visible: root.screen === "library" }
    Sources { anchors.fill: parent; app: root; visible: root.screen === "sources" || root.screen === "entries" }
    GameScreen { anchors.fill: parent; app: root; visible: root.screen === "game" }
}
