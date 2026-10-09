#!/usr/bin/env python3
"""Render docs/progress.svg from docs/progress.json (run after every success)."""
import json
import pathlib
from xml.sax.saxutils import escape

root = pathlib.Path(__file__).resolve().parent.parent
data = json.loads((root / "docs" / "progress.json").read_text(encoding="utf-8"))

W, PAD = 880, 36
ROW_H, BAR_H = 78, 18
head_h = 92
H = head_h + ROW_H * len(data["bars"]) + 30
BAR_W = W - 2 * PAD - 150

out = []
a = out.append
a(f'<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" viewBox="0 0 {W} {H}" role="img" aria-label="{escape(data["title"])}">')
a('<defs>')
a('<linearGradient id="bg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#14162a"/><stop offset="1" stop-color="#1d1a33"/></linearGradient>')
for i, b in enumerate(data["bars"]):
    a(f'<linearGradient id="g{i}" x1="0" y1="0" x2="1" y2="0"><stop offset="0" stop-color="{b["color"]}" stop-opacity="0.75"/><stop offset="1" stop-color="{b["color"]}"/></linearGradient>')
a('<filter id="glow" x="-5%" y="-50%" width="110%" height="200%"><feGaussianBlur stdDeviation="3" result="b"/><feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge></filter>')
a('</defs>')
a(f'<rect width="{W}" height="{H}" rx="18" fill="url(#bg)"/>')
a(f'<rect x="1" y="1" width="{W-2}" height="{H-2}" rx="17" fill="none" stroke="#ffffff" stroke-opacity="0.08"/>')
a(f'<text x="{PAD}" y="46" font-family="-apple-system,Segoe UI,Helvetica,Arial,sans-serif" font-size="26" font-weight="700" fill="#ffffff">{escape(data["title"])}</text>')
a(f'<text x="{PAD}" y="72" font-family="-apple-system,Segoe UI,Helvetica,Arial,sans-serif" font-size="13" fill="#b8b9d6">{escape(data["subtitle"])}</text>')
a(f'<text x="{W-PAD}" y="46" text-anchor="end" font-family="-apple-system,Segoe UI,Helvetica,Arial,sans-serif" font-size="12" fill="#8c8eb0">updated {escape(data["updated"])}</text>')

y = head_h
for i, b in enumerate(data["bars"]):
    frac = b["done"] / b["total"]
    pct = frac * 100
    label = f'{pct:.0f}%' if pct >= 10 or pct == 0 else f'{pct:.1f}%'
    a(f'<text x="{PAD}" y="{y+16}" font-family="-apple-system,Segoe UI,Helvetica,Arial,sans-serif" font-size="15" font-weight="600" fill="#f2f2ff">{escape(b["label"])}</text>')
    a(f'<text x="{W-PAD}" y="{y+16}" text-anchor="end" font-family="ui-monospace,SFMono-Regular,Menlo,monospace" font-size="14" fill="#d8d9f2">{b["done"]:,} / {b["total"]:,}</text>')
    by = y + 28
    a(f'<rect x="{PAD}" y="{by}" width="{W-2*PAD-70}" height="{BAR_H}" rx="9" fill="#ffffff" fill-opacity="0.07"/>')
    full = W - 2 * PAD - 70
    fw = max(frac * full, 6 if b["done"] else 0)
    if fw:
        a(f'<rect x="{PAD}" y="{by}" width="{fw:.1f}" height="{BAR_H}" rx="9" fill="url(#g{i})" filter="url(#glow)"/>')
    a(f'<text x="{W-PAD}" y="{by+14}" text-anchor="end" font-family="-apple-system,Segoe UI,Helvetica,Arial,sans-serif" font-size="17" font-weight="700" fill="{b["color"]}">{label}</text>')
    a(f'<text x="{PAD}" y="{by+BAR_H+17}" font-family="-apple-system,Segoe UI,Helvetica,Arial,sans-serif" font-size="12" fill="#9fa1c4">{escape(b["note"])}</text>')
    y += ROW_H

a('</svg>')
(root / "docs" / "progress.svg").write_text("\n".join(out), encoding="utf-8")
print("wrote docs/progress.svg")
