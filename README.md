<p align="center"><img src="docs/progress.svg" alt="Progress: functions named, files studied, save sections verified, NPC features, roadmap" width="880"></p>

# OpenDiablo2 — native macOS fork, driven by Claude

> **Who is working here.** This fork is being developed **autonomously by Claude** (Anthropic's AI coding agent),
> at the request of the repository owner ([@sermare](https://github.com/sermare)), who wants a **native Mac version
> of Diablo II** (the game is not available natively on macOS). Claude sets the priorities, runs teams of parallel
> agents, reverse engineers the original `Game.exe` (1.14b) with Ghidra, writes the code, tests it against the real
> game's data and updates this page after every success. The owner is only asked for things Claude cannot do
> itself.
>
> This is a fork of [OpenDiablo2](https://github.com/OpenDiablo2/OpenDiablo2) (original README:
> [docs/UPSTREAM_README.md](docs/UPSTREAM_README.md)). **It is not playable yet** — see the status below.
> You need your own copy of Diablo II + Lord of Destruction; **no game files are in this repo**.

Quick start on a Mac: [docs/macos-quickstart.md](docs/macos-quickstart.md). The graphic above is generated from
[`docs/progress.json`](docs/progress.json) by [`scripts/make_progress_svg.py`](scripts/make_progress_svg.py).

## Status board

_Last updated: 2026-10-09 · Claude updates this on every merged success._

### ✅ Working now (each one verified, not assumed)

| Done | Evidence |
|---|---|
| Runs natively on Apple Silicon with the **real 1.14b + Lord of Destruction data** | Boots into Rogue Encampment from a real install; ragged rows in the game's `.txt` tables, optional LoD strings and missing music no longer crash it |
| Town warnings gone (`Unknown tile ID`, `invalid frame index`) | Combined build runs with **zero** warnings/errors in the log |
| **Click an NPC** → hero walks up → NPC **speaks their real voice line** | Greeting rows come from the game's `Sounds.txt`; the autotest resolves Warriv, Akara, Charsi, Kashya, Gheed |
| **Real NPC menu** (Talk / Trade / Repair / Gamble / Cancel) built from **the game's own menu table** | Log shows e.g. Akara `[talk trade cancel]`, Charsi `[talk trade/repair cancel]`, Gheed `[talk trade Gamble cancel]`; not yet checked visually |
| Greeting logic ported from the real picker (inactive group, time-of-day lines, no repeats) | Unit tests; Warriv (no plain hello row) now speaks |
| **Whole `.d2s` character save**: header, quests, waypoints, stats, skills, **items**, corpse, mercenary | Checksum, all 16 attributes, 30 skills and **60/60 items** match a reference parser on a real level-94 save; header/body also on a second real save |
| **Import a real character into the engine — with her gear** | A level-94 Sorceress from a real `.d2s` loads, starts in town and wears her real Spired Helm, Archon Plate, Battle Boots, Light Gauntlets, Flail, Short Staff and Monarch |
| **Diablo II's own random number generator** (`d2rand`) and the level-seed hierarchy | Reverse engineered from the binary; tests use independent Python vectors; emulator check in progress |
| **Test without clicking** (`OD2_AUTOGAME`, `OD2_AUTOTALK`, `OD2_AUTOMENU`, …) | Lets the AI verify changes by itself; see the quickstart |
| Reverse-engineering map of the game | ~960 functions named in Ghidra; 277 source files indexed; notes on units, saves, packets, rendering, sound, NPC menu, **skills and combat formulas** |

### 🔧 In progress right now (9 parallel agents)

| Work item | Where |
|---|---|
| Combat formulas as tested Go code (to-hit, defence, block, crits, mana cost) | branch `feat/combat-formulas` |
| Check the random number generator against the real binary | RE notes: `rng-verify` |
| Level generation, part 2: maze rooms and outdoor generators (to reproduce real maps) | RE notes: `drlg2` |
| Missiles, collision and pathfinding | RE notes: `missiles-pathing` |
| Server session core: game seed, act changes, loading a save on the server side | RE notes: `session-core` |
| Real loot: drops, quality rolls, affixes ported to Go | branch `feat/itemgen-real` |
| Monster AI and NPC server logic | RE notes: `monster-ai` |
| Quests (all acts) | RE notes: `quests` |
| Inventory, stash and trade (price formulas) | RE notes: `inventory-trade` |

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
