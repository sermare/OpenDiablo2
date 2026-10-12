# Known gaps

Everything we know is missing, approximate or wrong in this fork, in plain words, so play-testers do not have to
report it again. First written 2026-10-09, refreshed 2026-10-12 for waves 36-43 (tree `land/w38`, fork/integration 20a31960) from [STATUS_MATRIX.md](STATUS_MATRIX.md), [PLAYTEST.md](PLAYTEST.md) (Part 2 logs),
[tile-diagnosis.md](tile-diagnosis.md), [verification-status.md](verification-status.md) and
[wave2-gameplay-changes.md](wave2-gameplay-changes.md). Where those pages disagree with the code, the code wins.
"UNVERIFIED" means ported from reverse-engineering notes and never compared with the running original game.

How to test: [PLAYTEST.md](PLAYTEST.md). If you find something not on this list, report it as described there.

## Levels, tiles and maps

| Gap | What you may see | Detail |
|---|---|---|
| Cave and maze border cells, about 19 levels | A few wrong wall or floor tiles along room edges inside caves, crypts, tombs, sewers and similar | 51 of 70 maze levels match the original generator in every room; about 19 levels (ids 23, 51, 53-60, 66-68, 70, 72, 100, 101) differ in a few border cells where neighbouring rooms merge. 92 percent of all maze rooms are record for record exact. |
| Maze levels 35, 92, 69, 71 | Old-style patchwork floor and walls | Levels 35 (Catacombs Level 2) and 92 stop with a generator error in the tile builder; 69 and 71 have a different room count. They fall back to the older stamped tiles. `OD2_MAZE_STAMP=1` forces the old path for any maze level. |
| Tile objects of Act 1/2 maze rows | Missing small wall decorations in some Act 1/2 dungeons | Flag 0x20 records (rows of the tile object table for Acts 1 and 2) are not ported; it affects no floor. |
| Act 1 town to Blood Moor border | A hard seam where the town ends and the field starts | The original blends the two presets; here the border is the first walkable border cell, and outside the preset the map is black and blocked. |
| Outdoor tile pick | Occasional wrong tile patches in Act 1-2 outdoors, the desert and Kurast | The random tile choice is modelled as one room-seed step; the original's tile library is not emulated. Jungle and Kurast keep the approximation documented in `drlgoutdoor/doc.go`. Rocky Waste has patches of the wrong tile set. |
| Stale pictures after level changes | A tile from the previous level drawn in the new one | Seen once in the jungle and fixed by a cache reset; if you see it, report it with the two level names. |
| Lut Gholein | Town darker than the original at night; special tiles of styles 31-34 unused | |
| Forgotten Sands (level 134) | Cannot be generated | Its generator (the Act 2 desert style) is not ported. |
| Unreachable ground | Bare patches where monsters were removed | Monsters that would stand on floor islands the hero cannot reach (5-9 per desert level, a third of the Great Marsh and jungle, 25-50 in the River of Flame) are removed on purpose. |
| Chaos Sanctuary | Lava between the arms is not ground; the sanctuary is the cross of its floor | Only about a dozen monsters (density over the reachable area). Leaving and re-entering rebuilds it without the seal state of the objects. |
| River of Flame | Hellforge room stamped but its objects are not played | The bridge exit has six tiles of three styles; which the original uses is undecided (all lead to level 108). |
| Levels generated but never walked by a script | Anything, they are least tested | Halls of Anguish, Pain and Death's Calling (122-124), Nihlathak's Temple (121), the hell levels 125-127, Frozen River (114), Drifter Cavern (116), Icy Cellar (119), Worldstone Chamber (132), Durance of Hate 2 and 3 (101, 102), treasure rooms and the second Swampy Pit levels. Levels are tested from the data, not by a human play-through. |
| Exits and stairs | Stairs or caves that lead to the wrong level, or arrival on the warp tile instead of beside it | Dungeon and warp destinations follow the Vis slot of Levels.txt (observed, UNVERIFIED against the original); in Act 3 two entrances of Sewers 1 are not told apart, and in Act 5 ice caves you arrive on the way-back tile. |
| Level persistence | A level you leave and return to keeps its monsters, chests and ground items for as long as the game runs | This is an engine choice (`OD2_NOPERSIST=1` turns it off) and differs from the original. Not stored in the character file. |
| Light map, day and night | Lighting and the day/night colour blend are approximate | The line-of-sight painter and exact blend are not implemented; the pace of the day cycle is UNVERIFIED. |

