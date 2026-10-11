# Amazon, Barbarian and Assassin skill coverage

Skills of the three classes (30 each) against what the skill engine (`d2common/d2skill` pipeline, `d2core/d2skills` engine, `d2common/d2missile` simulation) does. Skill rows come from the 1.14 patch_d2 skills table (`D2_TABLES/skills/patch_d2/skills.txt`, the decompiled form of the compiled `.bin` that the exe reads; the old `d2exp/skills.bin` has a different layout and is not used). srvst/srvdo/clt* are the function ids of the exe tables (0x72f8d8 / 0x72fa48 / 0x726898 / 0x7269b8, see `skills-2.md` of the reverse engineering notes).

Status words: **fully simulated** (a ported function plus the missiles/states it needs, the rules read from the exe or documented), **partial** (runs, but the listed part is not simulated or is inferred "U"), **stub** (accepted but only a placeholder effect), **missing** (no do function, `Implemented` false). The status is from reading the handlers and the missile closure of each skill, not from an exhaustive per-skill exe audit; passives count as simulated when their stats reach the combat code.

| | before | after |
|---|---|---|
| fully simulated | 64 | 69 |
| partial | 17 | 21 |
| stub | 1 | 0 |
| missing | 8 | 0 |
| total | 90 | 90 |

## Amazon

| id | skill | st/do | clt st/do | before | after | notes |
|---|---|---|---|---|---|---|
| 6 | Magic Arrow | -/- | -/- | fully simulated | fully simulated |  |
| 7 | Fire Arrow | 4/- | 11/- | fully simulated | fully simulated |  |
| 8 | Inner Sight | -/6 | -/- | fully simulated | fully simulated |  |
| 9 | Critical Strike | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 10 | Jab | 5/7 | 12/16 | fully simulated | fully simulated |  |
| 11 | Cold Arrow | 4/- | 11/- | fully simulated | fully simulated |  |
| 12 | Multiple Shot | 4/8 | 11/17 | fully simulated | fully simulated |  |
| 13 | Dodge | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 14 | Power Strike | 6/2 | -/- | fully simulated | fully simulated |  |
| 15 | Poison Javelin | 4/- | 11/- | partial | partial | poison over time and the javelin work; the lingering poison cloud (poisonjavcloud, missile do function 3) is not simulated |
| 16 | Exploding Arrow | 4/- | 11/- | fully simulated | fully simulated |  |
| 17 | Slow Missiles | -/6 | -/- | fully simulated | fully simulated |  |
| 18 | Avoid | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 19 | Impale | 7/2 | -/- | fully simulated | fully simulated |  |
| 20 | Lightning Bolt | 4/- | 11/- | fully simulated | fully simulated |  |
| 21 | Ice Arrow | 4/- | 11/- | fully simulated | fully simulated |  |
| 22 | Guided Arrow | 4/10 | 11/18 | fully simulated | fully simulated |  |
| 23 | Penetrate | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 24 | Charged Strike | 6/11 | -/19 | fully simulated | fully simulated |  |
| 25 | Plague Javelin | 4/- | 11/- | partial | partial | javelin and poison work; the plague cloud (plaguejavcloud, do function 3) is not simulated |
| 26 | Strafe | 8/12 | 13/20 | fully simulated | fully simulated |  |
| 27 | Immolation Arrow | 4/- | 11/- | partial | partial | arrow and explosion work; the burning ground (immolationfire, do function 5, hit function 9) is not simulated |
| 28 | Dopplezon | -/15 | -/- | missing | partial | summon with life = calc3 percent of the owner, lifetime calc2 frames (SRVDO_015 0x5dad10); the decoy AI that copies the owner is the monster AI of "dopplezon", not checked |
| 29 | Evade | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 30 | Fend | 9/13 | 14/21 | fully simulated | fully simulated |  |
| 31 | Freezing Arrow | 4/- | 11/- | fully simulated | fully simulated |  |
| 32 | Valkyrie | -/16 | -/- | missing | partial | summon (SRVDO_016 0x5daf10, player only, petmax 1, life bonus calc1); the pet's Dodge/Avoid/Evade/Critical Strike monster skills (sumskill1..4) are not wired |
| 33 | Pierce | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 34 | Lightning Strike | 10/14 | -/22 | fully simulated | fully simulated |  |
| 35 | Lightning Fury | 4/- | 11/- | partial | partial | missile flies; its hit function 20 (periodic effect) is not in the missile simulation |

## Barbarian

