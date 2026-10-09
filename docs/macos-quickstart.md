# OpenDiablo2 on macOS (Apple Silicon) with real Diablo II 1.14b data

This fork runs natively on an M-series Mac. You need your own copy of Diablo II
and Lord of Destruction; no game files are included or distributed here.

## 1. Get the game data

OpenDiablo2 reads the original MPQ archives. On a Mac the easiest way to obtain
them is Blizzard's official downloader, run once under Wine:

1. Download Blizzard's Diablo II (and Lord of Destruction) installers from your
   Battle.net account. They are Windows `.exe` downloaders.
2. Install Wine (`brew install --cask wine-stable`) and run each downloader in a
   fresh prefix: `WINEPREFIX=~/.wine-d2 wine Downloader_Diablo2_enUS.exe`, then
   run the generated `Installer.exe` (and the Lord of Destruction one) the same
   way, installing to the same folder.
3. You should now have `d2data.mpq`, `d2exp.mpq`, `d2char.mpq`, `d2music.mpq`,
   `d2sfx.mpq`, `d2speech.mpq`, `d2video.mpq`, `d2xmusic.mpq`, `d2xtalk.mpq`,
   `d2xvideo.mpq` and `patch_d2.mpq` in
   `~/.wine-d2/drive_c/Program Files (x86)/Diablo II/`.

## 2. Build

```sh
brew install go
go build -o od2 .
```

## 2b. Or build the app bundle

```sh
ZIP=1 scripts/make-app.sh      # dist/OpenDiablo2.app and dist/OpenDiablo2-<version>-macos-arm64.zip
```

The bundle is arm64, ad-hoc signed and contains no game files. The
`Package macOS app` GitHub Actions workflow runs the same command and keeps the
zip as a workflow artifact only (Actions tab, run, Artifacts; kept 14 days); no
release is published. Unzip it, right-click the app and choose Open the first
time (it is not notarised), then point it at your own Diablo II folder when it
asks (or set `MpqPath` as in step 3). Crash logs go to
`~/Library/Logs/OpenDiablo2/OpenDiablo2.log`.

## 3. Configure

Run `./od2` once; it writes
`~/Library/Application Support/OpenDiablo2/config.json`. Set `MpqPath` to the
folder with the archives and list the MPQs in `MpqLoadOrder` (patch first):

```json
"MpqLoadOrder": ["patch_d2.mpq", "d2exp.mpq", "d2xmusic.mpq", "d2xtalk.mpq",
  "d2xvideo.mpq", "d2data.mpq", "d2char.mpq", "d2music.mpq", "d2sfx.mpq",
  "d2video.mpq", "d2speech.mpq"],
"MpqPath": "/Users/you/.wine-d2/drive_c/Program Files (x86)/Diablo II/"
```

## Testing without the UI

Some environment variables make changes testable without clicking through the
menus (launch from a GUI session, e.g. `open script.command`; running the binary
from a non-GUI shell fails with a Cocoa display error):

