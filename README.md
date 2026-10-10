<p align="center"><img src="docs/progress.svg?v=1791626416" alt="Progress: functions named, files studied, save sections verified, NPC features, roadmap" width="880"></p>
<p align="center"><img src="https://raw.githubusercontent.com/sermare/OpenDiablo2/progress-history/docs/progress-history.svg" alt="Progress over time: one line per part, snapshot every 10 minutes" width="880"></p>

[![CI](https://github.com/sermare/OpenDiablo2/actions/workflows/ci.yml/badge.svg)](https://github.com/sermare/OpenDiablo2/actions/workflows/ci.yml)

# OpenDiablo2 — native macOS fork, driven by Claude

> **Who is working here.** This fork is being developed **autonomously by Claude** (Anthropic's AI coding agent),
> at the request of the repository owner ([@sermare](https://github.com/sermare)), who wants a **native Mac version
> of Diablo II** (the game is not available natively on macOS). Claude sets the priorities, runs teams of parallel
> agents, reverse engineers the original `Game.exe` (1.14b) with Ghidra, writes the code, tests it against the real
> game's data and updates this page after every success. The owner is only asked for things Claude cannot do
> itself.
>
> This is a fork of [OpenDiablo2](https://github.com/OpenDiablo2/OpenDiablo2) (original README:
> [docs/UPSTREAM_README.md](docs/UPSTREAM_README.md)). **It is an early play-test preview** (Acts 1 to 5 can be walked, with known gaps) — see [Try it](#try-it-for-non-developers) and [docs/KNOWN_GAPS.md](docs/KNOWN_GAPS.md).
> You need your own copy of Diablo II + Lord of Destruction; **no game files are in this repo**.

> **Working on the code?** [CONTRIBUTING.md](CONTRIBUTING.md) · [Architecture](docs/ARCHITECTURE.md) ·
> [Testing](docs/TESTING.md) · [Reverse engineering](docs/REVERSE_ENGINEERING.md)

## Try it (for non-developers)

You do not need to read code. You need a Mac with Apple Silicon (macOS 14 or newer), your **own** copy of Diablo II
and Lord of Destruction, and about 30 minutes.

1. **Install.** Open the `OpenDiablo2-<version>-macos-arm64.dmg` you were given and drag OpenDiablo2 onto
   Applications. First launch: right-click the app, choose Open, then Open again (it is not notarised).
2. **Point it at your game files.** The app looks for your Diablo II folder and offers it; otherwise pick the folder
   that holds `d2data.mpq` and the other `.mpq` files. Your files are only read.
3. **Start the game.** Pick or create a hero and play. Real maps are the default; there is nothing to switch on.
4. **Play-test.** Follow the guided checklist in [docs/PLAYTEST.md](docs/PLAYTEST.md#part-1-guided-30-minute-play-test)
   (what to look for in each act and how to report what you find). Step by step install help:
   [docs/macos-quickstart.md](docs/macos-quickstart.md). What is known to be missing or wrong:
   [docs/KNOWN_GAPS.md](docs/KNOWN_GAPS.md).

No game files, screenshots of game data or CD keys belong in this repository or in your report; describe what you saw in words.

Quick start on a Mac: [docs/macos-quickstart.md](docs/macos-quickstart.md). The graphic above is generated from
[`docs/progress.json`](docs/progress.json) (what each bar measures: [docs/progress-targets.md](docs/progress-targets.md)) by [`scripts/make_progress_svg.py`](scripts/make_progress_svg.py).

## Status board

_Last updated: 2026-10-10 (after the pass6 landing; Game v1 complete 68%, 70% with held branches; third full verify run 63179: all jobs passed; earlier run 47389: 71 of 74 sections green, 3 red) · Claude updates this on every merged success._

### ✅ Working now (each one verified, not assumed)

| Done | Evidence |
|---|---|
| **Skills for all seven classes** (~130 skills): the status-effect engine (poison, burn, chill, stun, fear, curses), auras, summons (skeletons, golems, wolves, ravens), traps, Paladin/Druid/Assassin/Barbarian/Amazon/Necromancer/Sorceress castings | Scenario casts 30 skills across the 7 classes; several formulas spot-checked in the binary, others marked unverified |
| **Skill hotkeys and the skill selection screen**: 16 hotkey slots, the popup laid out like the real game, spending points with prerequisites, all saved in the `.d2s` | Scenario selects, assigns F-keys, spends points and checks the exported header; screenshots read |
| **Remaining boss and monster behaviours**: Summoner, Duriel, Mephisto, Diablo, Izual, Vulture, Baal minions and the forced states (fear, confuse, attract, charm) | Real Summoner skill slots confirmed in the binary; monsters fight monsters when charmed or confused |
| **Nightmare and Hell**: a difficulty screen (only unlocked difficulties), the real scaling tables (a Nightmare Fallen: level 36, defence 369, damage 13-27, 1,669 xp), per-difficulty quests and waypoints | Checked against the real tables; saved in the `.d2s` active-difficulty byte |
| **Multiplayer**: two instances join over TCP using the real game's packet framing (Huffman table read from the binary); both build the same level from the host's map seed, see each other, walk, cast and chat | Two-process scenario passes: both saw the other's name and position, walk/cast/chat arrived, clean leave |
| **The quest system**: the intro quests, all of Act 1 and Radament's Lair — states, NPC speech with the real voice lines, rewards, quest log, saved in the `.d2s` | A scripted Act 1 line passes 70 of 70 checks; Den of Evil with real killed cave monsters; the quest bits appear in the exported `.d2s` |
| Runs natively on Apple Silicon with the **real 1.14b + Lord of Destruction data** | Boots into Rogue Encampment from a real install; ragged rows in the game's `.txt` tables, optional LoD strings and missing music no longer crash it |
| Town warnings gone (`Unknown tile ID`, `invalid frame index`) | Combined build runs with **zero** warnings/errors in the log |
| **Click an NPC** → hero walks up → NPC **speaks their real voice line** | Greeting rows come from the game's `Sounds.txt`; the autotest resolves Warriv, Akara, Charsi, Kashya, Gheed |
| **Real NPC menu** (Talk / Trade / Repair / Gamble / Cancel) built from **the game's own menu table** | Log shows e.g. Akara `[talk trade cancel]`, Charsi `[talk trade/repair cancel]`, Gheed `[talk trade Gamble cancel]`; not yet checked visually |
| Greeting logic ported from the real picker (inactive group, time-of-day lines, no repeats) | Unit tests; Warriv (no plain hello row) now speaks |
| **Engine saves back to a real `.d2s`** (autosave every ~5.5 min, on exit): an unchanged export is byte-identical; edited gold/level/quests re-parse with a valid checksum | The original file is never modified; the in-game autosave test wrote `gold=31337 checksum=ok` |
| **Whole `.d2s` character save — read and WRITE**: header, quests, waypoints, stats, skills, **items**, corpse, mercenary | Parse → write of a real save is **byte-for-byte identical** (2,663 bytes); Checksum, all 16 attributes, 30 skills and **60/60 items** match a reference parser on a real level-94 save; header/body also on a second real save |
| **Import a real character into the engine — with her gear** | A level-94 Sorceress from a real `.d2s` loads, starts in town and wears her real Spired Helm, Archon Plate, Battle Boots, Light Gauntlets, Flail, Short Staff and Monarch |
| **Diablo II's own random number generator** (`d2rand`) and the level-seed hierarchy | Reverse engineered from the binary; tests use independent Python vectors; checked instruction-by-instruction against the real code: no differences |
| **Test without clicking** (`OD2_AUTOGAME`, `OD2_AUTOTALK`, `OD2_AUTOMENU`, …) | Lets the AI verify changes by itself; see the quickstart |
| Reverse-engineering map of the game | all 11,257 functions named in Ghidra (the last 22 Warden/module names are partly unverified); 277 source files indexed; notes on units, saves, packets, rendering, sound, NPC menu, **skills and combat formulas** |

### 🔧 In progress right now (agents run in waves; the machine is the limit)

| Work item | Where |
|---|---|
| **Acts 2 to 5 towns and travel between acts** (merging with the latest code) | branch `feat/act-towns` |
| **A scripted first hour of Act 1**, finding and fixing the seams between systems | branch `feat/act1-playthrough` |
| **Exact outdoor tile ids**, proven against the real game | branch `feat/drlg-tiles` |
| **Act 4 and 5 generators in Go** (spec verified against the real game) | next |
| Research: Act 2 desert and Act 3 jungle generators | RE notes: `drlg-act23-outdoor` |
| Imported-character UI, last merge pending | `feat/imported-hero-ui` |

### 🎯 Plan and priorities (set by Claude)

1. **Real characters load fully** — items, corpse, mercenary — so a player's actual saves work in the engine.
2. **NPCs work like the real game** — menu ✓, voice greeting logic ✓, trade, quest dialogue.
3. **Combat is correct** — real skill, mana, hit, damage and defence formulas (now documented and verified).
4. **Real maps** — reproduce the original level generator from a seed.
5. **Loot and monsters** — item generation, monster AI, quests.
6. **Protocol and polish** — packet layer parity, rendering parity (lighting, palette shifts).

### 📋 Backlog (not started)

Missiles, collision and pathing (server simulation) · hirelings · stash/cube · automap · sound engine parity ·
server session core · D2Common data tables · key bindings from `default.key` · loose-file fallback for mods ·
"welcome back" greeting flag · day/night phase.

## Install on a Mac

You need your own copy of Diablo II (and Lord of Destruction). No game files are included.

1. **Get the game files.** The easiest way is Blizzard's official downloader run once under Wine; the steps are in
   [docs/macos-quickstart.md](docs/macos-quickstart.md). You end up with a folder holding `d2data.mpq`, `d2char.mpq`,
   `d2music.mpq`, `d2sfx.mpq`, `d2speech.mpq`, `d2video.mpq`, `patch_d2.mpq` (plus `d2exp.mpq`, `d2xmusic.mpq`,
   `d2xtalk.mpq`, `d2xvideo.mpq` for Lord of Destruction).
2. **Build the app** (needs `brew install go` and the Xcode command line tools, once):
   ```sh
   scripts/make-app.sh          # creates dist/OpenDiablo2.app (arm64, ad-hoc signed)
   INSTALL=1 scripts/make-app.sh   # same, and copies it to /Applications
   ```
   `UNIVERSAL=1` also tries an Intel slice (best effort: the engine needs CGO).
3. **Double-click `OpenDiablo2.app`** (first time: right-click, Open, because it is not notarised).
   On first launch it looks for the game files in `/Applications/Diablo II`, `~/Library/Application Support/Diablo II`,
   Wine prefixes (`~/.wine*/drive_c/Program Files (x86)/Diablo II`), CrossOver bottles and similar, offers what it finds,
   and otherwise shows a folder picker. It checks the files and writes
   `~/Library/Application Support/OpenDiablo2/config.json` for you. Missing files give a clear dialog.
   It does not scan Documents, Desktop or Downloads (macOS would ask for permission); pick the folder by hand if needed.
4. **Your characters.** Real `.d2s` characters are found automatically in `~/Library/Application Support/Diablo II*`,
   Wine prefixes (`Saved Games/Diablo II`) and the game folder's `Save`, and imported into the character list. Originals are
   only read. To use another folder set `"D2SDir"` in config.json (or `OD2_D2S_DIR`).
5. **Settings** without editing files: press the console key (`` ` ``) in game and type `fullscreen`, `musicvolume 0.5`,
   `soundvolume 1`, or `windowscale 2` (start window 1x to 4x, applies next launch). They are saved to config.json.
6. **Logs and problems:** logs go to `~/Library/Logs/OpenDiablo2/OpenDiablo2.log` when started from Finder. If the game
   crashes a dialog shows the log path. Config, saves and the log are the only things the app writes.

Command line and `OD2_*` variable workflows are unchanged. `OD2_CONFIG_DIR=<dir>` moves config.json and Saves (for tests),
`OD2_NO_SETUP=1` skips the first-run dialogs.

## How this is built

- **Ghidra + an MCP server** let the AI read and name functions in the original binary. The decompiler had to be
  compiled natively for macOS.
- The original game leaves its own source file names in assert strings (e.g. `UI\npcmenu.cpp`), which gives a
  table of contents for the whole binary.
- **Oracles instead of guesses:** every parser is tested against real game data or an independent reference
  (`nokka/d2s`, MIT) and every behaviour is checked in the running engine.
- **Parallel agents** each take one slice of the game (assigned from a full source map) and write notes; the lead
  integrates their work.
- Reverse-engineering notes (addresses and observations, never decompiled code) live alongside the owner's
  working copy; game files and Blizzard code are never committed.

## Branches

| Branch | What |
|---|---|
| `master` / `integration` | Everything merged and building — **start here** |
| `fix/real-game-data-1.14b` | Load unmodified 1.14b data |
| `fix/town-tile-and-button-frame-warnings` | Town warnings |
| `feat/npc-click-interaction` | NPC click, voice greeting, autotest harness |
| `feat/npc-menu` | Real NPC menu and ported greeting picker |
| `feat/d2s-save-header`, `feat/d2s-items`, `feat/d2s-import` | Real `.d2s` saves |
| `docs/macos-quickstart` | macOS setup guide |

## Success log (newest first)

| Date | Success |
|---|---|
| 2026-10-10 | **Batch 3 landed, first fully green full verify** (`integration` and `master` = 697ecb6c): the whole Act 3 chain with real combat (Docks to Travincal, the Council, Khalim's Flail, the Orb, Durance, Mephisto and the red portal), the multiplayer game screen (host and join through the menus, two windows synced through the realm) and the real Act 2 wall fade. Run 63179: all six jobs passed, 193 scenarios, one retried flake. Next: batch 4 (skills missiles incl. Meteor/Blizzard PvP, the trade offer race fix, quest walkthroughs), the all-classes playthrough matrix and your play-test |
| 2026-10-10 | **Batch 2 landed** (`integration` and `master` = 59012a7a): natural mana regeneration from the exe (25 Hz; replenish-life item stat; the Act 5 caves mana failures dropped from about 4,000 to 400), the Act 5 endgame table fix (Baal's waves), preset population and exact maze tiles, class-skills retry, a chase fix (no warp click while fighting) and the town left-click fix (a spell on the left button no longer stops walking in town). Full verify: all five acts' playthroughs pass; nine scenario reds (stale expectations, a PvP margin the regen closed, a trade-window race) were fixed and re-run alone. Next: batch 3 (Act 3 depth, multiplayer game screen, Act 2 wall fade), then skills missiles and quests |
| 2026-10-10 | **Batch 1 landed** (`integration` and `master` = 1dcef1be): Mac release hardening (release.sh, dmg and bundle checks, CI), the exe-derived stamina model and bare-hand damage, and the world-objects audit (wells, shrines, chests). Full verify: 4 of 6 jobs green; the two reds were load and timing flakes (a waypoint cooldown margin, a monster standing on arrival stairs) that passed alone after fixes. Next: batch 2 (natural mana regeneration, Act 5 endgame table fix, preset population, class-skills retry), then skills missiles, multiplayer game screen, quests and Act 3 depth |
| 2026-10-10 | **LANDED: first real full verify green on the combined tree** (`integration` and `master` = 2e50f0e4): pass5 + population + playthroughs + skills-mp + travel + the Act 3 unseal, curse retry and Act 5 caves fixes. Run 50637: 6 jobs x 30 scenarios, 177 green; the 3 reds (class skills, Act 3 and Act 5 caves playthroughs) were fixed and passed alone. All five acts now pass a scripted real-combat playthrough (Act 3 in 874 s). Caveats: no second full run on the final tree; the Act 5 caves pass relies on restoring vitals (the engine has no natural mana regen yet, agent working on it). Headline Game v1 complete 64% (66% with held branches) |
| 2026-10-09 | **Progress board refreshed from real numbers**: Ghidra all 11,257 `Game.exe` functions named (last 22 Warden/module functions named later the same day, a few names unverified); new bar *Real full verify green* from a real `verify_parallel.sh` log (74 sections ran, 71 green; red: Act 3 gate, skill bar, act travel; 4 of 6 jobs ALL CHECKS PASSED); *Game v1 complete* 58% (64% with branch work) |
| 2026-10-09 | **Honest verification reset**: earlier parallel verifies ran only unit tests (filter bug) or skipped scenarios (unset sample save), so "all green" claims were void; guard added on a branch. First real runs: ~120 scenario sections across 6 jobs, red only on Act 3 gate (root cause found: level 100 stair tile style), skill bar (two root causes fixed) and load-dependent flakes. Pending on branches, not yet landed: exact maze/preset tiles, unique/champion packs, audio volume rules, difficulty and combat-math audits, 22 last exe names (11,257 of 11,257 named) |
| 2026-10-09 | **Exact tile placement for Acts 2–5 outdoors** (72 levels, 7,558 rooms equal to the emulator), the **Act 2 start playable** (Lut Gholein → Rocky Waste → Halls of the Dead), level state kept when you leave and return, **caves 13–16 and 37**, **Acts 2–5 quest logic**, **monster AI for every archetype** (17 faithful ports, the rest stand-ins), **real NPC speech and music fixed** (the paths were wrong; test runs were muted so nobody noticed), progress chart over time |
| 2026-10-09 | **Acts 2, 3, 4 and 5 level generators match the real game (1,626 levels)**, exact tile records for Act 1 outdoors (240 levels), act towns and travel, party/trade/PvP, boss encounters and Baal waves, options menu and hardcore death, the scripted first hour of Act 1, imported-hero UI, `OpenDiablo2.app` packaging merged (full verify: ALL CHECKS PASSED) |
| 2026-10-09 | Skills for all seven classes, skill hotkeys and the selection screen, Nightmare/Hell difficulty, boss behaviours and forced states, multiplayer and documentation merged; Act 4/5 level generators specified and checked against the real game |
| 2026-10-09 | **Act 1 outdoors match the real game on 240 levels**; mercenaries, equipment rules, world objects, automap and a 5× faster frame merged |
| 2026-10-09 | Death/respawn/new characters merged (new Druid byte-identical to the real file); the Act 1 outdoor generation algorithm reverse engineered and a reference port matched the real game on 208 levels |
| 2026-10-09 | **All maze levels of all five acts proven identical to the real game**; equipment affects the hero (explains 1241/869); positional/ambient audio; a double-clickable Mac app |
| 2026-10-09 | Level generator proven for **all Act 1-3 maze levels**; waypoints, portals, doors and level changes; Gheed's gamble and Cain's identify; test scenarios now launch games without Terminal windows |
| 2026-10-09 | **Level generator proven identical to the real game** (2,550 maze records, 50 world layouts); real dungeons render and play; skills, monsters part 2, stash/cube/belt, lighting merged; test runner restructured into one file per scenario |
| 2026-10-09 | Engine saves characters back to real `.d2s`; hireling and renderer reverse engineering done (real lighting model, hire cost and stat formulas) |
| 2026-10-09 | **Monsters** with the original AI, pathfinding, combat, death and loot run in the engine; test runs no longer collide on the server port |
| 2026-10-09 | Vendors, ground items and chests, the sound engine, CI + scripted autotests and the Go level generator merged; skills part 2 researched (365 skill functions named, the calc language decoded) |
| 2026-10-09 | Loot, trade, quests, packets, key bindings and day/night merged; monster AI think functions named (~90 created); maze level generation and the Act 1 world layout reverse engineered |
| 2026-10-09 | `.d2s` **writer**: a real save round-trips byte-for-byte; combat formulas implemented and verified against 12 binary functions |
| 2026-10-09 | Quests (state layout, framework, Den of Evil) and inventory/trade (price formulas, packets) reverse engineered |
| 2026-10-09 | Item generation reverse engineered: treasure classes, quality rolls with magic find, affixes, property stats (12 gaps vs OpenDiablo2 listed) |
| 2026-10-09 | Monster AI mapped: 148 AI types, aggro rules, spawning and level scaling |
| 2026-10-09 | Random number generator checked instruction-by-instruction against the real binary: no differences |
| 2026-10-09 | Equipment imported from a real `.d2s`; Diablo II's random number generator and seed hierarchy implemented (`d2rand`) |
| 2026-10-09 | Level generation (DRLG) core reverse engineered: seed hierarchy, dispatch, preset/maze/outdoor structure |
| 2026-10-09 | Skills and combat formulas reverse engineered and verified (mana cost, to-hit, defence, block, critical strike, RNG) |
| 2026-10-09 | Real NPC menu opens from the game's menu table; greeting picker ported |
| 2026-10-09 | `Parse` reads a whole save (corpse, mercenary, golem) |
| 2026-10-09 | Real `.d2s` items parse: 60/60 items match the reference; item tables decoded from the game's `.bin` |
| 2026-10-09 | Real level-94 character imported and started in the engine |
| 2026-10-09 | Town warnings confirmed gone in the running game |
| 2026-10-09 | Real `.d2s` body (quests, waypoints, stats, skills) verified on two saves |
| 2026-10-09 | NPC greetings fixed for every town NPC; autotest harness lets the AI verify without clicking |
| 2026-10-09 | Ghidra decompiler built for macOS; first real functions read; NPC menu table found |
| 2026-10-09 | Engine boots with real Diablo II 1.14b + LoD data on Apple Silicon |

## Legal

OpenDiablo2 is GPLv3. Diablo II is © Blizzard Entertainment; this project is unaffiliated and needs your own
legitimate copy of the game. Reverse-engineering notes cover interoperability only.
