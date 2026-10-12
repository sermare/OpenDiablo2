<p align="center"><img src="docs/progress.svg?v=1791780876" alt="Progress: functions named, files studied, save sections verified, NPC features, roadmap" width="880"></p>
<p align="center"><img src="https://raw.githubusercontent.com/sermare/OpenDiablo2/progress-history/docs/progress-history.svg" alt="Progress over time: one line per part, snapshot every 10 minutes" width="880"></p>

[![CI](https://github.com/sermare/OpenDiablo2/actions/workflows/ci.yml/badge.svg)](https://github.com/sermare/OpenDiablo2/actions/workflows/ci.yml)

# OpenDiablo2 for macOS: a native Apple-Silicon Diablo II

A native macOS (Apple Silicon) build of **Diablo II: Lord of Destruction 1.14b**, built on the open-source
[OpenDiablo2](https://github.com/OpenDiablo2/OpenDiablo2) engine (Go + [Ebitengine](https://ebitengine.org)). You bring
**your own legitimate copy** of Diablo II and Lord of Destruction; the app reads your `.mpq` files and your `.d2s`
characters and never modifies them. **No game files, no Blizzard code and no decompiled output are in this repository.**

Diablo II has no native Mac version. This fork exists to make one. Behaviour is not guessed: it is ported from the
original `Game.exe` (1.14b), which is read (read-only) in Ghidra, and checked in the running engine against the game's
own data. The fork is developed autonomously by Claude (Anthropic's coding agent) at the request of the repository owner
([@sermare](https://github.com/sermare)); every claim below is meant to be checkable.

**Early preview, not a finished game.** Acts 1 to 5 can be played through by scripted heroes, but nobody except the
author has played it by hand for long. Expect rough edges. See [honest status](#honest-status).

> Not affiliated with or endorsed by Blizzard Entertainment. Diablo II is (c) Blizzard Entertainment.

## Contents

[What works](#what-works) · [Honest status](#honest-status) · [Requirements](#requirements) · [Build, run, make the app](#build-run-and-make-the-app) ·
[Config and display options](#config-and-display-options) · [Controls on a Mac](#controls-on-a-mac) · [How it is verified](#how-it-is-verified) ·
[How to help](#how-to-help) · [Docs map](#docs-map) · [Credits and licenses](#credits-and-licenses)

## What works

All of this is checked by a scripted scenario or unit test, not assumed (evidence per feature:
[docs/STATUS_MATRIX.md](docs/STATUS_MATRIX.md)):

* Boots on Apple Silicon with the real 1.14b + LoD data, shows the Rogue Encampment and the other four act towns, and
  travels between acts. Real level generation: the Go generator matches the original record for record on 427 levels.
* Real `.d2s` characters: read **and** written back (an unchanged save re-exports byte for byte), with items, quests,
  waypoints, mercenary and corpse. Your original files are only read; the engine saves its own copy.
* All seven classes, the status-effect engine, 177 of 210 skills with engine behaviour, auras, summons, traps.
* Monsters with the original AI (134 of 148 archetypes ported faithfully), bosses, Nightmare and Hell scaling.
* Quests of Acts 1 to 5 with the real NPC speech lines, vendors, gamble, stash, cube, trade.
* Multiplayer over TCP with the real packet framing (two instances see each other, walk, cast, chat).
* Display from 800x600 up to ultrawide; a Mac-style mouse scheme; a double-clickable `OpenDiablo2.app`.

## Honest status

The numbers come from [`docs/progress.json`](docs/progress.json) (what each bar means:
[docs/progress-targets.md](docs/progress-targets.md)); the graphic at the top is generated from it. Highlights:

| Area | Status |
|---|---|
| `Game.exe` functions named in Ghidra | 11,257 of 11,257 (a few names are marked unverified) |
| Level generator equal to the real game | 427 levels, 30,129 rooms, 0 mismatches |
| Acts played end to end by scripted heroes | 5 of 5, all 7 classes pass (35 of 35 in the class matrix) |
| Player skills with engine behaviour | 177 of 210 |
| Monster AI archetypes ported faithfully | 134 of 148 |
| In-game UI panels matching the original | 12 of 14 |
| Full verify | 39 of 39 scenarios in each of 6 jobs on the last landed tree |

What is **not** done or not confirmed (the full list is [docs/KNOWN_GAPS.md](docs/KNOWN_GAPS.md) and
[docs/HELP_WANTED.md](docs/HELP_WANTED.md)):

* **No human has confirmed sound, music or a full by-hand play-through of any act.** Scripted heroes pass; people have not.
* 33 skills and 14 monster AIs are approximate or open; some summon caps and a few numbers are labelled `UNVERIFIED`
  guesses in the code. `UNVERIFIED` always means a guess, never a claim.
* Cinematics (Bink video) are off by default and the audio sync was only checked by clock.
* Not notarised by Apple (right-click, Open on first launch); Apple Silicon only, macOS 14 or newer. No Windows or Linux build.
* Battle.net does not exist; multiplayer is direct TCP between instances.

The long history (every landing, dated) is in [docs/STATUS_LOG.md](docs/STATUS_LOG.md).

## Requirements

* A Mac with **Apple Silicon**, **macOS 14 or newer**.
* **Your own Diablo II + Lord of Destruction 1.14b files**: a folder holding `d2data.mpq`, `d2char.mpq`, `d2music.mpq`,
  `d2sfx.mpq`, `d2speech.mpq`, `d2video.mpq`, `patch_d2.mpq` plus `d2exp.mpq`, `d2xmusic.mpq`, `d2xtalk.mpq`,
  `d2xvideo.mpq`. The easiest way to get them on a Mac is Blizzard's official downloader run once under Wine:
  [docs/macos-quickstart.md](docs/macos-quickstart.md).
* Windows (amd64) is an experimental cross-compiled build (`scripts/make-windows.sh`, [docs/WINDOWS.md](docs/WINDOWS.md)); nobody has run it on Windows yet. Wave 36 rebuilt it from the current tree and fixed the Alt+Enter full screen toggle; a newer Ebiten (v2.10.5, also enables windows/arm64) builds on branch `feat/w36-ebiten-upgrade`, see [docs/WINDOWS.md](docs/WINDOWS.md).
* To build from source: `brew install go` and the Xcode command line tools (the engine needs CGO).
* Optional: a real `.d2s` character from the original game (found automatically, see below).

## Build, run and make the app

```sh
git clone https://github.com/sermare/OpenDiablo2 && cd OpenDiablo2
scripts/make-app.sh              # dist/OpenDiablo2.app (arm64, ad-hoc signed)
INSTALL=1 scripts/make-app.sh    # same, and copy it to /Applications
```

Double-click `OpenDiablo2.app` (first time: right-click, Open, then Open again; it is not notarised). On first launch it
looks for your game files in `/Applications/Diablo II`, `~/Library/Application Support/Diablo II`, Wine prefixes and
CrossOver bottles, offers what it finds, otherwise shows a folder picker, checks the files and writes
`~/Library/Application Support/OpenDiablo2/config.json`. It does not scan Documents, Desktop or Downloads.

From source without the app: `go build -o od2 . && ./od2` (needs a valid `config.json`; the app flow creates it).
`scripts/make-dmg.sh` and `scripts/release.sh` build a disk image; see [docs/macos-quickstart.md](docs/macos-quickstart.md).

Where things go: characters (`.d2s`) are found in `~/Library/Application Support/Diablo II*`, Wine prefixes and the game's
`Save` folder and imported read-only (`"D2SDir"` or `OD2_D2S_DIR` to point elsewhere); logs are in
`~/Library/Logs/OpenDiablo2/OpenDiablo2.log` when started from Finder. Config, saves and the log are the only things the app writes.

## Config and display options

Settings need no file editing: press the console key (`` ` ``) in game and type `fullscreen`, `musicvolume 0.5`,
`soundvolume 1` or `windowscale 2`; they are saved to `config.json`. Environment variables override:

| Variable | Effect |
|---|---|
| `OD2_DISPLAY=WxH` | pin the logical screen size, e.g. `1920x1080`, `3440x1440`, `5120x1440`; `800x600` is the legacy original. Default is 1512x982 (a MacBook). |
| `OD2_UI_SCALE=1..4` | integer interface scale (2 on a 4K screen shows the 1920x1080 layout doubled) |
| `OD2_BAR_FILL=none\|black\|tile` | what to draw beside the 800-px-wide bottom bar on a wide screen (default: none, the world shows there; tile repeats the bar art) |
| `OD2_CONFIG_DIR=<dir>` | move `config.json` and the saves folder (useful for testing) |
| `OD2_D2S_DIR=<dir>` | where to import your `.d2s` characters from |
| `OD2_NO_SETUP=1` | skip the first-run dialogs |
| `OD2_PLAY_VIDEOS=1` | play the Bink cinematics (off by default) |

Cmd+Enter toggles full screen. Everything about sizes, the 800x600 interface column and widescreen:
[docs/DISPLAY.md](docs/DISPLAY.md). The full variable list (including the test harness `OD2_AUTO*`):
[docs/TESTING.md](docs/TESTING.md) and [docs/macos-quickstart.md](docs/macos-quickstart.md).

## Controls on a Mac

| Input | Action |
|---|---|
| Left click on the ground | walk there (hold to keep walking) |
| Left click on a monster | use the **left skill** on it |
| **Shift** + left click | stand still and use the left skill at the cursor (ground spells like Fire Wall, Blizzard) |
| Right click, two-finger click, **Ctrl** + click, **Cmd** + click | use the **right skill** at the cursor |
| `Tab` / `I` / `C` / `T` / `Q` / `Esc` | automap / inventory / character / skill tree / quest log / menu |
| F-keys | skill hotkeys (16 slots, saved in the `.d2s`) |
| Cmd+Enter, Cmd+Q | full screen, quit (the hero is saved) |

The complete list is in [docs/macos-quickstart.md](docs/macos-quickstart.md#controls-mouse-trackpad-keyboard) and [docs/play.md](docs/play.md).

## How it is verified

"It works" means a scenario ran the real engine against the owner's real game files and a log line proved it.

* **Unit tests**: `D2_TABLES=<extracted tables> go test ./...` (tests that need game data skip themselves when it is absent,
  so CI on a machine without Diablo II is green but proves less).
* **`scripts/verify.sh`**: builds, runs the tests, the real-`.d2s` oracle tests and the save-back round trip, then every
  scenario in `scripts/verify.d/` (one small zsh file each, driven by the `OD2_AUTO*` scripting variables). `OD2_VERIFY_ONLY="<scenario>"`
  runs one; `scripts/verify_parallel.sh 6` runs the whole set in six jobs. A scenario that fails once is retried once.
* **Turbo mode** (`OD2_TURBO`) speeds up the long scripted playthroughs; the clock-sensitive checks stay in normal-speed scenarios.
* **Oracles, not guesses**: the level generator is compared with the original's output record for record, the RNG
  instruction by instruction, saves against an independent parser (`nokka/d2s`, MIT).

Details, variables, and how the evidence levels (`VERIFIED`, `inferred`, `UNVERIFIED`) are used:
[docs/TESTING.md](docs/TESTING.md), [docs/REVERSE_ENGINEERING.md](docs/REVERSE_ENGINEERING.md#4-evidence-levels).
Running the full verify needs a GUI session on a Mac with the game files; it is meant for contributors, not for players.

## How to help

* **Play it** and send a play report (Issues, "Play report"). A guided 30-minute checklist is in
  [docs/PLAYTEST.md](docs/PLAYTEST.md). Describe what you saw in words; do not attach screenshots of game data.
* **Pick a task**: [docs/HELP_WANTED.md](docs/HELP_WANTED.md) lists what is verified, guessed and parked, and where a Mac
  player or a Ghidra reader helps most.
* **Add a verify scenario**: [docs/ADDING_A_SCENARIO.md](docs/ADDING_A_SCENARIO.md).
* **Read the original binary**: [docs/GHIDRA_WORKFLOW.md](docs/GHIDRA_WORKFLOW.md) (addresses, names and constants in your own words, never Blizzard code).
* Rules, branches and the gate before a change is ready: [CONTRIBUTING.md](CONTRIBUTING.md). Never commit game files or saves.

## Docs map

[Architecture](docs/ARCHITECTURE.md) · [Testing](docs/TESTING.md) · [Display](docs/DISPLAY.md) · [macOS quickstart](docs/macos-quickstart.md) ·
[Play-test guide](docs/PLAYTEST.md) · [Known gaps](docs/KNOWN_GAPS.md) · [Status matrix](docs/STATUS_MATRIX.md) ·
[Reverse engineering](docs/REVERSE_ENGINEERING.md) · [FAQ](docs/faq.md) · [Status log](docs/STATUS_LOG.md) ·
[Original OpenDiablo2 README](docs/UPSTREAM_README.md)

## Credits and licenses

* **License: GPLv3** ([LICENSE](LICENSE)). Diablo II is (c) Blizzard Entertainment; this project is unaffiliated and needs
  your own legitimate copy. Reverse-engineering notes cover interoperability only.
* **[OpenDiablo2](https://github.com/OpenDiablo2/OpenDiablo2)** and its contributors ([CONTRIBUTORS](CONTRIBUTORS)): the engine this fork is built on.
* **[Ebitengine](https://ebitengine.org)** (Apache-2.0) for rendering and input; **[Ghidra](https://ghidra-sre.org)** (NSA, Apache-2.0) for reading the binary.
* **[nokka/d2s](https://github.com/nokka/d2s)** (MIT) is used as an independent reference parser to check save handling.
* **Bink 1 video and audio decoder** (`d2common/d2data/d2video`): the numeric format tables (scan orders, Huffman trees,
  patterns, quantisation matrices) come from the open [FFmpeg](https://ffmpeg.org) Bink decoder (libavcodec `bink.c` /
  `binkdata.h`, LGPL-2.1-or-later); the decoding code itself is original Go. No Blizzard or RAD code, and no game video or
  audio, is in this repository. The feature is off by default (`OD2_PLAY_VIDEOS=1`).
* No game data is in the repository: `scripts/check_repo_hygiene.sh` checks that.