| Variable | Effect |
|---|---|
| `OD2_AUTOGAME=<file>` | Start that character directly. A `.d2s` file is imported first. |
| `OD2_D2S_DIR=<dir>` | Import real `.d2s` characters from a folder into the character list. |
| `OD2_AUTOTALK=Warriv,Akara` | After a few seconds, resolve and log each NPC's greeting voice line. |
| `OD2_AUTOSOUND=akara_greeting_1,3480` | After a few seconds, play each Sounds.txt handle or index through the voice bank and log `AUTOSOUND` lines (row, file, priority, decision). Silent with `OD2_AUTOTEST_MUTE`. |
| `OD2_AUTOMONSTER=skeleton1,3` | Spawn monsters (id, name or class number, optional count; a pack led by the first) 12 subtiles from the hero, let the hero fight them, log `MONSTER spawn/aggro/attack/hit/death/drop` lines and an `AUTOMONSTER summary`. `OD2_AUTOMONSTER_SECONDS` (default 30), `OD2_AUTOMONSTER_DIFF=0..2`, `OD2_AUTOMONSTER_PASSIVE=1` (hero does not fight back). |
| `OD2_AUTOMERC=1` | Mercenary scenario: uses the save's merc or hires the first affordable offer of Kashya (logging `MERC offers/offer/hire`), spawns hostile monsters (`OD2_AUTOMERC=skeleton1,4`: id and count), lets the merc fight (hero stands by) and logs `MERC spawn/attack/skill/death/revive/exp/levelup`, then walks the hero away (`AUTOMERC follow dist=`) and prints an `AUTOMERC summary`. `OD2_AUTOMERC_KILL=1` kills and revives the merc (`=dead` leaves it dead so the exported .d2s shows the dead flag), `OD2_AUTOMERC_HEROLEVEL`, `OD2_AUTOMERC_EXP`, `OD2_AUTOMERC_SECONDS`. Also `OD2_AUTOMENU=Kashya OD2_AUTOMENU_CHOOSE=Hire` logs the offer list. |
| `OD2_AUTOAI=Vulture,state=fear+confuse+charm` | AI archetype / forced-state scenario: spawns one monster of an AI archetype (a monai name such as Summoner, Duriel, Diablo) or of a monster class (`skeleton1`) 12 subtiles from the hero, lets its own AI run, then injects each listed state (fear, blind, taunt, confuse, attract, charm) with the console command `forcestate <monster id> <state> <frames>` and logs `AUTOAI start/inject/summary`, `MONSTER state` and `MONSTER aistate ... from= to=` (AI transitions) plus monster-versus-monster `MONSTER attack ... target=name(id)` lines; hostile skeletons stand by for the confuse/attract/charm states. The hero is a spectator that cannot die. `OD2_AUTOAI_SECONDS` bounds the run. |
| `OD2_AUTOAMBIENT=<level id\|name>` | Put the sound environment of that Levels.txt row in place and visit day phases (`OD2_AUTOAMBIENT_PHASES=2,4`, `OD2_AUTOAMBIENT_SECONDS=30` each, `OD2_AUTOAMBIENT_SPEED=2` scales event timers). Logs `AUTOAMBIENT`/`AMBIENT` lines (SoundEnviron row, phase, music, ambience, scheduled events) and `SOUNDAT` lines (distance, gain, volume, pan of each positional sound). |
| `OD2_AUTOMONSTER_FAR=<subtiles>` | With `OD2_AUTOMONSTER`, put every second monster on a far ring; `SOUNDAT` lines show their sounds' volume and pan as they approach. |
| `OD2_AUTOCAST=Fire Bolt,5` | With `OD2_AUTOMONSTER` set: the hero casts the named skill (name or id, optional cast count, default 5) at the nearest monster through the skill pipeline instead of swinging, logging `CAST start/do`, `MANA`, `MISSILE create/hit/end`, `DAMAGE`, `STATE` lines and an `AUTOCAST summary`. A skill the hero does not have is granted at `OD2_AUTOCAST_LEVEL` (default 10); `OD2_AUTOCAST_MANA=<n>` sets max and current mana. Town restrictions and arrow ammo are waived. |
| `OD2_AUTOCAST=<skill>[,n][;<skill>[,n]...]` | A list of skills (of any class) cast one after the other, each granted when the hero lacks it. Monsters are spawned again when all died; corpse skills get a corpse (one is made if none lies around); melee skills walk up to the target or, after 6 s, the hero is put next to it. Each skill ends with an `AUTOCAST skill_done` line (casts, missiles, hits, melee, area hits, kills, damage, poison damage) 2.5 s after its last cast, the scenario ends after the last one. New log lines: `STATE aura/storm/totem/hit/clear`, `SKILL area/strikes/splash`, `SUMMON`, `MINION spawn/attack`, `TRAP armed/fire`, `MOVE`, `DEFENSE`, `DAMAGE dot`. |
| `OD2_AUTOCAST_CLASS=<class>` | Start the scenario with a fresh hero of that class (barbarian, necromancer, paladin, assassin, sorceress, amazon, druid) instead of the save, at `OD2_AUTOCAST_CLVL` (default 18). The hero is saved to a temporary file. |
| `OD2_AUTODEATH=1` | Move the hero away from the town start, spawn Andariel (`OD2_AUTODEATH_MONSTER`) next to it and log the death (`DEATH hero=... exp_lost= gold_dropped=`), the respawn in town (`DEATH respawn ...`), the corpse recovery and a summary. `OD2_AUTODEATH_LEVEL=<n>` makes the hero that level (experience penalty), `OD2_AUTODEATH_HP=<n>` starts with n life, `OD2_AUTOMONSTER_DIFF` sets the difficulty (penalty 0/5/10%), `OD2_AUTODEATH_HARDCORE=1` plays a final hardcore death. With `OD2_D2S_WRITEBACK` the exported `.d2s` is parsed after every save (`died=`, `corpse=`). Rules and what is unverified: `d2core/d2hero/death.go`. |
| `OD2_AUTONEWCHAR=Druid[,hardcore][,classic][,ladder]` | Create a new character through the hero creation path, write its real `.d2s` (335 byte header, status new) to `OD2_D2S_WRITEBACK` without replacing existing files, log `NEWCHAR parse:` and, with `OD2_AUTONEWCHAR_REF=<real new character .d2s>`, `NEWCHAR diff` and `NEWCHAR oracle byte_identical=`. The hero list entry it makes is removed again. |
| `OD2_AUTOTEST_MUTE=1` | Do not play sound during the autotest. |
| `OD2_AUTOTRADE=Akara,Charsi` | Open each Act 1 vendor's trade window, log the stock with computed buy prices, run one scripted buy and sell (and a repair for Charsi) with the gold before and after, then restore the gold. `OD2_AUTOTRADE_SEED=<n>` fixes the stock, `OD2_AUTOTRADE_LEVEL=<n>` generates it for that hero level (a high level only gets magic items). |
| `OD2_AUTOGAMBLE=Gheed` | Open the vendor's gamble window and log `AUTOGAMBLE` lines: the 14 stock items (slot 0 ring, slot 1 amulet; unidentified name, quality, item level, gamble price), a scripted buy that reveals the item (gold before/after), a sell back, and a restock check (clock moved past four minutes). Uses `OD2_AUTOTRADE_SEED` / `OD2_AUTOTRADE_LEVEL`. |
| `OD2_AUTOIDENTIFY=1` | Open Deckard Cain's identify window (directly if he is not in the camp) and log `AUTOIDENTIFY` lines: a few inventory items are marked unidentified, then identified one by one (100 gold each), with a Tome and a Scroll of Identify, and all at once, with the gold before/after. |
| `OD2_AUTOPANEL=stash,cube,belt,inventory` | Open each container panel and log its items with grid positions (`AUTOPANEL panel=stash item #n code= x= y= w= h=`), then check that every saved item rebuilds identically from its spec. Compare with the save's own parse (nokkasorc.json: alt_position_id 1 inventory, 4 cube, 5 stash). `OD2_AUTOPANEL_HOLD=<s>` keeps it open. Scripts can use `panel:stash`, `panel:cube`, `panel:belt` too. |
| `OD2_AUTOSTASH=1` | Walk to the town stash object (objects.txt id 267) as a click does and log whether the stash opened. |
| `OD2_AUTOBELT=1,2,3,4` | Drink the belt potion of each column (hotkeys 1 to 4) at 40% life and mana and log the restored amounts; nothing is saved. |
| `OD2_AUTOQUEST=den\|burial\|tools\|cain\|tower\|andariel\|radament\|act1` (or a quest id 1-6, 8) | Run a scripted quest line through the quest system (`d2common/d2quest`): NPC talks, area changes, kills, object use and item pickups are injected, and the log shows `QUEST <quest> slot=.. bits before->after (reason) state=..` for every flag change, `QUEST SPEECH` (NPC, message id, mode, Sounds.txt row, string key, text) for every spoken line, `QUEST EFFECT` for rewards, `AUTOQUEST expect ...: ok\|FAIL` checks and a final `AUTOQUEST RESULT PASS\|FAIL`. The quest slots it exercises are reset in the hero's record first (its predecessors are marked done), so any save works; the quest bits reach the `.d2s` (`act1quests=[..]` in the `D2S EXPORT reparse` line). `OD2_AUTOQUEST_REAL=1` spawns real monsters for Den of Evil and lets the hero kill them (otherwise kill events are injected), `OD2_AUTOQUEST_AREA=<levels.txt id>` makes the map stand for that area (e.g. `OD2_REALMAPS=1 OD2_AUTOLEVEL=9 OD2_AUTOQUEST_AREA=8`), `OD2_AUTOQUEST_DIFF=0..2` picks the quest record. |
| `OD2_AUTOEXIT=1` | Quit when the autotest finishes. |
| `OD2_AUTOSPEED=<1..4>` | Run the game clock that many times faster (movement, fights, timers, script waits); a long scripted playthrough then fits a verification run. |
| `OD2_NOPOPULATE=1` / `OD2_POPULATE=1` or `=0` | With `OD2_REALMAPS=1` the levels are filled with their natural monsters from Levels.txt (`POPULATE` lines show what was placed) in a normal game and in scripts with `walkto:` or `kill:` steps; scripts that test something else, and the monster, merc, death and quest scenarios, keep the levels empty. `OD2_POPULATE` forces it on or off. |
| `OD2_AUTOSHOT=<file.png>` | Save the rendered game frame (UI included) to a PNG after `OD2_AUTOSHOT_SECONDS` (default 10) and log `AUTOSHOT saved`; quits afterwards with `OD2_AUTOEXIT` unless a script or monster test is running. More shots inside a script: `say:capframe <file.png>`. |
| `OD2_REALMAPS=1` + `OD2_AUTOLEVEL=<id>` | Start directly in that Act 1 maze level (cave 8-12, crypt 18-25, jail 29-31, catacombs 34-37) generated by the DRLG port from the hero's map seed (`OD2_AUTOMAP=<id>` also selects it). The DS1 rooms are stamped, empty space is blocked, the hero spawns at the entry stairs, DS1 monster placements become real monsters, and the hero's click paths route around walls. `OD2_AUTOMAP_ASCII=1` logs the room list and a walkability map; `LEVELSTATUS` lines report hero tile and monster counts. Exact equality with the real game's maps is not proven. |
| `OD2_REALMAPS=1` + `OD2_AUTOLEVEL=<2-7,17,39>` | Start in an Act 1 wilderness level (Blood Moor ... Tamoe Highland, Burial Grounds, Moo Moo Farm) built by the `drlgoutdoor` port: world layout, border/cliff/road/feature placement and the room list are proven equal to the real game (emulator golden); the tiles of every room (plain rooms and presets) are the exact records of the original's room tile build (DT1 library of the room, rarity pick with the room seed, tree markers; emulator golden for 240 levels), preset DS1 files still provide the entities and the cave-entrance marker tiles (see `d2common/d2drlg/drlgoutdoor/doc.go`). |
| `OD2_AUTOSHOT=<file.png>` | Save the game screen (ebiten image, no OS permission needed) after `OD2_AUTOSHOT_DELAY` seconds (default 10) in the world; with `OD2_AUTOEXIT` it then quits. |
| `OD2_AUTOTIME=<phase>[@degree]` | Force and freeze the day/night clock: phase 0..5 or `night`, `dawn`, `day`, `afternoon`, `dusk`, `midnight`; `@185` starts that many degrees into the day (128 ticks per degree). |
| `OD2_LIGHTING=0` | Turn the light map rendering (on by default) off. |

