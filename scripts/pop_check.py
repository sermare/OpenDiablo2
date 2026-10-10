#!/usr/bin/env python3
"""Checks the population lines of a game log against the real tables, for any level.

usage: pop_check.py <log.txt> <level id> [<min monsters>]

Reads from the log (written by the game, OD2_REALMAPS=1 OD2_AUTOLEVEL=<id> OD2_POPULATE=1):
  "POPULATE types level N: key:count ..."   the natural monsters that survived the reachability filter
  "POPULATE groups level N: G"              pack leaders (groups, unique packs included) of the planned population
  "population: ... D density rolls at MonDen M"   the density rolls the level's rooms get (real maze / outdoor / preset)

Checks, against D2_TABLES (drlg/patch_d2/Levels.txt, monsters/patch_d2/monstats.txt):
  - every drawn class is legal: mon1..mon10 of the Levels.txt row, the minion1/minion2 followers of those classes,
    or their Spawn replacement (monstats "spawn", PlaceSpawn); each must have isSpawn = 1;
  - at most NumMon of the listed types are drawn (the leaders; followers come from minion1/minion2);
  - MonDen = 0 spawns no natural monster; otherwise at least <min monsters> (default 1) spawn;
  - density (when the log has the roll count): the groups made stay within a few standard deviations of
    rolls * MonDen / 100000 (each roll passes with that chance; a roll makes at most one group), and not far below
    that expectation for the rolls that land on walkable ground (sparse classes and crowded placements lose some);
  - NumMon = 0 / no listed class and MonDen > 0 is flagged as a table that cannot spawn anything.
Prints one POPCHECK line; exit 0 = pass, 2 = no tables (skipped), 1 = failure.
"""
import csv
import itertools
import math
import os
import re
import sys


def table(path):
    with open(path, encoding="latin1", newline="") as f:
        rows = list(csv.reader(f, delimiter="\t"))
    return rows[0], rows[1:]


