# Missile / damage oracle (Game.exe 1.14b run in an emulator)

Goldens under `d2common/d2combat/testdata/` hold numbers only. They are produced by running the real functions of Game.exe
under an x86 emulator (unicorn) with fake units (stat reads served from tables, heavy helpers hooked). The generator scripts
(`dm_res.py`, `dm_cb.py`, `dm_bld.py`, `dm_mis.py`, `dm_state.py`, `dm_mel.py`, and, second round, `dm_wep.py`, `dm_leech.py`, `dm_react.py`, `dm_e2e.py`, `dm_pierce.py`, `dm_chain.py`, `dm_coll.py`, `dm_knock.py`) live next to the other oracle tools in
`~/git/drlg-oracle/` (outside this repo, they need the game install). Tests: `d2common/d2combat/*_oracle_test.go`.

| Golden | Real function(s) | Go | Cases | Result |
|---|---|---|---|---|
| dmg_res | 0x579c90 per-type resist + flat + absorb (0x579b10, 0x579c20), MulDiv 0x47f2c0 | `ResolveComponent`, `MulDiv`, `Absorb` | 3000 | equal |
| dmg_cb (crush) | 0x5bdbf0 crushing blow callback | `RollCrushingBlow`, `CrushingDivisor`, `PlayerCountBonus` | 2500 | equal, incl. RNG state |
| dmg_cb (wounds) | 0x5bda80 open wounds callback (+0x5bd9c0) | `RollOpenWounds`, `OpenWoundsBase` | 2500 | equal |
| dmg_bld | 0x5a63c0 missile damage struct builder, 0x5a6690 physical case | `BuildMissileDamage`, `BuildMissilePhysical` | 4000 | equal, incl. RNG state |
| missile_create | 0x59d5d0 velocity / lifetime / accel / maxvel | `MissileCreateTiming` | 6000 (about 5000 created) | equal |
| states | stun 0x578830, poison 0x578990, burn 0x578b00, chill 0x578ca0, freeze 0x578f50 | `StateUnit.Apply*` | 3000 sequences | equal (ends, stats, bits, RNG) |
| melee | 0x5794e0 attacker damage (elementals, poison, burn, leech, stun, conversion) | `BuildAttackerDamage` | 3000 | equal |
| wep | 0x579120 weapon physical roll | `RollPhysical` | 5000 | equal, incl. RNG state |
| leech | 0x57a3b0 leech transfer | `ApplyLeech` | 4000 | equal, incl. RNG state and events |
| react | 0x57ae50 reaction (effect trace), 0x57aa60 stagger gate | `ReactionEffects`, `StaggerGate` | 6000 + 6000 | equal, incl. RNG state |
| e2e | 0x5794e0 + 0x579120, 0x579ef0, 0x57a650 (flag 0, flags 0x20), crushing blow, open wounds | `BuildAttackerDamage`, `ResolveStruct`, `ApplyHit`, `RollCrushingBlowExe`, `RollOpenWoundsExe` | 1500 | equal at every stage, final life/mana/stamina of both units |
| pierce | 0x59d4e0 creation roll, 0x5ab550 spend | `RollPierceCharges`, `SpendPierce` | 3000 + 2000 | equal |
| chain | target pick callback 0x569a40 (Chain Lightning hit function 12) | `d2missile.ChainNext` | 4000 | equal |
| collide | 0x5a6160 common predicate, 0x5a61d0 / 0x5a6210 / 0x5a6270 | `d2missile.AcceptCommon`, `AcceptTyped` | 10000 | equal |
| knock | 0x5bd580 knockback event | `RollKnockback` | 4000 | equal, incl. RNG state |

