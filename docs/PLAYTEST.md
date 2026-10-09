# Act 1 playthrough log

Branch `feat/act1-playthrough`. The first hour of Act 1 is played by `OD2_AUTOSCRIPT`:

| Scenario | What it plays |
|---|---|
| `scripts/verify.d/9b-act1-playthrough.sh` | a brand new Sorceress (made by `OD2_AUTONEWCHAR`): Akara (intro, Den of Evil quest, topic), the gate to Blood Moor, fights and loot, the Den of Evil cave, clear it, back to Akara for the skill point, exit (saves the `.d2s`) |
| `scripts/verify.d/9c-act1-reload.sh` | loads the `.d2s` of 9b again: same hero, quests, waypoints; Akara has nothing more to say about the Den of Evil |
| `scripts/verify.d/9d-act1-sample-hero.sh` | the level 94 sample hero (a revived copy: the sample is a dead hardcore character, see `scripts/d2s-revive.go`) walks town, Blood Moor, Den, back |

By hand (the game needs a GUI session, see `docs/macos-quickstart.md`):

```sh
OD2_REALMAPS=1 OD2_AUTOSPEED=3 OD2_AUTOGAME=hero.d2s OD2_AUTOEXIT=1 \
  OD2_AUTOSCRIPT='wait:1;walkto:exit=2;expect:level=2;walkto:exit=8;expect:level=8;kill:all,200;exit' ./od2
```

New script steps (`d2game/d2autoscript`): `walkto:exit=<level>` (a border or an exit tile that leads there;
the hero fights what attacks him on the way), `walkto:object=<name>`, `kill:all[,<s>]` /
`kill:near=<tiles>[,<s>]` (melee, casts the best attack spell, drinks healing potions, retries monsters he
could not reach), `loot:<tiles>[,<s>]` (walks to the items, stores them), `until:<log substring>,<s>`
(only sees lines logged since the last step that did something), `menu:<NPC menu row or topic>`.
`OD2_AUTOSPEED=1..4` runs the game clock faster so the hour fits a verify run; `OD2_NOPOPULATE=1`
leaves the levels without natural monsters.

## What the level change is now

Walking out of a level is the real exit: the border between two Act 1 outdoor levels (the world layout of
`drlgworld` places their rectangles; the hero crosses the shared edge and arrives at the same world
position, `d2common/d2level/edges.go`) or the exit tile of a preset (cave entrance, stairs;
`TileDestination`). The town is the preset the world layout chose (TownN1/E1/S1/W1) in a map of its own.

## Bugs found by playing, and their fixes

