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

Quick start on a Mac: [docs/macos-quickstart.md](docs/macos-quickstart.md).

## Status board

_Last updated: 2026-10-09. Updated by Claude on every merged success._

### ✅ Working now (each one verified, not assumed)

| Done | Evidence |
|---|---|
| Runs natively on Apple Silicon with the **real 1.14b + Lord of Destruction data** | Boots into Rogue Encampment from a real install; fixes: ragged rows in the game's `.txt` tables, optional LoD strings, missing music no longer crashes |
| Town warnings gone (`Unknown tile ID`, `invalid frame index`) | Combined build runs with **zero** warnings/errors in the log |
| **Click an NPC** → hero walks up → NPC **speaks their real voice line** | Greeting rows come from the game's `Sounds.txt`; autotest resolves Warriv, Akara, Charsi, Kashya, Gheed |
| **Real `.d2s` character saves**: header, quests, waypoints, stats, skills | Checksum + all 16 attributes + 30 skill allocations match a reference parser on two real 1.14b saves |
| **`.d2s` item lists** (properties, set bonuses, runewords, sockets) | All 60 items of a real level-94 save match the reference |
| **Import a real character into the engine** | A level-94 Sorceress from a real `.d2s` loads and starts in town |
| **Test without clicking** (`OD2_AUTOGAME`, `OD2_AUTOTALK`, …) | Lets the AI verify changes by itself; see the quickstart |
| Reverse-engineering map of the game | ~100 functions named in Ghidra; notes cover the NPC menu table, save format, packet tables, greeting picker, unit structs, frame order |

### 🔧 In progress right now (parallel agents)

| Work item | Branch / notes |
|---|---|
| Finish the `.d2s` reader (corpse, mercenary, golem) and import **equipment** into heroes | `feat/d2s-import` |
| **Real NPC menu** (Talk / Trade / Gamble / Hire …) from the game's own menu table | `feat/npc-menu` |
| Skills, combat and damage formulas | notes: `skills-combat` |
| Level generation from a seed (DRLG) | notes: `drlg` |
| Item generation and affixes | notes: `itemgen` |
| Monster AI and NPC server logic | notes: `monster-ai` |
| Quests (all acts) | notes: `quests` |
| Inventory, stash and trade | notes: `inventory-trade` |

### 🎯 Plan and priorities (set by Claude)

1. **Real characters load fully** — items, corpse, mercenary — so a player's actual saves work in the engine.
2. **NPCs work like the real game** — menu, voice greeting logic (day/night, "welcome back"), trade.
3. **Combat is correct** — real skill, damage and defence formulas.
4. **Real maps** — reproduce the original level generator from a seed.
5. **Loot and monsters** — item generation, monster AI, quests.
6. **Protocol and polish** — packet layer parity, rendering parity (lighting, palette shifts).

### 📋 Backlog (not started)

Missiles, collision and pathing · hirelings · stash/cube · automap · sound engine parity · server session core ·
D2Common data tables · key bindings from `default.key` · loose-file fallback for mods · welcome-back greeting flag.

### 📈 Progress metric

Named functions in `Game.exe` (Ghidra): **~800 of 10,819** (7%) and growing; ~45 of 277 original source files
covered in depth so far.

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
| `feat/d2s-save-header`, `feat/d2s-items`, `feat/d2s-import` | Real `.d2s` saves |
| `feat/npc-menu` | Real NPC menu (in progress) |
| `docs/macos-quickstart` | macOS setup guide |

## Success log (newest first)

| Date | Success |
|---|---|
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
