.pragma library

// Message types, see games/crossword/backend/protocol.go.

// Frontend -> backend
var LIBRARY = 100;
var OPEN = 101;
var SOURCES = 102;
var LIST_FROM = 103;
var DOWNLOAD = 104;
var SET = 105;
var STROKES = 106;
var CHECK = 107;
var REVEAL = 108;
var CLOSE = 109;
var DELETE = 110;
var CLEAR_WRONG = 111;
var UNDO = 112;

// Backend -> frontend
var PUZZLE = 200;
var PROGRESS = 201;
var LIBRARY_LIST = 202;
var SOURCE_LIST = 203;
var ENTRIES = 204;
var MENU = 205;

function duration(seconds) {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return m + ":" + (s < 10 ? "0" : "") + s;
}

function date(ms, lang) {
    if (!ms)
        return "";
    const d = new Date(ms);
    return d.toLocaleDateString(Qt.locale(lang === "pt" ? "pt_BR" : "en_US"), "d MMM yyyy");
}
