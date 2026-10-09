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
| `OD2_AUTOAMBIENT=<level id\|name>` | Put the sound environment of that Levels.txt row in place and visit day phases (`OD2_AUTOAMBIENT_PHASES=2,4`, `OD2_AUTOAMBIENT_SECONDS=30` each, `OD2_AUTOAMBIENT_SPEED=2` scales event timers). Logs `AUTOAMBIENT`/`AMBIENT` lines (SoundEnviron row, phase, music, ambience, scheduled events) and `SOUNDAT` lines (distance, gain, volume, pan of each positional sound). |
| `OD2_AUTOMONSTER_FAR=<subtiles>` | With `OD2_AUTOMONSTER`, put every second monster on a far ring; `SOUNDAT` lines show their sounds' volume and pan as they approach. |
| `OD2_AUTOCAST=Fire Bolt,5` | With `OD2_AUTOMONSTER` set: the hero casts the named skill (name or id, optional cast count, default 5) at the nearest monster through the skill pipeline instead of swinging, logging `CAST start/do`, `MANA`, `MISSILE create/hit/end`, `DAMAGE`, `STATE` lines and an `AUTOCAST summary`. A skill the hero does not have is granted at `OD2_AUTOCAST_LEVEL` (default 10); `OD2_AUTOCAST_MANA=<n>` sets max and current mana. Town restrictions and arrow ammo are waived. |
| `OD2_AUTODEATH=1` | Move the hero away from the town start, spawn Andariel (`OD2_AUTODEATH_MONSTER`) next to it and log the death (`DEATH hero=... exp_lost= gold_dropped=`), the respawn in town (`DEATH respawn ...`), the corpse recovery and a summary. `OD2_AUTODEATH_LEVEL=<n>` makes the hero that level (experience penalty), `OD2_AUTODEATH_HP=<n>` starts with n life, `OD2_AUTOMONSTER_DIFF` sets the difficulty (penalty 0/5/10%), `OD2_AUTODEATH_HARDCORE=1` plays a final hardcore death. With `OD2_D2S_WRITEBACK` the exported `.d2s` is parsed after every save (`died=`, `corpse=`). Rules and what is unverified: `d2core/d2hero/death.go`. |
| `OD2_AUTONEWCHAR=Druid[,hardcore][,classic][,ladder]` | Create a new character through the hero creation path, write its real `.d2s` (335 byte header, status new) to `OD2_D2S_WRITEBACK` without replacing existing files, log `NEWCHAR parse:` and, with `OD2_AUTONEWCHAR_REF=<real new character .d2s>`, `NEWCHAR diff` and `NEWCHAR oracle byte_identical=`. The hero list entry it makes is removed again. |
| `OD2_AUTOTEST_MUTE=1` | Do not play sound during the autotest. |
| `OD2_AUTOTRADE=Akara,Charsi` | Open each Act 1 vendor's trade window, log the stock with computed buy prices, run one scripted buy and sell (and a repair for Charsi) with the gold before and after, then restore the gold. `OD2_AUTOTRADE_SEED=<n>` fixes the stock, `OD2_AUTOTRADE_LEVEL=<n>` generates it for that hero level (a high level only gets magic items). |
| `OD2_AUTOGAMBLE=Gheed` | Open the vendor's gamble window and log `AUTOGAMBLE` lines: the 14 stock items (slot 0 ring, slot 1 amulet; unidentified name, quality, item level, gamble price), a scripted buy that reveals the item (gold before/after), a sell back, and a restock check (clock moved past four minutes). Uses `OD2_AUTOTRADE_SEED` / `OD2_AUTOTRADE_LEVEL`. |
| `OD2_AUTOIDENTIFY=1` | Open Deckard Cain's identify window (directly if he is not in the camp) and log `AUTOIDENTIFY` lines: a few inventory items are marked unidentified, then identified one by one (100 gold each), with a Tome and a Scroll of Identify, and all at once, with the gold before/after. |
| `OD2_AUTOPANEL=stash,cube,belt,inventory` | Open each container panel and log its items with grid positions (`AUTOPANEL panel=stash item #n code= x= y= w= h=`), then check that every saved item rebuilds identically from its spec. Compare with the save's own parse (nokkasorc.json: alt_position_id 1 inventory, 4 cube, 5 stash). `OD2_AUTOPANEL_HOLD=<s>` keeps it open. Scripts can use `panel:stash`, `panel:cube`, `panel:belt` too. |
| `OD2_AUTOSTASH=1` | Walk to the town stash object (objects.txt id 267) as a click does and log whether the stash opened. |
| `OD2_AUTOBELT=1,2,3,4` | Drink the belt potion of each column (hotkeys 1 to 4) at 40% life and mana and log the restored amounts; nothing is saved. |
| `OD2_AUTOEXIT=1` | Quit when the autotest finishes. |
| `OD2_AUTOSHOT=<file.png>` | Save the rendered game frame (UI included) to a PNG after `OD2_AUTOSHOT_SECONDS` (default 10) and log `AUTOSHOT saved`; quits afterwards with `OD2_AUTOEXIT` unless a script or monster test is running. More shots inside a script: `say:capframe <file.png>`. |
| `OD2_REALMAPS=1` + `OD2_AUTOLEVEL=<id>` | Start directly in that Act 1 maze level (cave 8-12, crypt 18-25, jail 29-31, catacombs 34-37) generated by the DRLG port from the hero's map seed (`OD2_AUTOMAP=<id>` also selects it). The DS1 rooms are stamped, empty space is blocked, the hero spawns at the entry stairs, DS1 monster placements become real monsters, and the hero's click paths route around walls. `OD2_AUTOMAP_ASCII=1` logs the room list and a walkability map; `LEVELSTATUS` lines report hero tile and monster counts. Exact equality with the real game's maps is not proven. |
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
| `expect:level=<id>` | Wait up to 4 s, then fail unless the hero is in that level; logs `AUTOSCRIPT level=<id> hero=(x,y)`. |
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

