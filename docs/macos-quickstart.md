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
| `OD2_AUTOMONSTER=skeleton1,3` | Spawn monsters (id, name or class number, optional count; a pack led by the first) 12 subtiles from the hero, let the hero fight them, log `MONSTER spawn/aggro/attack/hit/death/drop` lines and an `AUTOMONSTER summary`. `OD2_AUTOMONSTER_SECONDS` (default 30), `OD2_AUTOMONSTER_DIFF=0..2`, `OD2_AUTOMONSTER_PASSIVE=1` (hero does not fight back). `OD2_AUTOMONSTER=fallen1,pack` (or a superuniques key such as `The Countess,pack`) spawns one natural group planned from MinGrp/MaxGrp and the minion columns, logs `MONSTER pack ... composition=[fallen1:3 fallen1+:2]` and an `AUTOMONSTER world` line (packs, shots, shot_hits, blocked_steps, max_stack, hit_recoveries); `OD2_AUTOMONSTER_AREA=<levels.txt id>` scales non-boss monsters to that area's MonLvl. |
| `OD2_AUTOTEST_MUTE=1` | Do not play sound during the autotest. |
| `OD2_AUTOTRADE=Akara,Charsi` | Open each Act 1 vendor's trade window, log the stock with computed buy prices, run one scripted buy and sell (and a repair for Charsi) with the gold before and after, then restore the gold. `OD2_AUTOTRADE_SEED=<n>` fixes the stock, `OD2_AUTOTRADE_LEVEL=<n>` generates it for that hero level (a high level only gets magic items). |
| `OD2_AUTOPANEL=stash,cube,belt,inventory` | Open each container panel and log its items with grid positions (`AUTOPANEL panel=stash item #n code= x= y= w= h=`), then check that every saved item rebuilds identically from its spec. Compare with the save's own parse (nokkasorc.json: alt_position_id 1 inventory, 4 cube, 5 stash). `OD2_AUTOPANEL_HOLD=<s>` keeps it open. Scripts can use `panel:stash`, `panel:cube`, `panel:belt` too. |
| `OD2_AUTOSTASH=1` | Walk to the town stash object (objects.txt id 267) as a click does and log whether the stash opened. |
| `OD2_AUTOBELT=1,2,3,4` | Drink the belt potion of each column (hotkeys 1 to 4) at 40% life and mana and log the restored amounts; nothing is saved. |
| `OD2_AUTOEXIT=1` | Quit when the autotest finishes. |

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
| `exit` | Finish; with `OD2_AUTOEXIT=1` the process exits 0 on PASS, 1 on FAIL. |

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

