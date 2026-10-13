.pragma library

// Message types, see games/mahjong/backend/protocol.go.

// Frontend -> backend
var NEW_GAME = 100;
var CONTINUE = 101;
var REMOVE = 102;
var UNDO = 103;
var HINT = 104;
var SHUFFLE = 105;
var QUIT = 106;

// Backend -> frontend
var GAME_STATE = 200;
var MENU = 201;

// Faces 34-37 are flowers and 38-41 seasons; each group matches among itself.
function matchClass(face) {
    return face >= 38 ? 38 : face >= 34 ? 34 : face;
}

function duration(seconds) {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return m + ":" + (s < 10 ? "0" : "") + s;
}