def main():
    log, level = sys.argv[1], sys.argv[2]
    min_mon = int(sys.argv[3]) if len(sys.argv) > 3 else 1
    root = os.environ.get("D2_TABLES")
    tag = os.environ.get("POPCHECK_TAG", "POPCHECK")
    if not root:
        print("%s level %s: SKIP (D2_TABLES unset)" % (tag, level))
        return 2
    lh, lrows = table(os.path.join(root, "drlg/patch_d2/Levels.txt"))
    mh, mrows = table(os.path.join(root, "monsters/patch_d2/monstats.txt"))
    lc = {n: i for i, n in enumerate(lh)}
    mc = {n: i for i, n in enumerate(mh)}
    stats = {r[mc["Id"]]: r for r in mrows if len(r) > mc["isSpawn"]}
    row = next(r for r in lrows if len(r) > lc["Id"] and r[lc["Id"]] == level)
    listed = [row[lc["mon%d" % i]].strip() for i in range(1, 11) if row[lc["mon%d" % i]].strip()]
    legal = set(listed)
    for k in listed:
        st = stats.get(k)
        if st:
            for col in ("minion1", "minion2", "spawn"):
                v = st[mc[col]].strip() if col in mc else ""
                if v:
                    legal.add(v)
    name = row[lc["LevelName"]] if "LevelName" in lc else row[0]
    nmon = int(row[lc["NumMon"]] or 0)
    den = int(row[lc["MonDen"]] or 0)

    types = packs = groups = trials = walk_trials = None
    den_log = None
    for l in open(log, encoding="utf-8", errors="replace"):
        m = re.search(r"POPULATE types level %s: ?(.*)$" % level, l)
        if m:
            types = m.group(1).split()
        m = re.search(r"POPULATE packs level %s: ?(.*)$" % level, l)
        if m:
            packs = m.group(1).split()
        m = re.search(r"POPULATE groups level %s: (\d+)" % level, l)
        if m:
            groups = int(m.group(1))
        m = re.search(r"(\d+) density rolls at MonDen (\d+)", l)
        if m:
            trials, den_log = int(m.group(1)), int(m.group(2))
            m = re.search(r"\((\d+) on walkable ground\)", l)
            walk_trials = int(m.group(1)) if m else trials
    if types is None:
        print("%s level %s (%s): FAIL no POPULATE types line in the log" % (tag, level, name))
        return 1

    counts = {}
    for item in types:
        k, n = item.rsplit(":", 1)
        counts[k] = int(n)
    total = sum(counts.values())
    problems = []
    for k in counts:
        if k not in legal:
            problems.append("illegal class %s" % k)
        elif k not in stats or stats[k][mc["isSpawn"]].strip() != "1":
            problems.append("class %s has no isSpawn" % k)
    # the drawn types are a subset of the listed ones of at most NumMon; everything seen must come from them (the type
    # itself, its Spawn replacement, or the minion1/minion2 followers of either): look for such a subset
    def reach(k):
        out = {k}
        st = stats.get(k)
        for col in ("spawn", "minion1", "minion2"):
            v = st[mc[col]].strip() if st else ""
            if v:
                out.add(v)
                sp = stats.get(v)
                if sp:
                    out.update(x for x in (sp[mc["minion1"]].strip(), sp[mc["minion2"]].strip()) if x)
        return out

    # unique / champion packs draw their class from the umon list (Levels.txt umon1..10; the original draws it apart
    # from the level's drawn types), so they only have to be a class of the umon / mon rows or a follower of one
    pack_counts = {}
    for item in packs or []:
        k, n = item.rsplit(":", 1)
        pack_counts[k] = int(n)
    umon = [row[lc["umon%d" % i]].strip() for i in range(1, 11) if "umon%d" % i in lc and row[lc["umon%d" % i]].strip()]
    pack_legal = set()
    for k in set(umon) | set(listed):
        pack_legal |= reach(k)
    for k in pack_counts:
        if k not in pack_legal:
            problems.append("illegal pack class %s" % k)
    packs_total = sum(pack_counts.values())
    if den == 0:
        if total or packs_total:
            problems.append("MonDen is 0 but %d natural monsters spawned" % (total + packs_total))
    else:
        if not listed or nmon == 0:
            problems.append("MonDen %d but the row lists no monster type (NumMon %d)" % (den, nmon))
        if total + packs_total < min_mon:
            problems.append("only %d natural monsters (need %d)" % (total + packs_total, min_mon))
    if counts and nmon:
        uniq = sorted(set(listed))
        ok = any(set(counts) <= set().union(*(reach(k) for k in sub))
                 for n in range(1, min(nmon, len(uniq)) + 1) for sub in itertools.combinations(uniq, n))
        if not ok:
            problems.append("no choice of %d listed types explains %s" % (nmon, " ".join(sorted(counts))))
    dens = ""
    if trials is not None:
        if den_log != den:
            problems.append("engine MonDen %d differs from Levels.txt %d" % (den_log, den))
        mean = trials * min(den, 10000) / 100000.0
        sd = math.sqrt(mean * (1 - min(den, 10000) / 100000.0))
        lo, hi = mean - 4 * sd - 2, mean + 4 * sd + 2
        wmean = walk_trials * min(den, 10000) / 100000.0
        dens = " groups=%s expected=%.1f (%.1f on walkable ground)" % (groups, mean, wmean)
        if groups is not None:
            # a roll makes at most one group (or a unique pack); sparse classes and blocked placements only lose some
            if groups > hi:
                problems.append("%d groups, more than the %d density rolls allow (expected %.1f)" % (groups, trials, mean))
            if den > 0 and wmean >= 8 and groups < wmean * 0.4:
                problems.append("%d groups, far below the %.1f expected on the walkable ground" % (groups, wmean))
    print("%s level %s (%s): monsters=%d +%d in unique/champion packs, types=%d (NumMon %d, MonDen %d)%s legal=%s spawned=%s -> %s" % (
        tag, level, name, total, packs_total, len(counts), nmon, den, dens, ",".join(sorted(legal)),
        " ".join("%s:%d" % (k, counts[k]) for k in sorted(counts)), "FAIL " + "; ".join(problems) if problems else "PASS"))
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
