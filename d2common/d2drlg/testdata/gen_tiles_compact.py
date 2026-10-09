#!/usr/bin/env python3
"""Make the compact tile golden from the big emulator dumps (~/git/drlg-oracle/gen_tiles2.py,
parts gt_part*.json, not committed).

usage: gen_tiles_compact.py out.json big1.json [big2.json ...]

Per (seed, level), for the plain rooms and then the preset rooms (creation order) of the
level after ALL its rooms were built (plain rooms first, then presets; every level in a fresh
game): [x, y, seedLo, seedHi, digest, nWalls, nFloors, nShadows] where digest is the first 12
hex chars of sha256 over the canonical record string (see canonRoom in tiles_oracle_test.go).
The first level also keeps all its records ("full": per room lists of
[x, y, orientation, flags, dt1 file, tile index]) so a mismatch can be explained."""
import sys, json

out = []
for path in sys.argv[2:]:
    out += json.load(open(path))
res = []
for i, L in enumerate(out):
    def row(r):
        return [r['x'], r['y'], r['seed'][0], r['seed'][1], r['dig']] + r['n']
    d = dict(seed=L['seed'], level=L['level'], rooms=[row(r) for r in L['rooms']], presets=[row(r) for r in L['presets']])
    if i == 0:
        d['full'] = dict(rooms=[r['tiles'] for r in L['rooms']], presets=[r['tiles'] for r in L['presets']])
    res.append(d)
json.dump(res, open(sys.argv[1], 'w'), separators=(',', ':'))