| # | What a player saw | Root cause | Fix | Commit |
|---|---|---|---|---|
| 1 | The gate of the Rogue Encampment leads nowhere; walking to the edge of Blood Moor does nothing | no code changed the level at a seamless outdoor border; with `OD2_REALMAPS=1` the town was still the old overworld with a fake wilderness | `d2level.EdgeExit/EdgeArrival` (+tests), `MapEngine.World`, `generateRealTown` (preset by `TownFile`), `advanceEdges` | a9833485 |
| 2 | `LEVEL 1:` logged while standing in Blood Moor after `OD2_AUTOLEVEL=2` | the client reported the Rogue Encampment as its level at start | `d2client.AddPlayer` uses the start level | a9833485 |
| 3 | Entering the Den of Evil did not move the Den quest (only scripted runs did) | the quest system only learned of area changes from `OD2_AUTOQUEST` | `afterLevelBuilt` calls `questArea` for every level change | a9833485 |
| 4 | After Talk the NPC menu pops up again at once (and the greeting plays again); the Talk submenu with the quest topic closes the moment it opens | `onNPCMenuChoice` kept `npcTarget` and closed the topic menu that `questTalk` had just opened | Talk closes the menu first and forgets the NPC unless a topic menu is open | a9833485 |
| 5 | The hero gives up on a far stair, item or chest after 12 s although he is walking | the time-out counted seconds, not progress | the give-up clock restarts when the hero gets closer (`walkProgressStep`, `groundState.progress`, test) | 7d0c4d08, fd00fa43 |
| 6 | The cave entrance of Blood Moor is not an exit (two preset variants carry style 5 and 6); the tile styles of caves, crypts and jail are not the LvlWarp ids | `Destination(level, style)` assumed style == LvlWarp id | cave entrance presets are resolved by their path (`CaveEntranceDestination`, `SetWarpDestination`); dungeon tiles by the observed style rule (`TileDestination`, tests with the styles of levels 8..35) | 13ff15f5, fd00fa43 |
| 7 | Blood Moor, the other wilderness levels and the Den have no monsters | outdoor DS1 presets carry no monster markers; `PopulateRoom` was never called | `populateLevel` fills blocks of 16x16 tiles from the Levels.txt list (`mon1..`, density), not on top of the arrival; monsters the hero cannot walk to are removed (`MapEngine.CanWalkTo`) so a level can be cleared | a9833485, fd00fa43 |
| 8 | Monster gold drops are an item called `gld` that lands in the inventory grid | `Director.dropLoot` used `DropItems`, which turns gold into an item | `DropLoot` + `NewGoldPile` | 13ff15f5 |
| 9 | Experience grows (606 of 500) but the hero never reaches level 2; the Den reward skill point is the only one | nothing promoted the hero: only the `levelup` console command | `advanceHeroLevel` (`levelsGained`, tested): level, skill and stat points, full life and mana, sound, save | fd00fa43 |
| 10 | A new character has no potions or scrolls, so a Den fight at 40 life is lost | the lobby's character has no items and the import left it so | `StartingContainers` from CharStats.txt for a hero that was never played (`freshHero`, tests) | 13ff15f5, fd00fa43 |
| 11 | A hero who dies in the Den respawns **in the Den map** at the town coordinates, and the next level change fails to load `SODDstf.COF` | the death system assumed the town map was loaded; the level change set the animation of a dead hero | `respawnLevel` builds the town of the act, the hero stands up first, the corpse waits in the level where it lies and is placed when that level is built again | 13ff15f5, fd00fa43 |
| 12 | The sample hero (level 94) dies at once | the sample `.d2s` is a dead hardcore character with 0 life | `scripts/d2s-revive.go` makes a living copy for the playtest | 78ea6d51 |
| 13 | After a level change the new level is shown from where the old hero stood (the camera glides), and Akara's speech banner stays on screen in Blood Moor | the camera only eases towards the hero; the speech bubble is not tied to the level | `snapCamera`, `SpeechBubble.Clear` in `afterLevelBuilt` | 78ea6d51 |
| 14 | The scripted hero ignored the zombie biting him while he chased a far monster, gave up on a monster he was hitting, and chased monsters into the level border (so he walked back into the town) | the fight kept its first target and measured progress by distance only | retarget when another monster is 3 tiles nearer, "in reach and swinging" counts as progress, monsters near a border are left alone | fd00fa43, 78ea6d51 |
| 15 | A script step `until:<line>` was satisfied by the same line from ten steps ago, or missed the line its own action logged | the log search had no notion of "since" | `LogMarkHost`: lines since the last step that did something (tests) | 13ff15f5, 78ea6d51 |
| 16 | Monsters ran at the hero's difficulty only through an environment variable | -- | superseded by the difficulty work merged from `fork/integration` (`v.difficulty()`); the merge kept its version | 0dd97de0 |

## Limits found, not fixed

* Items created in the game (loot, starting potions) are not written into the `.d2s` (the exporter never
  fabricates item bits: `d2hero/d2s_export.go`); they live in the `.od2` save only. A hero reloaded from the
  exported `.d2s` has the items of the file (9c compares `items=` before and after: both 0).
* Levels are rebuilt at every level change: a cleared Blood Moor is populated again when the hero returns, a
  dungeon forgets its dead monsters (the Den quest remembers it is done).
* Levels 13-16 and 37 (the second cave levels, the Tristram area) are not generated by the maze port yet
  (`OD2_AUTOLEVEL=13` falls back to the town), so the game cannot be played deeper than the first cave level.
  Act 2 and later are not built at all.
* A new Sorceress has no skill points and attacks with the staff; the real one starts with Fire Bolt
  (CharStats `StartSkill`, the rule that grants it is not looked up). The scripted hero casts Fire Bolt or the
  best attack spell he has points in (the level 94 sample throws Fire Ball).
* Outdoor tiles keep the approximation documented in `drlgoutdoor/doc.go`; the town preset's border towards
  Blood Moor is the first walkable border cell, the original blends the two presets. Where the town stamp
  ends the map is black (blocked), as is the rest of the level rectangle around the preset.
