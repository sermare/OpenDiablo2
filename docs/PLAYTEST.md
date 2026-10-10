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


# Quest walkthroughs (Acts 2-5 in the real world)

Branch `feat/quest-walkthroughs`. The Acts 2-5 quest logic (`d2common/d2quest`) was verified only by `OD2_AUTOQUEST`,
which injects events. These scenarios play the first real steps of quests through the real game: the level 94 sample
hero (a revived copy, quests reset to the start of the act with `say:resetquests`, Normal monsters) talks to the NPCs,
walks through the real exits, opens the quest objects, kills the quest monsters, picks up and uses the quest items, and
the checks read the quest bits, the NPC speech lines (message id, Sounds.txt row, text), the quest log panel
(`questpanel <act> <quest>` opens the real panel on that quest and logs the title and the page text), the rewards and the
quest slots of the exported `.d2s`. The helpers are in `scripts/quest_walk_lib.zsh`.

| Scenario | What it plays | Quests, speech |
|---|---|---|
| `9j-quest-radament.sh` | Lut Gholein, Atma, the manhole into the Sewers 1-3, Radament, the Book of Skill (pick up, read: +1 skill point), Atma's reward | A2Q1, msg 304 and 334, slot 9 |
| `9j-quest-staff.sh` | the staff chest (Maggot Lair 3), the cube chest (Halls of the Dead 3), the scroll chest (Sewers 3), Deckard Cain, the cube makes the staff | A2Q2, msg 335-339, slot 10 |
| `9j-quest-taintedsun.sh` | Lost City (the sun darkens), Drognan, Valley of Snakes, Claw Viper Temple, the Tainted Sun altar gives the Amulet of the Viper | A2Q3, msg 348, slot 11 |
| `9k-quest-lamesen.sh` | Alkor, Kurast Bazaar, the temple entrance, the Ruined Temple, the tome on its altar, Alkor's +5 stat points | A3Q1, msg 549 and 564, slot 17 |
| `9k-quest-blade.sh` | Hratli, Flayer Jungle (Gidbinn), Flayer Dungeon, Ormus, Asheara, Ormus again (Iron Wolves) | A3Q3, msg 571 587 589 593, slot 19 |
| `9k-quest-izual.sh` | Tyrael, Outer Steppes, Plains of Despair, kill Izual, Tyrael's +2 skill points | A4Q1, msg 664 670 676 681, slot 25 |
| `9k-quest-siege.sh` | Larzuk, Bloody Foothills, kill Shenk the Overseer, Larzuk's reward (the socket dialog itself is not played) | A5Q1, msg 20077 and 20090, slot 35 |
| `9k-quest-rescue.sh` | Qual-Kehk (Rite of Passage line first, then Rescue), Bloody Foothills, Frigid Highlands with its three cages | A5Q2, msg 20153 and 20096, slot 36 (first steps only) |

New script/console pieces: `kill:name=<text>[,<s>]` (the monsters of that name anywhere in the level, also destroyable props),
console `useitem <code>` (right click on an inventory item), `cubeput <code>`, `lootquest [tiles] [s]` (only quest items),
`clearinv`, `questpanel <act> <quest>`; `loot:` now goes on when an item does not fit (it puts it back) and takes quest items
first; `resetquests` rebuilds the quest runtime; `LEVEL objects:` / `LEVEL npcs:` / `LEVEL quest object` log lines at every
level build.

## Bugs found by playing the quests, and their fixes