### Scripted scenarios (`OD2_AUTOSCRIPT`)

With `OD2_AUTOGAME` set, `OD2_AUTOSCRIPT` drives the hero without a mouse. It is
a semicolon-separated list of steps, run a few seconds after the game starts:

| Step | Effect |
|---|---|
| `wait:<seconds>` | Pause the script. |
| `move:<x>,<y>` | Walk to a world position (tile units) through the normal move path. |
| `move:npc=<name>` | Walk up to an NPC like a click would; its menu opens on arrival. |
| `cast:<skill>[@x,y]` | Cast a skill by its skills.txt name (target defaults to the hero). |
| `panel:inventory\|character\|skills\|quest\|close` | Open a panel, or close all. |
| `say:<command>` | Run an in-game console command. |
| `expect:log=<substring>` | Fail the run unless the game log already contains it. |
| `use:<object name or id>` | Walk to the nearest matching object (objects.txt name or id: `Waypoint`, `Door`, `Portal`, `119`) and operate it. Doors open/close and change collision, the waypoint opens its panel, portals change level. The next step waits until the walk or level change is over. |
| `waypoint:<level id>` | Choose that level in the open waypoint panel (greyed entries fail the step). Logs `WAYPOINT travel` and `LEVEL CHANGE`. |
| `travel:<act 1-5>` | Travel to the town of that act as the act's travel NPC/portal would (Warriv, Meshif, the Mephisto portal, Tyrael), with the quest and expansion rules of `d2level/acttravel.go`; a refused trip fails the step. Logs `TRAVEL`, `ACT CHANGE` (LoadAct packet), `ACT arrival`, `TOWN NPC`. |
| `refuse:<act 1-5>` | Passes only if the travel rules refuse that trip (`TRAVEL refused`). |