## Findings (observations)
- MulDiv (0x47f2c0) truncates toward zero; it is not Win32 rounding. Large operands (a > 0x100000 or b > 0x10000) use other paths.
- Absorb percent is capped at 40; flat absorb is stat*256 limited to what is left; a flat reduction larger than the damage leaves a NEGATIVE component (not floored); absorb and flat are skipped in "ignore" mode.
- Missile builder rolls in the order physical, fire, magic, lightning, cold, poison, burn; a component with min <= 0 or max <= 0 is 0 and consumes no step; min > max swaps; the mastery percent scales both ends. The percent bonus (stat 0x19, +0x79 demons, +0x7a undead, floored at -90) only applies when a target is given; deadly strike (stat 0x8d on the missile) doubles physical after it.
- Poison length is divided by the poison count stat 0x146 when it is greater than 1.
- Missile create: velocity = ((VelLev*level)/8 + Vel) << 8 (or the given value, shifted unless flag 0x10), scaled by the slow stat 0xa1 when the record can slow and the owner has state 0x57, then x75/100; lifetime = LevRange*level + Range (+(SubStop-SubStart)*loops with flag 8), minus Param40 with flag 0x200, the 0x64b850 value is that minus Param44 (flag 0x800) or record byte +0x136; lob (0x400): life2 = (dist<<16)/(vel<<4).
- Timed states: ONE list per unit and state. Poison/burn replace (end and value) only when the new per-tick value is >= the old one; poison and burn are separate lists. Stun is capped at 250 frames, special monster classes at 13, a live stun's end is overwritten (shorter replaces longer); for monsters with data flag 8 the roll Roll(100) < 90 on the ATTACKER's generator makes the stun fail (the earlier notes said "10 percent skip": it is 90 percent skip, 10 percent proceed). Chill/freeze only extend; chill length is divided by the difficulty divisor (min 1), freeze length by its divisor (no minimum), chill always consumes one roll on the TARGET's generator afterwards.
- Melee builder: burn damage becomes burn + (burn*scale + 0x13c + Roll(1))/128, so every hit adds about 2 (8.8 units) and consumes one RNG step even with no burn stats; element conversion classes 1..5 and 11/12 (10 = random 1..5), poison part /8, lengths floored at 50 frames.
- Crushing blow divisor: players and "special" classes 10, bosses/flag-2 monsters 8, others 4, plus the player count bonus for non-special monsters, doubled by the missile event; physical resist uses the RAW stat 36.

## Second round findings (see also docs/verify-combat-oracle-2.md)
- Pierce charges use a PRIVATE generator seeded (owner's stat 0x148 = 0, 0x29a), not the owner's: the four attempts compare 66, 70, ... against the chance, so a chance of 66 or less never gives a charge (d2missile.PierceCharges was rolling the owner's generator).
- Chain Lightning: the cast stores calc1 in the bolt's data field 0x28; hit function 12 spawns the next bolt while the field is >= 2, with field - 1, from the BOLT's position, at the candidate with the smallest unit id above the one just hit (wrapping to the smallest): not the nearest. Lightning Fury (hit function 20) sends one HitSubMissile1 bolt at each of up to calc1 enemies within aurarangecalc of the missile.
- Damage struct fields: +8 physical (leech is a percent of it, capped at the defender's life), +0x4c Total; the Total adds +0x38 (the leech field) only for a PLAIN MONSTER attacker (d2combat.Damage.SumTotal took the defender kind: fixed).
- Leech: players/special attackers: life/mana leech percent of physical damage (x64 fixed point, divided by the difficulty steal divisors and the monster's Drain column); the same ManaLeech/StaminaLeech struct value is also BURNED from the defender's mana/stamina (in 8.8 units, after the x64 shift). Monsters heal from life+mana+stamina capped by what the defender has.
- Knockback: stat > 0 only enables the roll; chance is 64/128 (32/128 or 128/128 for monster records with byte +5 bits 8 / 4).
- Hit recovery: StaggerGate uses masks 1 and 3 of the defender's own generator (new low word); the reaction effect trace now matches 12000 cases. Monsters with a block/knockback fall through to mode 0x13 ("setmode") when the animation is missing.
- Undead damage: +50 percent for a wielded weapon of item type 0x39 plus stat 0x7a (0x579380).

## Not covered (still unverified)
- The weapon strike chance and mastery bonus helper 0x646bc0 (inputs only, per weapon item data); the per monster class bonus list (stat 0xb4, helper 0x5793c0); the order in which the attacker's item events (open wounds before crushing blow) are dispatched by 0x5be860 is ASSUMED.
- The monster double (0x5a2f90, ported earlier without an oracle), the event dispatcher 0x5be860 itself, wall collision (verify-missiles-2 only), the unit scan 0x569510 filter (line of sight).
- The engine (d2core) is not rewired to the new pure functions except d2state (poison/burn replace, stun cap, chill/freeze extend) and d2missile.DamageDesc.Roll (exe order and zero rules): crushing blow, open wounds, ResolveComponent with flat/absorb and BuildAttackerDamage are available but unused by the engine.
