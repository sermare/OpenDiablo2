# Hit resolution verification (Game.exe 1.14, observations only)

Copy for ~/git/d2-re-notes/verify-hit-resolution.md (the agent could not write outside the worktree). Verified by reading the exe. Package d2common/d2combat.

## Pipeline
- Queue time (skill do-func -> 0x57bc00 FinalizeDamageStruct): 0x579ef0 ApplyResistsToDamageStruct runs ALL resist/flat/absorb work. 0x57b4b0 does none of it.
- 0x579ef0: reads defender stat 0x22 (34) and 0x23 (35), each <<8; if value>0 and damage struct dword [21] (+0x54) >0, value = MulDiv(value, [21], 0x400). Then 0x579d70 scale percent, event 0xb on the defender, 0x579e50 (freeze/stun length tweaks: stat 0x99 zeroes cold/freeze lengths, stat 0x76 halves, states 0x85/0x83 clear fields), then per type 0x579c90.
- Descriptor table 0x72ff38, 12 entries x 0x2c bytes: [0] struct byte offset, [1] resist stat, [2] max stat, [3] pierce stat, [4] absorb pct stat, [5] absorb flat stat, [6] flat slot. Physical uses slot 1 (stat 34). Fire, lightning, cold, magic use slot 2 (stat 35). Poison and length types: slot 0 (none).
- 0x579c90 per type: dmg<1 -> 0. If the ignore flag (struct flags 0x100/0x200/0x400 vs undead/demon/other class of the defender) is clear: dmg -= flat; if set: positive resist -> 0, flat skipped, absorb skipped. If dmg>0 and res!=0: res=min(res,100), dmg=MulDiv(dmg,100-res,100). Then 0x579c20 absorb. No floor: a flat larger than the damage leaves a negative component. 0x57a650 ApplyDamageToUnit subtracts life only when Total (struct [19]) > 0.
- 0x579c20 absorb: percent stat capped at 40, MulDiv(dmg,pct,100), then flat*256 capped to what is left; everything absorbed is heal.
- 0x579b10 effective resist matches the Go code, plus: positive PHYSICAL resist becomes 0 when the ATTACKER has state 0x2f and the DEFENDER is undead (helper 0x63f9e0), also in ignore mode. Pierce from the attacker, resist and max from the defender.

## 0x57b4b0 (resolve queued hit)
Find queued hit by (attacker type,id,defender type,id) in the defender list (+0xac); copy 0x70 bytes; range check 0x622e40; if result bit 1: an attacker player/monster in mode 0 aborts; struct flags := 0x20; 0x57a650 ApplyDamageToUnit (resists skipped); if struct [27]>0 call 0x622020(def,value); if hit class byte [25]==0 and class low nibble ==0, class |= 0x623e20(attacker); 0x57b380; then event 7 on the ATTACKER, event 3 on the DEFENDER; if hit: 0x5cf650; then 0x57ae50, then 0x57a960. Event 6 (missile) is dispatched elsewhere; callbacks test edx==6.

## Item event callbacks (0x5be860; table 0x72fd48, index = ItemStatCost itemeventfunc)
Callback args: (unit with the stat, other unit, damage struct, packed stat id<<16|layer), edx = event id. Stat 135 open wounds = func 15 = 0x5bda80 (Ghidra name "ITEM_PotionApplyTimedRecovery" is wrong). Stat 136 crushing blow = func 16 = 0x5bdbf0 (no Ghidra function there). Both run AFTER the base damage was subtracted. Others: 7 knockback (sets result bit 8), 8 howl, 14 freeze.
- Crushing blow: chance = attacker stat 136; Roll(100)<chance on the attacker rng. Divisor: defender player, or helper 0x63fed0 != 0 (monster classes 0x10f,0x152,0x167,0x230,0x231) -> 10; else boss (monstats +0xc & 0x40, helper 0x63fa40) or monster data +0x16 & 2 -> 8; else 4. For monsters only (not the 10 case): divisor += divisor*bonus/100, bonus from 0x571720 on stat 100 (monster_playercount, min 1): [0,0,50,100,150,200,250,300,350] for <9 else (n-2)*50. Event 6 doubles the divisor. removed = currentLife/divisor (8.8, idiv); phys = defender RAW stat 36 (not effective): >=100 -> 100, >0 -> removed -= removed*phys/100. life -= removed, floor 0; life<=0 sets struct result |= 2; removed>0 calls 0x622020(def,0x93).
- Open wounds: chance = stat 135, Roll(100)<chance on the attacker rng; L = attacker level (min 1); value = 0x5bd9c0(L,{9,18,27,36,45}) + 40: 0 for L<=1, else (L-1)*9 to 15, +18/level to 30, +27 to 45, +36 to 60, +45 above. Defender player: /4, and /2 more for event 6. Defender monster with monster data +0x16 & 0xc: /2 (any event). Creates state 0x3e on the defender via 0x56c740: 200 frames, skill 0, level 1, stat 0x4a (hp regen) = -value. Same state/skill/level only refreshes the expiry.

