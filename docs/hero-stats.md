# Hero stats from equipment (d2common/d2statlist)

The hero's equipment, inventory charms, socketed items and set bonuses now
affect life, mana, stamina, defense, attack rating, damage, resistances and
block, and combat against monsters uses those values.

## Where things live

| Piece | Package |
|---|---|
| Pure stat list, aggregation, derivations, property expansion, set/gem tables | `d2common/d2statlist` |
| .d2s item -> stat item adapter, `RecalcStats`, `StatsSummary` | `d2core/d2hero` (`stat_items.go`, `hero_totals.go`) |
| Character panel values | `d2game/d2player/hero_stats_panel.go` |
| Monster vs hero and hero vs monster combat inputs | `d2core/d2monsters/combat.go` |
| In-game scenario | `scripts/verify.d/89-hero-stats.sh` (`panel:character` logs `PANEL character: ...`) |

## The finding about the saved maxima

A .d2s stores life, mana and stamina maxima WITHOUT item bonuses. For the real
level 94 Sorceress (`nokkasorc`): stored max 869 / 221 / 525, stored current
1241 / 464 / 801. The class formula gives 849 / 221 / 525 (the missing +20 life
is the Act 3 Potion of Life quest reward, kept as `LifeBonus`). Adding the
equipped items and the charms in the inventory page gives totals of
1241 / 477 / 801: life and stamina equal the saved current values exactly (the
hero was at full life and stamina), mana 464 is below its total 477 (it was not
full). `TestRealSaveTotalsMatchStoredCurrentValues` checks this.

`HeroStatsState.MaxHealth/MaxMana/MaxStamina` are now the totals; the item-free
values are `BaseMax*`, and `ExportD2S` writes those back so a save does not
double count items on re-import.

## Formulas

Verified = read from the binary (skills-combat.md / itemgen.md) or reproduced
with the real save. Unverified = community documentation or inference.

Verified:
- Maxima: `life = hpadd + initVit + (lvl-1)*lifePerLevel/4 + (vit-initVit)*lifePerVit/4`,
  same shape for mana (`initEne`, `manaPerEne`) and stamina; items add `vitality
  bonus * perVit/4` (ItemStatCost op 9), `energy bonus * manaPerEne/4` (op 8),
  flat life/mana/stamina, then life and mana percent (stats 76, 77, op 11).
  Reproduces the real save to the point; starting life of all seven classes.
- Defense `= (armor + dex/4)` (`d2combat.Defense`); attack rating
  `toHit + (dex-7)*5 + class ToHitFactor`, plus the AR percent (stat 119);
  block `(toblock + class BlockFactor) * (dex-15) / (2*lvl)` capped at 75;
  resistance cap `75 + max resist`, at most 95, difficulty penalty applied
  before the cap, physical cap 50; deadly strike and critical strike are
  independent chances that double physical damage; to-hit chance in [5,95].
- Per-level stats (`stat * clvl >> param`) and their targets are checked
  against the real ItemStatCost by `TestRealTables`.
- Stat ids, Save Add handling (the .d2s reader already subtracts it), skill
  tab parameter `class*8 + tab` (seen in the real save).

Unverified (community knowledge or inferred; marked in code):
- Enhanced defense multiplies the base defense only; flat `+defense` is not
  multiplied.
- The stored defense of ethereal armor already contains the +50%; for weapons
  the +50% is applied to the table damage (the reference reader does so).
- Weapon damage `(base*(100+ED)/100 + added) * (100 + str*strbonus/100 + ...)/100`.
- Set tiers: the item's list `i` is active with at least `i+2` pieces; the
  Sets.txt partial/full bonuses (`SetDefs`) are only used for generated items.
- Broken items (durability 0) contribute nothing.
- Order of flat damage reduction and physical resist; block while running
  (the /3 rule is not applied); the weapon switch slots (11, 12) are inactive
  (the active weapon set is not read from the save).
- Difficulty resist penalty -40/-100 (LoD DifficultyLevels).
- Property functions 8-24 meanings (`PropertyTable.Expand`); the handlers were
  not decompiled.

Not done: time-of-day (`bytime`) stats, avoid/dodge/evade in monster attacks,
elemental damage and leech in the hero's attacks, IAS/FCR breakpoint tables
(the raw percentages are reported), requirement checks for equipping, generated
items in the UI (they have no property list yet; only imported saves and base
items contribute), crushing blow and open wounds effects.

## Natural regeneration (exe observations, `d2herostats/regen.go`)

The server runs one vitals event per unit per game frame (event type 3, handler
0x57e800, which re-schedules itself for frame+1; the frame rate is the value at
0x72ee74, initially 25). For a living unit it runs the life routine 0x57e600, the
stamina routine 0x57e4f0 and the mana routine 0x57e6e0, in 1/256 ("raw") units:

- mana per frame = max(1, maxRaw / (charstats ManaRegen * 25)) * (100 + stat 27) / 100
  + stat 26; the first part is skipped under state 85 (NOMANAREGEN); the sum is
  clamped to [0, max]. ManaRegen is 120 for every class, so a full pool is back
  in 120 s (a 477 mana sorceress gets 40 raw per frame, 3.9 mana per second).
  Stat 27 is "regenerate mana %" (property regen-mana, the mana shrine), stat 26
  the flat per-frame amount of the mana potion states.
- life per frame = stat 74 (replenish life) raw, capped at the maximum and never
  below 1 point; with no stat there is no natural life regeneration (10 points
  of replenish life are 0.98 life per second).

Unverified: the exact fractional part of the maximum (the exe's maxima are raw,
ours are whole points), state 85 is not tracked, the sources of stat 74 other
than worn items (potions use their own regeneration in `d2inventory`).
