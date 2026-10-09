# Animation speed oracle (IAS / FCR / FHR / FBR / FRW)

Code: d2common/d2animspeed (branch feat/anim-speed-oracle). Observations only, no decompiled code.
(Written to docs/ because the worktree could not write to ~/git/d2-re-notes; copy there.)

## Exe addresses (Game.exe 1.14b)
- 0x624150 PATH_UpdateUnitVelocityAndAnimRate: picks the animation rate (8.8, short at unit+0x4c, also unit+0x3c) per unit type/mode. Base rate = AnimData speed (anim record +0xc, record pointer at unit+0x50). Result 0 stays 0, cap 0x7fff.
- 0x6218d0 (is-cast-mode test), 0x621690 (walk/run test), 0x621770 (attack test), 0x621820 (a "keep base rate" mode).
- PATH_ScaleSpeedByStatTableEntry (index in EAX): table of 12-byte rows at 0x6ea3d4 = {flag, K, stat}. Rows: 0 -> K120 stat 93 (item IAS); 1 -> K120 stat 99 (FHR); 2 -> K120 stat 105 (FCR); 3 -> K120 stat 102 (FBR); 4 -> K150 stat 96 (FRW). Value = K*p/(K+p) (C truncation) when flag==1 and p != 0. VERIFIED.

## Rules (VERIFIED unless marked)
- Hit recovery (player mode 4 GH, monster 3): rate = S*(50 + dim(FHR))/100, no clamp. With 0 FHR the animation runs at HALF the AnimData speed (Amazon: 6 frames -> 12 ticks; community lists 11).
- Block (player mode 9, monster 6): rate = S*(50 + dim(FBR))/100; 100 instead of 50 while state 0x65 (meaning unknown, U); min 1.
- Cast (index 2): rate = S*(100 + dim(FCR))/100, cap 175 percent, no lower clamp.
- Attack (index 0): pct = stat0x44 + dim(item IAS); dual wield adds (avg of both weapons' 0x44 minus the first's); player mode 0x12 (kick) adds -30; clamp 15..175; rate = S*pct/100. That stat 0x44 = 100 - WSM + skill bonus is UNVERIFIED (community rule).
- Walk/run (index 4): pct = max(25, stat0x43 + dim_K150(FRW)); rate = S*pct/100; path velocity = baseVelocity*pct/100. stat 0x43 base 100 assumed (U).
- Other modes: stat 0x45 clamped 15..175.
- Ticks for N frames = ceil(256*N/rate) (matches the engine tickStepper). The community "-1" on frame counts is UNVERIFIED (breakpoints do not depend on it).

## Data check (VERIFIED; expansion animdata.d2 from d2exp.mpq; patch_d2 has none)
Frames (HTH): GH Ama6 Sor8 Nec7 Pal5 Bar5 Dru7 Ass5; BL Ama3 Sor5 Nec6 Pal3 Bar4 Dru6 Ass3; SC Ama20 Sor14 Nec16 Pal16 Bar14 Dru15@208 Ass17; walk speeds Pal 288, Bar 168, Dru 136. CharStats #gethit/#swing/#spell columns are not what the exe uses.
The rule reproduces d2herostats: FHR Ama/Sor/Pal/Ass, all 7 FCR rows, FBR Ama/Bar/Ass.

## Discrepancies in d2common/d2herostats (not edited)
- Necromancer FHR should be 0,5,10,16,26,39,56,86,152,377 (current row = Sorceress FCR).
- Barbarian FHR should be 0,7,15,27,48,86,200.
- Druid FHR (human): animdata A=7 gives the Necromancer row; current row unexplained (U).
- Sorceress FBR should be 0,7,15,27,48,86,200 (current = her FHR); Necromancer FBR 0,6,13,20,32,52,86,174,600.
- Paladin FBR with shield / Sorc lightning FCR need the shield / skill-specific AnimData record (U).

## Engine today
- d2asset createMode takes frameCount and speed straight from the AnimData record of token+mode+weaponclass; no modifier is ever applied. The player never enters GetHit/Block modes, so hit/block recovery animations do not exist.
- Differential test d2core/d2asset/anim_rate_oracle_test.go: for speeds 1..1024 the engine rate equals the oracle walk/run/attack/cast rate with no modifiers. Hit/block differ (oracle = half speed), so not wired.

## Edits needed to wire it (not done: engine has no source for the stats)
1. d2asset.Composite: add SetAnimRate(rate) (seconds = 1/(rate*speedUnit)); call after createMode with d2animspeed.{WalkRate,AttackRate,CastRate}(record.Speed(), ...). Needs stats 0x43/0x44 and item stats 93/96/99/102/105 summed in the hero stat list.
2. d2mapentity.Player: add hit-recovery and block modes using HitRate/BlockRate; velocity via WalkVelocity.
3. Fix the d2herostats rows above.
