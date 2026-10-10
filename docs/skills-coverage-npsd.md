# Player skill coverage: Necromancer, Paladin, Sorceress, Druid

Generated from `d2common/d2skill/coverage_npsd.go` (`D2_WRITE_COVERAGE=<file> go test ./d2common/d2skill -run CoverageNPSD`).
Source of truth for the skill rows: the patch_d2 `skills.txt` (the export of the compiled `skills.bin`; `D2_TABLES=$HOME/git/d2-tables`).
`do` is the srvdofunc (server do table 0x72fa48 in `skills-2.md`); `-` is the generic missile path.

Status: **F** fully simulated, **P** partial (runs, a documented part is approximated or missing), **S** stub (only a state nothing reads), **M** missing (cast refused), **T** passive (no cast, feeds other skills).
The statuses come from reading the handlers and the scenario logs (`scripts/verify.d/86-class-skills.sh`), not from diffing every skill against the exe. `VERIFIED` notes were read in the decompilation.

| | F | P | S | M | T |
|---|---|---|---|---|---|
| before (start of feat/skills-npsd) | 59 | 42 | 3 | 8 | 8 |
| after | 68 | 44 | 0 | 0 | 8 |


## Necromancer

| id | skill | do | before | after | note |
|---|---|---|---|---|---|
| 66 | Amplify Damage | 30 | F | F | state with damageresist -100 on enemies in range |
| 67 | Teeth | 8 | F | F | missile fan (do 8) |
| 68 | Bone Armor | 18 | F | F | absorb state, Director HeroDefense hook |
| 69 | Skeleton Mastery | - | T | T | feeds the Raise Skeleton order (hp, damage) |
| 70 | Raise Skeleton | 31 | F | F | d2summon stats, roster limits |
| 71 | Dim Vision | 30 | S | F | before: state only; now forces monster AI state 10 (blind) |
| 72 | Weaken | 30 | F | F | state with damageresist / damage stats |
| 73 | Poison Dagger | 32 | F | F | melee + poison stream |
| 74 | Corpse Explosion | 55 | P | P | level scale vs corpse added (verified 0x5c2c60); radius halving (ln34+1)/2 and calc3 element not ported |
| 75 | Clay Golem | 56 | F | F | golem order, one at a time |
| 76 | Iron Maiden | 30 | F | F | reflect via afterHit |
| 77 | Terror | 30 | F | F | flee state |
| 78 | Bone Wall | 60 | P | P | wall pieces as blocking monsters; exe piece layout (SKILL_SpawnBonePrisonWalls offsets) approximated |
| 79 | Golem Mastery | - | T | T | feeds golem orders |
| 80 | Raise Skeletal Mage | 31 | P | P | raised, but the mage melees (no ranged attack AI) |
| 81 | Confuse | 61 | S | F | before: state only; now forces confuse (list 9, mode 3) |
| 82 | Life Tap | 30 | F | F | heal via afterHit |
| 83 | Poison Explosion | 63 | P | P | area poison hit instead of the poisonexplosioncloud sub-missile fan |
| 84 | Bone Spear | - | F | F | generic missile path |
| 85 | BloodGolem | 56 | P | P | golem; life-leech-to-owner behaviour not checked against the exe |
| 86 | Attract | 59 | S | F | before: state only; now forces attract (list 9, alignment 1) |
| 87 | Decrepify | 30 | F | F | state, slow and damage stats |
| 88 | Bone Prison | 62 | P | P | ring of 8 pieces (exe uses a 12 cell offset table 0x6e45e0/0x6e4610 and links pieces to a leader) |
| 89 | Summon Resist | - | T | T |  |
| 90 | IronGolem | 57 | M | P | do 57 added as a golem order; the consumed item and its derived stats are not simulated |
| 91 | Lower Resist | 30 | F | F | state with resist stats |
| 92 | Poison Nova | 22 | F | F | nova missile ring (do 22) |
| 93 | Bone Spirit | 10 | F | F | homing missile (do 10) |
| 94 | FireGolem | 56 | P | P | golem; the fire absorb / damage aura stats come from the order only |
| 95 | Revive | 58 | P | P | uses the corpse type; level cap + life scaling to caster level added (verified 0x5c35a0); life roll at the corpse level not ported |

