# Monster scaling and hero stats: remaining items (Game.exe, Ghidra)

Branch: feat/monster-scaling-complete (fork). Observations only. Follows verify-monster-hero.md.

## Player-count scaling (VERIFIED)
- FUN_00571760 (called from MONAI_InitMonsterStatsFromMonstats 0x00571af0, also from 0x005e7710, 0x005fb630, 0x005efbf0) fills
  out[0]=HP pct, [1]=XP pct, [2]=difficulty record +0x10, [3]=difficulty, [4]=player count (min 1).
  Only for monster units whose monstats byte +0x4c (Align column) is 0; otherwise pcts 0 and count 1.
- Pct from FUN_00571720 (HP) / FUN_00571740 (XP): count<9 indexes identical tables 0x006e2888 / 0x006e28ac =
  0,0,50,100,150,200,250,300,350; count>=9: HP (n-2)*50, XP (n*5+0x82)*2.
- Count = FUN_005331a0: iterates the player table (FUN_00551930, callback 0x00533170 counts players passing predicate 0x00552234);
  if game byte +0x6a is 1..3 the result is max(count, global 0x0087bdb0). 0x0087bdb0 is the forced "players X" value, set 0..8
  by the setter at 0x00533190.
- Use in 0x00571af0: hp = (min+roll) + MulDiv(min+roll, hpPct, 100), then cap 0x7fffff (the bonus is BEFORE the cap), stored <<8.
  XP stat 0xd = xp + MulDiv(xp, xpPct, 100). Stat 0x64 gets the player count at spawn.
- Align (+0x4c) column identity inferred: all noRatio summons have Align 1 and 0x0063ff30 skips Align == 1.

## Classic adjustment 0x0063ff30 (VERIFIED)
Table 0x006ebfa0, 5 ints per entry {stat, num, den(N), num, den(H)}: stat 7 HP 1/2 both; stat 0x1f AC 10/12 both; stat 0xd XP 10/17 N,
10/26 H. Applied after player scaling, truncating MulDiv, then stat 0xc (level) = monstats Level (+0xaa, normal) + 25*diff.
Gate: game+0x70 == 0, difficulty != 0, Align != 1. That game+0x70 == "classic game" is inferred (the same field feeds the
area-level lookup), not proven. The Go model is an option (Options.Classic).

## Level-exclusion flag at 0x006cf268 (CORRECTION, partly unverified)
- Masks at 0x006cf250.. are 1,2,4,8,16,32,64,128. 0x006cf258 = 4 (bit 2), 0x006cf268 = 0x40 (bit 6), both tested against the byte at
  monstats +0xc. DATATBL_GetMonStatsByteFlag 0x00452b20 takes a bit index over the flag dword at +0xc.
- Flag column strings (0x006edaa0..0x006edcb0, stored in reverse of table order): isSpawn, isMelee, noRatio, SetBoss, BossXfer, boss,
  primeevil, opendoors, npc, interact, ... Checks: noRatio = bit 2 (matches the code) and interact = bit 9 (NPCSRV_AddPlayerToInteractList
  0x00570a50 passes 9). So bit 6 is primeevil and boss would be bit 5 (mask 0x6cf264). The earlier "assumed boss" is probably wrong.
  Not settled: one flag test in the same function (+0xf bit 0, bit 24, selects inventory creation) does not fit that order, and the
  loader table was not located (no xref tool available). Go keeps both (Class.Boss, Class.PrimeEvil); LevelExclusionFlag defaults to primeevil.
  Affected classes (boss=1, primeevil blank): radament, summoner, izual, bloodraven, griswold, nihlathakboss, putriddefilers, uberizual.

## Hero (VERIFIED)
- Attack rating COMBAT_GetPlayerAttackRating 0x00622710 (players only): (stat 2 dexterity - 7) * 5 + stat 0x13 (gear to-hit) +
  charstats byte +0x3c (ToHitFactor). Matches d2combat.PlayerAttackRating.
- Vitality point (FUN_0056ea50; allocate handler 0x0056ec40, stat reset 0x0056eb60): max life (stat 7) += charstats +0x46 (LifePerVit) * n * 0x40;
  max stamina (0xb) += +0x47 * n * 0x40; current life (6) and stamina (10) get the same grant only when n > 0; current clamped to max.
- Energy point (FUN_0056e970): max mana (9) += charstats +0x48 (ManaPerEnergy) * n * 0x40; current mana (8) only when n > 0; clamped.
- Stat reset 0x0056eb60 sets strength and dexterity to charstats +0x30 / +0x31 and refunds vit/ene through the two functions above.
- Stat-change callback FUN_005595b0 rescales current life/mana/stamina proportionally when the maximum changes (stats 7, 9, 0xb).
