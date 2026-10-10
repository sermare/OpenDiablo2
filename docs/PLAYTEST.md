# Play-test guide

This page has two parts. **Part 1 is for anyone who wants to help test** (no programming needed). **Part 2** is the
developers' playthrough log, kept for reference; skip it if you are only testing.

# Part 1: Guided 30-minute play-test

## Before you start

* Install and first launch: [macos-quickstart.md](macos-quickstart.md) (drag the app to Applications, right-click, Open,
  point it at your own Diablo II folder). The game starts with real maps; you change nothing.
* Read [KNOWN_GAPS.md](KNOWN_GAPS.md) once. Anything listed there is already known; you do not need to report it again
  (but tell us if it is worse than described).
* Keep a text file or notes app open to write findings as you go. Do not attach game screenshots or game files; describe
  what you saw (see "How to report" below).
* Use a throw-away hero (a new character is fine). The game saves your hero as you play; your original Diablo II
  saves are only read, never changed.

Useful keys: `Tab` automap, `I` inventory, `C` character, `T` skill tree, `Q` quest log, `Esc` menu, Cmd+Enter full screen,
Cmd+Q quit. A trackpad right click is a two-finger click or Control+click. Full list in the quickstart.

Helper commands (press `` ` `` to open the console, type, press Enter) so you do not need to play for hours to reach later acts:

| Command | What it does |
|---|---|
| `levelup 10` | Grants 10 levels (1 skill point and 5 stat points each). Spend them so your hero can fight. |
| `completequest <act> <quest>` | Marks a quest done so that travel to the next act is allowed (for example `completequest 1 6`). |
| `travel <act>` | Go to the town of that act (needs the quests above, or `travelfree 1` to skip the rules). |
| `setwaypoint <level id> 1` | Activate a waypoint. |
| `spawnportal <level id>` | Make a portal to that level next to you. |

These are test tools; using them is expected and not a bug.

## How the 30 minutes are split

| Minutes | Act | Where |
|---|---|---|
| 0-6 | Act 1 | Rogue Encampment, Blood Moor, Cold Plains, a cave, Den of Evil |
| 6-12 | Act 2 | Lut Gholein, Rocky Waste, Dry Hills, a tomb or sewer |
| 12-18 | Act 3 | Kurast Docks, jungle, Kurast, a dungeon |
| 18-24 | Act 4 | Pandemonium Fortress, Plains of Despair, City of the Damned, River of Flame |
| 24-30 | Act 5 | Harrogath, Bloody Foothills, Frigid Highlands, an ice cave |

Take the same quick look in every act. At each stop answer these four questions and write down only the "no" answers:

1. **Ground and walls.** Do floors and walls look like one connected place? Look for wrong or mismatched tiles, patches of
   a different tile set, floating walls, black holes inside the walkable area, bright seams along room edges.
2. **Walking.** Can you walk everywhere that looks open? Are you ever stuck on empty floor, or able to walk through a
   wall? Do exits, stairs, doors and waypoints take you where the name says?
3. **Monsters and loot.** Are there monsters (packs, a few loners)? Do they come at you, attack, die and drop things?
   Are any monsters standing still, in walls, or in places you cannot reach?
4. **Sound and music.** Is there music and ambient sound? Do NPCs speak when clicked? Do monsters, spells, doors and
   footsteps make sound? Does it stutter, cut off or stay silent?

### Act 1 (about 6 minutes)

1. Walk around the Rogue Encampment. Click Akara, Charsi, Gheed, Kashya, Warriv: each should speak and open a menu.
2. Leave through the gate to Blood Moor, then Cold Plains. Fight a few monsters, pick up something, look at the automap.
3. Enter the Den of Evil cave (or any cave) and go down one level. Caves are where tile problems are most likely.
4. Talk to Akara again after the Den: she should react to the quest.

Look for: tile seams along cave walls, the town gate and the first outdoor border, cave levels with very few or zero
monsters, Quill Rats or other monsters that never attack, no champion or unique (named, coloured) packs ever
appearing (known gap), music that does not change between town and field.

### Act 2 (about 6 minutes)

1. `levelup 10`, `completequest 1 6`, `travel 2` (or take Warriv's boat after Andariel). Walk Lut Gholein: Fara, Atma, Drognan, Greiz.
2. Go out of the city gate into Rocky Waste and Dry Hills.
3. Enter a tomb or the sewers and go in two levels deep.

Look for: patches of wrong ground in the desert, a town that is too dark at night, the gate facing the wrong way,
monsters on unreachable islands (they are removed on purpose, so a bare area is worth a note), stairs that lead to the wrong level.

### Act 3 (about 6 minutes)

1. `completequest 2 6`, `travel 3`. Talk to Alkor, Ormus, Hratli, Asheara, Meshif in Kurast Docks.
2. Walk the jungle: Spider Forest, Great Marsh, Flayer Jungle.
3. Enter a dungeon (Spider Cavern or a Flayer Dungeon).
4. If you reach Travincal, try the stairs down to the Durance of Hate.

Look for: jungle tile mix-ups, empty Kurast levels (Kurast Causeway has no monsters, a known gap), quest NPCs that only
talk, the Travincal stairs (they should be sealed until the Compelling Orb is smashed; Act 3 quests are not
playable yet, so being blocked there is expected; getting through without the orb is worth a note).

### Act 4 (about 6 minutes)

1. `completequest 3 6`, `travel 4`. Walk the Pandemonium Fortress, talk to Tyrael, Deckard Cain, Jamella, Halbu.
2. Walk the Outer Steppes, Plains of Despair, City of the Damned, River of Flame.
3. If you get far: the Chaos Sanctuary and the five seals.

Look for: lava drawn as walkable ground or ground drawn as lava, the Chaos Sanctuary shape, exits from the River of Flame,
bosses that appear at odd places, no way back to Act 3 (normal: the original has none).

### Act 5 (about 6 minutes)

1. `completequest 4 6` (or `travelfree 1`), `travel 5`. Walk Harrogath: Larzuk, Malah, Anya, Qual-Kehk, Nihlathak.
2. Go out to Bloody Foothills, Frigid Highlands, Arreat Plateau.
3. Enter an ice cave and take the stairs down.

Look for: Arreat Plateau and Bloody Foothills with few or no monsters (known), stairs that put you in a strange spot,
ice cave walls and floors that do not match.

### Always, in every act

* Open the inventory, character sheet and skill tree once. Is anything overlapping, cut off or in the wrong place?
* Press `Tab` for the automap: does it match what you walked?
* Quit with Cmd+Q and start again with the same hero. Are your gold, level, items and waypoints still there? (Items
  picked up in the game may not be written back to the original `.d2s`; see KNOWN_GAPS.)

## How to report

Write one short entry per finding. Use this shape (words only, no game files, no screenshots of game data):

```
Act / level:     Act 1, Cave Level 2
What I did:      walked from the entrance to the east side
What I saw:      a row of floor tiles of a different colour along the south wall
Kind:            tiles | missing monsters | walking/collision | audio | UI | crash | other
How bad:         cosmetic | annoying | blocks progress | crash
Can you repeat:  every time | sometimes | once
Hero:            Sorceress level 20, Normal
```

Kinds, with what to write:

* **Bad tiles:** the level name and roughly where (north/south of the entrance, near a door); the colour or shape that is wrong; whether it is along an edge or in the middle.
* **Missing monsters:** the level, how long you walked, how many monsters you met, whether any were frozen or in walls. "No monsters at all in the whole level" is a valuable report.
* **Audio:** what should have played (NPC voice, spell, footsteps, music, ambient), what you heard instead (silence, wrong sound, crackle, cut off), and whether it happens after switching windows.
* **Crash or freeze:** what you were doing in the last ten seconds.

For crashes and anything odd, attach the log file as text: `~/Library/Logs/OpenDiablo2/OpenDiablo2.log` (the run before is
`OpenDiablo2.previous.log`). The log contains game messages and file paths, no game data and no key. Open it in a text editor and
look it over before sending. Send reports to the maintainer
([@sermare](https://github.com/sermare)) as a GitHub issue on the fork, or however you were asked to.

---

# Part 2: Developer playthrough log (historical)

The text below is the working log of the scripted playthroughs. Some "does not do yet" lists in it are older than
the code; [KNOWN_GAPS.md](KNOWN_GAPS.md) is the current list.

## Act 1 playthrough log

Branch `feat/act1-playthrough`. The first hour of Act 1 is played by `OD2_AUTOSCRIPT`:

| Scenario | What it plays |
|---|---|
| `scripts/verify.d/9b-act1-playthrough.sh` | a brand new Sorceress (made by `OD2_AUTONEWCHAR`): Akara (intro, Den of Evil quest, topic), the gate to Blood Moor, fights and loot, the Den of Evil cave, clear it, back to Akara for the skill point, exit (saves the `.d2s`) |
| `scripts/verify.d/9c-act1-reload.sh` | loads the `.d2s` of 9b again: same hero, quests, waypoints; Akara has nothing more to say about the Den of Evil |
| `scripts/verify.d/9d-act1-sample-hero.sh` | the level 94 sample hero (a revived copy: the sample is a dead hardcore character, see `scripts/d2s-revive.go`) walks town, Blood Moor, Den, back |

By hand (the game needs a GUI session, see `docs/macos-quickstart.md`):

```sh
OD2_AUTOSPEED=3 OD2_AUTOGAME=hero.d2s OD2_AUTOEXIT=1 \
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
| 1 | The gate of the Rogue Encampment leads nowhere; walking to the edge of Blood Moor does nothing | no code changed the level at a seamless outdoor border; by default the town was still the old overworld with a fake wilderness | `d2level.EdgeExit/EdgeArrival` (+tests), `MapEngine.World`, `generateRealTown` (preset by `TownFile`), `advanceEdges` | a9833485 |
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


## Act 2 playthrough log

Branch `feat/act2-playthrough`. The start of Act 2 by default, played by `OD2_AUTOSCRIPT`:

| Scenario | What it plays |
|---|---|
| `scripts/verify.d/9e-act2-playthrough.sh` | the revived level 94 sample hero travels east (`say:completequest 1 6; travel:2`), talks to Warriv, Fara, Atma, Drognan and Greiz in Lut Gholein (the preset the Act 2 world layout chose), walks out of the gate into Rocky Waste (41) and Dry Hills (42) with fights and loot, enters the Halls of the Dead (56) through the tomb entrance, takes the stairs down to level 57 and up again, and walks back through the desert to town; the exported `.d2s` is an Act 2 save |
| `scripts/verify.d/9f-act2-lutn.sh` | `OD2_AUTOMAPSEED=1`: the other town variant (LutN, Rocky Waste to the north): out of the gate and back |

By hand:

```sh
OD2_AUTOSPEED=3 OD2_AUTOGAME=hero.d2s OD2_AUTOEXIT=1 \
  OD2_AUTOSCRIPT='wait:1;say:completequest 1 6;travel:2;expect:level=40;walkto:exit=41;expect:level=41;kill:near=30,45;loot:30,40;walkto:exit=42;expect:level=42;walkto:exit=56;expect:level=56;exit' ./od2
```

`OD2_AUTOMAPSEED=<n>` (new) replaces the game seed of the hero, to play the other world layouts.

## Bugs found by playing Act 2, and their fixes

| # | What a player saw | Root cause | Fix | Regression test |
|---|---|---|---|---|
| 17 | `walkto:exit=41` in Lut Gholein: "level 40 has no exit towards level 41"; the desert levels cannot be walked from one to the next | the seamless borders of the Act 2 chain 40-41-42-43-44-45 were not in the link table and the Act 2 world placement was never given to the map engine (no level rectangles) | the five links in `drlgEdges` (`d2level/links.go`), `act23Rects` hands the `PlaceAct2World` rectangles to `SetWorld` for the desert levels and the town | `TestAct2DesertEdges`, `TestAct2WorldBordersForEverySeed` (40 seeds, needs `D2_TABLES`) |
| 18 | Lut Gholein was a random one of its two presets in a 57x57 map without a world position: the gate did not face Rocky Waste | `GenerateActTown` rolled the file with the level seed; the world layout (`World23.TownFile`: LutW when Rocky Waste lies to the west, LutN when to the north) decides | by default the town uses the layout's file, its rectangle and `UseCollisionPaths`, like the Rogue Encampment | 9e, 9f (both variants walk out of the gate) |
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

## Act 4 and the start of Act 5 playthrough

Branch `feat/act45-playthrough`. `scripts/verify.d/9h-act45-playthrough.sh` plays Act 4 (Pandemonium Fortress, Outer
Steppes, Plains of Despair, City of the Damned, River of Flame, Chaos Sanctuary, and the waypoint back to the Fortress)
and the first outdoor levels of Act 5 (Harrogath, Bloody Foothills, Frigid Highlands, Arreat Plateau, the cave stairs
into the Crystalline Passage). `scripts/verify.d/9i-act5-caves-playthrough.sh` plays the ice caves on to the Worldstone
Keep (waypoint to the Crystalline Passage, Glacial Trail, Frozen Tundra, Ancients' Way, Arreat Summit, Worldstone Keep
1-3, Throne of Destruction; a waypoint hop from level 129 back to the Ancients' Way). Both use the revived level 94
sample hero with the act travel quest flags forced (`say:completequest`), default real maps, `OD2_AUTOSPEED=4`.

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

## Act 3 playthrough log

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
| 34 | `walkto:exit=76` in Kurast Docks: the gate leads nowhere; the jungle and Kurast levels cannot be walked from one to the next | the seamless borders of Act 3 (town - Spider Forest - Great Marsh - Flayer Jungle - Lower Kurast .. Travincal) were not in the link table, and the town had no world rectangle | five fixed links plus the three jungle pairs in `drlgEdges` (the jungle levels hang off each other at random; a border exists only where the rectangles of the seed touch), `GenerateActTown` hands the `PlaceAct3World` rectangles to the map engine for Kurast Docks | `TestAct3WorldBordersForEverySeed` (40 seeds, all of 75..83 reachable over borders; needs `D2_TABLES`), 9g |
| 35 | the scripted walk from the south end of the Great Marsh to Flayer Jungle left through the *west* border into Spider Forest | the marsh touches both neighbours and any border crossing counted while the hero walked | `advanceEdges` crosses only the border, and `advanceWarpUse` enters only the warp tile, the walk heads for (`edgeWanted`; a fight next to the Ruined Temple's entrance once sent the walk to Upper Kurast into the Disused Fane) | `TestEdgeWantedOnlyTheBorderOfTheWalk` |
| 36 | arriving in Upper Kurast from the Bazaar the hero cannot take a single step ("EXIT no way found") | the edge arrival keeps the world position; the approximated Kurast tiles put a roof there and the nearest free sub-tile was a 5x5 alcove closed on all sides | `MapEngine.NearestOpen`: the arrival moves to the nearest free sub-tile of a region of at least 600 sub-tiles (used by level changes and the provider default) | `TestNearestOpenCellSkipsClosedPockets` |
| 37 | the entrances of Spider Forest, Flayer Jungle, Kurast and Travincal lead to level 0 | their special tile styles are not LvlWarp ids and `SingleTileDestination` is Act 2 only (Spider Forest has two caves, Bazaar a sewer and two temples) | the style of the entrance tile is the Vis slot of Levels.txt (OBSERVED, exe rule UNVERIFIED): `d2level.Act3SlotDestination`, also used by `TileDestination` | `TestAct3SlotDestination`, `TestAct3SlotsMatchLevelsTxt` (compares with the real Levels.txt), 9g (`exit tile ... leads to level 0` fails it) |
| 38 | Spider Cavern, Flayer Dungeons, Sewers, the Durance of Hate: "no level provider" / "the engine cannot load that level yet" | the maze provider only probed levels up to 72 | `maxMazeLevel = 102` | `TestAct3DungeonsAreBuildable` (every Act 3 maze level generates), 9g |
| 39 | the six temples (94..99), Sewers 2, the treasure rooms and the Durance level 3 are not built either | they are DrlgType 2 (a single preset DS1), not mazes | `isAct3Preset` joins `isPresetLevel` (`GenerateRealPreset`), with its own `Params` (no world, rectangle from Levels.txt) | `TestAct3DungeonsAreBuildable` (each DrlgType 2 level of 84..102 is a preset level), 9g (Ruined Temple, Forgotten Reliquary) |
| 40 | in Spider Cavern the exit tile does nothing: the hero is locked in the cave | the lair presets carry the exit with style 1; `TileDestination` read style 1 as "second up exit or first down" and the cave has neither | a dungeon with a way out and no way down returns up for the low styles; the Act 3 "up" warp ids (52, 55, 58, 59, 62, 63, 65, 66) join `upWarps` | `TestAct3DungeonTileDestination` |
| 41 | Lower Kurast, the Bazaar, Upper Kurast and Travincal have not a single monster | `GroupsForRoom` truncates: a 16x16 block of a level with MonDen 325 holds 0.83 monsters, which became 0 groups in every block | `GroupsForRoomFrac`: where the rounding gives 0 the fraction is the chance of one group (levels that had groups are untouched) | `TestGroupsForRoomFrac`, 9g (79, 80, 81, 83 get monsters) |

## What Act 3 does not do yet

* The quests of Act 3 (Lam Esen's Tome, Khalim's Will, Blade of the Old Religion, the Golden Bird, Mephisto) are not played; the NPCs of Kurast Docks only talk (menus and greetings). Hratli is not in the DS1 and stands at a guessed place (`townExtras`).
* Kurast Causeway (82) has no natural monsters: no 16x16 block of the generated level reaches the walkable share of 40 % (the bridge between the canals). Travincal has its council monsters only as far as the DS1 names them.
* The tiles of the jungle and of Kurast keep the approximation of `drlgoutdoor/doc.go`; the special (wall) tiles of the Kurast presets with styles 8, 9, 12, 16 are not exits and stay unresolved on purpose (38 of them in Upper Kurast).
* The Swampy Pits and Flayer Dungeons (86..91) are mazes with DS1 monsters: the natural groups skip their blocks, the second level of each pit (87, 89) and the treasure rooms were generated but are not part of 9g.
* Durance of Hate 2 (101) is generated but not walked; level 102 (Durance 3, Mephisto) is a preset level that nothing enters yet, and the red portal to the Pandemonium Fortress is only the act-travel rule.
* Sewers 1 has two entrances in each of Kurast Bazaar and Upper Kurast (slots 0 and 1); coming back up, the hero arrives at the first one (which one the original uses is UNVERIFIED).
* The Great Marsh and the jungle lose a third of their monsters to the "hero cannot walk there" filter (islands of floor between water and undergrowth).

## Quest rewards through the NPCs (scenario 9k)

Branch `feat/reward-npc-ui`. Larzuk (sockets), Anya (personalise) and Charsi (imbue) take the item the hero holds on the cursor when the hero clicks them (`OnItemDropOnNPC`; scripts use `say:pickitem <code>` then `move:npc=`); while a reward is owed their menu also gets a row (Add Sockets / Personalize / Imbue) that opens the inventory (a row of this fork, UNVERIFIED against the exe). Akara gets the "Reset Stat/Skill Points" row (string 0x2ba0, gated by quest slot 41). Debug aids: `questpending <act> <quest>`, `giveitemq`, `pickitem`, `putitem`, `freeinv`. Not persisted: the owed sockets/personalisation (only the imbue and the reset follow the quest record). Hratli and Jerhyn have no travel rows in the exe table; act travel stays Warriv, Meshif, the Mephisto portal and Tyrael talk.
