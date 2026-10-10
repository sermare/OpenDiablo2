#!/usr/bin/env python3
"""Checks the "POPULATE types level N: key:count ..." line of a game log against the real tables.

usage: act3_pop_check.py <log.txt> <level id> [<min monsters>]

Legal classes of a level: mon1..mon10 of its Levels.txt row plus the minion1/minion2 followers of those classes
(natural groups bring PartyMin..PartyMax of them); each must have isSpawn. Tables come from D2_TABLES
(drlg/patch_d2/Levels.txt, monsters/patch_d2/monstats.txt). Prints one ACT3POP line; exit 0 = pass,
2 = no tables (skipped), 1 = failure.
"""
import csv
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
    if not root:
        print("ACT3POP level %s: SKIP (D2_TABLES unset)" % level)
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
            for col in ("minion1", "minion2"):
                v = st[mc[col]].strip()
                if v:
                    legal.add(v)
    name = row[lc["LevelName"]] if "LevelName" in lc else row[0]

    line = None
    for l in open(log, encoding="utf-8", errors="replace"):
        m = re.search(r"POPULATE types level %s: ?(.*)$" % level, l)
        if m:
            line = m.group(1).split()
    if line is None:
        print("ACT3POP level %s (%s): FAIL no POPULATE types line in the log" % (level, name))
        return 1

    counts = {}
    for item in line:
        k, n = item.rsplit(":", 1)
        counts[k] = int(n)
    total = sum(counts.values())
    problems = []
    for k in counts:
        if k not in legal:
            problems.append("illegal class %s" % k)
        elif k not in stats or stats[k][mc["isSpawn"]].strip() != "1":
            problems.append("class %s has no isSpawn" % k)
    if total < min_mon:
        problems.append("only %d natural monsters (need %d)" % (total, min_mon))
    nmon = int(row[lc["NumMon"]] or 0)
    # only the leaders count against NumMon: followers come from minion1/minion2
    # (a listed class that is also the minion of another spawned class may be there only as a follower)
    followers = set()
    for k in counts:
        st = stats.get(k)
        if st:
            followers.update(v for v in (st[mc["minion1"]].strip(), st[mc["minion2"]].strip()) if v and v != k)
    leaders = [k for k in counts if k in listed and k not in followers]
    if len(leaders) > nmon:
        problems.append("%d listed types drawn, NumMon is %d" % (len(leaders), nmon))
    print("ACT3POP level %s (%s): monsters=%d types=%d (NumMon %d) legal=%s spawned=%s -> %s" % (
        level, name, total, len(counts), nmon, ",".join(sorted(legal)),
        " ".join("%s:%d" % (k, counts[k]) for k in sorted(counts)), "FAIL " + "; ".join(problems) if problems else "PASS"))
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