Debug console commands for act travel (use with `say:`): `completequest <act> <quest>`, `resetquests`, `travelfree 0|1` (skip the rules; needed for the trips back from acts 4 and 5, which have no NPC in the original), `travel <act>`.
| `expect:level=<id>` | Wait up to 4 s, then fail unless the hero is in that level; logs `AUTOSCRIPT level=<id> hero=(x,y)`. |
| `automap:on\|off\|toggle\|full\|mini\|stats` | Set the automap (Tab); `stats` logs `AUTOMAP state ... cells=N floor= wall= object=` and the markers. See `docs/automap.md`. |
| `skill:left=<name>` / `skill:right=<name>` | Select a skill on the left/right button (same checks as the popup: learned, not passive, left-capable on the left); logs `SKILLBAR select ...` and a `SKILLBAR state` line with the hotkeys. |
| `skill:popup=left\|right\|close` | Open the skill popup of a button (logs the grid: `SKILLBAR popup ... [Fire Ball@r1c1(lvl20) ...]`, row/column per skilldesc page/row/column). |
| `skill:hover=<name>` / `skill:click=<name>` | Put the mouse on an icon of the open popup (or skill tree) / click it (selects, plays the click, closes the popup). |
| `hotkey:F1=<name>[@left]` / `press:F1` | Assign a skill to a hotkey slot / press the key. `press` takes the same path as the keyboard: while an icon is hovered it assigns it, otherwise it selects the assigned skill (`SKILLBAR hotkey`, `SKILLBAR press`). |
| `skill:use=left\|right` | Cast the active left/right skill through the normal skill pipeline (as a click would). |
| `skill:spend=<name>` / `skill:nospend=<name>` | Put an unused skill point into a skill (skills.txt reqlevel/reqskill1-3 apply) / pass only if the point is refused. Console: `levelup <n>` grants level-ups (1 skill point and 5 stat points each). |
| `walkto:exit=<level>` | Walk to the exit that leads to that level (the shared border of two outdoor levels, a cave entrance or stair tile) and through it. Needs `OD2_REALMAPS=1`. The hero fights what attacks him on the way. Waits until the walk and the level change are over. |
| `walkto:object=<name>` | Walk to the nearest object of that name and operate it like a click (door, waypoint, chest, quest object...). |
| `kill:all[,<s>]` / `kill:near=<tiles>[,<s>]` | Fight every monster of the level / those near the hero (at most `<s>` seconds, default 120): melee, casts the best attack spell the hero has, drinks belt healing potions below 40% life, retries monsters it could not reach. Logs `KILL start/done`. |
| `loot:<tiles>[,<s>]` | Walk to the items on the ground within that distance, pick them up and store them (belt, inventory). Logs `LOOT`. |
| `until:<log substring>,<s>` | Wait up to `<s>` seconds for a log line logged since the last step that did something (not a wait or a check); fails the script if it does not come. |
| `menu:<row>` | Choose a row of the open NPC menu by label or action (`Talk`, `Trade`) or a Talk topic (`Den of Evil`). |
| `exit` | Finish; with `OD2_AUTOEXIT=1` the process exits 0 on PASS, 1 on FAIL. |

