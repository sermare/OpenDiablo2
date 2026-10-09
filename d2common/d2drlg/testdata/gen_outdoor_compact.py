#!/usr/bin/env python3
"""Make the compact Act 1 outdoor golden (numbers only) from the big emulator golden
(d2-re-notes/drlg3-ref/golden_outdoor_act1.json, 30 seeds, ~10 MB, not committed).

usage: gen_outdoor_compact.py big.json out.json [nfull]

Every level of every seed keeps its key numbers plus sha256 digests (first 16 hex chars)
of the large arrays; the first `nfull` seeds (default 2) are kept in full.
Digest input is a little-endian uint32 stream, see digest_* below and the Go test."""
import sys, json, hashlib, struct

def u32(vals):
    return struct.pack('<%dI' % len(vals), *[v & 0xFFFFFFFF for v in vals])

def dg(*chunks):
    h = hashlib.sha256()
    for c in chunks:
        h.update(c)
    return h.hexdigest()[:16]

def grid_words(rows):
    h = len(rows); w = len(rows[0]) if h else 0
    return u32([w, h] + [v for r in rows for v in r])

def room_words(r):
    w = [r['type'], r['x'], r['y'], r['w'], r['h'], r['s4'], r['flags'], r['r50']]
    if r['type'] == 1:
        w += [r['info54'], r['info58'], r['subtype'], r['subtheme'], r['mask'], r['seedAfterMask'][0], r['seedAfterMask'][1]]
    else:
        w += [r['prestDef'], r['file']]
    return u32(w)

def river_words(rv):
    w = [rv['n']]
    for k in ('ends', 'start', 'end', 'junc'):
        for p in rv[k]:
            w += list(p)
    for p in rv['paths']:
        w += [len(p)]
        for q in p:
            w += list(q)
    return u32(w)

def compact_level(L, full):
    c = {k: L[k] for k in ('rect', 'levelType', 'odFlagsInit', 'vis', 'neighbors', 'polygonAtAct', 'seedAfterStages', 'odFlagsAfterStages', 'seedAfterRooms', 'counters')}
    c['stages'] = [[s['name'], s['lo'], s['hi'], s['odflags']] for s in L['stages']]
    c['gridDigest'] = {k: dg(grid_words(L['grids'][k])) for k in ('def', 'B', 'flag', 'D')}
    c['riverDigest'] = dg(river_words(L['river']))
    c['riverN'] = L['river']['n']
    rooms = L['rooms']
    c['nRooms'] = len(rooms)
    c['roomsDigest'] = dg(*[room_words(r) for r in rooms])
    t1 = [r for r in rooms if r['type'] == 1]
    c['nPlainRooms'] = len(t1)
    c['roomGridsDigest'] = dg(*[grid_words(r['gridsABC'][k]) for r in t1 for k in 'ABC'] + [u32(r['seedAfterBuild']) for r in t1])
    if full:
        c['full'] = dict(grids=L['grids'], river=L['river'],
                         rooms=[{k: v for k, v in r.items() if k != 'roomFlagsAfter'} for r in rooms])
    return c

def main():
    big = json.load(open(sys.argv[1]))
    nfull = int(sys.argv[3]) if len(sys.argv) > 3 else 2
    out = []
    for i, rec in enumerate(big):
        out.append(dict(seed=rec['seed'], base=rec['base'],
                        levels={k: compact_level(L, i < nfull) for k, L in rec['levels'].items()}))
    json.dump(out, open(sys.argv[2], 'w'), separators=(',', ':'))

main()