| id | skill | st/do | clt st/do | before | after | notes |
|---|---|---|---|---|---|---|
| 126 | Bash | 32/2 | -/- | fully simulated | fully simulated |  |
| 127 | Sword Mastery | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 128 | Axe Mastery | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 129 | Mace Mastery | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 130 | Howl | -/22 | -/25 | fully simulated | fully simulated |  |
| 131 | Find Potion | 33/69 | 26/38 | missing | fully simulated | SRVDO_069 0x5d6c40 + the act/difficulty potion table 0x73e988 (golden test) |
| 132 | Leap | 40/77 | 29/43 | partial | fully simulated | VERIFIED SRVDO_077 0x5d8e60: a landing player throws back everything hostile within calc1 (result flags 9, no damage; the radius unit and centre are U); the jump itself is still an instant move (no arc, clamped to 18 subtiles) |
| 133 | Double Swing | -/70 | 27/39 | fully simulated | fully simulated |  |
| 134 | Pole Arm Mastery | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 135 | Throwing Mastery | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 136 | Spear Mastery | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 137 | Taunt | -/71 | -/- | partial | partial | curse-style state with a synthetic stat; AI reaction is U |
| 138 | Shout | -/68 | -/25 | fully simulated | fully simulated |  |
| 139 | Stun | 32/2 | -/- | fully simulated | fully simulated |  |
| 140 | Double Throw | -/74 | 11/42 | fully simulated | fully simulated |  |
| 141 | Increased Stamina | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 142 | Find Item | 34/72 | 28/40 | missing | fully simulated | SRVDO_FindItem 0x5d7210: bucket roll over Param1..4 picks monstats TreasureClass1..4 and drops it at the corpse |
| 143 | Leap Attack | 41/78 | 30/44 | partial | fully simulated | VERIFIED SRVDO_078 0x5d9320 / 0x5d9170 / 0x5d8f90: ONE victim (the target in melee range, else the nearest in a scan), to-hit roll, calc1 percent, knockback bit, bash overlay, the victim's stun removed (skills-batch3.md) |
| 144 | Concentrate | 32/2 | -/- | fully simulated | fully simulated |  |
| 145 | Iron Skin | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 146 | Battle Cry | -/68 | -/25 | fully simulated | fully simulated |  |
| 147 | Frenzy | -/9 | -/39 | fully simulated | fully simulated |  |
| 148 | Increased Speed | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 149 | Battle Orders | -/68 | -/25 | fully simulated | fully simulated |  |
| 150 | Grim Ward | 33/75 | 26/41 | missing | partial | corpse consumed, terror state on enemies within aurarangecalc every 6 frames for 200 frames (SRVDO_075 0x5d7430, missile table values); the small/medium/large variant by corpse size is not modelled |
| 151 | Whirlwind | 38/76 | 31/45 | missing | partial | hero walks to the point one subtile per frame hitting the nearest other enemy every weapon-delay frames (SRVST_038/SRVDO_076, 0x5d7a00/0x5d8010); walk speed and the scan radius unit are U |
| 152 | Berserk | 39/2 | -/- | fully simulated | fully simulated |  |
| 153 | Natural Resistance | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 154 | War Cry | -/68 | -/25 | fully simulated | fully simulated |  |
| 155 | Battle Command | -/68 | -/25 | fully simulated | fully simulated |  |

## Assassin

