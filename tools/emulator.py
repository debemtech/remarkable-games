#!/usr/bin/env python3
"""Minimal AppLoad stand-in for running the app on a PC.

It loads resources.rcc like AppLoad does (mounted at /<app id>), provides the
`net.asivery.AppLoad 1.0` QML module and speaks AppLoad's socket protocol to
the real backend binary.

    pip install PySide6-Essentials
    ./build.sh chess
    go build -o build/chess-host ./games/chess/backend
    CHESS_STOCKFISH=$(games/chess/stockfish.sh host) \
        python3 tools/emulator.py build/chess-rmpp --backend build/chess-host

Without --backend it runs <app dir>/backend/entry, which works for the arm64
build too if qemu-user with binfmt is set up.

Optional: --script FILE runs actions (tap X Y / swipe X1 Y1 X2 Y2 /
draw X1 Y1 X2 Y2 ... (a stroke with the mouse) / pen X1 Y1 ... and
eraser X1 Y1 ... (the same with a stylus, as tablet events) / pendown X Y, penup (to hold) /
touch X1 Y1 ... (a finger) / touchdown X Y, touchup /
touch2 X1 Y1 X2 Y2
(a two finger tap) / shot NAME / sleep SECONDS) in device coordinates
(1620x2160) and quits afterwards.
"""
import argparse
import json
import os
import socket
import struct
import subprocess
import sys
import tempfile

from PySide6.QtCore import (QObject, Property, QPoint, QResource, QSocketNotifier, QTimer, QUrl,
                            QEvent, QPointF, Qt, Signal, Slot)
from PySide6.QtGui import QGuiApplication, QInputDevice, QPointingDevice, QTabletEvent
from PySide6.QtQml import QmlElement
from PySide6.QtQuick import QQuickView
from PySide6.QtTest import QTest

QML_IMPORT_NAME = "net.asivery.AppLoad"
QML_IMPORT_MAJOR_VERSION = 1

MSG_SYSTEM_TERMINATE = 0xFFFFFFFF
MSG_SYSTEM_NEW_COORDINATOR = 0xFFFFFFFE
DEVICE_W, DEVICE_H = 1620, 2160


class Host:
    endpoints = []
    conn = None
    app = None


def send_to_backend(msg_type, text):
    if Host.conn is None:
        print(f"[host] no backend connected, dropping message {msg_type}")
        return
    body = text.encode()
    Host.conn.send(struct.pack("=II", msg_type & 0xFFFFFFFF, len(body)))
    Host.conn.send(body)


@QmlElement
class AppLoad(QObject):
    messageReceived = Signal(int, str, arguments=["type", "contents"])

    def __init__(self, parent=None):
        super().__init__(parent)
        self._id = ""

    def _get_id(self):
        return self._id

    def _set_id(self, value):
        self._id = value
        Host.endpoints.append(self)
        send_to_backend(MSG_SYSTEM_NEW_COORDINATOR, str(len(Host.endpoints)))

    applicationID = Property(str, _get_id, _set_id)

    @Slot(int, str)
    def sendMessage(self, msg_type, text):
        print(f"[ui -> backend] {msg_type} {text}")
        send_to_backend(msg_type, text)

    @Slot()
    def terminate(self):
        print("[host] terminate()")
        send_to_backend(MSG_SYSTEM_TERMINATE, "close")
        QTimer.singleShot(0, Host.app.quit)