| # | What a player saw | Root cause | Fix | Regression test |
|---|---|---|---|---|
| 34 | The manhole and the dock of Lut Gholein do nothing: the sewers (Radament) cannot be entered | the Act 2 town special tiles (style 2 manhole, style 3 dock stairs) are not LvlWarp ids | `townTileDestinations` in `d2level/links.go` (observed in screenshots of the tiles) | `TestLutGholeinSewerTiles`; 9j-quest-radament |
| 35 | Sewers Level 1 and 2: the stairs down lead nowhere, the dock end does not lead back | the slot rule gives the wrong tile for the sewer rooms | `mazeRoomExit`: `SewSDown` leads to the next sewer level, `SewNSDock` to the town | `TestSewerRoomExits`; 9j-quest-radament |
| 36 | After `resetquests` Atma still said nothing / quests kept their old state | the quest runtime held its own states and heard lines | `resetquests` builds the runtime again from the cleared record | 9j-quest-* |
| 37 | The quest chests, the Tainted Sun altar, Lam Esen's tome, Gidbinn... did nothing when clicked: the quest never saw them | `IsQuestObject` knew the Act 1 objects only, so only those were walked to as quest objects | all Act 2-4 quest objects; chests, altar, tome and Gidbinn open once (`questObjectOperated`) | `TestLaterQuestObjectsAreQuestObjects`; 9j-quest-staff, 9j-quest-taintedsun |
| 38 | Many quest lines are heard but show no subtitle (Qual-Kehk and Cain of Act 3, Cain's Staff lines, Greiz, intros of Tyrael, Hratli, the Ancients, Act 3 Khalim lines ...) | `TextKey` derived the string.tbl key from the sound handle and got `Qualkehk` / `Cain` / `SuccessfulCain` ... wrong | the spelling rules and a table of the exceptions in `d2quest/speech.go`; the lines that really have no key are listed in the test | `TestTextKeysExist` (needs `D2_STRINGTBL`: the game's `string.tbl` files concatenated) |
| 39 | The Horadric Staff log said "Take the artifacts to Cain" after Cain had confirmed the finished staff, and page 4 after the third report | the page was the number of reports + 1 | page 2 searching, 3 all parts reported (use the cube), 4 staff assembled, 5 parts carried but not reported | `TestHoradricStaffLogPageAfterAssembly`; 9j-quest-staff |
| 40 | Lam Esen's tome: the object stands in the Ruined Temple and gives nothing | the quest only listened for the item pick-up | operating object 193 spawns the item | `TestLamEsenTomeObjectGivesTheItem`; 9k-quest-lamesen |
| 41 | The Ruined Temple (and levels 90, 91, 93, 95-99): "the engine cannot load that level yet" | the preset provider only knew the Act 1 caves and Acts 4/5 | `isAct3Preset` (Levels.txt gives the rectangle, LvlPrest the DS1) | 9k-quest-lamesen |
| 42 | Kurast Bazaar, Upper Kurast, the Causeway: the temple entrances are not exits | two links with the same LvlWarp id (61) | `TempleEntranceByPreset`: `BurbsTemple2.ds1` (tile style 2) is the first temple link, `BurbsTemple3.ds1` (style 3) the second. UNVERIFIED which temple is which in the original | `TestTempleEntranceByPreset` |
| 43 | The way out of a temple (and Spider Cavern, dungeons, sewers, Durance up stairs) leads nowhere | no Act 3 "up" LvlWarp ids in `upWarps` | 52, 55, 58, 59, 62, 63, 65, 66 | `TestTempleUpStairsLeadBack` |
| 44 | The cave entrances of Flayer Jungle and Spider Forest lead nowhere | their presets (`PygW.ds1`, `PygW2.ds1`) carry the styles 0 and 1 of the slot order | slot rule for levels 76 and 78 | `TestJungleCaveEntranceStyles` |
| 45 | The Blade of the Old Religion cannot be played: no Gidbinn; the log said "Look for the Gidbinn" to the end | no item from the Gidbinn objects (251 altar, 252 blade), pages followed the first area change only | the objects spawn `g33`; `logPage` follows the state (pages 1-5 of `qstsa3q31..35`) | `TestGidbinnObjectsGiveTheBlade`; 9k-quest-blade |
| 46 | Shenk the Overseer (and every DS1 super unique) is a plain "Overseer": no name, no followers, so "kill Shenk" never fired | `placeMonsters` turned a super unique marker into a plain monster of the class | the NPC placement keeps the SuperUniques.txt key; the director spawns the boss with followers and its own name | `TestNPCSuperUniqueKey`; 9k-quest-siege |
| 47 | Harrogath has no Larzuk with real maps: the Siege quest cannot be taken | the legacy act town provider adds the NPCs the DS1 lacks, the preset provider did not | `placeTownExtras` also for preset towns | 9k-quest-siege |
| 48 | The prison doors of Rescue on Mount Arreat cannot be attacked | AI "Idle" so not hostile, so never adopted by the director | `IsDestructibleProp` (prison and barricade doors/towers) are adopted and attackable by name | `TestIsDestructibleProp` |
| 49 | The quest log shows `Rescue %d more Soldiers ...` | a printf token in the string table | `fillCount` (15 soldiers, 4 seals: the counts are not kept by the quest system yet, UNVERIFIED) | `TestFillCount`; 9k-quest-rescue |
| 50 | A script check placed after `kill:all,200` ran in the middle of the fight | the runner stopped waiting for a busy host after 120 s | `BusyTimeout = 400` | `TestBusyTimeoutCoversLongKills` |
| 51 | `loot:` stopped at the first item that did not fit in the inventory, quest items could stay on the floor | no room = end of the step | the item is put back and the step goes on; quest items first; `lootquest` | 9j-quest-staff |

| 52 | `scripts/verify_parallel.sh` printed "ALL JOBS PASSED" in a minute: no game scenario ran | it passes scenario names without `.sh`, `verify.sh` matched `OD2_VERIFY_ONLY` against the file name with `.sh`, so every scenario was skipped | match with and without the suffix | the 9j/9k scenarios appear in the parallel logs |

## Limits found, not fixed

* The prison doors (class 434) of Rescue on Mount Arreat are not in the DS1 monster lists of the generated Frigid Highlands
  (no `place_prisondoor`-like marker; 3 `Cage` objects exist), so the kill step of that quest cannot be played; the quest
  moves to its "entered the area" state and shows its pages, nothing more. The count on the log page is a fixed number.
* Outdoor levels of Acts 2 and 3 (Far Oasis, Lost City, Valley of Snakes, ...) have no waypoint object when the hero walks
  in; the panel lists them and travel there works (the hero lands at the map centre), but the way back needs a walk.
* The Act 3 temple entrances map by preset name, the Flayer Jungle caves by slot order (both UNVERIFIED); Gidbinn lies in
  the Flayer Jungle (the decoy object) and an altar object stands in Kurast Docktown, the DS1 object placement is taken as it is.
* The Siege on Harrogath log page has no text: the game's string table has no `qstsa5q1x` keys.
* Khalim's Will, the Golden Bird, Terror's End, the Hellforge, Prison of Ice, Betrayal and Rite of Passage were not played;
  Larzuk, Anya and Malah reward dialogs are another branch's work.
