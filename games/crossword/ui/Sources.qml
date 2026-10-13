import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

// Where puzzles come from, and what each source offers.
Item {
    id: screen
    property var app
    readonly property var sortedSources: {
        // The user's language first, otherwise in the backend's order
        const lang = app.lang;
        const rank = s => (s.language === lang ? 0 : 1000) + app.sources.indexOf(s);
        return app.sources.slice().sort((a, b) => rank(a) - rank(b));
    }

    PagedList {
        anchors.fill: parent
        app: screen.app
        visible: screen.app.screen === "sources"
        title: screen.app.tr("Get puzzles", "Baixar cruzadas")
        heading: screen.app.tr("Free puzzles from the web (needs Wi-Fi)", "Cruzadas grátis da internet (precisa de Wi-Fi)")
        backText: screen.app.tr("Back", "Voltar")
        emptyText: screen.app.tr("Loading...", "Carregando...")
        items: screen.sortedSources.map(s => s.name + "  ·  " + s.language.toUpperCase())
        onItemClicked: (index) => screen.app.openSource(screen.sortedSources[index])
        onBack: screen.app.screen = "menu"
    }

    PagedList {
        anchors.fill: parent
        app: screen.app
        visible: screen.app.screen === "entries"
        title: screen.app.entries.name || ""
        heading: screen.app.downloading ? screen.app.tr("Downloading...", "Baixando...") : screen.app.tr("Choose a puzzle", "Escolha uma cruzada")
        backText: screen.app.tr("Sources", "Fontes")
        emptyText: screen.app.entries.loading ? screen.app.tr("Loading...", "Carregando...") : (screen.app.entries.error || screen.app.tr("Nothing here.", "Nada aqui."))
        items: screen.app.entries.entries.map(e => (e.title || Protocol.date(e.date, screen.app.lang) || e.id)
            + "  ·  " + e.size + (e.downloaded ? "  ·  ✓" : ""))
        onItemClicked: (index) => {
            screen.app.downloading = true;
            screen.app.send(Protocol.DOWNLOAD, { source: screen.app.entries.source, id: screen.app.entries.entries[index].id });
        }
        onBack: screen.app.openSources()
    }
}