WRAPPER_QML = """
import QtQuick
Rectangle {
    id: wrapper
    color: "#808080"
    Loader {
        id: loader
        source: "%s"
        width: %d; height: %d
        anchors.centerIn: parent
        transform: Scale {
            origin.x: loader.width / 2; origin.y: loader.height / 2
            xScale: Math.min(wrapper.width / loader.width, wrapper.height / loader.height)
            yScale: xScale
        }
        Connections {
            target: loader.item
            function onClose() { Qt.quit() }
        }
    }
}
"""


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("app_dir", help="AppLoad application directory (with manifest.json and resources.rcc)")
    parser.add_argument("--backend", help="backend binary (defaults to <app_dir>/backend/entry)")
    parser.add_argument("--scale", type=float, default=0.5, help="window scale relative to the Paper Pro screen")
    parser.add_argument("--script", help="file with actions to run, then quit")
    parser.add_argument("--out", default=".", help="directory for screenshots")
    args = parser.parse_args()

    manifest = json.load(open(os.path.join(args.app_dir, "manifest.json")))
    app_id = manifest["id"]

    Host.app = QGuiApplication(sys.argv)
    if not QResource.registerResource(os.path.join(args.app_dir, "resources.rcc"), "/" + app_id):
        sys.exit("Failed to register resources.rcc")

    # Backend socket, like appload::management::createListeningSocket
    sock_path = os.path.join(tempfile.mkdtemp(), app_id + ".sock")
    server = socket.socket(socket.AF_UNIX, socket.SOCK_SEQPACKET)
    server.bind(sock_path)
    server.listen(1)
    backend = os.path.abspath(args.backend or os.path.join(args.app_dir, "backend", "entry"))
    proc = subprocess.Popen([backend, sock_path], cwd=args.app_dir)

    def on_readable():
        try:
            header = Host.conn.recv(8)
        except OSError:
            header = b""
        if len(header) < 8:
            print("[host] backend disconnected")
            reader.setEnabled(False)
            return
        msg_type, length = struct.unpack("=II", header)
        body = Host.conn.recv(length).decode() if length else ""
        print(f"[backend -> ui] {msg_type} {body[:160]}")
        for endpoint in Host.endpoints:
            endpoint.messageReceived.emit(msg_type, body)

    def on_connect():
        nonlocal reader
        Host.conn, _ = server.accept()
        acceptor.setEnabled(False)
        reader = QSocketNotifier(Host.conn.fileno(), QSocketNotifier.Read)
        reader.activated.connect(on_readable)
        if Host.endpoints:
            send_to_backend(MSG_SYSTEM_NEW_COORDINATOR, str(len(Host.endpoints)))

    reader = None
    acceptor = QSocketNotifier(server.fileno(), QSocketNotifier.Read)
    acceptor.activated.connect(on_connect)

    wrapper_path = os.path.join(tempfile.mkdtemp(), "wrapper.qml")
    with open(wrapper_path, "w") as f:
        f.write(WRAPPER_QML % ("qrc:/" + app_id + manifest["entry"], DEVICE_W, DEVICE_H))

    view = QQuickView()
    view.setResizeMode(QQuickView.SizeRootObjectToView)
    view.engine().quit.connect(Host.app.quit)
    view.setSource(QUrl.fromLocalFile(wrapper_path))
    if view.status() == QQuickView.Error:
        for error in view.errors():
            print(error.toString())
        sys.exit(1)
    view.resize(int(DEVICE_W * args.scale), int(DEVICE_H * args.scale))
    view.show()

    if args.script:
        actions = [line.split() for line in open(args.script) if line.strip() and not line.startswith("#")]
        pos = lambda x, y: QPoint(int(int(x) * args.scale), int(int(y) * args.scale))

        stylus = {}

        def device(kind):
            if kind not in stylus:
                ptype = QPointingDevice.PointerType.Eraser if kind == "eraser" else QPointingDevice.PointerType.Pen
                stylus[kind] = QPointingDevice(kind, 100 + len(stylus), QInputDevice.DeviceType.Stylus, ptype,
                                               QInputDevice.Capability.Position | QInputDevice.Capability.Pressure, 1, 2)
            return stylus[kind]

        def tablet_event(kind, t, p):
            # Moves carry no button of their own, or Qt takes them for presses
            button = Qt.NoButton if t == QEvent.TabletMove else Qt.LeftButton
            buttons = Qt.NoButton if t == QEvent.TabletRelease else Qt.LeftButton
            pressure = 0 if t == QEvent.TabletRelease else 0.5
            QGuiApplication.sendEvent(view, QTabletEvent(t, device(kind), p, view.mapToGlobal(p), pressure,
                                                         0, 0, 0, 0, 0, Qt.NoModifier, button, buttons))

        def tablet(kind, points):
            path = [points[0]]
            for a, b in zip(points, points[1:]):
                path += [QPointF(a.x() + (b.x() - a.x()) * k / 5, a.y() + (b.y() - a.y()) * k / 5) for k in range(1, 6)]
            tablet_event(kind, QEvent.TabletPress, path[0])
            for p in path[1:]:
                tablet_event(kind, QEvent.TabletMove, p)
            tablet_event(kind, QEvent.TabletRelease, path[-1])

        touchscreen = QTest.createTouchDevice()
        held = {}

        def step():
            if not actions:
                Host.app.quit()
                return
            cmd, *params = actions.pop(0)
            delay = 400
            if cmd == "tap":
                QTest.mouseClick(view, Qt.LeftButton, Qt.NoModifier, pos(*params))
            elif cmd == "draw":
                points = [pos(params[i], params[i + 1]) for i in range(0, len(params) - 1, 2)]
                QTest.mousePress(view, Qt.LeftButton, Qt.NoModifier, points[0])
                for a, b in zip(points, points[1:]):
                    for k in range(1, 6):  # in small steps, like a pen
                        QTest.mouseMove(view, QPoint(a.x() + (b.x() - a.x()) * k // 5, a.y() + (b.y() - a.y()) * k // 5))
                QTest.mouseRelease(view, Qt.LeftButton, Qt.NoModifier, points[-1])
                delay = 150
            elif cmd in ("pen", "eraser"):
                tablet(cmd, [QPointF(pos(params[i], params[i + 1])) for i in range(0, len(params) - 1, 2)])
                delay = 150
            elif cmd == "pendown":
                held["pen"] = QPointF(pos(params[0], params[1]))
                tablet_event("pen", QEvent.TabletPress, held["pen"])
            elif cmd == "penup":
                tablet_event("pen", QEvent.TabletRelease, held.pop("pen"))
            elif cmd == "touchdown":
                held["touch"] = pos(params[0], params[1])
                QTest.touchEvent(view, touchscreen).press(0, held["touch"], view).commit()
            elif cmd == "touchup":
                QTest.touchEvent(view, touchscreen).release(0, held.pop("touch"), view).commit()
            elif cmd == "touch":  # one finger, pressed at the first point, moved through the rest
                points = [pos(params[i], params[i + 1]) for i in range(0, len(params) - 1, 2)]
                QTest.touchEvent(view, touchscreen).press(0, points[0], view).commit()
                for p in points[1:]:
                    QTest.touchEvent(view, touchscreen).move(0, p, view).commit()
                QTest.touchEvent(view, touchscreen).release(0, points[-1], view).commit()
            elif cmd == "touch2":
                a, b = pos(params[0], params[1]), pos(params[2], params[3])
                QTest.touchEvent(view, touchscreen).press(0, a, view).press(1, b, view).commit()
                QTest.touchEvent(view, touchscreen).release(0, a, view).release(1, b, view).commit()
            elif cmd == "swipe":
                QTest.mousePress(view, Qt.LeftButton, Qt.NoModifier, pos(params[0], params[1]))
                QTest.mouseMove(view, pos(params[2], params[3]))
                QTest.mouseRelease(view, Qt.LeftButton, Qt.NoModifier, pos(params[2], params[3]))
            elif cmd == "shot":
                path = os.path.join(args.out, params[0] + ".png")
                view.grabWindow().save(path)
                print(f"[host] screenshot {path}")
                delay = 50
            elif cmd == "sleep":
                delay = int(float(params[0]) * 1000)
            QTimer.singleShot(delay, step)

        QTimer.singleShot(1500, step)

    code = Host.app.exec()
    if Host.conn is not None:
        Host.conn.close()
    try:
        proc.wait(3)
    except subprocess.TimeoutExpired:
        proc.kill()
    sys.exit(code)


if __name__ == "__main__":
    main()