## Monsters and combat

| Gap | What you may see | Detail |
|---|---|---|
| Unique and champion packs | Packs may be rarer, or their modifiers different, than in the original | Natural champion and unique packs with `monumod` modifiers are planned by `d2monreg` (`PopulateNatural`, `assignMods`, `uniquePack`) and counted in the population log line (`packs with modifiers`). Frequencies are not compared with the original; nests and some group kinds are still skipped (`real_pop.go`). |
| Monsters with a stand-in AI | Behaviour that is recognisable but not the original's | Every `monai.txt` AI has at least a stand-in (melee chaser, caster, flyer, turret or inert, `d2monster/ai_rest.go`); Tentacle, TentacleHead and Frog Demon are ported. An AI that is not registered at all still logs `AI not ported: stands still`. |
| Unique and champion packs | Natural champion and rare packs appear with their modifiers; some modifier effects are missing | The init callbacks of the exe (life bonus per rank, +3 levels, Extra Strong/Fast, Fire/Lightning/Cold/Poison Enchanted, Mana Burn, resist caps, Stone Skin, Ghostly, Fanatic, Possessed, Berserk) are ported and applied (`d2monreg/umodfx.go`, `d2monsters/umodfx.go`). Not yet: the skills of Teleportation and Aura Enchanted are recorded but the AI does not use them, Cursed and Multishot, the random name, and the timer event of modifier 41. |
| Monsters with no AI | None known (stale row removed in wave 38) | Every `monai.txt` AI is registered; Quill Rat, Tentacle and Frog Demon are ported from the exe. The Hireable (mercenary) think is an approximation of the exe's 1108-byte function; the quest-log hooks of Izual, Summoner and Nihlathak are not hosted. |
| Monsters in some levels | Empty or nearly empty levels | Kurast Causeway (82): earlier notes disagree (empty, or natural monsters added in wave 20); **unverified, check it**. Arreat Plateau on Normal and Bloody Foothills have none or only preset monsters; Swampy Pits and Flayer Dungeons (86-91) skip their blocks; Travincal only has the council monsters its data names. |
| Summons and their caster | Monster-cast summons (nest spawn, minion spawner, Hydra) stay alive when their caster dies; player summons follow or are dropped when the owner changes level | Verified against the exe: a monster leader's death hands its minions to the first minion or detaches them, nothing is killed (0x58d4a0). Player minions, totems, traps and Hydras vanish when the owner dies (0x57dcc0 -> 0x573980) and on an act change; inside an act, pets that warp follow the owner and range-type pets farther than 40 subtiles are dropped (0x5732b0; the `warp` column is flag bit 0 and `range` bit 1, VERIFIED from the table loader 0x618a30). The rule is in `d2summon.LevelChangeFate` and `Director.OwnerChangedLevel`, and the game screen applies it on a level change inside an act (`pet_levelchange.go`: `TakePetsForLevelChange` / `AdoptPets`; warp pets are placed next to the hero, others released; a range pet within 40 subtiles cannot stay behind because pets are not parked, so it is released too). Wired statically, not yet run in a live scenario. |
| Monster versus monster | Monsters do not fight each other, except when charmed, converted, attracted or confused | No general infighting is implemented; the forced states are in `d2monsters/forced.go`. Whether the original has any other monster hostility is not recorded. |
| Combat accuracy | Damage, hit chance, defence and some skill numbers may differ from the original | The formulas are read from the game's code but only partly compared with the running game. Hero defence uses dexterity only in the monster glue. Telekinesis is only a damage hit on a targeted monster (item pickup and object use are not simulated); Find Item is implemented (`d2skills/effects_abc.go`). Missile homing exists; periodic area effects are partial. Several skill formulas are marked unverified. |
| Cube and crafting | Recipes run from the installed `CubeMain.txt` | The Transmute button (`d2player/cube_transmute.go`, `d2cube`) covers the Horadric Staff and runeword socketing. A few condition ops and output level rules are UNVERIFIED (`d2cube/doc.go`). |

