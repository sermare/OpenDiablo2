# Reverse engineering: method, evidence and rules

This fork reproduces the behaviour of Diablo II 1.14b (Lord of Destruction) rather than inventing it. The facts come
from studying the original `Game.exe` for **interoperability**: so that this engine can read the same data files,
load the same character saves and behave the same way. This page explains how that was done, how much to trust each
claim, and the rules for contributing without crossing legal or quality lines.

The detailed notes (function addresses, structure offsets, constants, control-flow summaries) are kept in a separate
notes repository next to the author's working copy, not in this repository. They record addresses and observations
only. Code comments in this repo carry the conclusions and their evidence level.

## 1. Tooling

### 1.1 Ghidra and an MCP server

* The binary is analysed in [Ghidra](https://ghidra-sre.org/) (12.1.x), project loaded at the executable's image base
  (0x400000, 32-bit x86). One Ghidra window is open and driven by an AI agent through a **Ghidra MCP server**
  (Model Context Protocol), which exposes search, decompile, cross-reference, rename and memory-read operations as tools.
* The window is shared: calls are serialised, so agents batch their requests and avoid decompiling the same function twice.
* Policy for agents that only investigate: read-only. They do not run Ghidra scripts, close the window or import other
  programs. Renaming functions is allowed only for functions whose purpose is understood, using a module prefix and a
  verb-first PascalCase name (for example `UI_BuildNpcMenu`, `TRADE_CalcItemPrice`). About 2,400 functions are named at the
  time of writing (the README status board has the current count).

### 1.2 The decompiler on macOS

Ghidra's release does not ship a native decompiler for Apple Silicon. It was built from the source bundled with Ghidra:
in `Ghidra/Features/Decompiler/src/decompile/cpp`, build the `ghidra_opt` target for arm64 with the `mac_arm_64`
OS directory, and copy the resulting `decompile` binary into `Ghidra/Features/Decompiler/os/mac_arm_64/`. After that the
decompiler works natively. (A JDK is also needed to run Ghidra itself.)

### 1.3 Other tools

* An MPQ command-line tool to list and extract archives (the engine's own `d2mpq` reads them at runtime).
* Python with Capstone and Unicorn for disassembly and emulation (section 3).
* Independent open-source references for formats, used as cross-checks: the `nokka/d2s` save parser (MIT), and a
  clean-room re-implementation of server logic (D2MOO) whose source is compared against the binary but never copied.

## 2. Techniques

### 2.1 Assert strings give the table of contents

Blizzard left the original source file paths inside assert and allocation helper calls, for example `.\UI\npcmenu.cpp`.
Searching the binary's strings for `\.(cpp|cxx|h)$` returns about 277 original file names (`Main.cpp`, `GAME\Game.cpp`,
`UNIT\Monster.cpp`, `UI\inv.cpp`, `SKILLS\Skills.cpp`, `Sound\SoundHdr.cpp`, `Env.cpp`, ...).

The method:

1. Search strings for the source file names of your slice.
2. Take cross-references to a path string: the functions that pass it to an assert belong to that source file.
3. Decompile those functions (with callers and callees) and follow callees only as needed.
4. Name what you understand; cross-check the result against game data (`monstats.txt`, `npc.txt`, `Sounds.txt`, ...) and
   against this engine's code: what does the fork lack or do differently?

The original game was built from separate libraries (D2Common, D2Game, D2Client, ...) that the 1.14 executable links
into one binary, so file names also show where a function lived (for example D2Common `LvlTbls.cpp`).

### 2.2 Slicing

Work is split by source file group (units, skills and combat, monster AI, DRLG, inventory/trade, quests, hirelings,
rendering and sound, game loop and packets, menus). Each slice produces one notes file with: a table of named functions
(address, name, purpose), key structures and offsets, how the slice maps to missing or wrong behaviour in the fork,
and open questions. The port is written afterwards from the notes, not directly from decompiler output.

### 2.3 Oracles instead of guesses

Wherever possible a claim is checked against something that is not the author's reading of the code:

* **Real data**: parsers are tested on unmodified game files and real character saves (for example a level-94 save;
  a parse-write round trip must be byte-identical).
* **Independent implementations**: the `.d2s` item parse is compared with a reference parser written by someone else.
* **The emulator oracle** (next section): the original generator itself, executed, supplies the expected numbers.
* **Instruction-level comparison**: the random number generator was compared with the real code instruction by instruction.
* **Negative controls**: perturb the expected value and make sure the test fails.

## 3. The emulator oracle

For a deterministic subsystem whose output is a pure function of a seed (the level generator is the main case), reading
the code is not enough to be sure a port is exact. The oracle runs the original code:

1. A harness loads the 32-bit PE image into a Unicorn x86 emulator at its preferred base address, with a heap, stack,
   segment registers (so exception-handling prologues work) and stubs for imported Windows functions.
2. Hooks replace only things that cannot influence the random streams: the allocator, MPQ file opening (files are served
   from a folder of extracted data), string-table lookups, critical sections.
3. The real table loaders run, then the real generator is called for a chosen seed, difficulty and level.
4. The harness reads the resulting structures from emulated memory (level rectangle, level seed, room list, grids) and
   writes derived **numbers** to JSON ("golden" files).
5. The Go port must reproduce the numbers exactly, including the final seed; any missing or extra random draw changes it.

Where the original touches a subsystem the emulator does not implement (the DT1 tile library), the hook consumes the
documented number of random steps. This is recorded as a **model**, not an observation, in the notes and in the code.

Procedure for regenerating goldens and the exact test commands are in [TESTING.md](TESTING.md#3-oracle-tests-the-level-generator-ground-truth).
The harness itself is not part of this repository because it depends on extracted game files.

## 4. Evidence levels

Every behavioural claim carries a level, in the notes and in comments of the Go code. Use the same words.

| Tag | Meaning |
|---|---|
| `VERIFIED` | Control flow, constants or layout read from the decompiled binary and confirmed (for example the constant is visible in the disassembly, the table bytes were read from the image). |
| `VERIFIED-EMU` (notes) | Checked bit for bit against the real code running in the emulator oracle. The strongest level. |
| `VERIFIED-DISASM` (notes) | Traced in raw x86 and internally consistent, but not separately executed. |
| `inferred` | Derived from structure or from how the data behaves. Plausible; not read directly. |
| `UNVERIFIED` | A hypothesis, an interpretation of a vague note, or a shortcut. Kept small and replaceable. |
| `NOTE (binary)` | A place where the binary disagrees with an earlier note or a reference implementation. The binary wins. |

Rules for the tags:

* Never upgrade a claim without new evidence. Say what evidence ("spot-checked in Ghidra at address X", "golden of N seeds").
* Parts of the original that were not read are left as documented `TODO` interfaces or listed under "not ported". They are
  never invented to make something look complete. The package comments in `d2monster`, `d2drlg`, `d2skill` and
  `d2missile` are good models.
* Engine-level choices that are not from the original (for example the corpse lifetime) are labelled as engine choices.

## 5. Legal stance and what may be committed

This is a technical description, not legal advice.

* **You need your own legitimate copy** of Diablo II and Lord of Destruction. The engine reads the original archives from
  your install. Nothing is downloaded or distributed by this project.
* **No game files in the repository.** That includes MPQ archives, extracted tables and string files, DS1/DT1/DC6/DCC
  assets, audio, video, palettes, executables and character saves (yours or anyone's). Test fixtures are synthetic, or
  read from paths in environment variables at test time and skipped when absent.
* **No Blizzard code in the repository.** No decompiler output is committed or pasted into notes, comments or tests.
  Notes and comments record addresses, names, structure offsets, constants, control-flow summaries in your own words,
  and porting implications. A short constant or table value that is needed for interoperability (a multiplier, a table of
  small numbers, a file-format field offset) is acceptable; a function body transcribed from the decompiler is not.
* **Golden test data contains only numbers derived from the generator** (room rectangles, seeds, hashes), never extracted
  game content.
* **Purpose is interoperability**: reading the original data formats and saves and reproducing observable behaviour so a
  native macOS client can use players' own game data. The project is GPLv3 and is not affiliated with or endorsed by
  Blizzard Entertainment. Diablo II is copyright Blizzard Entertainment.

## 6. Rules for contributors

1. **Own your slice.** Know which notes and which packages your change concerns. Keep changes localised; the integrator
   merges many branches, so avoid drive-by edits and reformatting of files you do not own.
2. **Read before you port.** Port from understanding (notes, names, structure offsets), then cite the evidence level in a
   comment next to the code. Do not paste decompiler output anywhere in the repo.
3. **Do not guess.** If you cannot confirm a behaviour, implement the smallest replaceable version and tag it `UNVERIFIED`
   with what you would check.
4. **Prefer a pure package with a small interface.** The rule goes in `d2common` (or a pure `d2core` package), the engine
   adapter in a glue package. See [ARCHITECTURE.md](ARCHITECTURE.md).
5. **Add a test that could fail.** Table-driven unit tests; real-data tests behind environment variables; for generators,
   oracle numbers. See [TESTING.md](TESTING.md).
6. **Never commit game data or saves**, and check `git status` before every commit.
7. **Read-only on the shared Ghidra window**: no renames without confidence, no scripts, no closing the application,
   and be economical (one request at a time is all it can answer).
8. **Record what you learn** in the notes (address, structure, evidence), not only in code, and update the README status
   board and `docs/progress.json` only for things that were verified, not assumed.
9. **Be honest in reports**: what is implemented, what evidence exists (test names, log lines), what is unverified or
   skipped and why.
