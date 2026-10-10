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

## Act 2 finish (branch `feat/act2-finish`)

| Scenario | What it plays |
|---|---|
| `scripts/verify.d/9i-act2-full.sh` | Dry Hills -> Far Oasis (43) -> Lost City (44) -> Valley of Snakes (45) over the seamless borders; Maggot Lair 1-3 (62-64), Ancient Tunnels (65), Claw Viper Temple 1-2 (58, 61) and back |
| `scripts/verify.d/9j-act2-dungeons.sh` | Lut Gholein -> Sewers 1-3 (47-49), Harem (50) -> Harem 2 -> Palace Cellar 1 (51-52), portals to Palace Cellar 3 (54) and the Arcane Sanctuary (74) with a teleport pad, the Canyon of the Magi (46, by portal) and its seven tomb entrances (66-72) |
| `scripts/verify.d/9k-act2-quests.sh` | `OD2_AUTOQUEST=act2` (also `sun`, `staff`, `arcane`, `summoner`, `tombs`): Radament, Tainted Sun, Horadric Staff, Arcane Sanctuary, Summoner, Seven Tombs through the real speech tables, Sounds.txt rows and string.tbl text; Fara, Drognan, Greiz, Jerhyn, Lysander, Warriv, Cain, Meshif and Tyrael speak |

Log lines: `real maze: Tal Rasha's tomb level N: real=... orifice objects=...`, `act town: exit tile style=...`, `OBJECT teleport pad ...`.

### Bugs found by playing the rest of Act 2

| # | What a player saw | Root cause | Fix | Regression test |
|---|---|---|---|---|
| 24 | Lut Gholein: `walkto:exit=47` / `=50`: "level 40 does not border level 47"; no way into the sewers or the palace | the town DS1 special tiles carry the styles 2, 3, 4 (Vis slots of Levels.txt), which no rule resolved | `d2level.Act2VisDestination`: the style of a special tile of an Act 2 file is the Vis slot of Levels.txt (UNVERIFIED rule; every style seen in the files fits; table `vis_act2.go`) | `TestPresetTileDestinations`, `TestAct2VisSlotsMatchLevelsTxt` (D2_TABLES), 9j |
| 25 | Harem Level 1 (50): "the engine cannot load that level yet" | Levels.txt DrlgType 2 (one preset file, `Harem2.ds1`); the maze provider rejects it and no provider took it | `actPresetPrest` in `act_towns.go` builds it like a town | 9j |
| 26 | Sewers 1: "level 47 has no exit towards level 48" | the down stairs carry style 2, the dungeon stair rule knew only 0, 1 and 4+ | the Vis slot rule | `TestAct2VisSlotsMatchLevelsTxt`, 9j |
| 27 | Canyon of the Magi: entrances to the tombs 70-72 unreachable ("level 46 does not border level 70"), 66-69 only by accident (styles 4-7) | the seven King Tomb tiles carry the styles 1-7 = Vis1..Vis7 of level 46 (Vis0 is empty) | `CanyonTombDestination`; `markWarpTiles` records the destinations | `TestCanyonTombEntrances`, 9j (all seven tombs entered and left) |
| 28 | No tomb had Tal Rasha's chamber or the Horadric orifice; the special tombs were not enlarged | `drlgmaze.Params.TombA/TombB` were never passed by the engine | `GenerateRealMaze` passes `DrawActExtras`; `d2drlg.RealTomb(gameSeed)` = TombA (the maze finisher stamps `Talrasha` on TombA) | `TestRealTomb`, `TestAct2MazeLevelsGenerate` (D2_TABLES), 9j log |
| 29 | Arcane Sanctuary teleport pads, the sanctuary portal and Duriel's portal did nothing | stubs in `d2object` | fn 27: the hero lands next to the nearest other pad (`PadPartner`; the original uses the same or an adjacent room); fn 34: Palace Cellar 3 <-> sanctuary; fn 43: Duriel's lair | `TestPadPartner`, 9j |
| 30 | Warriv, Fara, Drognan, Greiz, Jerhyn, Elzix, Lysander, Cain had no quest speech after Radament | quests A2Q2-A2Q6 had no nodes (Seven Tombs only the Duriel kill) | `d2quest/a2_later.go`: Horadric Staff, Tainted Sun, Arcane Sanctuary, Summoner and the speech chain of the Seven Tombs; `CubeHoradricStaff`, orifice, `TravelToAct3` | `TestAct2QuestChain`, `TestHoradricStaffCain`, autoquest stages, 9k |
| 31 | Cain's lines 335-339, Tyrael 302 and Greiz 397 showed no text | the handle -> string.tbl key rule does not fit them | `TextKey` special cases | `TestTextKeysAct2`, `TestTextKeysExistInStringTbl` (D2_STRING_TBL) |