## Quests and acts

| Gap | What you may see | Detail |
|---|---|---|
| Act 3 gate and quests | Cutscenes and some exact rules are missing | Played end to end by scenarios 9m and 9n (`act3_live.go`, `d2act3`): Khalim's chests, Lam Esen's Tome, Gidbinn, the Council and Khalim's Flail, the cube recipe for Khalim's Will, the Compelling Orb and the Durance stairs it seals, Mephisto, the red portal to the Fortress. Still open: the cutscenes (the Mephisto bridge is only an object); which Council member carries the Flail, the red portal delay, the Sewers lever rule and the Orb with a carried Will are UNVERIFIED; Hratli stands at a guessed place (not in the town data). |
| Act 2 quests | Quest effects that only appear in the log | Darkened sun, harem blocker, Tyrael's portal and Duriel's portal delay are logged but not simulated. Quest handlers other than the speech are UNVERIFIED. |
| Acts 3-5 quest triggers | Quests may not advance as in the original | Table-driven guesses (`generic.go`). Act 1 and Radament's Lair are the verified lines. |
| Act travel | Trips back from Act 4 and 5 need `travelfree 1` | The original has no NPC for them. The red portal to the Pandemonium Fortress is only the travel rule. |
| Boss encounters | Delays, positions and portal objects differ from the original | Duriel, Mephisto, the Diablo seals (the position of three seal bosses is UNVERIFIED) and the Baal waves use read ids and guessed timing. |
| Cow level and Pandemonium portals | Not reachable the normal way | Creators of the Cow Level and level 133-136 portals are unknown. |

## Items, trade and saves

| Gap | What you may see | Detail |
|---|---|---|
| Items found in the game and the `.d2s` | Items you pick up or buy may be missing after reloading the original `.d2s` | They live in the engine's own `.od2` save; the `.d2s` exporter never writes bits it cannot prove. Use the same hero in this game to keep them. Your original `.d2s` is never modified. |
| Loot | Item levels and affixes may differ in feel | Item level of object drops is the monster level of the current area (Levels.txt MonLvl1..3, Ex columns in an expansion game), falling back to the hero level only when the area is unknown; drops are checked for plausible frequencies, not against the original. |
| New Sorceress | Starts with one point in Fire Bolt | `CreateHeroSkillsState` grants the CharStats `StartSkill` when it names a skill, else Fire Bolt for the Sorceress (`startSkillName`). UNVERIFIED: how the exe turns `StartSkill` into a skill is not recorded. |
| Prices, vendor stock and gambling | Printed prices and stock may differ | Binary-read and unit tested, never compared with the running game. |

## Interface, audio and performance

| Gap | What you may see | Detail |
|---|---|---|
| Audio | Wrong, missing or cut-off sounds; music or ambience that does not match the original | Voice allocation and ambient rules are read from the game but not compared by ear. Report these. |
| Interface layout | Panels slightly off | Moved to the original's numbers but not verified side by side. NPC menus are checked in the log, not visually. |
| Speed | Stutter after switching away from the window | macOS App Nap; see the quickstart for the setting. |
| Mac only | Apple Silicon only, macOS 14 or newer | The app is not notarised (right-click Open the first time). No Intel, Windows or Linux builds are tested. |

## Multiplayer

| Gap | What you may see | Detail |
|---|---|---|
| TCP join | Works between two copies of this game only | Not compatible with the real Diablo II servers or clients; the join packet layout is unverified. Party invitations, the level 9 requirement and player trade rules are modelled on visible behaviour. |
| Real Battle.net | Not supported and not planned | |

