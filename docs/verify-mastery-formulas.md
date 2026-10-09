# Weapon mastery formulas in Game.exe (1.14b): verified

Addresses and observations only (read-only Ghidra, disassembly decoded by hand). Branch feat/verify-mastery-formulas. Copy to ~/git/d2-re-notes/verify-mastery-formulas.md (the agent could not write outside its worktree).

## Lookup 0x646bc0 (unit, weapon, skill or 0, mode) and throw helper 0x646a90
- Helper 0x646a90 takes the skill argument, else the unit's current skill (0x620500). Throw family only when: weapon is throwable (0x62bbd0), the skill record word +0x18 (itypea1) inherits from item type 0x30 (= thro, row 48 of ItemTypes), and skill record byte +0x14 == 2 (the range column; "rng" = 2 is inferred from the layout, consistent with every skill below).
- When that holds it reads ONLY stats 0x159 / 0x15a / 0x15b (mode 0/1/2), max over entries whose param the weapon is of (0x629d70), sets "handled" and returns even 0. Otherwise 0x646bc0 reads only 0x156..0x158.
- Real rows that select the throw family (itypea1 thro or a descendant, range rng): Throw, Left Hand Throw, Double Throw, Poison Javelin, Plague Javelin, Lightning Bolt, Lightning Fury. Not: Attack, Left Hand Swing, Jab, Power Strike, Impale, Fend, Charged Strike, Fire Arrow, melee skills.
- ItemTypes: comb = {mele, thro}; tkni = {comb, knif}; taxe = {comb, axe}; jave = {comb, spea}. So a thrown javelin or knife is "of thro" (throw family when thrown with a throw skill) and, when used in melee, is of spea / knif / axe / mele (melee masteries then apply).

## Call sites
- To-hit, COMBAT_RollToHit 0x57b9c0, call at 0x57ba63 (mode 0, player attackers, weapon from 0x5335c0, skill 0). The result seeds a percent sum together with stat 0x77 and the function's first stack argument (the skill's to-hit bonus). Final: AR = baseAR + MulDiv(baseAR, sum, 100) (0x47f2c0 is a*b/c with overflow care). So the mastery is a PERCENT of attack rating, additive with item_tohit_percent. The lookup runs only when the second stack argument is 0: COMBAT_RollAttackOutcome (0x57ccb8) passes 0; MISSILE_ProcessHitOrExpire (0x5abb74) passes 1, so missile hits never get the mastery to-hit.
- Damage, COMBAT_RollPhysicalDamage 0x579120, call at 0x5792ab (mode 1): the value is added to the same accumulator as the stat 0x19 inputs, the weapon strength bonus (0x629a50 * stat 0 / 100) and dexterity bonus (0x629a80 * stat 2 / 100). The accumulator is clamped at -90 and min/max are each scaled once by (100 + acc)/100. So damage mastery is ADDITIVE with enhanced damage, not multiplicative. Same shape in MISSILE_BuildDamageDescriptor 0x64ca60 (call 0x64cd32, mode 1), whose output goes to descriptor field +0x78 (the percent). That branch is taken only when the missile record byte +0x12d (SrcDam) is non-zero and the owner has a weapon item.
- Crit, COMBAT_BuildAttackerDamage 0x5794e0, call at 0x5795d7 (mode 2), skipped when the third stack argument is non-zero (0 at the COMBAT_FinalizeDamageStruct call 0x57bc4a). Value > 0 and rand(100) (0x46deb0 with 0x64) < value -> doubles physical (+0x8 *2, result bit 0x2000). If that fails it falls through to a second independent roll against stat 0x151 and then one against stat 0x8d. So it is a SEPARATE roll (combined chance 1 - prod(1 - p)), mastery first. Missile equivalent FUN_0064ba70 (called at 0x64cce0 from MISSILE_BuildDamageDescriptor, sets descriptor flag 0x2): order is stat 0x151, stat 0x8d, then the mastery (call 0x64bb06, mode 2).
- Other callers by mode: 0x5816b4 FUN_00581650 mode 0, 0x5e9724 MONAI_Think_BladeCreeper mode 0, skilldesc callers (display only).

## Engine changes
- to-hit mastery now goes into AttackRatingPct (percent), was a flat AR add.
- damage mastery stays in the additive percent (verified) and is now also added to missile descriptors (castMissile and cast.desc) when SrcDam > 0.
- mastery lookup picks the throw family only for a throwable weapon + throwing skill (SkillThrows), else the melee family only (was: both).
- Not wired: missile crit (flag 0x2 consumer not traced) and the missile roll order.
