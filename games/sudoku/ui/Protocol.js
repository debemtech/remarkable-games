.pragma library

// Message types, see games/sudoku/backend/protocol.go.

// Frontend -> backend
var NEW_GAME = 100;
var CONTINUE = 101;
var SET = 102;
var NOTE = 103;
var UNDO = 104;
var HINT = 105;
var CHECK = 106;
var RESTART = 107;
var QUIT = 108;

// Backend -> frontend
var GAME_STATE = 200;
var MENU = 201;

var DIFFICULTIES = ["easy", "medium", "hard", "expert"];

function title(s) {
    return s ? s.charAt(0).toUpperCase() + s.slice(1) : "";
}

function duration(seconds) {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return m + ":" + (s < 10 ? "0" : "") + s;
}