## Paladin

| id | skill | do | before | after | note |
|---|---|---|---|---|---|
| 96 | Sacrifice | 64 | F | F | self damage + strike |
| 97 | Smite | 150 | P | P | shield bash damage; knockback/stun not simulated |
| 98 | Might | 65 | F | F | friendly aura (do 65): stats re-applied every pulse |
| 99 | Prayer | 65 | F | F | friendly aura (do 65): stats re-applied every pulse |
| 100 | Resist Fire | 65 | F | F | friendly aura (do 65): stats re-applied every pulse |
| 101 | Holy Bolt | - | P | P | bolt damages undead; healing of allies not simulated |
| 102 | Holy Fire | 66 | F | F | damage aura (do 66) |
| 103 | Thorns | 65 | F | F | friendly aura (do 65): stats re-applied every pulse |
| 104 | Defiance | 65 | F | F | friendly aura (do 65): stats re-applied every pulse |
| 105 | Resist Cold | 65 | F | F | friendly aura (do 65): stats re-applied every pulse |
| 106 | Zeal | 13 | F | F | multi hit |
| 107 | Charge | 67 | P | P | rush + strike, movement approximated |
| 108 | Blessed Aim | 65 | F | F | friendly aura (do 65): stats re-applied every pulse |
| 109 | Cleansing | 65 | F | F | friendly aura (do 65): stats re-applied every pulse |
| 110 | Resist Lightning | 65 | F | F | friendly aura (do 65): stats re-applied every pulse |
| 111 | Vengeance | 2 | F | F | elemental melee |
| 112 | Blessed Hammer | 73 | P | P | missile without the spiral movement |
| 113 | Concentration | 65 | F | F | friendly aura (do 65): stats re-applied every pulse |
| 114 | Holy Freeze | 81 | F | F | enemy aura, cold damage + slow |
| 115 | Vigor | 65 | F | F | friendly aura (do 65): stats re-applied every pulse |
| 116 | Conversion | 79 | M | P | do 79 added (roll, state, level/life scale, charm AI); boss exclusion U |
| 117 | Holy Shield | 18 | F | F | self state (do 18) |
| 118 | Holy Shock | 66 | F | F | damage aura |
| 119 | Sanctuary | 66 | P | P | damages all enemies, exe damages undead only and knocks back |
| 120 | Meditation | 65 | F | F | friendly aura (do 65): stats re-applied every pulse |
| 121 | Fist of the Heavens | 80 | P | P | bolt at the aim; the follow-up holy bolts are not simulated |
| 122 | Fanaticism | 65 | F | F | friendly aura (do 65): stats re-applied every pulse |
| 123 | Conviction | 66 | F | F | enemy resist aura |
| 124 | Redemption | 82 | P | P | corpse consumption heals; exe radius / cost rules approximated |
| 125 | Salvation | 65 | F | F | friendly aura (do 65): stats re-applied every pulse |

## Sorceress

| id | skill | do | before | after | note |
|---|---|---|---|---|---|
| 36 | Fire Bolt | - | F | F | generic missile path |
| 37 | Warmth | - | T | T |  |
| 38 | Charged Bolt | 17 | F | F | do 17 |
| 39 | Ice Bolt | - | F | F | generic missile path |
| 40 | Frozen Armor | 18 | F | F | do 18 self state with chill on melee attackers |
| 41 | Inferno | 19 | F | F | stream (do 19) |
| 42 | Static Field | 20 | F | F | do 20 |
| 43 | Telekinesis | 21 | M | P | do 21 added: damage on a targeted monster; item pickup and object use are not simulated |
| 44 | Frost Nova | 22 | F | F | do 22 |
| 45 | Ice Blast | - | F | F | generic missile path |
| 46 | Blaze | 23 | F | F |  |
| 47 | Fire Ball | - | F | F | generic missile path |
| 48 | Nova | 22 | F | F | do 22 |
| 49 | Lightning | - | F | F | generic missile path |
| 50 | Shiver Armor | 18 | F | F | do 18 self state with chill on melee attackers |
| 51 | Fire Wall | 24 | F | F |  |
| 52 | Enchant | 25 | F | F |  |
| 53 | Chain Lightning | 26 | F | F |  |
| 54 | Teleport | 27 | P | F | levels.txt Teleport flag (0 refuse, 2 no walls) added (verified 0x5c84d0) |
| 55 | Glacial Spike | - | F | F | generic missile path |
| 56 | Meteor | 28 | P | P | impact only; the fire trail of meteorcenter is not simulated |
| 57 | Thunder Storm | 29 | P | P | strikes the nearest enemy, no random targets |
| 58 | Energy Shield | 23 | P | P | mana absorb ratio passed unverified |
| 59 | Blizzard | 28 | P | P | random shards approximated |
| 60 | Chilling Armor | 18 | F | F | do 18 self state with chill on melee attackers |
| 61 | Fire Mastery | - | T | T |  |
| 62 | Hydra | 144 | M | P | do 144 added: 3 stationary fire shooters in the exe triangle; hydra AI timing U |
| 63 | Lightning Mastery | - | T | T |  |
| 64 | Frozen Orb | - | P | P | orb missile; the nova of bolts every few frames is a hit/Do detail not simulated |
| 65 | Cold Mastery | - | T | T |  |

