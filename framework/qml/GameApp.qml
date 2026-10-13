import QtQuick 2.5
import net.asivery.AppLoad 1.0
import "Framework.js" as Framework

// Root item of a game. It connects to the backend, shows toasts and provides
// a size unit. A game's main.qml looks like:
//
//   GameApp {
//       id: app
//       appId: "mygame"            // the id from manifest.json
//       screen: "menu"
//       onMessage: (type, data) => { ... }
//       MyMenu { anchors.fill: parent; visible: app.screen === "menu" }
//   }
//
// Screens call app.send(type, payload) and use app.u for all sizes.
//
// Set orientation: "landscape" for a game played with the tablet on its side.
// The content is then turned by 90° (flipLandscape turns it the other way).
//
// Preferences: app.prefs is an object the backend saves if it calls
// App.UsePrefs; change it with app.setPref(key, value). The landscape flip is
// kept there ("flip").
//
// The pen: xochitl only passes a late, mouse-like copy of it to apps. If the
// backend calls pen.Serve, set penActive while a screen wants the pen; the
// backend then reads the pen itself and the pen signal brings its events
// ({e: "near"/"far"/"down"/"move"/"up", t: "pen"/"eraser", pts: [x, y, ...]},
// points in window pixels). penAvailable says whether that works; if not,
// use Qt's pointer events.
Rectangle {
    id: app
    anchors.fill: parent
    color: "white"

    property string appId
    // Name of the visible screen, for games to switch on.
    property string screen
    property string orientation: "portrait"
    property var prefs: ({})
    property bool flipLandscape: !!prefs.flip
    // Language of the interface: "en" or "pt", kept in the preferences
    readonly property string lang: prefs.lang === "pt" ? "pt" : "en"
    readonly property bool rotated: orientation === "landscape" && height > width
    // Size of the game area, after rotation
    readonly property real contentWidth: rotated ? height : width
    readonly property real contentHeight: rotated ? width : height
    // 1% of the shorter screen side (16.2 px on the Paper Pro)
    readonly property real u: Math.min(width, height) / 100
    // Space the toast bar takes at the bottom
    readonly property real toastHeight: 7 * u
    // Stop the backend when the app is closed
    property bool terminateOnClose: true

    property string toastText: ""
    property string toastKind: "info"

    default property alias content: contentArea.data

    property bool penActive: false
    property bool penAvailable: false
    onPenActiveChanged: send(Framework.PEN, { on: penActive })

    // A message from the backend, with its JSON body decoded
    signal message(int type, var data)
    signal pen(var event)

    // Required by AppLoad
    signal close
    function unloading() {
        if (terminateOnClose)
            endpoint.terminate();
    }

    function send(type, payload) {
        endpoint.sendMessage(type, JSON.stringify(payload === undefined ? {} : payload));
    }

    function setPref(key, value) {
        const p = Object.assign({}, prefs);
        p[key] = value;
        prefs = p;
        const changed = {};
        changed[key] = value;
        send(Framework.PREFS, changed);
    }

    // The text in the user's language: app.tr("Check", "Verificar")
    function tr(en, pt) {
        return lang === "pt" ? pt : en;
    }

    function toggleFlip() {
        setPref("flip", !flipLandscape);
    }

    function showToast(text, kind, ms) {
        toastText = text;
        toastKind = kind || "info";
        toastTimer.stop();
        if (ms > 0) {
            toastTimer.interval = ms;
            toastTimer.start();
        }
    }

    function hideToast() {
        toastText = "";
    }

    AppLoad {
        id: endpoint
        applicationID: app.appId
        onMessageReceived: (type, contents) => {
            let data;
            try {
                data = contents.length > 0 ? JSON.parse(contents) : {};
            } catch (e) {
                console.log(app.appId + ": invalid message from the backend:", type, contents);
                return;
            }
            if (type === Framework.PEN) {
                if (data.e === "ready" || data.e === "none") {
                    app.penAvailable = data.e === "ready";
                    console.log(app.appId + ": pen", data.e, data.t || "", "window", app.Window.width, "x", app.Window.height);
                    return;
                }
                if (data.pts) {
                    // Screen proportions to window pixels
                    const w = app.Window.width, h = app.Window.height;
                    for (let i = 0; i + 1 < data.pts.length; i += 2) {
                        data.pts[i] *= w;
                        data.pts[i + 1] *= h;
                    }
                }
                app.pen(data);
            } else if (type === Framework.TOAST)
                app.showToast(data.text, data.kind, data.ms);
            else if (type === Framework.PREFS)
                app.prefs = data;
            else
                app.message(type, data);
        }
    }

    // If the backend isn't connected yet, AppLoad drops this. The backend then
    // syncs when it connects (new coordinator), so both orders work.
    Component.onCompleted: {
        send(Framework.HELLO);
        if (penActive)
            send(Framework.PEN, { on: true });
    }

    Item {
        id: rotator
        anchors.centerIn: parent
        width: app.contentWidth
        height: app.contentHeight
        rotation: app.rotated ? (app.flipLandscape ? 270 : 90) : 0

        Item {
            id: contentArea
            anchors.fill: parent
        }

        Rectangle {
            anchors.bottom: parent.bottom
            width: parent.width
            height: app.toastHeight
            visible: app.toastText !== ""
            color: "white"

            Rectangle {
                width: parent.width
                height: Math.max(2, app.u / 5)
                color: "black"
            }

            Text {
                anchors.centerIn: parent
                width: parent.width - 4 * app.u
                horizontalAlignment: Text.AlignHCenter
                elide: Text.ElideRight
                text: app.toastText
                font.pixelSize: 3.5 * app.u
                font.bold: app.toastKind === "error"
            }

            MouseArea {
                anchors.fill: parent
                onClicked: app.hideToast()
            }
        }
    }

    Timer {
        id: toastTimer
        onTriggered: app.hideToast()
    }
}
