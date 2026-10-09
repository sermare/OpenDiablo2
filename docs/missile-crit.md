# Missile critical strike in Game.exe 1.14b: verified

Addresses and observations only (read-only Ghidra, disassembly decoded by hand). Branch feat/missile-crit. Copy of ~/git/d2-re-notes/missile-crit.md.

## Where it is rolled
- MISSILE_BuildDamageDescriptor 0x64ca60 (only caller 0x59d4bc in the create path 0x59d4a0; the descriptor is a 0x7c byte struct, memset at the start) rolls the crit ONCE when the missile is created, not per hit.
- Effective SrcDam byte (stack slot esp+0x10): if the missile record has a Skill (record word +0x194 > 0) or the record flag bit tested at 0x64cadd/0x64cbb4 is set, it is the SKILL record byte +0x1a5 (skills.txt SrcDam) unless the missile record byte +0x12d (missiles.txt SrcDamage) is 0xff (-1), which forces 0 (0x64cbde..0x64cbf6). Otherwise it is the record byte +0x12d itself.
- Call site A, 0x64cce0: effective SrcDam non-zero (descriptor flag 0x1 is set just before at 0x64ccb2). Weapon argument = owner's weapon: player 0x623b70(unit,1), monster with owner data 0x6229e0, else null. Result success -> descriptor flag 0x2 (0x64cce9).
- Call site B, 0x64cf2a: effective SrcDam 0 but record byte +0x12e (SrcMissDmg) non-zero on the non-skill branch and an owner present: same helper with weapon 0 (no mastery roll).
- So the gate is "weapon-based physical missile" (effective SrcDam != 0), NOT "ranged attacks only". With real tables: Magic Arrow, Fire Arrow, Cold/Ice/Immolation/Freezing/Exploding Arrow, Multiple Shot (SrcDam 96), Guided Arrow, Strafe (96), Poison/Plague Javelin, Lightning Bolt (96), Lightning Fury and monster bow skills qualify; Fire Bolt, Charged Bolt, Fireball etc. have SrcDam 0; poisonjavcloud, plaguejavcloud and furylightning have SrcDamage -1 so they do not.

## Helper 0x64ba70 (weapon in eax, unit in esi; returns 1 on success)
- Order: stat 0x151 (passive_critical_strike), stat 0x8d (item_deadlystrike), then, only when a weapon was passed, the mastery lookup 0x646bc0 mode 2 (call 0x64bb06). First success returns 1. Each roll is (LCG 0x6ac690c5 on the unit seed at unit+0x20/+0x24) % 100 < chance, the same generator as the other rolls.
- The 0x151 roll is NOT guarded by a chance test: it consumes one random step even when the chance is 0. The 0x8d and mastery rolls are skipped when the chance is 0. (The port skips the 0x151 roll at 0 so a hero without crit stats keeps an identical random stream: deliberate deviation.)

## Consumer of flag 0x2
- 0x64be80 (called at 0x59d4c9 right after the build) copies the descriptor into the missile unit's stats: +4/+8 -> stats 0x15/0x16 (min/max damage), elemental fields, +0x78 -> stat 0x19; descriptor flags 0x100/0x200/0x400 -> stats 0x67/0x68/0x6a = 1; descriptor flag 0x2 -> missile stat 0x8d = 1 (0x64c0d7..0x64c0e6).
- The physical damage struct builder 0x5a6690 (type switch, case 0 at 0x5a66ba) reads min/max through 0x5a6330, applies the attacker percent (stat 0x19 plus demon/undead bonuses 0x79/0x7a, clamp at -90), THEN, if the missile's stat 0x8d is non-zero, doubles the physical value (0x5a674a) and sets result bit 0x2000. So the crit is a plain x2 of physical after the percent, shared by every hit of that missile (pierce, repeated hits).
- The descriptor flag 0x2 also makes COMBAT_FinalizeDamageStruct 0x57bc00 skip the melee builder 0x5794e0 for the damage struct that carries it (0x57bc3a), so no second roll.

## Unverified
- Whether every missile hit function reaches 0x5a6690's physical case (it was read for hit function 1; other damage functions' use of missile stat 0x8d was not traced).
- Predicate 0x628b90 at 0x64cd15 (halves the effective SrcDam when the record flag at 0x6cf268 is set and the weapon item satisfies it, probably Half2HSrc with a two-hander); not modelled.
- Meaning of the record flag bit tested through 0x6cf26c / 0x6cf250 (skill-damage branch selection).

## Engine changes
- d2combat.RollMissileStrike (order critical, deadly, mastery), d2missile.DamageDesc.Crit (doubles physical after DamagePct in Roll, sets ResultCritical), d2missile.Spec.SrcDam, d2skill.missileCrit called from castMissile. Area-hit descriptors (cast.desc) are not missiles and do not roll it.
