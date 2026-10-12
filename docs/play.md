# Playing on a Mac

This page is the short version for someone who just wants to play. Step by step install:
[macos-quickstart.md](macos-quickstart.md). Everything below needs **your own** Diablo II + Lord of Destruction 1.14b
files; none are included. Early preview: nobody but the author has played it by hand for long, so expect rough edges
(see [KNOWN_GAPS.md](KNOWN_GAPS.md)).

## Start

1. Open the app (first time: right-click, Open, Open; it is not notarised).
2. First launch only: pick the folder with your `d2data.mpq`, `d2exp.mpq`, `patch_d2.mpq` and the other `.mpq` files.
   The app offers what it finds (`/Applications/Diablo II`, Wine prefixes, CrossOver bottles) and otherwise shows a
   folder picker. It only reads these files.
3. Choose a character: your real `.d2s` saves are found and imported (the originals are never modified), or create a
   new one. Pick a difficulty if more than one is unlocked.
4. The game starts **full screen**. Cmd+Enter toggles a window; `OD2_WINDOWED=1` starts windowed (see [ENV.md](ENV.md)).

## Controls

Mouse or trackpad with one button is enough: left click walks or uses the left skill, right click, two-finger click,
Control+click or Command+click uses the right skill, Shift+click casts standing still. Keys: `I` inventory, `C`
character, `T` skill tree, `L` quest log, `Tab` automap, `Esc` menu, F-keys skill hotkeys. Full list and rebinding:
[CONTROLS.md](CONTROLS.md). Console (`` ` ``): `fullscreen`, `musicvolume 0.5`, `soundvolume 1`, `windowscale 2`.

## Where things are

| What | Where |
|---|---|
| Settings | `~/Library/Application Support/OpenDiablo2/config.json` |
| Your characters in this game | `~/Library/Application Support/OpenDiablo2/Saves/` (the engine's own `.od2` plus an exported `.d2s`, with rotating backups `.d2s.bak1` to `.bak3`) |
| Log of this run | `~/Library/Logs/OpenDiablo2/OpenDiablo2.log` (previous run: `OpenDiablo2.previous.log`) when started from Finder |
| Your original game files and saves | Never written to |

Items you find in this game live in the engine's own save; the exporter only writes what it can prove, so open the same
hero in this game to keep them ([KNOWN_GAPS.md](KNOWN_GAPS.md#items-trade-and-saves)).

## Something is wrong

Use the report format in [PLAYTEST.md](PLAYTEST.md#how-to-report): act and level, what you did, what you saw, how bad,
whether you can repeat it, and attach the log file as **text**. The log holds game messages and file paths, no game data
and no key; read it before sending. No screenshots of game data.

## What is and is not confirmed

Scripted heroes play all five acts, in all seven classes, in the automated checks. A person has not confirmed sound, music or
a long by-hand play-through. Multiplayer is direct TCP between two copies of this game, not Battle.net.
