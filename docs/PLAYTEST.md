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

* The Chaos Sanctuary (108, 25 preset stamps) is walkable as the cross of its floor (star, four arms, entry); the lava
  between is not ground (the exact tiles are in place, `OD2_AUTOMAP_ASCII=1` marks walkable-but-cut-off ground with
  `o`). Populating now draws packs on ground the hero can reach (0 removed as unreachable, but only a dozen monsters:
  Levels.txt density over the reachable area). The five seals are operable (`chaos.go`, `d2boss.Seals`): the seal
  bosses appear at the exe's dummy offset, or on the nearest reachable ground when that falls behind a wall (the Grand
  Vizier, Infector and De Seis dummies all do here, UNVERIFIED which is right), Diablo arrives at dummy 255 after the
  last boss dies and can be killed. `9j-chaos-sanctuary.sh` plays all of it. Leaving and re-entering the level
  rebuilds it without the seal state of the objects (the encounter state is kept).
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
