import QtQuick
import "framework"
import "Protocol.js" as Protocol

// The grid to write in with the pen, and all the clues under it. The buttons
// (menu, check, reveal) hide behind a tap on the top edge.
Item {
    id: screen
    property var app
    readonly property real u: app.u
    readonly property var pz: app.puzzle
    readonly property var pr: app.progress
    readonly property int w: pz.width || 1
    readonly property int h: pz.height || 1

    // Clues in the grid, no list: cruzadas diretas always; other puzzles if
    // their clues are short enough to read in a square ("auto"), or if asked.
    // They get a row of clue squares above and a column left of the grid:
    // every answer then has a block right before it to hold its clue.
    property string layout: "auto" // auto, grid, list
    readonly property bool inGrid: !!pz.arrows || layout === "grid" || (layout === "auto" && cluesFit)
    readonly property int off: inGrid && !pz.arrows ? 1 : 0
    readonly property int vw: w + off // the grid as shown
    readonly property int vh: h + off
    // A square shown (d) and the puzzle's square (i, -1 for the added ones)
    function orig(d) {
        const r = Math.floor(d / vw) - off, c = d % vw - off;
        return r < 0 || c < 0 ? -1 : r * w + c;
    }
    function view(i) {
        return (Math.floor(i / w) + off) * vw + i % w + off;
    }
    // Where an answer's clue goes in the added row and column
    function clueSquare(c, acr, shift) {
        const first = (Math.floor(c.cells[0] / w) + shift) * (w + shift) + c.cells[0] % w + shift;
        return acr ? first - 1 : first - (w + shift);
    }

    // Black and white, gray for the selection, red only for wrong letters
    readonly property color selectedColor: "#c8c8c8"
    readonly property color mistakeColor: "#d32020"

    // Selection: a square and a direction
    property int cell: -1
    property bool across: true
    property bool showBar: false
    property double touchTime: 0     // last finger on the grid
    property double twoFingerTime: 0 // last two finger tap
    property int peek: -1            // clue square shown big while held (square shown)

    readonly property var clue: findClue(cell, across) || findClue(cell, !across)
    // (objects read from a var property are new wrappers each time, so no === on them)
    readonly property bool clueAcross: findClue(cell, across) ? across : !across

    function findClue(i, acr) {
        const list = acr ? pz.across : pz.down;
        if (!list || i < 0)
            return null;
        for (let k = 0; k < list.length; k++)
            if (list[k].cells.indexOf(i) >= 0)
                return list[k];
        return null;
    }
    function entry(i) {
        return pr.entries && pr.entries[i] ? pr.entries[i] : "";
    }
    function isWrong(i) {
        return !!pr.wrong && pr.wrong.indexOf(i) >= 0;
    }

    // The clues in each block (by square shown, with shift 1 for an added
    // row and column), the one whose answer starts first on top
    function placeClues(shift) {
        const out = {};
        for (const [list, acr] of [[pz.across || [], true], [pz.down || [], false]])
            for (let k = 0; k < list.length; k++) {
                const c = list[k];
                const at = pz.arrows ? c.at : clueSquare(c, acr, shift);
                (out[at] = out[at] || []).push({ clue: c, across: acr });
            }
        for (const at in out)
            out[at].sort((a, b) => a.clue.cells[0] - b.clue.cells[0]);
        return out;
    }
    readonly property var cellClues: inGrid ? placeClues(off) : ({})
    // The arrow at the start of each answer (by the puzzle's square): from
    // which side its clue is, and whether it goes right (across) or down
    readonly property var arrows: {
        const out = {};
        for (const at in cellClues)
            for (const item of cellClues[at]) {
                const first = item.clue.cells[0], d = view(first);
                const dr = Math.floor(d / vw) - Math.floor(at / vw), dc = d % vw - at % vw;
                const from = dr === 0 ? (dc > 0 ? "left" : "right") : dc === 0 ? (dr > 0 ? "top" : "bottom") : "topleft";
                (out[first] = out[first] || []).push({ from: from, across: item.across });
            }
        return out;
    }

    // Whether the clues can be read in the squares: Qt lays out every clue in
    // its part of a square (measure) with letters at least 1.5 mm tall.
    readonly property real clueMargin: Math.max(2, grid.cellSize * 0.025)
    property bool cluesFit: false
    function checkFit() {
        if (pz.arrows || !pz.cells) {
            cluesFit = false;
            return;
        }
        const size = Math.floor(Math.min((width - 4 * u) / (w + 1), (height - 8 * u) / (h + 1), 10 * u));
        const margin = Math.max(2, size * 0.025);
        const placed = placeClues(1);
        let fit = true;
        for (const at in placed) {
            const n = placed[at].length;
            measure.width = size - 2 * margin;
            for (const item of placed[at]) {
                measure.text = item.clue.text;
                if (measure.contentWidth > measure.width + 0.5 || measure.contentHeight > size / n - 2 * margin)
                    fit = false;
            }
        }
        cluesFit = fit;
    }
    onWidthChanged: checkFit()
    onHeightChanged: checkFit()

    Text {
        id: measure
        visible: false
        wrapMode: Text.WordWrap
        textFormat: Text.PlainText
        renderType: Text.NativeRendering
        font.pixelSize: 14
    }

    // A tap on a square shown: selects it; on the selected one, switches
    // direction; on a block with clues, selects its answer.
    function select(d) {
        if (d < 0 || !pz.cells)
            return;
        const written = cellClues[d];
        if (written) {
            // A block with clues: its answer (the other one, if it has two)
            let k = 0;
            for (let j = 0; j < written.length; j++)
                if (clue && written[j].clue.number === clue.number && written[j].across === clueAcross)
                    k = (j + 1) % written.length;
            selectClue(written[k].clue, written[k].across);
            return;
        }
        const i = orig(d);
        if (i < 0 || pz.cells[i].b)
            return;
        if (i === cell || !findClue(i, across))
            across = !across;
        cell = i;
    }

    function selectClue(c, acr) {
        across = acr;
        // Start at the first empty square of the answer
        let target = c.cells[0];
        for (let k = 0; k < c.cells.length; k++) {
            if (entry(c.cells[k]) === "") {
                target = c.cells[k];
                break;
            }
        }
        cell = target;
    }

    onVisibleChanged: app.penActive = visible

    onPzChanged: {
        cell = -1;
        layout = "auto";
        Qt.callLater(checkFit); // once w and h have the new puzzle's size
        showBar = false;
        if (pz.across && pz.across.length > 0)
            selectClue(pz.across[0], true);
    }

    // Two fingers tap: undo (or drop the letter being written). It's under
    // everything else, or it would take single taps too.
    MultiPointTouchArea {
        anchors.fill: parent
        mouseEnabled: false
        minimumTouchPoints: 2
        maximumTouchPoints: 2
        property double start: 0
        property bool moved: false
        property bool done: false

        onPressed: (points) => {
            start = Date.now();
            screen.twoFingerTime = start;
            moved = false;
            done = false;
        }
        onUpdated: (points) => {
            for (let k = 0; k < points.length; k++)
                if (Math.abs(points[k].x - points[k].startX) + Math.abs(points[k].y - points[k].startY) > 4 * screen.u)
                    moved = true;
        }
        onReleased: (points) => {
            screen.twoFingerTime = Date.now();
            if (done || moved || Date.now() - start > 700 || ink.penNear || Date.now() - ink.penTime < 800)
                return;
            done = true;
            if (ink.group.length > 0 || ink.current.length > 2)
                ink.discard();
            else
                screen.app.send(Protocol.UNDO);
        }
    }

    // The top edge: tap to show the buttons
    Item {
        id: topEdge
        anchors.top: parent.top
        width: parent.width
        height: 4 * screen.u

        Rectangle {
            anchors.centerIn: parent
            width: 12 * screen.u
            height: Math.max(3, 0.5 * screen.u)
            radius: height / 2
            color: "#909090"
        }
        MouseArea {
            anchors.fill: parent
            onClicked: screen.showBar = !screen.showBar
        }
    }

    // The grid
    Rectangle {
        id: grid
        readonly property real room: screen.inGrid ? screen.height - 2 * topEdge.height : screen.height * 0.535
        // At most about 12 mm, like a paper crossword; bigger with the clues
        // in the grid, so they can be read
        readonly property real cellSize: Math.floor(Math.min((screen.width - 4 * screen.u) / screen.vw, room / screen.vh,
                                                             (screen.inGrid ? 10 : 6.5) * screen.u))
        // With the clues in the grid it sits in the middle of the screen
        y: screen.inGrid ? Math.max(topEdge.height, (screen.height - height) / 2) : topEdge.height
        anchors.horizontalCenter: parent.horizontalCenter
        width: cellSize * screen.vw + 2
        height: cellSize * screen.vh + 2
        color: "black"

        Grid {
            x: 1
            y: 1
            columns: screen.vw

            Repeater {
                model: screen.pz.cells ? screen.vw * screen.vh : 0

                // Black and white (the selected square gray): e-ink redraws
                // that fastest, so the ink keeps up with the pen.
                Rectangle {
                    id: square
                    readonly property int cell: screen.orig(index) // the puzzle's square
                    readonly property var c: cell >= 0 && screen.pz.cells[cell] || { b: true }
                    readonly property var written: screen.cellClues[index] || []
                    readonly property bool clueSquare: written.length > 0
                    width: grid.cellSize
                    height: grid.cellSize
                    color: c.b && !clueSquare ? "black" : cell >= 0 && cell === screen.cell ? screen.selectedColor : "white"
                    border.width: c.b && !clueSquare ? 0 : 1
                    border.color: "black"

                    // The clues written in a block, one above the other
                    Repeater {
                        model: square.written.length
                        Item {
                            y: index * square.height / square.written.length
                            width: square.width
                            height: square.height / square.written.length
                            Rectangle {
                                visible: index > 0
                                width: parent.width
                                height: 1
                                color: "black"
                            }
                            Text {
                                anchors.fill: parent
                                anchors.margins: screen.clueMargin
                                horizontalAlignment: Text.AlignHCenter
                                verticalAlignment: Text.AlignVCenter
                                wrapMode: Text.WordWrap // long words shrink rather than break
                                textFormat: Text.PlainText
                                fontSizeMode: Text.Fit
                                font.pixelSize: grid.cellSize * 0.2
                                renderType: Text.NativeRendering
                                text: square.written[index].clue.text
                            }
                        }
                    }

                    // Arrows from the clues to where their answers start
                    Repeater {
                        model: square.cell >= 0 ? (screen.arrows[square.cell] || []).length : 0
                        Canvas {
                            width: grid.cellSize * 0.24
                            height: width
                            readonly property var arrow: (screen.arrows[square.cell] || [])[index] || { from: "left", across: true }
                            x: arrow.from === "left" || arrow.from === "topleft" ? 1 : arrow.from === "right" ? square.width - width - 1 : (square.width - width) / 2
                            y: arrow.from === "top" || arrow.from === "topleft" ? 1 : arrow.from === "bottom" ? square.height - height - 1 : (square.height - height) / 2
                            onPaint: {
                                const ctx = getContext("2d");
                                const s = width;
                                ctx.reset();
                                ctx.fillStyle = "black";
                                ctx.beginPath();
                                if (arrow.across) {
                                    ctx.moveTo(s * 0.15, s * 0.1); ctx.lineTo(s * 0.9, s * 0.5); ctx.lineTo(s * 0.15, s * 0.9);
                                } else {
                                    ctx.moveTo(s * 0.1, s * 0.15); ctx.lineTo(s * 0.5, s * 0.9); ctx.lineTo(s * 0.9, s * 0.15);
                                }
                                ctx.closePath();
                                ctx.fill();
                            }
                        }
                    }

                    Rectangle {
                        visible: !!c.c
                        anchors.fill: parent
                        anchors.margins: parent.width * 0.06
                        radius: width / 2
                        color: "transparent"
                        border.width: 1
                        border.color: "black"
                    }
                    Text {
                        visible: !!c.n && !screen.inGrid
                        x: grid.cellSize * 0.06
                        y: grid.cellSize * 0.02
                        text: c.n || ""
                        font.pixelSize: grid.cellSize * 0.26
                    }
                    Text {
                        anchors.centerIn: parent
                        anchors.verticalCenterOffset: grid.cellSize * 0.07
                        text: screen.entry(square.cell)
                        font.pixelSize: grid.cellSize * 0.58
                        font.bold: true
                        color: screen.isWrong(square.cell) ? screen.mistakeColor : "black"
                        font.italic: !!screen.pr.revealed && screen.pr.revealed.indexOf(square.cell) >= 0
                    }
                }
            }
        }

        // Writing with the pen straight into the squares. A letter's strokes
        // go to the square they're drawn over and are read once the pen has
        // rested for a moment or starts on another square. Ink is drawn as
        // small pieces, so the screen only redraws right under the pen.
        // The eraser end of the pen clears the letters it passes over.
        // Fingers don't write: a quick tap selects a square, a two finger tap
        // undoes (see the MultiPointTouchArea below).
        Item {
            id: ink
            anchors.fill: parent
            property var group: []       // strokes of the letter being written
            property var groupInk: []
            property int target: -1      // its square
            property var current: []     // the stroke being drawn: x0, y0, x1, y1, ...
            property var currentInk: []
            property string tool: ""     // "pen", "eraser" or "" while the pen is up
            property bool moved: false
            property real startX: 0
            property real startY: 0
            property double startTime: 0
            property real lastX: 0
            property real lastY: 0
            property bool erasedAny: false
            property double penTime: 0   // last pen event, to ignore its mouse copies
            readonly property real thickness: Math.max(3, Math.round(grid.cellSize * 0.045))

            function cellAt(x, y) {
                const c = Math.floor((x - 1) / grid.cellSize);
                const r = Math.floor((y - 1) / grid.cellSize);
                return c < 0 || r < 0 || c >= screen.vw || r >= screen.vh ? -1 : r * screen.vw + c;
            }
            function center(s) {
                let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity;
                for (let i = 0; i + 1 < s.length; i += 2) {
                    x0 = Math.min(x0, s[i]); x1 = Math.max(x1, s[i]);
                    y0 = Math.min(y0, s[i + 1]); y1 = Math.max(y1, s[i + 1]);
                }
                return { x: (x0 + x1) / 2, y: (y0 + y1) / 2 };
            }
            // Strokes of one letter may stick out of its square a bit.
            function nearTarget(x, y) {
                const size = grid.cellSize;
                const d = screen.view(target);
                const cx = (d % screen.vw + 0.5) * size + 1;
                const cy = (Math.floor(d / screen.vw) + 0.5) * size + 1;
                return Math.abs(x - cx) < size * 0.75 && Math.abs(y - cy) < size * 0.75;
            }
            function segment(x0, y0, x1, y1) {
                const dx = x1 - x0, dy = y1 - y0;
                const len = Math.sqrt(dx * dx + dy * dy);
                currentInk.push(piece.createObject(ink, {
                    x: (x0 + x1 - len - thickness) / 2,
                    y: (y0 + y1 - thickness) / 2,
                    width: len + thickness,
                    height: thickness,
                    rotation: Math.atan2(dy, dx) * 180 / Math.PI
                }));
            }
            function destroyAll(pieces) {
                for (let k = 0; k < pieces.length; k++)
                    pieces[k].destroy();
            }
            // Drops the letter being written without reading it.
            function discard() {
                idle.stop();
                destroyAll(groupInk);
                destroyAll(currentInk);
                group = [];
                groupInk = [];
                current = [];
                currentInk = [];
                target = -1;
            }
            function flush() {
                idle.stop();
                if (group.length > 0)
                    screen.app.send(Protocol.STROKES, { cell: target, strokes: group });
                destroyAll(groupInk);
                group = [];
                groupInk = [];
                target = -1;
            }
            function eraseAt(x, y) {
                const i = screen.orig(cellAt(x, y));
                if (i < 0 || screen.pz.cells[i].b)
                    return;
                if (group.length > 0 && i === target)
                    discard();
                if (screen.entry(i) !== "") {
                    screen.app.send(Protocol.SET, { cell: i, letter: "", join: erasedAny });
                    erasedAny = true;
                }
            }

            function penDown(kind, pos) {
                idle.stop();
                tool = kind;
                moved = false;
                startX = lastX = pos.x;
                startY = lastY = pos.y;
                startTime = Date.now();
                erasedAny = false;
                current = [pos.x, pos.y];
                currentInk = [];
                if (kind === "eraser")
                    eraseAt(pos.x, pos.y);
                else if (screen.cellClues[cellAt(pos.x, pos.y)])
                    hold.restart();
            }
            function penMove(pos) {
                if (tool === "")
                    return;
                if (!moved) {
                    if (Math.abs(pos.x - startX) + Math.abs(pos.y - startY) < 4)
                        return;
                    moved = true;
                    hold.stop();
                }
                const dx = pos.x - lastX, dy = pos.y - lastY;
                if (dx * dx + dy * dy < 9)
                    return;
                if (tool === "eraser") {
                    // Every square along the way, even if the pen moved fast
                    const steps = Math.ceil(Math.sqrt(dx * dx + dy * dy) / (grid.cellSize / 3));
                    for (let k = 1; k <= steps; k++)
                        eraseAt(lastX + dx * k / steps, lastY + dy * k / steps);
                } else {
                    segment(lastX, lastY, pos.x, pos.y);
                    current.push(pos.x, pos.y);
                }
                lastX = pos.x;
                lastY = pos.y;
            }
            function penUp() {
                const kind = tool;
                tool = "";
                hold.stop();
                if (screen.peek >= 0) {
                    screen.peek = -1;
                    current = [];
                    return;
                }
                if (kind !== "pen")
                    return;
                if (!moved) {
                    // A tap with the pen: a dot while writing, else selects
                    current = [];
                    if (group.length > 0 && nearTarget(startX, startY)) {
                        idle.restart();
                        return;
                    }
                    flush();
                    if (Date.now() - startTime < 350)
                        screen.select(cellAt(startX, startY));
                    return;
                }
                const s = current, pieces = currentInk;
                current = [];
                currentInk = [];
                const p = center(s);
                if (group.length > 0 && !nearTarget(p.x, p.y))
                    flush();
                if (group.length === 0) {
                    const i = screen.orig(cellAt(p.x, p.y));
                    if (i < 0 || screen.pz.cells[i].b) {
                        destroyAll(pieces);
                        return;
                    }
                    target = i;
                }
                group.push(s);
                for (let k = 0; k < pieces.length; k++)
                    groupInk.push(pieces[k]);
                idle.restart();
            }

            Component {
                id: piece
                Rectangle {
                    color: "black"
                    antialiasing: false
                }
            }

            // The pen as the backend reads it from the tablet (see GameApp.pen)
            property bool penNear: false
            Connections {
                target: screen.app
                function onPen(ev) {
                    ink.penTime = Date.now();
                    ink.handlePen(ev);
                }
            }
            function handlePen(ev) {
                    if (ev.e === "near" || ev.e === "far") {
                        ink.penNear = ev.e === "near";
                        return;
                    }
                    if (!screen.visible || screen.pr.solved || screen.showBar)
                        return;
                    const pts = ev.pts || [];
                    const at = (i) => ink.mapFromItem(null, pts[i], pts[i + 1]);
                    if (ev.e === "down") {
                        ink.penDown(ev.t, at(0));
                        return;
                    }
                    if (ink.tool === "")
                        return;
                    for (let i = 0; i + 1 < pts.length; i += 2)
                        ink.penMove(at(i));
                    if (ev.e === "up")
                        ink.penUp();
            }

            // Without the backend's pen (the emulator): Qt's pointer events.
            // The pen's tip and its eraser end
            PointHandler {
                acceptedDevices: PointerDevice.Stylus
                acceptedPointerTypes: PointerDevice.Pen | PointerDevice.Generic
                enabled: !screen.pr.solved && !screen.app.penAvailable
                onActiveChanged: active ? ink.penDown("pen", point.position) : ink.penUp()
                onPointChanged: {
                    ink.penTime = Date.now();
                    if (active)
                        ink.penMove(point.position);
                }
            }
            PointHandler {
                acceptedDevices: PointerDevice.Stylus
                acceptedPointerTypes: PointerDevice.Eraser
                enabled: !screen.pr.solved && !screen.app.penAvailable
                onActiveChanged: active ? ink.penDown("eraser", point.position) : ink.penUp()
                onPointChanged: {
                    ink.penTime = Date.now();
                    if (active)
                        ink.penMove(point.position);
                }
            }
            // A mouse is the pen too (the emulator, or a pen that comes as a
            // mouse), unless it's a copy of a pen or finger event.
            PointHandler {
                id: mouse
                acceptedDevices: PointerDevice.Mouse
                enabled: !screen.pr.solved && !screen.app.penAvailable
                property bool copy: false
                onActiveChanged: {
                    if (active) {
                        copy = Date.now() - ink.penTime < 300 || Date.now() - screen.touchTime < 300;
                        if (!copy)
                            ink.penDown("pen", point.position);
                    } else if (!copy) {
                        ink.penUp();
                    }
                }
                onPointChanged: if (active && !copy) ink.penMove(point.position)
            }
            // A finger: a quick tap selects a square. Not while the pen is
            // near, that's the hand resting on the screen.
            // Holding a clue square shows it big until let go.
            TapHandler {
                acceptedDevices: PointerDevice.TouchScreen
                longPressThreshold: 0.4
                onPressedChanged: {
                    screen.touchTime = Date.now();
                    if (!pressed)
                        screen.peek = -1;
                }
                onLongPressed: {
                    const d = ink.cellAt(point.position.x, point.position.y);
                    if (!ink.penNear && screen.cellClues[d])
                        screen.peek = d;
                }
                onTapped: (eventPoint) => {
                    screen.touchTime = Date.now();
                    if (ink.penNear || Date.now() - ink.penTime < 800)
                        return;
                    if (Date.now() - screen.twoFingerTime > 600)
                        screen.select(ink.cellAt(eventPoint.position.x, eventPoint.position.y));
                }
            }

            Timer {
                id: hold
                interval: 400
                onTriggered: screen.peek = ink.cellAt(ink.startX, ink.startY)
            }

            Timer {
                id: idle
                interval: 700
                onTriggered: ink.flush()
            }
        }
    }

    // All the clues, in columns: across, then down
    Item {
        id: clues
        anchors.top: grid.bottom
        anchors.topMargin: 1.5 * screen.u
        anchors.left: parent.left
        anchors.leftMargin: 2 * screen.u
        anchors.right: parent.right
        anchors.rightMargin: 2 * screen.u
        anchors.bottom: parent.bottom
        anchors.bottomMargin: 1 * screen.u
        clip: true
        visible: !screen.inGrid

        readonly property int columns: width > 90 * screen.u ? 3 : 2
        readonly property real gap: 1.5 * screen.u
        readonly property real columnWidth: (width - (columns - 1) * gap) / columns
        readonly property real fontSize: 1.4 * screen.u
        readonly property var items: {
            const out = [];
            const add = (title, list, acr) => {
                if (!list || list.length === 0)
                    return;
                out.push({ title: title });
                for (let k = 0; k < list.length; k++)
                    out.push({ clue: list[k], across: acr });
            };
            add(screen.app.tr("Across", "Horizontais"), screen.pz.across, true);
            add(screen.app.tr("Down", "Verticais"), screen.pz.down, false);
            return out;
        }

        Flow {
            anchors.fill: parent
            flow: Flow.TopToBottom
            spacing: 0

            Repeater {
                model: clues.items

                Rectangle {
                    // (the model holds copies of the clues, so compare by number)
                    readonly property bool current: !!modelData.clue && !!screen.clue
                        && modelData.clue.number === screen.clue.number && modelData.across === screen.clueAcross
                    width: clues.columnWidth + clues.gap
                    height: label.implicitHeight + 0.5 * screen.u
                    color: "transparent"

                    Rectangle {
                        anchors.fill: parent
                        anchors.rightMargin: clues.gap
                        visible: parent.current
                        color: screen.selectedColor
                        radius: 0.4 * screen.u
                    }
                    Text {
                        id: label
                        anchors.verticalCenter: parent.verticalCenter
                        x: 0.4 * screen.u
                        width: clues.columnWidth - 0.8 * screen.u
                        wrapMode: Text.Wrap
                        textFormat: Text.StyledText
                        text: modelData.title ? "<b>" + modelData.title + "</b>"
                            : "<b>" + modelData.clue.number + "</b> " + modelData.clue.text
                        font.pixelSize: modelData.title ? clues.fontSize * 1.1 : clues.fontSize
                        renderType: Text.NativeRendering
                    }
                    TapHandler {
                        enabled: !!modelData.clue
                        onTapped: if (Date.now() - screen.twoFingerTime > 600) screen.selectClue(modelData.clue, modelData.across)
                    }
                }
            }
        }
    }

    // A held clue square, big, next to it
    Rectangle {
        id: peekBox
        visible: screen.peek >= 0
        readonly property var written: screen.cellClues[screen.peek] || []
        readonly property real squareX: grid.x + 1 + (screen.peek % screen.vw) * grid.cellSize
        readonly property real squareY: grid.y + 1 + Math.floor(screen.peek / screen.vw) * grid.cellSize
        width: Math.min(60 * screen.u, screen.width - 4 * screen.u)
        height: peekColumn.height + 3 * screen.u
        x: Math.max(2 * screen.u, Math.min(screen.width - width - 2 * screen.u, squareX + grid.cellSize / 2 - width / 2))
        // Above the square if there's room, else below it
        y: squareY - height - screen.u > 0 ? squareY - height - screen.u : squareY + grid.cellSize + screen.u
        color: "white"
        border.width: Math.max(3, screen.u / 4)
        border.color: "black"

        Column {
            id: peekColumn
            x: 2 * screen.u
            y: 1.5 * screen.u
            width: parent.width - 4 * screen.u
            spacing: 1.5 * screen.u
            Repeater {
                model: peekBox.written.length
                Text {
                    width: peekColumn.width
                    wrapMode: Text.WordWrap
                    textFormat: Text.PlainText
                    renderType: Text.NativeRendering
                    font.pixelSize: 5 * screen.u
                    text: (peekBox.written[index].across ? "▶ " : "▼ ")
                        + peekBox.written[index].clue.text.replace(/-\n/g, "").replace(/\n/g, " ")
                }
            }
        }
    }

    // The buttons, shown after a tap on the top edge
    MouseArea {
        anchors.fill: parent
        visible: screen.showBar
        onClicked: screen.showBar = false
    }

    Rectangle {
        visible: screen.showBar
        width: parent.width
        height: barColumn.height + 3 * screen.u
        color: "white"

        Rectangle {
            anchors.bottom: parent.bottom
            width: parent.width
            height: Math.max(2, 0.3 * screen.u)
            color: "black"
        }
        MouseArea { anchors.fill: parent }

        Column {
            id: barColumn
            x: 2 * screen.u
            y: 1.5 * screen.u
            width: parent.width - 4 * screen.u
            spacing: 1.2 * screen.u

            Text {
                width: parent.width
                elide: Text.ElideRight
                text: (screen.pz.title || "") + "  ·  " + (screen.pr.filled || 0) + "/" + (screen.pr.total || 0)
                    + (screen.pz.author ? "  ·  " + screen.pz.author : "")
                font.pixelSize: 3 * screen.u
            }
            Row {
                spacing: 1.5 * screen.u
                MenuButton {
                    fontSize: 3.4 * screen.u
                    text: screen.app.tr("Menu", "Menu")
                    onClicked: screen.app.send(Protocol.CLOSE)
                }
                MenuButton {
                    fontSize: 3.4 * screen.u
                    text: screen.app.tr("Check", "Verificar")
                    enabled: !!screen.pz.hasSolution && !screen.pr.solved
                    onClicked: { screen.showBar = false; screen.app.send(Protocol.CHECK); }
                }
                MenuButton {
                    fontSize: 3.4 * screen.u
                    text: screen.app.tr("Clear wrong", "Apagar erros")
                    visible: !!screen.pr.wrong && screen.pr.wrong.length > 0
                    onClicked: { screen.showBar = false; screen.app.send(Protocol.CLEAR_WRONG); }
                }
                MenuButton {
                    fontSize: 3.4 * screen.u
                    visible: !screen.pz.arrows
                    text: screen.inGrid ? screen.app.tr("Clue list", "Lista de pistas") : screen.app.tr("Clues in grid", "Pistas na grade")
                    onClicked: { screen.showBar = false; screen.layout = screen.inGrid ? "list" : "grid"; }
                }
                MenuButton {
                    fontSize: 3.4 * screen.u
                    text: screen.app.tr("Reveal letter", "Revelar letra")
                    enabled: !!screen.pz.hasSolution && !screen.pr.solved && screen.cell >= 0
                    onClicked: { screen.showBar = false; screen.app.send(Protocol.REVEAL, { cell: screen.cell }); }
                }
            }
        }
    }

    // Solved
    Rectangle {
        anchors.centerIn: parent
        visible: !!screen.pr.solved
        width: 70 * screen.u
        height: solvedColumn.height + 6 * screen.u
        radius: 2 * screen.u
        color: "white"
        border.width: Math.max(3, screen.u / 3)
        border.color: "black"

        MouseArea { anchors.fill: parent }

        Column {
            id: solvedColumn
            anchors.centerIn: parent
            spacing: 2 * screen.u
            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                text: screen.app.tr("Solved!", "Resolvida!")
                font.pixelSize: 8 * screen.u
                font.bold: true
                color: "black"
            }
            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                text: Protocol.duration(screen.pr.seconds || 0)
                font.pixelSize: 4.5 * screen.u
            }
            MenuButton {
                anchors.horizontalCenter: parent.horizontalCenter
                width: 36 * screen.u
                fontSize: 4 * screen.u
                text: screen.app.tr("Back to the menu", "Voltar ao menu")
                onClicked: screen.app.send(Protocol.CLOSE)
            }
        }
    }
}
