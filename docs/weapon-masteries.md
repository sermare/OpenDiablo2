# Weapon mastery stats in Game.exe (1.14b)

Addresses and observations only. Copy to ~/git/d2-re-notes/weapon-masteries.md (the agent could not write outside its worktree). Engine: d2common/d2skill/mastery.go, strike() in class.go, d2core/d2skills/units.go (heroUnit.Mastery, Engine.MasteryFor).

## Stat ids (ItemStatCost, verified in the table)
- 342/343/344 = 0x156/0x157/0x158 passive_mastery_melee_th / _dmg / _crit
- 345/346/347 = 0x159/0x15a/0x15b passive_mastery_throw_th / _dmg / _crit
- skills.txt rows with passiveitype (only these 8): Sword (swor), Axe (axe), Mace (blun), Pole Arm (pole), Spear (spea), Claw (h2h) -> melee family; Throwing (thro) -> throw family; Weapon Block (h2h, passive_weaponblock). Slots: stat1 th (ln12), stat2 dmg (ln34), stat3 crit (dm56). Amazon passives (Critical Strike, Penetrate, Dodge, Avoid, Evade...) have NO passiveitype: they are plain unkeyed stats; there are no Amazon bow/crossbow mastery rows in 1.14b.

## The lookup: 0x646bc0 SKILL_Func_646bc0(unit, weapon item, skill or 0, mode) (verified from decompile)
- mode 0/1/2 selects stat 0x156/0x157/0x158. It fetches ALL (param, value) entries of that stat on the unit (STATS_Fwd_626400, max 0x20) and returns the largest value among entries whose param satisfies ITEM_IsOfType(weapon, param) (the type or an ancestor). No unit or weapon: 0.
- First it calls 0x646a90: for a throwable weapon (ITEM_IsThrowableType) with a skill whose record +0x18 type inherits from 0x30 and whose +0x14 byte is 2, it does the same lookup over the throw family 0x159..0x15b and reports "handled" (0x646bc0 then returns that).
- So the weapon gate is at the reader, not at apply time (matches true-passives.md).

## Callers (12 xrefs)
- 0x57ba63 COMBAT_RollToHit: hand weapon from 0x5335c0, mode 0, player attackers only. How the value enters the roll is not visible in the decompile: UNVERIFIED (engine adds it to attack rating, flat).
- 0x5792ab COMBAT_RollPhysicalDamage: mode 1 on the hand weapon; result use not visible: UNVERIFIED (engine adds it to the damage percent).
- 0x5795d7 COMBAT_BuildAttackerDamage: mode 2, the weapon strike chance before passive_critical_strike and item_deadlystrike (skills-combat.md).
- Others: MISSILE_BuildDamageDescriptor 0x64cd32, 0x64bb06, 0x5816b4, MONAI_Think_BladeCreeper 0x5e9724, skilldesc 0x4e5860 / 0x4e607d (SKILLDESC_CalcSkillAttackRating) / 0x4e62b0 / 0x4e7470 / 0x4e78cf. Ranged/missile paths are NOT wired in the engine yet.

## Engine
heroUnit.Mastery(kind) -> Engine.MasteryFor(unit, kind, weaponType): the hero's TruePassiveStats mods, MasteryValue = max over keyed mods of the kind whose param the weapon is of (ItemTypes Equiv1/2 chain, TypeIs). Both families are searched (exe picks the throw family by the condition above; params never overlap). Unkeyed passives stay in PassiveTotals; keyed ones stay out.
