#!/bin/bash
# Builds a game into an AppLoad application directory.
#   ./build.sh <game> [rmpp|rm2]     e.g. ./build.sh chess
#
# rmpp: reMarkable Paper Pro / Paper Pro Move (arm64, default)
# rm2:  reMarkable 2 / 1 with AppLoad (arm, ARMv7)
#
# Result: build/<game>-<device>/, copy it to /home/root/xovi/exthome/appload/<game>
#
# A game lives in games/<game>/ with
#   manifest.json, icon.png     copied as is
#   ui/                         QML, JS and images; the framework QML is added as ui/framework/
#   backend/                    Go main package (optional, for "loadsBackend": true)
#   build-extra.sh              optional hook, called as: build-extra.sh <app dir> <device>
set -euo pipefail
cd "$(dirname "$0")"
ROOT=$PWD

GAME=${1:?usage: $0 <game> [rmpp|rm2]}
DEVICE=${2:-rmpp}
SRC=$ROOT/games/$GAME
[ -d "$SRC" ] || { echo "no such game: $SRC" >&2; exit 1; }

case "$DEVICE" in
    rmpp) GOARCH=arm64; GOARM= ;;
    rm2) GOARCH=arm; GOARM=7 ;;
    *) echo "unknown device $DEVICE (rmpp or rm2)" >&2; exit 1 ;;
esac

RCC=${RCC:-$(command -v /usr/lib/qt6/libexec/rcc || command -v /usr/lib/qt6/bin/rcc || command -v rcc || true)}
[ -n "$RCC" ] || { echo "rcc not found (install qt6-base-dev-tools or set RCC)" >&2; exit 1; }

DEST=$ROOT/build/$GAME-$DEVICE
rm -rf "$DEST"
mkdir -p "$DEST"
cp "$SRC/manifest.json" "$SRC/icon.png" "$DEST/"

# Resources: every file in the game's ui/ plus the framework under ui/framework/
QRC=$ROOT/build/$GAME.qrc
{
    echo '<RCC><qresource>'
    (cd "$SRC" && find ui -type f | sort) | while read -r f; do
        echo "<file alias=\"$f\">$SRC/$f</file>"
    done
    (cd "$ROOT/framework/qml" && find . -type f | sort) | while read -r f; do
        echo "<file alias=\"ui/framework/${f#./}\">$ROOT/framework/qml/${f#./}</file>"
    done
    echo '</qresource></RCC>'
} > "$QRC"
"$RCC" --binary -o "$DEST/resources.rcc" "$QRC"

if [ -d "$SRC/backend" ]; then
    mkdir -p "$DEST/backend"
    CGO_ENABLED=0 GOOS=linux GOARCH=$GOARCH GOARM=$GOARM \
        go build -trimpath -ldflags="-s -w" -o "$DEST/backend/entry" "./games/$GAME/backend"
fi

if [ -x "$SRC/build-extra.sh" ]; then
    "$SRC/build-extra.sh" "$DEST" "$DEVICE"
fi

echo "Built $DEST"
