# Known gaps

Everything we know is missing, approximate or wrong in this fork, in plain words, so play-testers do not have to
report it again. Written 2026-10-09 from [STATUS_MATRIX.md](STATUS_MATRIX.md), [PLAYTEST.md](PLAYTEST.md) (Part 2 logs),
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
| Unique and champion packs | You never meet a named unique or a coloured champion pack in the field, no bonus modifiers (fast, extra strong, cursed, ...) | Natural champion and unique packs and the `monumod` modifiers are not spawned or ported. Only fixed bosses and super-uniques placed by the level data appear. Their flags are used in combat rules only. |
| Monsters with no AI | Monsters that stand still and never attack (Quill Rats and a few others) | Printed in the log as `AI not ported: stands still`. Tentacle, Frog Demon and some other behaviours are simple stand-ins. |
| Monsters in some levels | Empty or nearly empty levels | Kurast Causeway (82) has none; Arreat Plateau on Normal and Bloody Foothills have none or only preset monsters; Swampy Pits and Flayer Dungeons (86-91) skip their blocks; Travincal only has the council monsters its data names. Packs in other Act 3 levels were thin until recently. |
| Summons and their caster | Monster-cast summons (nest spawn, minion spawner, Hydra) stay alive when their caster dies; player summons stay when the owner changes level | Player minions vanish when the owner dies (D2MOO KillPlayerPets, see `ownerGone`). Open question, UNVERIFIED: no D2MOO code kills a monster's summons with their caster, so they are left alone; the exe was not checked (Ghidra had no program open). Owner leaving a level inside an act kills nothing in D2MOO (only classic act changes and leaving the game do); totems and traps on owner death are UNVERIFIED. |
| Monster versus monster | Monsters do not fight each other, except when charmed or confused | |
| Combat accuracy | Damage, hit chance, defence and some skill numbers may differ from the original | The formulas are read from the game's code but only partly compared with the running game. Hero defence uses dexterity only in the monster glue. Telekinesis and Find Item are not implemented. Missiles: no homing, no periodic area effects. Several skill formulas are marked unverified. |
| Cube and crafting | The Horadric Cube holds items but transmutes nothing; the cube panel does not call the Horadric Staff recipe | |

## Quests and acts

| Gap | What you may see | Detail |
|---|---|---|
| Act 3 gate and quests | Travincal and the Durance of Hate cannot be completed | Lam Esen's Tome, Khalim's Will, Blade of the Old Religion, the Golden Bird and Mephisto are not played; the Kurast Docks NPCs only talk. The stairs from Travincal to the Durance of Hate are sealed until the Compelling Orb is smashed (the quest basis is UNVERIFIED), and Durance level 3 (Mephisto) is not entered by anything. Hratli is not in the town data and stands at a guessed place. |
| Act 2 quests | Quest effects that only appear in the log | Darkened sun, harem blocker, Tyrael's portal and Duriel's portal delay are logged but not simulated. Quest handlers other than the speech are UNVERIFIED. |
| Acts 3-5 quest triggers | Quests may not advance as in the original | Table-driven guesses (`generic.go`). Act 1 and Radament's Lair are the verified lines. |
| Act travel | Trips back from Act 4 and 5 need `travelfree 1` | The original has no NPC for them. The red portal to the Pandemonium Fortress is only the travel rule. |
| Boss encounters | Delays, positions and portal objects differ from the original | Duriel, Mephisto, the Diablo seals (the position of three seal bosses is UNVERIFIED) and the Baal waves use read ids and guessed timing. |
| Cow level and Pandemonium portals | Not reachable the normal way | Creators of the Cow Level and level 133-136 portals are unknown. |

## Items, trade and saves

| Gap | What you may see | Detail |
|---|---|---|
| Items found in the game and the `.d2s` | Items you pick up or buy may be missing after reloading the original `.d2s` | They live in the engine's own `.od2` save; the `.d2s` exporter never writes bits it cannot prove. Use the same hero in this game to keep them. Your original `.d2s` is never modified. |
| Loot | Item levels and affixes may differ in feel | Item level of object drops is the hero level (the original uses the area level); drops are checked for plausible frequencies, not against the original. |
| New Sorceress | Starts without Fire Bolt | The start-skill rule is not looked up. |
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

## What is verified (so you know what to trust)

Verified against the original game: the character file format (byte-identical round trip), the random number
generator and level seeds, the level layout generators for all five acts (room lists, world layout, tile records for
the preset and outdoor levels), the monster scaling tables, and the packet size tables. Everything else listed in
[STATUS_MATRIX.md](STATUS_MATRIX.md) as "approximate" is a best reading of the original, not a proof.