## 0x57bd60 / 0x57bcb0 / 0x57cc10
- 0x57bd60 returns 8 evade, 0x10 anim block, 4 dodge, 2 avoid. Evade: player mode 3 or 2, monster mode 15 or 2; stat 0x154, Roll(100)<chance; a failed evade ends the chain. Not moving: anim block (0x57bcb0 chance>0 and animation action == 0xd, one roll), then avoid 0x153 if the arg flag (missile) else dodge 0x152.
- 0x57cc10: a player defender in mode 3 is auto-hit (to-hit not rolled). Codes 2->0x100, 4->0x80, 0x10->0x8000, shield 1->0x10, any non-zero clears bit 1; evade 8 sets no bit. A hit on a unit without state 0x36 gets bit 4.
- 0x57bcb0 is NOT the monster stat-20 block chance: it returns the max value among entries of layered stat 0x15c (348) whose item type matches the weapon in hand slots 4 or 5 (Weapon Block passive). Monster shield block stays as in the earlier notes.

## 0x57b8b0 (only from 0x57b9c0, player attackers)
Pointers to DEF and AR. Attacker stat 0x73 != 0 and defender is a plain monster (not monster data flag 0xa, not boss 0x63fa40, not special 0x63fed0): DEF=0. Stat 0x74: pct halved when defender is a player, boss, +0x16&2 monster or special; clamp 0..100; DEF -= MulDiv(DEF,pct,100). Stat 0x7b added to AR vs demon (0x63f990), 0x7c vs undead (0x63f9e0). Then in 0x57b9c0: AR pct = stat 0x77 + bonus arg + weapon part + stat 0xb3 typed entries; AR += MulDiv(AR,pct,100) (operands confirmed). Monster attackers: stat 0x13 + bonus + dex*5, no 0x57b8b0.

## 0x57ae50 (reaction)
Result word at struct+4. Defender with state 0x36: only result&2 toggles state 0x5c. Player: 0x80 dodge (state 0x41), 0x100 avoid (0x42), 0x200 evade (0x44), (0x10|0x8000) block unless 0x4000: plays only if frame - stat 0x5f > 15 + fbr(stat 0x66)/8 (trunc to zero), then stat 0x5f = frame; 2 death; 8 knockback (mode 0x13); 4 hit recovery (mode 4) if state 0x15 present or 0x57aa60 == 0, else only unit flag 0x8000; 0x4000 flag only. Monster (not mode 0/0xc): 8 becomes 4 when the monster lacks the mode; order death, knockback, block 0x10 (mode 6, not for class ids 0xf3,0x14d,0x2c1), hit 4 (mode 3), 0x4000, hp-bar refresh when stat 0x160 differs by >4.
- 0x57aa60 (1 = no hit recovery): frozen (state 1), poison-only damage, or Total<0x100 -> 1. D by hit class: 2,6,10,11 -> 8; 4,8 -> 0x20; 5 -> 0x40; else 0x10. maxLife/D > Total -> 1. Else 0 only if (maxLife/(D/2)<=Total or rand) and (maxLife/(D/4)<=Total or rand) and (not a monster or monster has the mode). rand = RAND_RollSeedPow2Mask, mask not read.

## Engine edits needed (d2core, not done)
1. d2core/d2monsters/combat.go:179, d2skills/engine.go:417,500,515-516: replace ApplyResist(dmg-reduce,res) with d2combat.ReduceComponent per type; flat via FlatReduction (stat 34 physical, 35 fire/lightning/cold/magic, none poison/burn) scaled by ScaleFlatReduction; pass the ignore flag; add absorb with the 40 cap. Do not floor components; floor Total at the end, subtract only when Total>0.
2. engine.go:485 resist(): set PhysicalNullified (attacker state 0x2f and defender undead) for physical.
3. Add crushing blow and open wounds as post-damage steps on the attacker rng: RollCrushingBlow with CrushingBlowDivisor(defender class, players in game, missile), then RollOpenWounds + OpenWoundsValue and a 200-frame state 0x3e with negative stat 0x4a (ApplyOpenWounds refresh).
4. effects.go:869 ResolveAttack: set AutoHit for a player defender in mode 3, DefenderLacksState36, map evade to no flag.
5. Player AR path: AdjustAROperands (stats 0x73,0x74,0x7b,0x7c) before the AR percent; monsters skip it.
6. Reactions: SelectReaction / BlockRecoveryReady / StaggerSuppressed; track stat 0x5f.
7. Weapon Block (stat 0x15c) feeds AvoidInput.AnimBlockChance.

## Unverified
Unit of stat 0x4a per tick; mask of RAND_RollSeedPow2Mask in 0x57aa60; what fills struct [21] and [27]; event 0xb and event 6 dispatch sites; names of states 0x2f, 0x3e, 0x15; classes behind 0x63fed0; event ids dispatched inside 0x57a650.
