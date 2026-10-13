#!/usr/bin/env bash
# Vectorize chessMarkable's piece PNGs so they stay sharp on the 229 DPI
# screen of the Paper Pro. Needs ImageMagick and potrace.
#   tools/trace_pieces.sh <chessmarkable>/res ui/res
set -euo pipefail

SRC=${1:?source directory with the piece PNGs}
DEST=${2:?destination directory}
SCALE=8
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

for piece in king queen rook bishop knight pawn; do
    for color in white black; do
        name="$piece-$color"
        png="$SRC/$name.png"
        read -r width height < <(identify -format "%w %h\n" "$png")

        # Black ink: dark, opaque pixels. Silhouette: every opaque pixel (gets filled white).
        convert "$png" -background white -flatten -colorspace Gray \
            -filter Lanczos -resize "$((SCALE * 100))%" -threshold 50% "$TMP/ink.pbm"
        convert "$png" -alpha extract -negate \
            -filter Lanczos -resize "$((SCALE * 100))%" -threshold 50% "$TMP/silhouette.pbm"

        potrace -s --flat -a 1.0 -O 0.4 "$TMP/ink.pbm" -o "$TMP/ink.svg"
        potrace -s --flat -a 1.0 -O 0.4 "$TMP/silhouette.pbm" -o "$TMP/silhouette.svg"

        python3 - "$TMP/silhouette.svg" "$TMP/ink.svg" "$DEST/$name.svg" "$width" "$height" "$SCALE" <<'EOF'
import re, sys
silhouette, ink, out, width, height, scale = sys.argv[1:]
width, height, scale = int(width), int(height), int(scale)

def group(path, fill):
    svg = open(path).read()
    g = re.search(r"<g .*?</g>", svg, re.S).group(0)
    return re.sub(r'fill="#000000"', f'fill="{fill}"', g, count=1)

w, h = width * scale, height * scale
with open(out, "w") as f:
    f.write(f'<svg xmlns="http://www.w3.org/2000/svg" width="{w}" height="{h}" viewBox="0 0 {w} {h}">\n')
    f.write(group(silhouette, "#ffffff") + "\n")
    f.write(group(ink, "#000000") + "\n")
    f.write("</svg>\n")
EOF
        echo "$DEST/$name.svg"
    done
done