## Waves 36-43: what changed, what is still open

Landed in the tree these pages describe (full verify run 813: 291 scenarios in 6 jobs, all passed; details in
[STATUS_LOG.md](STATUS_LOG.md) and [RELEASE_NOTES.md](RELEASE_NOTES.md)). "Passed" means a scripted check, not a person.

| Area | Still open or not confirmed |
|---|---|
| Not in this tree | `feat/w43-fullgame` (a Normal to Hell chain scenario) is on its own branch: it broke Nightmare scenarios and was reverted out of the landed tree. Nobody has shown a Normal to Hell run end to end. |
| Cinematics | Now on by default in normal play (opening movie on first launch, act-change movies, Baal). Audio and video sync were checked by clock, never by ear or eye. The Cinematics menu path has no human check. |
| Classic (non-LoD) game | The game mode is wired into monsters, loot, vendors, the cube and the Act V gate. Whether a classic game plays right by hand is unconfirmed. **unverified** |
| Full screen by default | Normal launches start full screen whatever `config.json` says; use Cmd+Enter or `OD2_WINDOWED=1` for a window. Multi-monitor and notch behaviour were not looked at. **unverified** |
| Mercenaries | The hire list, portrait with life bar and potion drop on it exist, and the Hireable thinker is ported from the exe. Look and feel is checked in logs, not side by side with the original. |
| Monster missiles | Boss and monster missile skills now hit the hero (log-checked). Damage numbers are only partly compared with the original. |
| Light and backdrop | The occluded light painter is ported from the exe and the neighbour-level backdrop hides the black void at the map edge; the look was not compared visually. Turn off with `OD2_LIGHT_QUALITY=0` / `OD2_BACKDROP=0` to see if it is the cause. |
| Lut Gholein | Jerhyn is a placeholder object (his town data is missing), found by name when a script walks to him. |
| Ubers | Uber Diablo and Uber Mephisto stand still, as the exe is read; fighting Ubers need two `register` lines restored in `ai_boss2.go`. |
| Windows | Experimental cross-compiled build only, never run on Windows; the newer Ebiten is on `feat/w36-ebiten-upgrade`, not landed. |
| Input | Mouse wheel and keyboard item moving exist; a first-attempt retry is still needed by a few real-input scenarios (a click that lands mid-cast becomes a walk). |
| Saves | One validated, locked, atomic `.d2s` writer with rotating backups and a recovery offer is in. Damaged-save recovery by hand was not tried by a person. |

## What is verified (so you know what to trust)

Verified against the original game: the character file format (byte-identical round trip), the random number
generator and level seeds, the level layout generators for all five acts (room lists, world layout, tile records for
the preset and outdoor levels), the monster scaling tables, and the packet size tables. Everything else listed in
[STATUS_MATRIX.md](STATUS_MATRIX.md) as "approximate" is a best reading of the original, not a proof.

## Open items

(Carried over from waves 20 to 26; not re-measured in waves 36 to 43.)

- Lava levels still thin (partly fixed in w20). The placement test used a square of half-width SizeX around the spot
  (3x3 to 7x7); the exe tests the spot, a cross or a 3x3 square by SizeX 1, 2, 3 (`d2monreg.Footprint`). River of
  Flame (107) now makes 104 groups (was 84) and the Chaos Sanctuary (108) 49 (was 44) from 393 and 303 density hits
  (the `population diag` log line has the causes: 269 and 245 hits found none of their 20 spots free). Whether the
  original also treats lava subtiles as walk-blocked at population time is not checked (needs a collision oracle).
- The `auto` minimum of `scripts/pop_check.py` is derived from the engine's own rolls and walkability, so it cannot
  catch a real under-population like the one above. Compare with the oracle for golden group counts when available.
- Levels 73, 110 and 120 keep a minimum of 0 (no natural monsters). Level 73 passed again with 0 after the rebase; 110 and 120
  were not re-run yet (game runs paused).
