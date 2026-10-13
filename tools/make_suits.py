#!/usr/bin/env python3
"""Writes the card suit symbols to framework/qml/res/suits/ as SVGs.

The shapes are outlines copied from DejaVu Sans (Bitstream Vera license), so
they look the same on every tablet. Needs fontTools:

    pip install fonttools
    python3 tools/make_suits.py [path/to/DejaVuSans.ttf]
"""
import os
import sys

from fontTools.pens.boundsPen import BoundsPen
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen
from fontTools.ttLib import TTFont

FONT = sys.argv[1] if len(sys.argv) > 1 else "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "framework", "qml", "res", "suits")
RED, BLACK = "#c62828", "#000000"
SUITS = [("spades", 0x2660, BLACK), ("hearts", 0x2665, RED), ("diamonds", 0x2666, RED), ("clubs", 0x2663, BLACK)]
SIZE = 100

font = TTFont(FONT)
glyphs = font.getGlyphSet()
cmap = font.getBestCmap()
os.makedirs(OUT, exist_ok=True)
for name, codepoint, color in SUITS:
    glyph = glyphs[cmap[codepoint]]
    bounds = BoundsPen(glyphs)
    glyph.draw(bounds)
    x0, y0, x1, y1 = bounds.bounds
    scale = SIZE / max(x1 - x0, y1 - y0)
    # Center in a square box, flipping the font's y axis.
    dx = (SIZE - (x1 - x0) * scale) / 2 - x0 * scale
    dy = (SIZE - (y1 - y0) * scale) / 2 + y1 * scale
    pen = SVGPathPen(glyphs)
    glyph.draw(TransformPen(pen, (scale, 0, 0, -scale, dx, dy)))
    with open(os.path.join(OUT, name + ".svg"), "w") as f:
        f.write(f'<svg xmlns="http://www.w3.org/2000/svg" width="{SIZE}" height="{SIZE}" viewBox="0 0 {SIZE} {SIZE}">'
                f'<path fill="{color}" d="{pen.getCommands()}"/></svg>\n')
print("wrote", OUT)
