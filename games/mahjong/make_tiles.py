#!/usr/bin/env python3
"""Draws the 42 tile faces as SVGs into ui/res/tiles/.

The Chinese characters are copied from Noto Sans CJK (SIL Open Font License) as
outlines, so the tablet doesn't need the font. Needs fontTools:

    pip install fonttools
    python3 games/mahjong/make_tiles.py [path/to/NotoSansCJK-Bold.ttc]

Faces: 0-8 dots, 9-17 bamboo, 18-26 characters (1-9 each), 27-30 winds
(east, south, west, north), 31-33 dragons (red, green, white), 34-37 flowers,
38-41 seasons. Matches games/mahjong/backend/board.go.
"""
import os
import sys

from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen
from fontTools.ttLib import TTFont

FONT = sys.argv[1] if len(sys.argv) > 1 else "/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc"
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "ui", "res", "tiles")
W, H = 300, 400  # tile face, 3:4

BLACK, RED, GREEN, BLUE = "#000000", "#b00000", "#006a20", "#0b3a8c"

font = TTFont(FONT, fontNumber=3)  # Traditional Chinese
glyphs = font.getGlyphSet()
cmap = font.getBestCmap()
EM = font["head"].unitsPerEm
ASCENT = 880  # roughly where the CJK glyph box tops out


def text(s, cx, cy, size, color=BLACK):
    """Path for text centered at (cx, cy), `size` = em height in SVG units."""
    widths = [glyphs[cmap[ord(c)]].width for c in s]
    scale = size / EM
    x = cx - sum(widths) * scale / 2
    # Glyph y axis points up; center vertically on the CJK box (-120..880).
    baseline = cy + (ASCENT - 380) * scale - size * 0.12
    out = []
    for c, w in zip(s, widths):
        pen = SVGPathPen(glyphs)
        glyphs[cmap[ord(c)]].draw(TransformPen(pen, (scale, 0, 0, -scale, x, baseline)))
        out.append(f'<path fill="{color}" d="{pen.getCommands()}"/>')
        x += w * scale
    return "".join(out)


def index(label, color=BLACK):
    """Small label in the top-left corner, for players who don't read the characters."""
    return text(label, 40, 40, 54, color)


def circle(cx, cy, r, center=BLACK, ring=BLUE):
    return (f'<circle cx="{cx}" cy="{cy}" r="{r}" fill="none" stroke="{ring}" stroke-width="{r * 0.28:.1f}"/>'
            f'<circle cx="{cx}" cy="{cy}" r="{r * 0.42:.1f}" fill="{center}"/>')


def stick(cx, cy, h, color=GREEN):
    w = max(h * 0.3, 30)
    return (f'<rect x="{cx - w / 2:.1f}" y="{cy - h / 2:.1f}" width="{w:.1f}" height="{h:.1f}" rx="{w / 2:.1f}" fill="{color}"/>'
            f'<rect x="{cx - w / 2:.1f}" y="{cy - h * 0.04:.1f}" width="{w:.1f}" height="{h * 0.08:.1f}" fill="white"/>')


# Positions on a 3x3-ish grid inside the face (below the index)
def grid_positions(n):
    cols = {1: [150], 2: [100, 200], 3: [80, 150, 220]}
    rows = lambda k: [215] if k == 1 else [130 + i * (180 / (k - 1)) for i in range(k)]
    layouts = {
        1: [(150, 215)],
        2: [(150, 130), (150, 300)],
        3: [(85, 120), (150, 215), (215, 310)],
        4: [(x, y) for y in rows(2) for x in cols[2]],
        5: [(x, y) for y in rows(2) for x in [85, 215]] + [(150, 215)],
        6: [(x, y) for y in rows(3) for x in cols[2]],
        7: [(85, 110), (150, 145), (215, 180)] + [(x, y) for y in [250, 330] for x in cols[2]],
        8: [(x, y) for y in rows(4) for x in cols[2]],
        9: [(x, y) for y in rows(3) for x in cols[3]],
    }
    return layouts[n]


# Which positions get a red accent, as on traditional tiles
ACCENT = {5: [4], 7: [0], 9: [4]}


def dots(n):
    if n == 1:
        return circle(150, 215, 95, center=RED)
    r = {2: 58, 3: 48, 4: 46, 5: 42, 6: 40, 7: 30, 8: 27, 9: 32}[n]
    return "".join(circle(x, y, r, center=RED if i in ACCENT.get(n, []) else BLACK)
                   for i, (x, y) in enumerate(grid_positions(n)))


def bamboo(n):
    if n == 1:  # traditionally a bird; a single big stick here
        return stick(150, 225, 250, GREEN) + stick(150, 225, 120, RED)
    h = {2: 150, 3: 110, 4: 150, 5: 110, 6: 80, 7: 64, 8: 52, 9: 78}[n]
    return "".join(stick(x, y, h, RED if i in ACCENT.get(n, []) else GREEN)
                   for i, (x, y) in enumerate(grid_positions(n)))


NUMERALS = "一二三四五六七八九"
faces = []
for n in range(1, 10):
    faces.append((str(n), dots(n)))
for n in range(1, 10):
    faces.append((str(n), bamboo(n)))
for n in range(1, 10):
    faces.append((str(n), text(NUMERALS[n - 1], 150, 150, 140) + text("萬", 150, 300, 150, RED)))
for label, c in zip("ESWN", "東南西北"):
    faces.append((label, text(c, 155, 225, 220)))
faces.append(("R", text("中", 155, 225, 220, RED)))
faces.append(("G", text("發", 155, 225, 220, GREEN)))
faces.append(("Wh", '<rect x="65" y="110" width="170" height="230" fill="none" stroke="%s" stroke-width="16"/>'
                    '<rect x="95" y="140" width="110" height="170" fill="none" stroke="%s" stroke-width="8"/>' % (BLUE, BLUE)))
for i, c in enumerate("梅蘭菊竹"):
    faces.append((f"F{i + 1}", text(c, 155, 230, 190, GREEN)))
for i, c in enumerate("春夏秋冬"):
    faces.append((f"S{i + 1}", text(c, 155, 230, 190, RED)))

os.makedirs(OUT, exist_ok=True)
for i, (label, art) in enumerate(faces):
    with open(os.path.join(OUT, f"{i}.svg"), "w") as f:
        f.write(f'<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" viewBox="0 0 {W} {H}">'
                f'{index(label)}{art}</svg>\n')
print(f"wrote {len(faces)} faces to {OUT}")