Console commands for these steps: `spawnportal <level>` (a portal object next to the hero), `setwaypoint <level> <0|1>`.
Without `OD2_REALMAPS=1` only the Rogue Encampment can be loaded; other waypoints are listed greyed out. With it the Act 1 maze levels (for example Jail Level 1 = 29, Catacombs Level 2 = 35) load.

Each step logs `AUTOSCRIPT step N: ...` and the end logs
`AUTOSCRIPT RESULT PASS` or `AUTOSCRIPT RESULT FAIL`. Example:

```sh
OD2_AUTOGAME=hero.d2s OD2_AUTOEXIT=1 OD2_AUTOSCRIPT='wait:1;move:npc=Akara;wait:8;expect:log=NPC menu opened;panel:inventory;exit' ./od2
```

The parser and state machine live in `d2game/d2autoscript` and are unit tested
without a display.

## Continuous integration

`.github/workflows/ci.yml` builds and tests on macOS arm64 (everything) and on
Linux (only the packages that need no display or game data, listed in the
workflow). New pure packages are gofmt-checked via `GOFMT_DIRS`.

## Real character saves

`d2common/d2fileformats/d2s` reads the header, quests, waypoints, NPC flags,
stats and skills of a Diablo II 1.14b `.d2s`; `HeroStateFactory.ImportD2S`
turns one into an OpenDiablo2 hero. Items are not imported yet.

## One-command verification

`scripts/verify.sh` builds the engine, runs every unit test, runs the real-save
oracle tests and starts a real `.d2s` character in the game to check the NPC
menus and a scripted scenario (walk to Akara, open panels), then prints `ALL CHECKS PASSED` or what failed. It needs a GUI session.

```sh
D2S_SAMPLE_BODY=/path/to/real.d2s D2S_SAMPLE_BODY_JSON=/path/to/expected.json \
D2_TABLES=/path/to/extracted/tables ./scripts/verify.sh
```

