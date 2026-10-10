#!/usr/bin/env python3
"""Render docs/progress.svg from docs/progress.json (run after every success)."""
import json
import pathlib
from xml.sax.saxutils import escape

root = pathlib.Path(__file__).resolve().parent.parent
data = json.loads((root / "docs" / "progress.json").read_text(encoding="utf-8"))

W, PAD = 880, 36
V1 = data.get("v1")
V1_H = 118 if V1 else 0
ROW_H, BAR_H = 78, 18
head_h = 92 + V1_H
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


if V1:
    wts = V1["weights"]
    byl = {b["label"]: b for b in data["bars"]}
    tot = sum(wts.values())
    vd = sum(wts[l] * byl[l]["done"] / byl[l]["total"] for l in wts if l in byl) / tot * 100
    vp = sum(wts[l] * min(byl[l]["total"], byl[l]["done"] + byl[l].get("pending", 0)) / byl[l]["total"] for l in wts if l in byl) / tot * 100
    hy = 92
    full_h = W - 2 * PAD
    a(f'<text x="{PAD}" y="{hy+22}" font-family="-apple-system,Segoe UI,Helvetica,Arial,sans-serif" font-size="19" font-weight="700" fill="#ffffff">{escape(V1["title"])}</text>')
    a(f'<text x="{W-PAD}" y="{hy+26}" text-anchor="end" font-family="-apple-system,Segoe UI,Helvetica,Arial,sans-serif" font-size="30" font-weight="800" fill="#ffd166">{vd:.0f}%</text>')
    a(f'<rect x="{PAD}" y="{hy+36}" width="{full_h}" height="26" rx="13" fill="#ffffff" fill-opacity="0.08"/>')
    if vp > vd:
        a(f'<rect x="{PAD}" y="{hy+36}" width="{full_h*vp/100:.1f}" height="26" rx="13" fill="#ffd166" fill-opacity="0.25" stroke="#ffd166" stroke-opacity="0.6" stroke-dasharray="5 4"/>')
    a(f'<rect x="{PAD}" y="{hy+36}" width="{max(full_h*vd/100, 10):.1f}" height="26" rx="13" fill="#ffd166" filter="url(#glow)"/>')
    a(f'<text x="{PAD}" y="{hy+84}" font-family="-apple-system,Segoe UI,Helvetica,Arial,sans-serif" font-size="11.5" fill="#b8b9d6">weighted average of the bars below (weights in docs/progress.json); dashed = +{vp-vd:.0f}% on branches, not yet verified</text>')
    a(f'<text x="{PAD}" y="{hy+100}" font-family="-apple-system,Segoe UI,Helvetica,Arial,sans-serif" font-size="11.5" fill="#8c8eb0">v1 = double-click the Mac app, make or load a character, play Acts 1-5 single player without a blocker</text>')

y = head_h
for i, b in enumerate(data["bars"]):
    frac = b["done"] / b["total"]
    pct = frac * 100
    label = f'{pct:.1f}%' if (pct < 10 and pct != 0) or 99 <= pct < 100 else f'{pct:.0f}%'
    a(f'<text x="{PAD}" y="{y+16}" font-family="-apple-system,Segoe UI,Helvetica,Arial,sans-serif" font-size="15" font-weight="600" fill="#f2f2ff">{escape(b["label"])}</text>')
    a(f'<text x="{W-PAD}" y="{y+16}" text-anchor="end" font-family="ui-monospace,SFMono-Regular,Menlo,monospace" font-size="14" fill="#d8d9f2">{b["done"]:,} / {b["total"]:,}</text>')
    by = y + 28
    a(f'<rect x="{PAD}" y="{by}" width="{W-2*PAD-70}" height="{BAR_H}" rx="9" fill="#ffffff" fill-opacity="0.07"/>')
    full = W - 2 * PAD - 70
    fw = max(frac * full, 6 if b["done"] else 0)
    if fw:
        a(f'<rect x="{PAD}" y="{by}" width="{fw:.1f}" height="{BAR_H}" rx="9" fill="url(#g{i})" filter="url(#glow)"/>')
    # work that exists only on branches that have not passed the full verify yet: a lighter segment after the bar
    pend = b.get("pending", 0)
    if pend:
        pw = min(pend / b["total"] * full, full - fw)
        if pw > 0:
            a(f'<rect x="{PAD + fw:.1f}" y="{by}" width="{pw:.1f}" height="{BAR_H}" rx="9" fill="{b["color"]}" fill-opacity="0.28" stroke="{b["color"]}" stroke-opacity="0.6" stroke-dasharray="4 3"/>')
        b = dict(b, note=f'+{pend / b["total"] * 100:.0f}% done on branches, not yet verified (dashed) · ' + b["note"])
    a(f'<text x="{W-PAD}" y="{by+14}" text-anchor="end" font-family="-apple-system,Segoe UI,Helvetica,Arial,sans-serif" font-size="17" font-weight="700" fill="{b["color"]}">{label}</text>')
    a(f'<text x="{PAD}" y="{by+BAR_H+17}" font-family="-apple-system,Segoe UI,Helvetica,Arial,sans-serif" font-size="12" fill="#9fa1c4">{escape(b["note"])}</text>')
    y += ROW_H

a('</svg>')
(root / "docs" / "progress.svg").write_text("\n".join(out), encoding="utf-8")
print("wrote docs/progress.svg")
