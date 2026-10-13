.pragma library

// Message types, see games/blackjack/backend/protocol.go.

// Frontend -> backend
var BET = 100;
var DEAL = 101;
var HIT = 102;
var STAND = 103;
var DOUBLE = 104;
var SPLIT = 105;
var SURRENDER = 106;
var HINT = 107;
var RESTART = 108;

// Backend -> frontend
var STATE = 200;

var RESULTS = {
    "win": "Win",
    "blackjack": "Blackjack!",
    "push": "Push",
    "lose": "Lose",
    "bust": "Bust",
    "surrender": "Surrendered"
};

function money(n) {
    return n.toLocaleString(Qt.locale("en_US"), "f", 0);
}

function signed(n) {
    return n > 0 ? "+" + money(n) : n < 0 ? "−" + money(-n) : "±0";
}
