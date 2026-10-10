#!/usr/bin/env python3
"""Unit tests of scripts/pop_check.py on population lines as the game logs them (counts copied from a red verify run,
split the way the game screen now logs them: natural types / rare+champion packs / super uniques).

usage: D2_TABLES=$HOME/git/d2-tables python3 -I scripts/test_pop_check.py   (skipped without D2_TABLES)
"""
import os
import subprocess
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))


def run(level, lines):
    with tempfile.NamedTemporaryFile("w", suffix=".txt", delete=False) as f:
        f.write("\n".join(lines) + "\n")
    try:
        p = subprocess.run([sys.executable, "-I", os.path.join(HERE, "pop_check.py"), f.name, str(level)],
                           capture_output=True, text=True)
    finally:
        os.unlink(f.name)
    return p.returncode, p.stdout


def log(level, types, packs="", supers="", groups=0, rolls=None):
    out = ["[Game Screen][INFO] POPULATE types level %d: %s" % (level, types),
           "[Game Screen][INFO] POPULATE packs level %d: %s" % (level, packs),
           "[Game Screen][INFO] POPULATE supers level %d: %s" % (level, supers),
           "[Game Screen][INFO] POPULATE groups level %d: %d" % (level, groups)]
    if rolls:
        out.insert(0, "[Map Generator][INFO] real outdoor: population: 1 rooms, %d density rolls at MonDen %d (%d on walkable ground)" % rolls)
    return out


@unittest.skipUnless(os.environ.get("D2_TABLES"), "D2_TABLES unset")
class PopCheck(unittest.TestCase):
    def test_super_unique_class_is_not_a_level_type(self):
        # Stony Field: Rakanishu (fallen2) is a super unique, the level draws cr_archer1 / goatman1 / skeleton1
        rc, out = run(4, log(4, "cr_archer1:19 goatman1:12 skeleton1:63", "skeleton1:14", "fallen2:9", 50, (9288, 520, 7692)))
        self.assertEqual(rc, 0, out)

    def test_old_format_fails(self):
        # the same monsters with the super unique counted as a natural type: 4 types for NumMon 3, fallen2 not listed
        rc, out = run(4, log(4, "cr_archer1:19 fallen2:9 goatman1:12 skeleton1:63", "skeleton1:14", "", 50, (9288, 520, 7692)))
        self.assertEqual(rc, 1, out)
        self.assertIn("illegal class fallen2", out)

    def test_monden_zero_allows_supers_only(self):
        # Bloody Foothills has MonDen 0: its super uniques come from presets
        rc, out = run(110, log(110, "", "", "overseer1:1 minion1:20 imp3:5", 0, (0, 0, 0)))
        self.assertEqual(rc, 0, out)

    def test_monden_zero_rejects_natural(self):
        rc, out = run(110, log(110, "imp1:3", "", "", 0))
        self.assertEqual(rc, 1, out)
        self.assertIn("MonDen is 0", out)

    def test_unknown_super_class_fails(self):
        rc, out = run(4, log(4, "skeleton1:63", "", "nosuchmonster:1", 50))
        self.assertEqual(rc, 1, out)

    def test_groups_exclude_party_members(self):
        # Tal Rasha's Tomb (level 66) logged 215 groups against 89 expected: the party-pack members were counted
        # as groups. 4056 rolls at MonDen 2200 -> 89.2 expected; 90 groups is fine, 215 is not
        types = "mummy4:188 skeleton4:114 unraveler3:25 vampire1:46"
        rc, out = run(66, log(66, types, "", "", 90, (4056, 2200, 1500)))
        self.assertEqual(rc, 0, out)
        rc, out = run(66, log(66, types, "", "", 215, (4056, 2200, 1500)))
        self.assertEqual(rc, 1, out)
        self.assertIn("more than the 4056 density rolls allow", out)


if __name__ == "__main__":
    unittest.main()
