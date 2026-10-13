#!/bin/bash
# Builds a statically linked Stockfish and prints its path.
#   games/chess/stockfish.sh rmpp|rm2|host
# Builds are cached in build/cache. Set STOCKFISH_BIN to use a binary you
# already have instead.
set -euo pipefail

VERSION=sf_19
DEVICE=${1:?usage: $0 rmpp|rm2|host}
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
CACHE=$ROOT/build/cache
SRC=$CACHE/stockfish-$VERSION

if [ -n "${STOCKFISH_BIN:-}" ]; then
    echo "$STOCKFISH_BIN"
    exit 0
fi

case "$DEVICE" in
    # Paper Pro: Cortex-A53, no dot product instructions
    rmpp) ARCH=armv8; CXX=aarch64-linux-gnu-g++; LDFLAGS=-static ;;
    rm2) ARCH=armv7-neon; CXX=arm-linux-gnueabihf-g++; LDFLAGS=-static ;;
    host) ARCH=x86-64-sse41-popcnt; CXX=g++; LDFLAGS= ;;
    *) echo "unknown device $DEVICE" >&2; exit 1 ;;
esac

OUT=$CACHE/stockfish-$VERSION-$DEVICE
if [ -x "$OUT" ]; then
    echo "$OUT"
    exit 0
fi

mkdir -p "$CACHE"
if [ ! -d "$SRC" ]; then
    git clone -q --depth 1 --branch "$VERSION" https://github.com/official-stockfish/Stockfish "$SRC" >&2
fi
(
    cd "$SRC/src"
    make -s net >&2
    make -s clean >&2
    make -s -j"$(nproc)" build ARCH="$ARCH" COMP=gcc CXX="$CXX" EXTRALDFLAGS="$LDFLAGS" >&2
    cp stockfish "$OUT"
    "${CXX%g++}strip" "$OUT" 2>/dev/null || true
)
echo "$OUT"