| id | skill | st/do | clt st/do | before | after | notes |
|---|---|---|---|---|---|---|
| 251 | Fire Trauma | -/- | -/- | fully simulated | fully simulated |  |
| 252 | Claw Mastery | -/- | -/- | fully simulated | fully simulated | passive stats (PassiveTotals / mastery / defense hooks) |
| 253 | Psychic Hammer | 22/33 | 5/3 | partial | partial | fixed radius 3 area hit; the knockback missile is not simulated (srvdo 33 is U) |
| 254 | Tiger Strike | 23/34 | -/- | fully simulated | fully simulated |  |
| 255 | Dragon Talon | 24/42 | 6/4 | partial | partial | kick count and charge release are U (srvdo 42) |
| 256 | Shock Field | -/43 | -/5 | fully simulated | fully simulated |  |
| 257 | Blade Sentinel | -/44 | -/- | partial | partial | summon works; the creeper missile (do function 20, hit function 37) path is not simulated |
| 258 | Quickness | -/18 | -/- | fully simulated | fully simulated |  |
| 259 | Fists of Fire | 23/35 | -/- | fully simulated | fully simulated |  |
| 260 | Dragon Claw | 25/46 | -/- | partial | fully simulated | VERIFIED SRVDO_046 0x5d4cf0 / 0x5d4ba0: one blow per action frame (two per cast here), to-hit bonus plus stat 0x145, calc1 damage bonus, charges released |
| 261 | Charged Bolt Sentry | -/45 | -/- | fully simulated | fully simulated |  |
| 262 | Wake of Fire Sentry | -/45 | -/- | partial | partial | sentry works; wake of destruction maker (do function 31) is not simulated |
| 263 | Weapon Block | -/- | -/- | partial | partial | passive stat is keyed to the weapon type, so it is left out of PassiveTotals; the block rule lives in d2combat/avoid.go |
| 264 | Cloak of Shadows | -/47 | 7/- | partial | partial | timed state that ends on hit; the blinding of nearby enemies is not simulated |
| 265 | Cobra Strike | 23/34 | -/- | fully simulated | fully simulated |  |
| 266 | Blade Fury | 26/48 | 8/6 | missing | partial | one bladefragment1 per do, repeat gate prgcalc1-1 frames (SRVDO_048 0x5d5260); the fragment flight/bounce rules of the missile are the generic ones |
| 267 | Fade | -/18 | -/- | fully simulated | fully simulated |  |
| 268 | Shadow Warrior | -/49 | -/- | fully simulated | fully simulated |  |
| 269 | Claws of Thunder | 23/35 | -/- | fully simulated | fully simulated |  |
| 270 | Dragon Tail | 27/50 | 9/7 | partial | fully simulated | VERIFIED SRVDO_050 0x5d5b90: the kick is the plain kick; the explosion is kick damage x (calc1 + fire mastery) percent in radius par3 around the target with the knockback bit (the damage field, +0x10 = fire, is U) |
| 271 | Lightning Sentry | -/45 | -/- | fully simulated | fully simulated |  |
| 272 | Inferno Sentry | -/45 | -/- | fully simulated | fully simulated |  |
| 273 | Mind Blast | -/51 | -/8 | partial | partial | area damage; stun/convert of the real function (srvdo 51) is U |
| 274 | Blades of Ice | 23/35 | -/- | fully simulated | fully simulated |  |
| 275 | Dragon Flight | 12/52 | 5/- | missing | fully simulated | VERIFIED SRVDO_052 0x5d6290: needs a target unit; the branch is the animation frame (first: teleport when levels.txt Teleport != 0, 2 refuses a blocked line; second: the kick with the skill bonus + stat 0x145 and the charge release); the port lands next to the target (the exe goes to its position) |
| 276 | Death Sentry | -/45 | -/- | fully simulated | fully simulated |  |
| 277 | Blade Shield | 28/54 | -/- | stub | partial | was a bare self state; now hurts enemies within par4 every perdelay frames while it lasts (SRVDO_054 0x5d6880 only starts the periodic effect; attachment missile hit rule U) |
| 278 | Venom | -/18 | -/- | fully simulated | fully simulated |  |
| 279 | Shadow Master | -/49 | -/- | fully simulated | fully simulated |  |
| 280 | Royal Strike | 23/34 | -/- | partial | partial | charge strike works; the chaos ice missile (do function 35) is not simulated |

## What was added

`d2common/d2skill/class_abc.go` registers do functions 15, 16, 48, 52, 69, 72, 75, 76 and replaces 54; `d2core/d2skills/effects_abc.go` applies the new effect kinds (`loot`, `ward`, `whirl`, `shield`); `d2core/d2monsters/findloot.go` drops a corpse's treasure class column or a potion code. Unit tests: `d2common/d2skill/class_abc_test.go`, `d2core/d2skills/effects_abc_test.go`. In-game check: `scripts/verify.d/86-class-skills.sh` (log lines `SUMMON`, `MISSILE create name=bladefragment1`, `LOOT`, `WARD`, `WHIRL`, `SHIELD`).

Numbers taken from the exe for this work: the potion table at 0x73e988 (15 entries of 3 item codes, index = act - 1 + 5 * difficulty), the Find Item bucket rule, the Whirlwind step delay by weapon frame count (0x5d7de0), the eight Blade Fury hit-sub offsets of hit function 52 (not used: Blade Fury throws bladefragment1 directly). No emulator golden was made: the drlg-oracle harness only runs the level generator, so the table golden was read straight from the memory of the exe (read-only Ghidra).
