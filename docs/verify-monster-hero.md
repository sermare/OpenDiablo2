# Verify monster scaling and hero stats (Ghidra, Game.exe)

Branch: feat/verify-monster-hero (fork). Observations only, no decompiled code.
(Intended home: ~/git/d2-re-notes/verify-monster-hero.md; copy there.)

## Monster scaling
- MONTBL_GetLevelScaledStats 0x006551e0: args (monster id, expansion flag, difficulty, level, field mask, out[]).
  Difficulty clamped 0..2; level clamped to last monlvl row (0x78-byte rows = 30 ints); negative fails.
- monlvl columns are interleaved as (normal, L- expansion) pairs per stat, 3 difficulties each: AC, TH, HP, DM, XP.
  A nonzero expansion flag picks the L- member of every pair. VERIFIED: L- columns used in LoD.
- noRatio (class flag bit 4): raw per-difficulty monstats shorts; else MulDiv(monlvl value, monstats percent, 100).
- MATH_MulDiv 0x0047f2c0 (a*b/c): plain IMUL + IDIV = TRUNCATES toward zero. VERIFIED, not nearest.
  Andariel normal HP 1024, hell 45023, LoD hell 60031.
- MONAI_InitMonsterStatsFromMonstats 0x00571af0:
  - level default = monstats Level of the difficulty. Only if diff > 0, an expansion/area value is set and the class has
    neither noRatio (bit 4) nor the bit tested against the global mask at 0x006cf268 (assumed boss) does it use
    LEVEL_GetMonsterLevel 0x0061dc00 (levels record +0x10 classic set / +0x16 expansion set, per difficulty).
    So Normal always uses monstats Level (old Go code used area level at Normal: fixed).
  - HP: min and max scaled separately; HP = min + uniform roll over (max-min+1), plus an extra percentage bonus
    (from FUN_00571760, not modelled), cap 0x7fffff, stored x256. XP stat 0xd gets the same bonus.
- FUN_0063ff30 is NOT player-count scaling. Runs only when expansion flag is 0, diff != 0 and monstats byte +0x4c != 1:
  classic-mode adjustment HP x1/2, AC x10/12, XP x10/17 (N) or x10/26 (H) (table 0x006ebfa0, 5 ints per entry), and
  level = monstats Level + 25*diff. Player-count scaling not found; still UNVERIFIED and unmodelled.

## Hero
- DATATBL_GetLevelFromExperience 0x00610b10: class clamped 0..6; counts consecutive Experience.txt rows (row 0 = 0)
  with threshold <= exp, bounded by class MaxLvl. Same as Go LevelFor. VERIFIED.
- Threshold accessor 0x00610ab0 indexes the file row directly. Gain handler 0x0057c510 caps experience at row MaxLvl-1
  (3520485254 for 99), NOT the last row; Go previously capped at row MaxLvl: fixed (ExpCap).
- Level-up 0x0056e770: stat 0x1e (next threshold) = row[level]. Per gained level n: max life += charstats byte +0x43
  * n * 0x40, max stamina += byte +0x45, max mana += byte +0x44, in 1/256 fixed point, i.e. exact quarter points with
  truncated display (floor of quarter total; matches d2statlist BaseMax). Stat points += byte +0x50 * n (5),
  skill points += n. Current life reset to max.
- Not checked: attack rating formula, per-vitality/energy point grants (covered by the level 94 save test).
