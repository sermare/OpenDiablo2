#!/usr/bin/env python3
"""Progress over time as a line chart (docs/progress-history.svg).

  progress_history.py backfill   rebuild history from the git log of docs/progress.json
  progress_history.py tick       append a snapshot of the current docs/progress.json
  progress_history.py render     (re)draw the chart from docs/progress-history.json

One line per bar of docs/progress.json, same colour as its bar, y axis 0-100 %, x axis real time.
"""
import datetime as dt
import json
import pathlib
import subprocess
import sys
from xml.sax.saxutils import escape

root = pathlib.Path(__file__).resolve().parent.parent
PROG = root / "docs" / "progress.json"
HIST = root / "docs" / "progress-history.json"
OUT = root / "docs" / "progress-history.svg"
FONT = "-apple-system,Segoe UI,Helvetica,Arial,sans-serif"


def pcts(data):
    out = {b["label"]: round(100.0 * b["done"] / b["total"], 2) for b in data["bars"]}
    v1 = data.get("v1")
    if v1 and v1.get("weights"):
        w = v1["weights"]
        num = sum(w[k] * out[k] for k in w if k in out)
        den = sum(w[k] for k in w if k in out)
        if den:
            out[v1.get("title", "Game v1 complete")] = round(num / den, 2)
            bars = {b["label"]: b for b in data["bars"]}
            numb = sum(w[k] * min(100.0, 100.0 * (bars[k]["done"] + (bars[k].get("pending") or 0)) / bars[k]["total"])
                       for k in w if k in bars)
            out[v1.get("title", "Game v1 complete") + " (incl. branch work)"] = round(numb / den, 2)
    return out


def load():
    return json.loads(HIST.read_text()) if HIST.exists() else {"points": []}


def save(h):
    HIST.write_text(json.dumps(h, indent=0, ensure_ascii=False))


def backfill():
    log = subprocess.run(["git", "log", "--reverse", "--format=%H %cI", "--", "docs/progress.json"],
                         cwd=root, capture_output=True, text=True, check=True).stdout.split("\n")
    pts = []
    for line in log:
        if not line.strip():
            continue
        h, t = line.split()
        try:
            raw = subprocess.run(["git", "show", f"{h}:docs/progress.json"], cwd=root, capture_output=True,
                                 text=True, check=True).stdout
            pts.append({"t": t, "v": pcts(json.loads(raw))})
        except Exception:
            pass
    save({"points": pts})


def tick():
    h = load()
    now = dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat()
    h["points"].append({"t": now, "v": pcts(json.loads(PROG.read_text()))})
    save(h)


def render():
    data = json.loads(PROG.read_text())
    pts = load()["points"]
    if not pts:
        return
    ts = [dt.datetime.fromisoformat(p["t"]) for p in pts]
    t0, t1 = ts[0], max(ts[-1], ts[0] + dt.timedelta(minutes=10))
    W, H = 880, 520
    L, R, T, B = 56, 24, 64, 150
    pw, ph = W - L - R, H - T - B
    span = (t1 - t0).total_seconds()

    def X(t):
        return L + pw * (t - t0).total_seconds() / span

    def Y(v):
        return T + ph * (1 - v / 100.0)

    o = []
    a = o.append
    a(f'<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" viewBox="0 0 {W} {H}" role="img" aria-label="Progress over time">')
    a(f'<rect width="{W}" height="{H}" rx="18" fill="#14162a"/>')
    a(f'<text x="{L}" y="36" font-family="{FONT}" font-size="22" font-weight="700" fill="#fff">Progress over time</text>')
    a(f'<text x="{W-R}" y="36" text-anchor="end" font-family="{FONT}" font-size="12" fill="#8c8eb0">snapshot every 10 min · latest {escape(ts[-1].strftime("%Y-%m-%d %H:%M UTC"))}</text>')
    for g in range(0, 101, 20):
        a(f'<line x1="{L}" y1="{Y(g):.1f}" x2="{W-R}" y2="{Y(g):.1f}" stroke="#fff" stroke-opacity="{0.22 if g in (0,100) else 0.08}"/>')
        a(f'<text x="{L-8}" y="{Y(g)+4:.1f}" text-anchor="end" font-family="{FONT}" font-size="11" fill="#9fa1c4">{g}%</text>')
    n = 6
    for i in range(n + 1):
        t = t0 + (t1 - t0) * i / n
        x = X(t)
        a(f'<line x1="{x:.1f}" y1="{T}" x2="{x:.1f}" y2="{T+ph}" stroke="#fff" stroke-opacity="0.05"/>')
        a(f'<text x="{x:.1f}" y="{T+ph+16}" text-anchor="middle" font-family="{FONT}" font-size="11" fill="#9fa1c4">{t.strftime("%m-%d %H:%M")}</text>')
    for b in data["bars"]:
        xy = [(X(t), Y(p["v"].get(b["label"], 0))) for t, p in zip(ts, pts) if b["label"] in p["v"]]
        if not xy:
            continue
        d = " ".join(f"{x:.1f},{y:.1f}" for x, y in xy)
        a(f'<polyline points="{d}" fill="none" stroke="{b["color"]}" stroke-width="2.4" stroke-linejoin="round" stroke-linecap="round"/>')
        a(f'<circle cx="{xy[-1][0]:.1f}" cy="{xy[-1][1]:.1f}" r="4" fill="{b["color"]}"/>')
    ly = T + ph + 40
    for i, b in enumerate(data["bars"]):
        col, row = i % 2, i // 2
        x = L + col * (pw // 2)
        y = ly + row * 20
        last = pts[-1]["v"].get(b["label"], 0)
        a(f'<rect x="{x}" y="{y-9}" width="14" height="4" rx="2" fill="{b["color"]}"/>')
        a(f'<text x="{x+22}" y="{y}" font-family="{FONT}" font-size="12" fill="#e6e7fa">{escape(b["label"])} — {last:.0f}%</text>')
    a('</svg>')
    OUT.write_text("\n".join(o))
    print("wrote", OUT.relative_to(root))


if __name__ == "__main__":
    cmd = sys.argv[1] if len(sys.argv) > 1 else "render"
    {"backfill": backfill, "tick": tick, "render": render}[cmd]()
    if cmd != "render":
        render()