Which tomb is real: the game seed draws two tomb levels (`DrawActExtras`); the first (TombA) gets the Talrasha chamber with the orifice (object 152), the second a Kaa chamber, the other five a chest. The entrance does not tell which tomb is real. Quest objects: Tainted Sun altar 149, orifice 152, Horazon's journal 357.

## What Act 2 does not do yet

* The Vis slot rule for special tiles (bug 24), the string.tbl keys of the palace guard barks (voice only), the object id of
  Horazon's journal (357, "Tome") and all A2Q2-A2Q6 handlers except the speech tables are UNVERIFIED against the binary.
* The Horadric Cube panel does not call `CubeHoradricStaff` (the recipe is in the quest layer); quest effects such as the
  darkened sun, the harem blocker, Tyrael's portal and the Duriel portal delay are logged (`QUEST EFFECT ... [not simulated]`) only.
* Lut Gholein: the special tiles of the styles 31-34 are not used.
* The desert tiles are the same approximation as the Act 1 outdoors (`drlgoutdoor/doc.go`, exact tiles unavailable for
  the level type): the ground of Rocky Waste has patches of the wrong tile set, and the town is dark at night.
* A few monsters of the desert stand on islands the hero cannot reach (5-9 per level) and are removed; the walkable
  area of the generated levels differs from the original's.

# Act 4 and the start of Act 5 playthrough