## Druid

| id | skill | do | before | after | note |
|---|---|---|---|---|---|
| 221 | Raven | 114 | P | F | summon level (calc2) now set through the order; stats from d2summon + MonLvl AC/AR (verified 0x5c2850) |
| 222 | Plague Poppy | 115 | M | P | do 115 added: vine shooter with calc1 life, calc2 level |
| 223 | Wearwolf | 116 | P | P | timed form state; the form attack skills are not rebound |
| 224 | Shape Shifting | - | T | T |  |
| 225 | Firestorm | 117 | P | P | fire bursts in a line approximated |
| 226 | Oak Sage | 119 | P | P | totem aura as a state; stats follow the order |
| 227 | Summon Spirit Wolf | 119 | P | F | summon level (calc2) + MonLvl AC/AR now applied; resist / armor / damage stats from the order |
| 228 | Wearbear | 116 | P | P | timed form state |
| 229 | Molten Boulder | - | P | P | emerge missile; the rolling boulder chain is a hit/Do function not simulated |
| 230 | Arctic Blast | 19 | F | F | stream (do 19) |
| 231 | Cycle of Life | 115 | M | P | do 115 added (attack only; the corpse cycler heal is not simulated) |
| 232 | Feral Rage | 120 | P | P | stacking state with leech stats; exe leech not applied |
| 233 | Maul | 120 | P | P | stacking state, stun on hit |
| 234 | Eruption | 28 | P | P | Fissure: random strikes approximated |
| 235 | Cyclone Armor | 18 | F | F | do 18 |
| 236 | Heart of Wolverine | 119 | P | P | totem aura as a state |
| 237 | Summon Fenris | 119 | P | F | summon level (calc2) + MonLvl AC/AR now applied |
| 238 | Rabies | 121 | P | P | poison bite; the spreading contagion of the exe is not ported |
| 239 | Fire Claws | 2 | F | F | do 2 melee with fire |
| 240 | Twister | 118 | P | P | missile chain approximated |
| 241 | Vines | 115 | M | P | do 115 added (attack only; the vine cycler is not simulated) |
| 242 | Hunger | 122 | M | F | do 122 added: melee roll, calc1 damage percent, life and mana steal (verified 0x5c5f40) |
| 243 | Shock Wave | 8 | P | P | missile fan with stun |
| 244 | Volcano | 123 | P | P | random eruptions approximated |
| 245 | Tornado | 118 | P | P | missile without the wander path |
| 246 | Spirit of Barbs | 119 | P | P | totem aura as a state |
| 247 | Summon Grizzly | 119 | P | F | summon level (calc2) + MonLvl AC/AR now applied |
| 248 | Fury | 13 | F | F | multi hit (do 13) |
| 249 | Armageddon | 124 | P | P | random meteors approximated |
| 250 | Hurricane | 124 | P | P | area ticks |

## Not in this list

Fire Blast / Fire Trauma (and the other trap bombs) are Assassin skills, handled by the Amazon/Barbarian/Assassin branch.
