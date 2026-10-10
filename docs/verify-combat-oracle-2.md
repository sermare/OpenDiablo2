# Combat oracle, second round (Game.exe 1.14b in the emulator; observations only)

Generators: `~/git/drlg-oracle/dm_wep.py dm_leech.py dm_react.py dm_e2e.py dm_pierce.py dm_chain.py dm_coll.py dm_knock.py`
(unicorn, fake units, peripheral helpers hooked). Goldens are numbers only (`d2common/d2combat/testdata/*_golden.json`,
`d2common/d2missile/testdata/*`). All Go ports are compared case by case, including the generator state.

## Weapon roll 0x579120 (RollPhysical)
min/max in 8.8 from stats 0x15/0x16 (0x17/0x18 for hand mode 2; 1..2 bare handed, min 1 / max 2 floors), + stat 0x6f*256 to both; min < 1 -> 256,
max <= min -> min + 256. Percent = a4 (struct pct) + stat 0x19 + weapon strength and dexterity bonus (stat * bonus / 100) + mastery (0x646bc0);
unarmed adds the strength stat as percent. Floored at -90. Low end scaled by (stat 0x12 + pct), high end by (stat 0x11 + pct); result =
lo + a5 + Roll(hi - lo); floored at 0; scaled by a6/128 unless 0x80. Called from 0x5794e0 with a4 = struct pct (demon/undead bonuses), a5 = struct physical.
Ethereal and per item enhanced damage are inside the item's min/max/stat values (item side, not in this function).

## Leech 0x57a3b0
See ApplyLeech. Players and special monsters: mana, then life, each MulDiv(physical, leech, 100) scaled by the monster's Drain percent,
x64 fixed point, divided by the difficulty row's divisors (player attackers); after a mana and a life transfer one Roll(2) on the attacker picks
the sound event 0x97/0x98. Plain monsters: life + mana + stamina leech, each capped by what the defender has, heal the attacker's life only.
0x57a650 then burns the defender's mana/stamina by the (shifted) struct values.

## Reaction 0x57ae50 and stagger 0x57aa60
ReactionEffects reproduces every call the function makes (names in reaction.go). Player: dodge/avoid/evade need their state and skill lookups,
block rate limited by frames since stat 0x5f (> 15 + fbr/8), death needs mode not 0/0x11, knockback mode 0x13, hit recovery gated by
StaggerGate unless state 0x15. Monster: death, knockback (0x8 -> 0x4 when the class lacks mode 0xd), block (mode 6, else mode 0x13), hit (mode 3),
0x4000 flag only, hp bar change > 4. The monster engine (d2monsters.damage) now uses the stagger gate.

## Melee pipeline
BuildAttackerDamage (now with the real weapon roll, weapon strike chance, undead blunt bonus) -> ResolveStruct (0x579d70 scale percent 17/25/boss
row/200/400, 0x579e50 freeze/stun/state tweaks, 12 descriptor rows, flat reduction, absorb, Total) -> ApplyHit (0x57a650 with flag 0: physical capped
at life, leech, absorb heal, Total subtracted with a floor to 0 below 1.0 life, mana/stamina burn, the five state requests in order stun, chill,
freeze, poison, burn, death bit) -> open wounds, crushing blow callbacks. Monster targets with class flag 0xf clear (0x452b20) take no damage.
Unmodelled: item event dispatcher 0x5be860 (order wounds before crush assumed), 0x5a2f90 monster double, 0x5793c0 class bonus list.

## Missiles
Pierce, chain, fury and acceptance: see the section in missile-damage-oracle.md. Wall collision stays as in verify-missiles-2.md.

## Engine wiring done
d2monsters: monster hit recovery uses StaggerGate; hero melee hits leech life and mana (new). d2missile: PierceCharges, hit functions 12 and 20.
d2skill: Chain Lightning is cast with the bolt count and jumps through hit function 12 (unit id order). Not wired: the full ApplyHit pipeline,
knockback motion (RollKnockback is ready), RollPhysical into the hero damage (the totals are pre-summed).