Branch `feat/act45-playthrough`. `scripts/verify.d/9h-act45-playthrough.sh` plays Act 4 (Pandemonium Fortress, Outer
Steppes, Plains of Despair, City of the Damned, River of Flame, Chaos Sanctuary, and the waypoint back to the Fortress)
and the first outdoor levels of Act 5 (Harrogath, Bloody Foothills, Frigid Highlands, Arreat Plateau, the cave stairs
into the Crystalline Passage). `scripts/verify.d/9i-act5-caves-playthrough.sh` plays the ice caves on to the Worldstone
Keep (waypoint to the Crystalline Passage, Glacial Trail, Frozen Tundra, Ancients' Way, Arreat Summit, Worldstone Keep
1-3, Throne of Destruction; a waypoint hop from level 129 back to the Ancients' Way). Both use the revived level 94
sample hero with the act travel quest flags forced (`say:completequest`), `OD2_REALMAPS=1`, `OD2_AUTOSPEED=4`.

## Bugs found by playing Acts 4 and 5, and their fixes

| # | What a player saw | Root cause | Fix | Regression test |
|---|---|---|---|---|
| 24 | `walkto:exit=104` in the Fortress: "level 103 does not border level 104"; the gate of the Fortress and the borders of the Act 4 chain are not exits. Same for Harrogath -> Bloody Foothills -> Frigid Highlands -> Arreat Plateau | no seamless links for the Act 4/5 world layouts (`drlgworld.GenerateAct4/5` already place the rectangles edge to edge) | `drlgEdges` gains 103-104-105-106 and 109-110-111-112 (the pass 3 registrations and `DRLG_LinkAdjacentLevelRange` of drlg-act45-outdoor.md section 2); Frozen Tundra (117) has none | `d2level.TestAct45EdgeNeighbours`, `TestAct45RouteReachable`; 9h checks `via=edge` for each border |
| 25 | The lava warp of the City of the Damned and the cave of the Arreat Plateau lead to level 0 ("exit tile ... leads to level 0"): stepping on them does nothing | `markWarpTiles` only resolved exit presets of the Act 1 caves and the Act 2 desert (`ActOfLevel == 2`) | outdoor levels of Act 4/5 with one tile link resolve their exit preset to it (`SingleTileDestination`, `ActOfLevel >= 2`) | 9h: no "leads to level 0", `WarpLava1.ds1 leads to level 107`, `WestEntrance_Dirt.ds1 leads to level 113` |
| 26 | River of Flame, ice caves, Worldstone Keep: "the engine cannot load that level yet"; the maze levels of Acts 4 and 5 are refused although the generators are ported | `maxMazeLevel = 72` | `maxMazeLevel = 135` (drlgmaze `maze_act45.go`) | `d2mapgen.TestMazeProviderRange`; 9h/9i build 107, 113-115, 118, 128-130 |
| 27 | In the dungeons of Act 4/5 the stairs and warp tiles lead nowhere ("has no known destination"): the tile style is not a LvlWarp id | `TileDestination` knew the Act 1 up/down rule only | `SlotDestination`: the style k of a tile is the k-th link of the level in Vis slot order (ice caves up/ahead/down floor, Arreat Summit, Worldstone Keep up/down, River of Flame south room = style 0); read off the DS1 files and checked against LvlWarp ids 73-75, 81-82. UNVERIFIED against the exe | `d2level.TestSlotDestination` (28 cases, also through `TileDestination`) |
| 28 | The Frozen Tundra has two cave exits (back to the Glacial Trail, on to the Ancients' Way) and both led to level 0 | two tile links, so the single-link rule does not apply | `OutdoorExitByPreset`: `WestEntrance_Snow.ds1` -> 115, `WestExit_Snow.ds1` -> 118 | `d2level.TestFrozenTundraExits`; 9i checks both log lines |
| 29 | There is no way from the River of Flame to the Chaos Sanctuary: "level 107 has no exit towards level 108" | the exit is walk-through floor of the bridge room (`Act4/Diab/BridgeLava.ds1`, six special tiles with styles 8, 12, 16) and the link has no LvlWarp id (`-1`) | `mazeRoomExit` gives the tiles of that room the destination 108 (`SetWarpDestination`) | `d2mapgen.TestMazeRoomExit`; 9h: `from=107 to=108 via=warp` |
| 30 | A hero who walks back out of the Chaos Sanctuary lands on the bridge tiles and, on the way to the south exit, leaves for the Chaos Sanctuary again: a defensive fight went for a monster standing on an exit tile (the order to walk there counts as a click on the exit) | `nearWarpTile` protected only the warp tiles of outdoor levels, and the defensive fight (`k.defend`) did not look at it at all | the fight leaves monsters near exit tiles alone in the Act 4/5 dungeons too, defence included | 9h: `from=107 to=106 via=warp` after `from=108 to=107` |
| 31 | The walk to a stair of the Worldstone Keep gives up ("LEVEL gave up walking to the warp tile ... 24 tiles away") although the maze path is simply winding | the warp-use clock (12 s without getting closer in a straight line) also ran during a scripted exit walk, which has its own 60 s clock | the warp-use give-up is suspended while an exit walk is active | 9i (128 -> 129) |
| 32 | `POPULATE level 112 ...: -2 monsters` | the count was the monsters of Levels.txt groups minus all removed ones, including preset monsters | the log counts the monsters that are on the level after the removal | 9h reads the count for levels 104-107, 111 and 113 |
| 33 | `EXIT` walks of 150 tiles looked frozen in the log (60 s of silence before "gave up") | no progress output | a line every 10 s of walking: position, distance to the candidate, seconds without progress | -- |

Waypoints (103, 106, 107, 109, 111, 112, 113, 115, 117, 118, 129 have their waypoint object and bit; the panel lists the
act's nine or three rows; travel lands next to the waypoint) were played by hand through the `waypoint:` script step
and need no fix, they only need the 10 s since the last level change that the original has (`wait:11`).

## What Acts 4 and 5 do not do yet

* The Chaos Sanctuary is built from 25 preset stamps of its 225 rooms (the exact tile path stops at "animated preset
  tiles (0x66ff50 / 0x6703e0)", the level has no plain rooms): only the entry area has a floor, the rest is black, and
  about 170 of its 200 monsters are removed as unreachable. The seals, Diablo and the star are not playable.
* The River of Flame has 25-50 unreachable monsters removed (the generated rooms are not all joined by floor), the
  Hellforge room is stamped but its objects are not played, and the bridge exit has six tiles of three styles
  (which one the original uses is not decided; all lead to 108).
* Arreat Plateau on Normal has no natural monsters: its Levels.txt list (`overseer2`, `minion2`) is not spawnable for the
  group picker (`IsLevelSpawnable`); Bloody Foothills has Levels.txt density 0 (only preset monsters, none placed yet).
* The Act 5 ice cave stairs resolve by the slot rule (`SlotDestination`, UNVERIFIED against the exe), and the hero
  arrives on the warp tile of the way back (the original places him beside it).
* Halls of Anguish/Pain/Death's Calling (122-124), Nihlathak's Temple (121), the hell levels 125-127, Frozen River (114),
  Drifter Cavern (116), Icy Cellar (119), the Worldstone Chamber (132) and Forgotten Sands (134) are generated but not
  walked by a script; the Throne of Destruction's link to the Worldstone Chamber is a portal in the original, not a tile.
* The level names come from the game's Levels.txt strings (`Rigid Highlands`, `Crystalized Cavern Level 1`, ...).
