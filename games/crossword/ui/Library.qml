import QtQuick 2.5
import "framework"
import "Protocol.js" as Protocol

// The puzzles on the tablet.
PagedList {
    id: lib
    title: app.tr("My puzzles", "Minhas cruzadas")
    heading: app.tr("Tap to play · hold to delete a download", "Toque para jogar · segure para apagar")
    backText: app.tr("Back", "Voltar")
    emptyText: app.tr("No puzzles yet. Get some with \"Get puzzles\", or copy .puz and .ipuz files to\n",
                      "Nenhuma cruzada ainda. Baixe algumas em \"Baixar cruzadas\" ou copie arquivos .puz e .ipuz para\n")
               + app.library.dir
    items: app.library.items.map(it => it.title + "  ·  " + it.origin
        + (it.solved ? "  ·  ✓" : it.percent > 0 ? "  ·  " + it.percent + "%" : ""))

    onItemClicked: (index) => app.send(Protocol.OPEN, { key: app.library.items[index].key })
    onItemHeld: (index) => app.send(Protocol.DELETE, { key: app.library.items[index].key })
    onBack: app.screen = "menu"
}
