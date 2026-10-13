.pragma library

// Message types, see games/chess/backend/protocol.go.

// Frontend -> backend
var NEW_GAME = 100;
var MOVE = 101;
var UNDO = 102;
var QUIT_GAME = 103;
var GET_SLOTS = 104;
var LIST_PGNS = 110;
var OPEN_PGN = 111;
var START_REPLAY = 112;
var REPLAY_NEXT = 113;
var REPLAY_UNDO = 114;
var REPLAY_RESET = 115;
var REPLAY_MOVE = 116;
var REPLAY_CLOSE = 117;

// Backend -> frontend
var GAME_STATE = 200;
var SLOTS = 201;
var PGN_LIST = 202;
var GAME_LIST = 203;
var REPLAY_STATE = 204;

var MODE_NAMES = {
    "pvp": "Player vs Player",
    "easy": "Player vs Bot (Easy)",
    "normal": "Player vs Bot (Normal)",
    "hard": "Player vs Bot (Hard)"
};
