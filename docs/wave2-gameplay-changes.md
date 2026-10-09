# Wave 2: gameplay-behaviour changes for owner review

Scope: every commit on the wave-2 `fork/feat/*` branches (versus `fork/integration`; `fork/integration-s2` does not exist on the fork) that changes what the game does. Pure tests, oracles, fuzzers, race fixes and docs are listed at the end only.
Sources: commit messages, code comments and tests of those commits, and `~/git/d2-re-notes/*.md`. Nothing here was run or re-verified for this document.

Refreshed against `integration-s9-rehearsal` (96a9c5d4): rows whose basis was later verified in the exe, or whose statement was corrected, are updated in place and marked "(Refreshed)". New gameplay changes since the first version are in [Added since the first version](#added-since-the-first-version). Verified rules per area are in [verification-status.md](verification-status.md); several verifications live on fork branches that this tree does not contain (listed there).

Notes on attribution: several commits are reachable from more than one branch (for example `feat/wire-states` carries the earlier oracle commits). Each commit is listed once, under the branch that `git log --source` reports first. "Revert" means `git revert <commit>`; no change below has a runtime flag unless stated.

## Review first

Highest risk, or a basis that is unverified or only partly verified.

| # | Branch | Commit | What changes for the player | Basis | Risk | Off / revert |
|---|---|---|---|---|---|---|
| 1 | feat/wire-summons | afd349a0 | Necromancer/Druid/Assassin summons are now planned by `Roster.Plan`, evicted per pettype limit, spawned with computed stats, expire on time, and run the ported NecroPet/Raven AI each tick. | NecroPet leash/engage constants verified; Raven, Vines, CycleOfLife and AssassinSentry AIs are "simplified, marked UNVERIFIED" (b23dd5a3). | high | Revert afd349a0 (the skill-effect hook now calls Summoner). |
| 2 | feat/summon-completion | d368fa9a | Summon skills' passive stats are now passed into minion stats (were nil). | Evaluates `passivestat` columns; no exe address given. | med | Revert d368fa9a. |
| 3 | feat/reconcile-resist | 2d13da6a, 5f3dfd8a | Monsters no longer capped at 75% resist, so 100% immune monsters take no damage and cannot be chilled. Attacker pierce is applied. Non-expansion heroes get classic -20/-50 resist penalties. Absorb percent capped at 40. | Monster ignore flag, pierce 333-336, MulDiv, absorb cap verified in Game.exe. (Refreshed) Item pierce stats 305-308 are never read by the exe's effective resist (`verify-resist.md`: the descriptor has one pierce stat per type, 333-336) and have no Op in ItemStatCost, so they do not add; their consumer elsewhere is unknown. | high | Revert 2d13da6a and 5f3dfd8a. |
| 4 | feat/wire-hit-resolution | 599907da | Avoided hits (dodge/avoid/evade) become real misses, rolled before damage. A running hero is auto-hit and consumes no to-hit seed step. Crushing blow and open wounds now run on melee skills and physical missiles. | Verified per verify-hit-resolution.md. (Refreshed) State 0x2f is States.txt row 47 `sanctuary`, 0x3e is row 62 `openwounds`, 0x15 is row 21 `stunned`. Open-wounds life drain per tick is still NOT applied (unit of stat 0x4a UNVERIFIED). The exe order is flat reduction, percent resist, absorb per type, with the pvp / merc scale before the loop. | high | Revert 599907da. |
| 5 | feat/wire-states | 0911cdab, cc66af98, 66f3193a | Monster chill/freeze lengths divided by difficulty divisors (Nightmare 2, Hell 4). Chill uses the monster's own ColdEffect instead of fixed -50 and no longer triples attack speed. Stun capped at 250. Poison/burn: one stream per kind. One curse at a time. Static field and DoT kills clear states. | State rules verified against the exe (docs/verify-states.md). Rules marked "U" are listed in d2state/doc.go. `ColorShiftLocal` has no caller. (Refreshed) Still unverified: removal of states on an area change (the exe has no clear-on-area caller; 0x0063b0d0 has only the two death callers), the meaning of state 0x39 and of monster data flag16 / flag40, the 13-frame boss stun cut (not modelled). | high | Revert the three commits. |
| 6 | feat/verify-xp-gain | 7a53477f | Kill XP now scales by level difference: 6+ below the hero gives reduced XP (207..13 /256); hero below 25 killing 6+ above gives reduced XP (225..5 /256); hero 25+ killing higher gives xp*clvl/mlvl; level 99 gets 0 XP. | Tables 0x6e2960 and 0x6e298c read from the exe. (Refreshed) The credit routine 0x0057c990, party split 0x0057c6b0 and merc rules (86/256 when the merc did not make the kill) are also verified. In this tree only the level scaling, the 0x7fffff clamp, the level-99 cap and the merc xp gate are wired; party split, merc share, item +% experience (stat 85) and the unknown-level clamp are on `fork/feat/wire-party-xp` (4722afa3), not here. ExpRatio is applied 1:1 (column missing from the extracted Experience.txt). The recipient alive / distance tests are UNVERIFIED as to tile or subtile. | med | Revert 7a53477f. |
| 7 | feat/wire-inventory | d0ea5bf0 | Stackables merge when dropped onto the same base item (surplus stays on cursor). Gold pickup capped at level*10000, overflow left on the ground. The .d2s loader zeroes carried gold above level*10000. | Gold caps, stack merge, belt tables verified (4160d52c). (Refreshed) The stash gold cap (2,500,000) is now applied on .d2s load and export (0a0a6b40); the gold pickup overflow flow is extracted and tested (cfd28df3); the extra-stack stat is looked up by id 254 (bc9918c6). Still UNVERIFIED: belt column +0x131 vs +0x130 for potions and the first-empty-cell fallback left ON. Verified in the exe: auto-place has no first-fit fallback when the scored search fails, and the 1.14b server does nothing for the unstack packet 0x22. | high (touches saves) | Revert d0ea5bf0. |
| 8 | feat/wire-monster-scaling | c9f845cc | Monster level resolves through d2monstats (difficulty above Normal, expansion, not noRatio/boss, known area). HP/XP player-count bonus; HP cap 0x7fffff; truncating MulDiv. | Verified: boss flag bit 6, player-count bonus 0x00571760, MulDiv truncates, HP cap (597f6599, 11b58e54). `Classic` column choice (plain vs L-HP) is UNVERIFIED, L- columns default. | high | Revert c9f845cc. |
| 9 | feat/players-and-level-dedupe | 0a3b86e4 | Monster HP/XP bonus now follows the live hero count at spawn time. `OD2_PLAYERS` (0..8) forces a larger count. | Larger-of-real-and-forced as in the exe. | med | Unset `OD2_PLAYERS`; otherwise revert 0a3b86e4. |
| 10 | feat/act3-durance-polish | 369b08b6 | Travincal-to-Durance stairs are sealed until the Compelling Orb is smashed. | The quest-state basis is UNVERIFIED against the binary. The 9g-act3-durance scenario was written but never run (no display). (Refreshed) `verify-level-graph.md` shows the Durance warp (level 100) is gated by the Blackened Temple node's private byte +0xc, restored on join from Khalim's Will (slot 18) bit 0, and allowed from source room 101; the Orb drives it. Whether the engine's stair seal matches that exactly is not checked. | med | Revert 369b08b6. |
| 11 | feat/level-graph-audit | fa1fb49f | Adds Act 3-5 outdoor borders, portal links and quest gates (Orb, Horadric Staff, Hellforge, Chaos seals, Anya portal) to the level graph so 40+ levels become reachable. | Built from drlg notes plus Levels.txt QuestFlag; gate semantics not exe-verified. (Refreshed) The exe reads QuestFlag only in the town portal object use (a record slot, bit 0); the warp tiles use their own node-based gates (levels 73, 100, 118, 128, 132). `gates.go` still models every gate as a QuestFlag slot. The town portal always leads to the act's town (verified). The notes are on `fork/feat/verify-level-graph`, not in this tree. | med | Revert fa1fb49f. |
| 12 | feat/death-area-cleanup | 1c46cecd | On area change the skill engine rebinds to the new area's Director and drops missiles, storms, traps, totem pulses, timers, summon bookkeeping, monster targets and DoT. Hero death stops the hero's auras pulsing unless the state has plrstaydeath. | Original behaviour for summons/storms/traps/missiles at hero death and summons through portals is UNVERIFIED (not changed). (Refreshed) Verified: states without plrstaydeath are destroyed at player death (0x0057d310 -> 0x00627890); no clear-on-area caller exists in the exe. | med | Revert 1c46cecd. |
| 13 | feat/skilldesc-hero-wiring | 747b6e8a | Skill tooltips show real damage rows instead of a synthesized damage line. | Kind 9/10/11/8/13 rows read in the exe (skilldesc-damage.md; kind 13 is summon life); kinds beyond 2/3/7 inferred from skilldesc.txt and marked UNVERIFIED (ed1c23b9). (Refreshed) Kinds 15, 16, 18, 25, 31, 63, 67 and 71 were decoded in 747b6e8a; the descdam header path (table 0x0072bf00) and about 40 other kinds are not modelled. | low (display) | Revert 747b6e8a and ed1c23b9. |

## Combat

| Branch | Change for the player | Why | Risk | Off / revert |
|---|---|---|---|---|
| feat/wire-hit-resolution (599907da) | See Review first #4. Also: monster-vs-hero physical damage applies flat reduction (stat 34) then physical resist in 8.8, floored at 0; numbers unchanged per differential test. Hero block animation replays after 15 + fasterblockrate/8 frames (logged only, no gameplay change yet). | verify-hit-resolution.md | high | Revert 599907da |
| feat/reconcile-resist (2d13da6a, 5f3dfd8a) | See Review first #3. | Game.exe | high | Revert both |
| feat/verify-undead-helper (88a97565) | Sanctuary's physical-resist zeroing keys on undead (monstats flag byte +0xd masks 0x08/0x10) instead of the boss flag. | Helper 0x63f9e0 verified in Game.exe (verify-undead-helper.md corrects verify-resist.md item 5, which said boss). | low | Revert 88a97565 |
| feat/wire-states (0911cdab, cc66af98, 66f3193a) | See Review first #5. | docs/verify-states.md | high | Revert |
| feat/wire-hero-death (59d4264e) | On hero death, states without plrstaydeath and all poison/burn streams end (before, they carried through the corpse and respawn). | Uses `Engine.HeroDied`; plrstaydeath rule from the states table. | low | Revert 59d4264e |

## Monsters

| Branch | Change | Why | Risk | Off / revert |
|---|---|---|---|---|
| feat/wire-monster-scaling (c9f845cc) | See Review first #8. | exe, see above | high | Revert |
| feat/players-and-level-dedupe (0a3b86e4) | See #9. Removes the unused `d2difficulty.MonsterLevel` (no callers). | 0x00571760 | med | `OD2_PLAYERS`, or revert |
| feat/verify-monster-ai (44fdb6e6) | Monster AI fixes: target modes 4/5, strict aggro radius, 4-step Wander, low-byte Circle split, rewritten SandRaider, Minion commanded-reach, Tentacle owner/expiry, one-shot Summoner wake-up. | Re-read 0x5af230, 0x5dc560, 0x5dcff0, 0x5de5e0, 0x5ef800, 0x5e09c0, 0x5ee920, 0x5aed10 in Game.exe. | med | Revert 44fdb6e6 |
| feat/monster-ai-oracle (75e1b96d) | Stand-in AIs now register with the exe target mode and look up their own target when the tick gives none. Baal wave, SandMaggot and pet AI divergences are listed in the test, not changed. | monster-ai.md 148-row AI table | low | Revert 75e1b96d |
| feat/monster-ai-remaining (882f621c) | Wounded teleport, threat re-target, idle wander, SandRaider overlay; Summoner wake gate becomes tick distance < 20 (not a counter). | 0x5aedc0, 0x5aefc0, 0x5dd6b0, 0x5dd7f0, 0x622020 | med | Revert 882f621c |
| feat/wire-monster-ai (e051311d) | Adds the post-target interfaces to the Director. Everything stays OFF: `CanTeleport`, `LevelThreat`, `IsTownLevel`, `OnOverlay` are nil unless a host sets them, and no code under d2game/d2app sets them. The levels.txt byte +0x2f column for LevelThreat is UNVERIFIED. | none needed while off | low | Default off; revert e051311d |
| feat/determinism-replay (43bf2b5e) | Map layout rolls come from seeded d2rand streams instead of global math/rand; monster placements, lists and shot targets are ordered by id/position. Same seed now gives the same map and monsters. | Own tests (map-order and time dependence). Map layouts differ from before for a given seed. | med | Revert 43bf2b5e |
| feat/trap-director (eaa9cae4) | Director.Cast of an armed trap calls the skill engine. Sentry/vine/totem drivers deliberately NOT switched (cadence and range differ). | Parity test only. | low | Revert eaa9cae4 |

## Summons

| Branch | Change | Why | Risk | Off / revert |
|---|---|---|---|---|
| feat/wire-summons (afd349a0) | See #1. | partly UNVERIFIED AIs | high | Revert |
| feat/summon-completion (d368fa9a) | See #2. | | med | Revert |
| feat/wire-states (b23dd5a3) | d2summon minion stats from monstats.txt per difficulty, level-scaled skill mods, per-pettype limits (shared golem/vine/totem/sentry groups, Raven top-up, oldest evicted, timed traps). Used by wire-summons. | petmax rules; Raven/Vines/CycleOfLife/AssassinSentry AIs UNVERIFIED | high | Revert b23dd5a3 |

## Items and inventory

| Branch | Change | Why | Risk | Off / revert |
|---|---|---|---|---|
| feat/wire-inventory (d0ea5bf0) | See #7. Belt size now from armor.txt belt column via `BeltBoxes` (same numbers for stock data). No split-stack UI added (test pins that nothing divides a stack). | verify-inventory.md; 4160d52c | high | Revert d0ea5bf0 |
| feat/verify-item-rules (23b4979c) | Generator can roll ethereal (5%) and sockets (33%) behind `DropOptions.RollExtras`, default OFF in this commit. (Refreshed) 258e5bfb turned `RollExtras` ON for monster drops, ground loot, chests and racks (see Added since the first version). Other changes are verified-rule helpers (MaxSock borders at ilvl 25/40 inclusive, durability loss 10% armor / 4% weapon, assassin claws need both hands). Ethereal +50% damage/defense is verified (FUN_00660a40, (v*3)/2) but implemented only on `fork/feat/ethereal-and-vendor-extras` (d5ada317), not in this tree. | 0x62bd70, 0x62ebf0, 0x62b720, 0x557d90, 0x62f100, 0x63ec50, 0x554d90, 0x554c60 | low (RollExtras off); item_property/item.go touched, check | Set `RollExtras=false` (enabled since 258e5bfb), or revert 23b4979c |
| feat/followup-fixes (1cb45104) | Bug fix: an ethereal item no longer becomes indestructible via the indestruct property handler. | Code bug (`ethereal || prop`). | low | Revert 1cb45104 |
| feat/vendor-stock-oracle (d456a95b) | Gamble quality/upgrade windows load from the 1.14b .bin values (the shipped difficultylevels.txt has no Gamble* columns, so gamble stock was all magic with no exceptional). Vendor stock seeds are now one session seed mixed with vendor class id (was a fresh time.Now() per open, identical across vendors). | (Refreshed) The vendor rules listed as unverified in d2vendor/doc.go were verified in the exe (verify-vendor.md: Min columns read, reqlevel gate, act-indexed item level cap on Normal only, potion tiers, restock on leaving town at 240000 ms); the fixes are on `fork/feat/verify-vendor` (3f981b52), not in this tree. Vendor items can be socketed but are never ethereal (d5ada317, same branch note). | med | Revert d456a95b |
| feat/skill-tooltips-colours (1b675ec5) | Ethereal normal items get a grey ground label like socketed ones. | Name colour rule; RGB of new tokens unverified. | low | Revert 1b675ec5 |
| feat/reconcile-objects (d3556095, 0be3dfd0) | Affix picker: alvl uses max(ilvl, qlvl); group 0 not exempt; zero-frequency skipped; rare-only flag also blocks tempered; classless items accept class affixes. Object spawn: per-slot ObjPrb roll, single weighted member per group, count ((w*h>>7)*density)>>8. | Verified against Game.exe (verify-affix-objects-path.md). | med | Revert both |

## Skills

| Branch | Change | Why | Risk | Off / revert |
|---|---|---|---|---|
| feat/wire-skill-levels (8563bd14) | Casts, mana cost and tooltip use skill points plus item +skills (stats 127, 83, 188, 107/97). The skill tree keeps base points. No change without +skill items. | `d2skill.EffectiveLevel` | med | Revert 8563bd14 |
| feat/skill-damage-oracle (782fdd18) | Skill mana calc fixed: no minmana floor, mps rounding order. | Verified at 0x6477d0 | low | Revert 782fdd18 |
| feat/death-area-cleanup (1c46cecd) | See #12. | | med | Revert |

## Quests

| Branch | Change | Why | Risk | Off / revert |
|---|---|---|---|---|
| feat/quest-rewards-audit (f7ae0165) | Act 3-5 spec-driven quest rewards survive death/reload (they became inert because load copied reward-pending to completed-before). Potion of Life now raises LifeBonus/max life. | Audit tests pin reward amounts, once-per-claim, once-per-difficulty. (Refreshed) The reward handlers were read in the exe (verify-quest-rewards.md): Radament gives nothing, Lam Esen 5 stat points, Fallen Angel 2 skill points, Golden Bird spawns the Potion of Life (the +20 life is paid on use), Prison of Ice gives the Scroll of Resistance (+10 per difficulty record on read). In this tree the Golden Bird claim still pays the life itself; the on-use payouts are on `fork/feat/verify-quest-rewards` (36e2fa9b). | med | Revert f7ae0165 |
| feat/act3-durance-polish (369b08b6) | See #10. | | med | Revert |

## Hero progression

| Branch | Change | Why | Risk | Off / revert |
|---|---|---|---|---|
| feat/wire-herostats (9c80cbad) | Experience is capped at Experience.txt row MaxLvl-1 before level-ups are counted (was unbounded). | exe 0x0057c510 (cap at row MaxLvl-1, verified) | low | Revert 9c80cbad |
| feat/derive-equivalence (d62bb21e) | `RecalcStats` now uses `d2herostats.Derive`. Differential test over 7 classes x levels 1-99 says it equals the old recalculation. | 0x0056e770; level-94 save oracle | low | Revert d62bb21e |
| feat/verify-xp-gain (7a53477f) | See #6. | | med | Revert |
| feat/breakpoints-from-oracle (911b77ce) | Hero FHR/FCR/FBR breakpoint tables replaced by rows derived from the animation-speed rule. Fixes Necromancer FHR, Barbarian FHR, Sorceress FBR, Necromancer FBR, Druid FBR. Adds werewolf/werebear, Paladin shield FBR and Assassin claw IAS rows flagged Unverified. Only matters where tables are displayed or consumed. | d2animspeed rule (verified in 0x00624150 and the table at 0x006ea3d4; the engine does not apply the rule itself) | low | Revert 911b77ce |

## UI and tooltips

| Branch | Change | Why | Risk | Off / revert |
|---|---|---|---|---|
| feat/verify-skilldesc (a3039111, ed1c23b9) | Skill tooltip is rebuilt on the real line-kind switch: fewer or different lines (zero values give no line, mana shows one decimal), string-table labels (StrSkill1/17/2/4), dsc2 block above Current Skill Level, dsc3 at level+1, new Next Level block and synergy lines. | Switch at 0x4eade0, jump table 0x4ec048. Damage line approximation UNVERIFIED. | low | Revert both |
| feat/skilldesc-hero-wiring (747b6e8a) | See #13. | | low | Revert |
| feat/skill-tooltips-colours (1b675ec5) | Original colour codes (0xFF 'c' N) in Label text are converted to engine tokens; adds tan, dark green, purple (RGB unverified); tooltip uses 'Current Skill Level: ' from the string table and adds the mana cost line. | RGB UNVERIFIED | low | Revert 1b675ec5 |
| feat/followup-fixes (91e12e9c) | Zone text and sound environment look up the level by level id, not RegionType (the wrong level was named); announces on level change. | Same bug class fixed earlier in lighting.go | low | Revert 91e12e9c |
| feat/localisation-audit (3a5ab319) | .tbl loading hardened: zero NameLength no longer fails the table; unknown or empty language code falls back to ENG; out-of-range label index returns -1; missing patchstring.tbl is skipped. | Verified on ENG/DEU/FRA/POL/KOR tables | low | Revert 3a5ab319 |

## Rendering

| Branch | Change | Why | Risk | Off / revert |
|---|---|---|---|---|
| feat/anim-render-exact (6f65e169) | Act 2 is no longer dark: base light now uses the hero's level id (was the LevelType row, reading cave rows for desert type 16 and town type 12). Animation speed steps at 25 Hz with an 8.8 fixed-point accumulator. DT1 shadows are lit like the floor. | renderer.md b5 | med | Revert 6f65e169 |

## World and levels

| Branch | Change | Why | Risk | Off / revert |
|---|---|---|---|---|
| feat/level-graph-audit (fa1fb49f) | See #11. | | med | Revert |
| feat/act3-durance-polish (369b08b6) | See #10. | | med | Revert |
| feat/reconcile-objects (eca4f708, 54823854) | `objgroup` records are now stored (the loader built the map and dropped it). Chest opening routes through `d2object.Open` and falls back to the old path for unknown rows. All 8 objgroup member slots are read (slot 7 was dropped) and SHRINES/WELLS read the right columns (they read column 0). Random room population is not yet called by map generation. | Header case bug found by the opt-in missing-header report (49e9ff83). | med | Revert both |
| feat/determinism-replay (43bf2b5e) | See Monsters. | | med | Revert |

## Saves

| Branch | Change | Why | Risk | Off / revert |
|---|---|---|---|---|
| feat/wire-inventory (d0ea5bf0) | .d2s loader zeroes carried gold above level*10000. A real level-94 save keeps its gold and byte-exact roundtrip tests are unaffected. | 4160d52c | high | Revert |
| feat/quest-rewards-audit (f7ae0165) | Pending quest rewards are kept across save/load. | Tests | med | Revert |
| feat/d2s-robustness | Tests and fuzz targets only; no loader behaviour change in the commit. | | none | n/a |

## Not gameplay (tests, oracles, docs, concurrency)

No change to play behaviour: d2monster oracle tests (75e1b96d test part), feat/hit-resolution-oracle a3dc714b (audit found no formula divergence), feat/xp-gain-oracle (05705e8c, c46ebe2e: awardKillXP and merc XP gate extracted and pinned; the level-difference penalty was wired later by 7a53477f, see Review first #6), feat/verify-boss-flag (b040e4d4, 11b58e54, 597f6599: pure d2monstats/d2herostats models and docs; the wiring is in feat/wire-monster-scaling), feat/wire-states test and race commits (e72f2b50, 9c9a5709, 8827a314, 78560be1: mutex guards, "behaviour unchanged"), d2animspeed and d2path movement oracles, d2object and d2drop oracles (becf96ce, 49accf9d), d2inventory rules (19d81f98), verify-item-rules docs/audit (080fa955, 8ad9f13d), vendor oracle tests, d2replay test fixes (9468fcc3, 00fb62b1), d2txt missing-header report (49e9ff83; opt-in via `OnMissing` or `OD2_TXT_WARN=1`), docs/verification-status.md (d44f9e9f), d2herostats and d2monstats first drops (778b2874, ca198407).

## Gaps in this review

- Not every wave-2 branch has a unique commit; the list above is derived from the commits reachable only from the named branches.
- Risk ratings are the author's judgement from the commit text, not from running anything.
- Items marked UNVERIFIED are quoted from commit messages; no Ghidra check was done here. The "(Refreshed)" notes quote `~/git/d2-re-notes/verify-*.md`; they were not re-checked in Ghidra either.

## Added since the first version

Gameplay changes that landed in `integration-s9-rehearsal` after the first version of this page.

| Branch | Commit | What changes for the player | Basis | Risk | Off / revert |
|---|---|---|---|---|---|
| feat/wire-attack-rating | 1187b0a5 | Hero melee and melee skill strikes apply the attack-rating operands before the to-hit roll: +AR vs undead (stat 0x7c) and demons (0x7b), ignore target defense (0x73), target defense percent (0x74). Monster-vs-hero to-hit adds the hero's armor vs melee (stat 0x21) or vs missile (stat 0x20). No-ops for a hero without those stats (fixed-seed regression test). | Operands read at 0x57b8b0 and the AR percent at 0x57b9c0 (verify-hit-resolution.md). Monster data word +0x16 bits (plain / halving) are NOT modelled (only the boss flag and special classes); PvP and missile to-hit untouched. | low | Revert 1187b0a5 |
| feat/wire-item-extras | 258e5bfb | `RollExtras` is on for monster drops, ground loot, chests and weapon / armor racks: 5% ethereal (halved durability) and 33% sockets on normal / superior bases, capped by difficulty, area and type. Rolled sockets and max durability persist in `Spec` / `StoredItem` (older hero files load unchanged; the .d2s writer is untouched, so real-save round trips are identical). Drop code, ilvl, seed, quality and affix counts are identical with extras on or off (300 fixed seeds). Vendor stock does not use these options. | Rolls verified at 0x554d90 and 0x554c60 (verify-item-rules.md). The ethereal +50% is verified but not implemented in this tree. | med | Set `RollExtras=false`, or revert 258e5bfb |
| feat/inventory-followups | 0a0a6b40 | Stash gold lives in `HeroState.StashGold` (nil = unknown); a .d2s import zeroes stashed gold above 2,500,000 and the exporter writes it back clamped (with a warning). Real saves still round-trip byte exact. | 0x531a50 save loader and 0x623640 stash cap (verify-inventory.md). | med (touches saves) | Revert 0a0a6b40 |
| feat/inventory-followups | bc9918c6, cfd28df3 | No behaviour change for stock data: the extra-stack stat is found by id 254 (a table without that row now adds nothing); the gold pickup flow is extracted into `pickUpGoldPile` with the same behaviour and tests for four cases. | verify-inventory.md | low | Revert |