* Quill Rats and a few other AI classes are not ported (`AI not ported: stands still`): they never attack.
* Dungeon stairs resolve by the style rule in `TileDestination` (UNVERIFIED against the exe's table).


# Act 2 playthrough log

Branch `feat/act2-playthrough`. The start of Act 2 with `OD2_REALMAPS=1`, played by `OD2_AUTOSCRIPT`:

| Scenario | What it plays |
|---|---|
| `scripts/verify.d/9e-act2-playthrough.sh` | the revived level 94 sample hero travels east (`say:completequest 1 6; travel:2`), talks to Warriv, Fara, Atma, Drognan and Greiz in Lut Gholein (the preset the Act 2 world layout chose), walks out of the gate into Rocky Waste (41) and Dry Hills (42) with fights and loot, enters the Halls of the Dead (56) through the tomb entrance, takes the stairs down to level 57 and up again, and walks back through the desert to town; the exported `.d2s` is an Act 2 save |
| `scripts/verify.d/9f-act2-lutn.sh` | `OD2_AUTOMAPSEED=1`: the other town variant (LutN, Rocky Waste to the north): out of the gate and back |

By hand:

```sh
OD2_REALMAPS=1 OD2_AUTOSPEED=3 OD2_AUTOGAME=hero.d2s OD2_AUTOEXIT=1 \
  OD2_AUTOSCRIPT='wait:1;say:completequest 1 6;travel:2;expect:level=40;walkto:exit=41;expect:level=41;kill:near=30,45;loot:30,40;walkto:exit=42;expect:level=42;walkto:exit=56;expect:level=56;exit' ./od2
```

`OD2_AUTOMAPSEED=<n>` (new) replaces the game seed of the hero, to play the other world layouts.

## Bugs found by playing Act 2, and their fixes

| # | What a player saw | Root cause | Fix | Regression test |
|---|---|---|---|---|
| 17 | `walkto:exit=41` in Lut Gholein: "level 40 has no exit towards level 41"; the desert levels cannot be walked from one to the next | the seamless borders of the Act 2 chain 40-41-42-43-44-45 were not in the link table and the Act 2 world placement was never given to the map engine (no level rectangles) | the five links in `drlgEdges` (`d2level/links.go`), `act23Rects` hands the `PlaceAct2World` rectangles to `SetWorld` for the desert levels and the town | `TestAct2DesertEdges`, `TestAct2WorldBordersForEverySeed` (40 seeds, needs `D2_TABLES`) |
| 18 | Lut Gholein was a random one of its two presets in a 57x57 map without a world position: the gate did not face Rocky Waste | `GenerateActTown` rolled the file with the level seed; the world layout (`World23.TownFile`: LutW when Rocky Waste lies to the west, LutN when to the north) decides | with `OD2_REALMAPS=1` the town uses the layout's file, its rectangle and `UseCollisionPaths`, like the Rogue Encampment | 9e, 9f (both variants walk out of the gate) |
| 19 | Halls of the Dead and every other Act 2 dungeon: "no level provider" | the maze provider only probed levels 2-37 | `maxMazeLevel = 72` (sewers, palace, tombs, lair, arcane sanctuary are in `drlgmaze/maze_act23.go`) | 9e (levels 56 and 57 are built and populated) |
| 20 | The tomb entrance of Dry Hills/Rocky Waste leads to level 0: stepping on it does nothing | the entrance presets (`Act2/Outdoors/TombEnt*.ds1`) carry the style 1 or 2, while the Levels.txt slots are LvlWarp 33..36; only `/caves/` presets were resolved | `d2level.SingleTileDestination`: the single dungeon behind an Act 2 desert level is the destination of its entrance preset | `TestAct2SingleTileDestination`, 9e (`TombEnt2.ds1 leads to level 56`) |
| 21 | While killing the monsters of Rocky Waste the hero ran into the Stony Tomb (a level change in the middle of the fight) | a scripted fight or pickup near a tomb entrance sends the hero to a point that lands on the warp tile, and a move order onto a warp tile counts as a click on it | `nearWarpTile`: outdoor warp tiles (like the level borders) keep the scripted fight and loot away (`warpChaseRadius`) | 9e fails on a `from=41 to=55` level change |
| 22 | The way back from the far side of Dry Hills: "EXIT gave up walking towards level 41 ... after 60 s" although the hero walked all the time | the exit walk's time-out counted seconds, not progress (the Act 1 fix of bug 5 covered stairs, items and chests only) | `exitWalk.progress`: the 60 s run only while the hero does not get closer | `TestExitWalkProgressRestartsTheClock`, 9e (`EXIT gave up` fails it) |
| 23 | A hero who ends the session in Lut Gholein is saved as an Act 1 hero (`act=1` in the exported `.d2s`, so the next load starts in the Rogue Encampment) | the server took the act from the client's save packet; the client's player entity never learns the act and always says 1, while `onChangeLevel` tracks it correctly | the server ignores the client's act | 9e checks the last `D2S EXPORT reparse` for `act=2` |

## What Act 2 does not do yet

* Warriv, Fara, Drognan and Greiz have a greeting voice and their menu rows (talk, trade/repair, hire, "go west"),
  but no quest speech: the quests of Act 2 start from Atma (Radament, message 304) and the palace; the rest of the
  Act 2 quest line needs the Sewers, the Palace, the Arcane Sanctuary and the tombs, which are generated but not played.
* Far Oasis (43), Lost City (44) and Valley of Snakes (45) are generated and have borders and entrances (the unit
  tests cover them), but 9e stops at Dry Hills: the monsters take a long time at the speed of a scripted hour.
* The Canyon of the Magi (46) has no seamless neighbour and seven tomb entrances (which tomb is the real one is
  not decided); its entrance tiles stay unresolved.
* The desert tiles are the same approximation as the Act 1 outdoors (`drlgoutdoor/doc.go`, exact tiles unavailable for
  the level type): the ground of Rocky Waste has patches of the wrong tile set, and the town is dark at night.
* A few monsters of the desert stand on islands the hero cannot reach (5-9 per level) and are removed; the walkable
  area of the generated levels differs from the original's.


# Act 3 playthrough log

Branch `feat/act3-playthrough`. The start of Act 3 with `OD2_REALMAPS=1`, played by `OD2_AUTOSCRIPT`:

| Scenario | What it plays |
|---|---|
| `scripts/verify.d/9g-act3-playthrough.sh` | the revived level 94 sample hero sails east (`say:completequest 1 6; travel:2; say:completequest 2 6; travel:3`), talks to Alkor, Ormus, Hratli, Asheara and Meshif in Kurast Docks (real preset `DockTown3.ds1`), walks the seamless borders Spider Forest (76), Great Marsh (77), Flayer Jungle (78), Lower Kurast (79), Kurast Bazaar (80), Upper Kurast (81), Kurast Causeway (82) and Travincal (83), with fights and loot in the jungle; takes the dungeon entrances Spider Cavern (85), Swampy Pit (86), Flayer Dungeon (88), Sewers (92), Ruined Temple (94), Forgotten Reliquary (96) and the Durance of Hate (100) and comes back out of each, then opens a town portal (`say:spawnportal 75; use:Portal`) from Travincal back to Kurast Docks; the exported `.d2s` is an Act 3 save |

By hand:

```sh
OD2_REALMAPS=1 OD2_AUTOSPEED=3 OD2_AUTOGAME=hero.d2s OD2_AUTOEXIT=1 \
  OD2_AUTOSCRIPT='wait:1;say:completequest 1 6;travel:2;expect:level=40;wait:3;say:completequest 2 6;travel:3;expect:level=75;walkto:exit=76;expect:level=76;walkto:exit=85;expect:level=85;exit' ./od2
```

## Bugs found by playing Act 3, and their fixes

| # | What a player saw | Root cause | Fix | Regression test |
|---|---|---|---|---|
| 24 | `walkto:exit=76` in Kurast Docks: the gate leads nowhere; the jungle and Kurast levels cannot be walked from one to the next | the seamless borders of Act 3 (town - Spider Forest - Great Marsh - Flayer Jungle - Lower Kurast .. Travincal) were not in the link table, and the town had no world rectangle | five fixed links plus the three jungle pairs in `drlgEdges` (the jungle levels hang off each other at random; a border exists only where the rectangles of the seed touch), `GenerateActTown` hands the `PlaceAct3World` rectangles to the map engine for Kurast Docks | `TestAct3WorldBordersForEverySeed` (40 seeds, all of 75..83 reachable over borders; needs `D2_TABLES`), 9g |
| 25 | the scripted walk from the south end of the Great Marsh to Flayer Jungle left through the *west* border into Spider Forest | the marsh touches both neighbours and any border crossing counted while the hero walked | `advanceEdges` crosses only the border the walk heads for (`edgeWanted`) | `TestEdgeWantedOnlyTheBorderOfTheWalk` |
| 26 | arriving in Upper Kurast from the Bazaar the hero cannot take a single step ("EXIT no way found") | the edge arrival keeps the world position; the approximated Kurast tiles put a roof there and the nearest free sub-tile was a 5x5 alcove closed on all sides | `MapEngine.NearestOpen`: the arrival moves to the nearest free sub-tile of a region of at least 600 sub-tiles (used by level changes and the provider default) | `TestNearestOpenCellSkipsClosedPockets` |
| 27 | the entrances of Spider Forest, Flayer Jungle, Kurast and Travincal lead to level 0 | their special tile styles are not LvlWarp ids and `SingleTileDestination` is Act 2 only (Spider Forest has two caves, Bazaar a sewer and two temples) | the style of the entrance tile is the Vis slot of Levels.txt (OBSERVED, exe rule UNVERIFIED): `d2level.Act3SlotDestination`, also used by `TileDestination` | `TestAct3SlotDestination`, `TestAct3SlotsMatchLevelsTxt` (compares with the real Levels.txt), 9g (`exit tile ... leads to level 0` fails it) |
| 28 | Spider Cavern, Flayer Dungeons, Sewers, the Durance of Hate: "no level provider" / "the engine cannot load that level yet" | the maze provider only probed levels up to 72 | `maxMazeLevel = 102` | `TestAct3DungeonsAreBuildable` (every Act 3 maze level generates), 9g |
| 29 | the six temples (94..99), Sewers 2, the treasure rooms and the Durance level 3 are not built either | they are DrlgType 2 (a single preset DS1), not mazes | `isAct3Preset` joins `isPresetLevel` (`GenerateRealPreset`), with its own `Params` (no world, rectangle from Levels.txt) | `TestAct3DungeonsAreBuildable` (each DrlgType 2 level of 84..102 is a preset level), 9g (Ruined Temple, Forgotten Reliquary) |
| 30 | in Spider Cavern the exit tile does nothing: the hero is locked in the cave | the lair presets carry the exit with style 1; `TileDestination` read style 1 as "second up exit or first down" and the cave has neither | a dungeon with a way out and no way down returns up for the low styles; the Act 3 "up" warp ids (52, 55, 58, 59, 62, 63, 65, 66) join `upWarps` | `TestAct3DungeonTileDestination` |
| 31 | Lower Kurast, the Bazaar, Upper Kurast and Travincal have not a single monster | `GroupsForRoom` truncates: a 16x16 block of a level with MonDen 325 holds 0.83 monsters, which became 0 groups in every block | `GroupsForRoomFrac`: where the rounding gives 0 the fraction is the chance of one group (levels that had groups are untouched) | `TestGroupsForRoomFrac`, 9g (79, 80, 81, 83 get monsters) |

## What Act 3 does not do yet

* The quests of Act 3 (Lam Esen's Tome, Khalim's Will, Blade of the Old Religion, the Golden Bird, Mephisto) are not played; the NPCs of Kurast Docks only talk (menus and greetings). Hratli is not in the DS1 and stands at a guessed place (`townExtras`).
* Kurast Causeway (82) has no natural monsters: no 16x16 block of the generated level reaches the walkable share of 40 % (the bridge between the canals). Travincal has its council monsters only as far as the DS1 names them.
* The tiles of the jungle and of Kurast keep the approximation of `drlgoutdoor/doc.go`; the special (wall) tiles of the Kurast presets with styles 8, 9, 12, 16 are not exits and stay unresolved on purpose (38 of them in Upper Kurast).
* The Swampy Pits and Flayer Dungeons (86..91) are mazes with DS1 monsters: the natural groups skip their blocks, the second level of each pit (87, 89) and the treasure rooms were generated but are not part of 9g.
* Durance of Hate 2 (101) is generated but not walked; level 102 (Durance 3, Mephisto) is a preset level that nothing enters yet, and the red portal to the Pandemonium Fortress is only the act-travel rule.
* Sewers 1 has two entrances in each of Kurast Bazaar and Upper Kurast (slots 0 and 1); coming back up, the hero arrives at the first one (which one the original uses is UNVERIFIED).
* The Great Marsh and the jungle lose a third of their monsters to the "hero cannot walk there" filter (islands of floor between water and undergrowth).
