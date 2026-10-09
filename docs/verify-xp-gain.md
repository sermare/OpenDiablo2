# Kill experience (Game.exe 1.14, addresses and observations only)

(Placed in docs/ because agents may only edit inside their worktree; copy to ~/git/d2-re-notes/verify-xp-gain.md.)

Chain: unit death handler 0x005a2900 -> credit routine 0x0057c990.

## 0x0057c990 credit routine (verified by disassembly)
- Victim experience = victim stat 0x0d; victim level = stat 0x0c.
- 0x0057c7b0 resolves the killer: a player is itself; a monster (pet/merc/summon) resolves to its owner via 0x0058cec0
  (must be a player unit, else no credit). So merc and pet kills credit the OWNER as a normal player kill; no extra penalty.
- The player's merc (lookup 0x00572cc0, relation 7) also gets credit: xp through 0x0057c490 with the MERC's level; if the merc did not
  make the kill itself, xp*0x56>>8 (86/256); then 0x0057c860.
- Player path: party id (0x00552690 != 0xffff) -> 0x0057c6b0 split with the UNSCALED victim xp; else 0x0057c490 then 0x0057c510.

## 0x0057c490 per-recipient pipeline
xp clamp 0x7fffff; xp<=0 returns 1; recipient level >= class max level returns 0; level scaling 0x0057c300; ExpRatio scale 0x0057c3a0
(xp*ExpRatio[level] >> ExpRatio[row 0]; Experience table record is 32 bytes/level, ratio the 8th dword); item stat 0x55 (+% experience):
xp += xp*pct/100; then 0x0057c400/0x0057c510 (cap at row MaxLvl-1, see verify-monster-hero.md).

## 0x0057c300 level scaling (args: monster level, xp, char level), tables int32 in 1/256
- monster <= char: idx=min(char-monster,10) in 0x006e2960 = 256,256,256,256,256,256,207,159,110,61,13
- monster > char and char >= 25 and monster > 0: xp*char/monster (helper 0x0047f2c0 = a*b/c)
- monster > char and char < 25: idx=min(monster-char,10) in 0x006e298c = 256,256,256,256,256,256,225,174,92,38,5
- multiplier 256 returns xp unchanged, else xp*mult/256.

## 0x0057c6b0 party split
- recipients collected by callback 0x0057c5a0 (max 8, assertion at 8): skips dead (via 0x00552230), requires squared distance <= 0x1900 (6400)
  between member and killer (tile vs subtile unit UNVERIFIED), records unit and level (stat 0xc), sums levels.
- n==1: solo. n>1: total = xp + ((n-1)*xp*0x59>>8); share_i = trunc(float32(total)/sumLevels * level_i); each share then goes through
  0x0057c490 with the member's own level and 0x0057c510.
- Per-member weights = character level; party size bonus 89/256 per extra member; distance limit yes.

## 0x0057c860 / 0x0057cb10 merc
- 0x0057c860: merc experience stat 0x0d += 2 * gain, then loops the hireling level table (0x00666140) to the new level, capped below class-0 max level; applies level-up.
- 0x0057cb10 (called from the player level-up 0x0056e5d0) and 0x0057cab0: catch-up of merc experience to the owner's level.

## Go status
d2herostats.KillXP / LevelScaleXP / MercKillShare / ExpRatioScale; d2party.SplitKillXP.
Wired: d2monsters scaleKillXP (0x7fffff clamp, level scaling, level-99 cap, item +% experience = stat 85 item_addexperience via
HeroStatsState.ItemExperiencePct); party split (client offers the unscaled xp + monster level, server Roster.ShareKillXP = SplitKillXP
then each share scaled with the member's level; the member's client adds its own item %); merc share (Director.creditOwnerMerc: the
merc's own level pipeline, x86/256 when the owner or a pet made the kill).
NOT wired: ExpRatio (column missing in the extracted Experience.txt; the exe keeps it in its own table; applied 1:1).
Unverified: the "alive" and "distance <= 6400" recipient tests and their unit (roster has no positions; the area stands in),
x87 vs float rounding in the split.
