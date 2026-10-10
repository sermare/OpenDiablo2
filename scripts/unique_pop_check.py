#!/usr/bin/env python3
"""Checks the unique / champion population of one level in a game log against the real tables.

usage: unique_pop_check.py <log.txt> <level id>

Log lines read (written by the real-map population and the game screen):
  ... population ranks level N diff D: champions=c rares=r minions=m supers=s    (generator, before the reachability filter)
  POPULATE ranks level N: champions=c rares=r minions=m supers=s                 (game screen, after it)
  POPULATE rank leaders level N: key:C key:R key:S ...
Table rules checked (Levels.txt, monstats.txt under D2_TABLES; the original's rules read from the exe):
  - at most max(MonUMin, MonUMax) of the difficulty unique + champion packs per level (the picker stops there);
  - a pack leader is a class of the level's umon list (Normal) or of its mon / nmon list or one of their
    spawn alternates (Nightmare / Hell, where the type list is used);
  - the leaders seen in the game are a subset of the generator's (the filter only removes);
  - a rare brings 3..6 minions in the generator.
Prints one UNIQPOP line; exit 0 = pass, 2 = no tables (skipped), 1 = failure.
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
    root = os.environ.get("D2_TABLES")
    if not root:
        print("UNIQPOP level %s: SKIP (D2_TABLES unset)" % level)
        return 2
    lh, lrows = table(os.path.join(root, "drlg/patch_d2/Levels.txt"))
    mh, mrows = table(os.path.join(root, "monsters/patch_d2/monstats.txt"))
    lc = {n: i for i, n in enumerate(lh)}
    mc = {n: i for i, n in enumerate(mh)}
    stats = {r[mc["Id"]]: r for r in mrows if len(r) > mc["isSpawn"]}
    row = next(r for r in lrows if len(r) > lc["Id"] and r[lc["Id"]] == level)

    gen = game = None
    leaders = None
    for l in open(log, encoding="utf-8", errors="replace"):
        m = re.search(r"population ranks level %s diff (\d): champions=(\d+) rares=(\d+) minions=(\d+) supers=(\d+)" % level, l)
        if m:
            gen = tuple(int(x) for x in m.groups())
        m = re.search(r"POPULATE ranks level %s: champions=(\d+) rares=(\d+) minions=(\d+) supers=(\d+)" % level, l)
        if m:
            game = tuple(int(x) for x in m.groups())
        m = re.search(r"POPULATE rank leaders level %s: ?(.*)$" % level, l)
        if m:
            leaders = m.group(1).split()
    if gen is None or game is None or leaders is None:
        print("UNIQPOP level %s: FAIL missing log lines (generator %s, game %s, leaders %s)" % (level, gen, game, leaders))
        return 1

    diff = gen[0]
    sfx = ["", "(N)", "(H)"][diff]
    umin = int(row[lc["MonUMin" + sfx]] or 0)
    umax = int(row[lc["MonUMax" + sfx]] or 0)
    limit = max(umin, umax)
    if diff == 0:
        legal = {row[lc["umon%d" % i]].strip() for i in range(1, 11) if row[lc["umon%d" % i]].strip()}
    else:
        legal = {row[lc["nmon%d" % i]].strip() for i in range(1, 11) if row[lc["nmon%d" % i]].strip()}
        legal |= {row[lc["mon%d" % i]].strip() for i in range(1, 11) if row[lc["mon%d" % i]].strip()}
        for k in list(legal):
            if k in stats and stats[k][mc["spawn"]].strip():
                legal.add(stats[k][mc["spawn"]].strip())

    problems = []
    packs_gen = gen[1] + gen[2]
    packs_game = game[0] + game[1]
    if packs_gen > limit:
        problems.append("%d unique/champion packs, the table allows %d (MonUMin %d MonUMax %d)" % (packs_gen, limit, umin, umax))
    if packs_game > packs_gen or game[0] > gen[1] or game[1] > gen[2]:
        problems.append("the game has more packs than the generator planned")
    if umin > 0 and packs_gen == 0:
        problems.append("MonUMin is %d but the generator made no pack" % umin)
    for t in leaders:
        k, r = t.rsplit(":", 1)
        if r in ("C", "R") and k not in legal:
            problems.append("%s leader %s is not in the level's list %s" % (r, k, sorted(legal)))
    if sum(1 for t in leaders if t.endswith(":C")) != game[0] or sum(1 for t in leaders if t.endswith(":R")) != game[1]:
        problems.append("leader list disagrees with the counts")
    if gen[2] and gen[3] < 3 * gen[2]:
        problems.append("%d rares but only %d minions (3..6 each)" % (gen[2], gen[3]))
    print("UNIQPOP level %s diff %d: generator champions=%d rares=%d minions=%d supers=%d; game champions=%d rares=%d minions=%d supers=%d; "
          "table MonUMin %d MonUMax %d -> %s" % ((level, diff) + gen[1:] + game + (umin, umax, "FAIL " + "; ".join(problems) if problems else "PASS")))
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
