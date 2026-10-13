#!/bin/bash
# Called by build.sh: puts Stockfish next to the backend.
set -euo pipefail
DEST=${1:?app dir}
DEVICE=${2:?device}
cp "$("$(dirname "$0")/stockfish.sh" "$DEVICE")" "$DEST/backend/stockfish"
