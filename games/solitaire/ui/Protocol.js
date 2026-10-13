.pragma library

// Message types, see games/solitaire/backend/protocol.go.

// Frontend -> backend
var NEW_GAME = 100;
var CONTINUE = 101;
var MOVE = 102;
var AUTO = 103;
var STOCK = 104;
var UNDO = 105;
var HINT = 106;
var RESTART = 107;
var QUIT = 108;

// Backend -> frontend
var GAME_STATE = 200;
var MENU = 201;

function duration(seconds) {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return m + ":" + (s < 10 ? "0" : "") + s;
}
