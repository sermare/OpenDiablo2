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

| # | What a player saw | Root cause | Fix |
|---|---|---|---|
| 1 | The gate of the Rogue Encampment leads nowhere; walking to the edge of Blood Moor does nothing | no code changed the level at a seamless outdoor border; with `OD2_REALMAPS=1` the town was still the old overworld with a fake wilderness | `d2level.EdgeExit/EdgeArrival` (+tests), `MapEngine.World`, `generateRealTown` (preset by `TownFile`), `advanceEdges` |
| 2 | `LEVEL 1:` logged while standing in Blood Moor after `OD2_AUTOLEVEL=2` | the client reported the Rogue Encampment as its level at start | `d2client.AddPlayer` uses the start level |
| 3 | Entering the Den of Evil did not move the Den quest (only scripted runs did) | the quest system only learned of area changes from `OD2_AUTOQUEST` | `afterLevelBuilt` calls `questArea` for every level change |
| 4 | After Talk the NPC menu pops up again at once (and the greeting plays again); the Talk submenu with the quest topic closes the moment it opens | `onNPCMenuChoice` kept `npcTarget` and closed the topic menu that `questTalk` had just opened | Talk closes the menu first and forgets the NPC unless a topic menu is open |
| 5 | The hero gives up on a far stair/item/chest after 12 s although he is walking | the time-out counted seconds, not progress | give-up clock restarts when the hero gets closer (`walkProgressStep`; unit test `TestGroundProgress...`) |
| 6 | The cave entrance of Blood Moor is not an exit; the tile styles of the other levels' stairs are not the LvlWarp ids | `Destination(level, style)` assumed style == LvlWarp id | cave entrance presets are resolved by their path (`CaveEntranceDestination`), dungeon tiles by the observed style rule (`TileDestination`, tests) |
| 7 | Blood Moor, Cold Plains ... and the Den have no monsters | outdoor DS1 presets carry no monster markers; `PopulateRoom` was never called | `populateLevel` fills blocks of 16x16 tiles from the Levels.txt monster list (`mon1..`, density), not near the arrival, never where the hero cannot walk (`MapEngine.CanWalkTo`), at the hero's difficulty |
| 8 | Zombie gold drops are an item called `gld` that lands in the inventory grid | `Director.dropLoot` used `DropItems`, which turns gold into an item | `DropLoot` + `NewGoldPile` |
| 9 | Experience grows (606/500) but the hero never reaches level 2; the quest skill point never shows | nothing promoted the hero: only the `levelup` console command | `advanceHeroLevel` (`levelsGained` tested): level, skill/stat points, full life and mana, sound, save |
| 10 | A new character has no potions or scrolls, so a Den fight at 40 life is lost | the new character file has no items and the import left it so | `StartingContainers` from CharStats.txt for a hero that was never played (`freshHero`) |
| 11 | A hero who dies in the Den respawns **in the Den map** at the town coordinates; the next level change crashes the animation (`SODDstf.COF`) | the death system assumed the town map was loaded | `respawnLevel` builds the town of the act, the corpse waits in the level where it lies and is placed when that level is built again |
| 12 | The sample hero (level 94) dies at once | the sample `.d2s` is a dead hardcore character with 0 life | `scripts/d2s-revive.go` makes a living copy for the playtest |
| 13 | Monsters in Hell for a Normal hero and the reverse | the director's difficulty came only from an env var | with `OD2_REALMAPS=1` it follows the hero (`gameClient.Difficulty`), also the death penalty |

(Commits of the fixes are on the branch; see `git log feat/act1-playthrough`.)

## Limits found, not fixed

* Items created in the game (loot, starting potions) are not written into the `.d2s` (the exporter never
  fabricates item bits); they live in the `.od2` save only. After a reload from the `.d2s` the hero has
  the items of the file.
* Levels are rebuilt at every level change: a cleared Blood Moor is populated again when the hero returns.
* Levels 13-16 and 37 (cave level 2 and deeper in the first caves, the Tristram area) are not generated by
  the maze DRLG port yet (`OD2_AUTOLEVEL=13` falls back to the town), so the game cannot be played past
  the first cave level of the wilderness caves. Act 2 and later are not built at all.
* The hero's Fire Bolt at level 1 does not exist (a new Sorceress has 0 skill points; the real one starts with
  Fire Bolt level 1 through `StartSkill`; not changed).
* Outdoor tiles are the approximation documented in `drlgoutdoor/doc.go`; the town preset's border towards
  Blood Moor is the first walkable border cell, the original blends the two presets.
