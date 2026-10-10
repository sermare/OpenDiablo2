# Testing

How this fork is checked, from fast unit tests to a full in-game run. Nothing here needs game files unless
it says so, and **no game file, extracted table or save is ever committed**: tests that need real data read it from
paths given in environment variables and `t.Skip` when those are unset.

Layers, cheapest first:

1. [Unit tests](#1-unit-tests) (always run, including on Linux CI)
2. [Real-data tests](#2-real-data-tests-environment-variables) (your own game data, skipped otherwise)
3. [Oracle tests](#3-oracle-tests-the-level-generator-ground-truth) (committed golden numbers from the real game)
4. [The in-game harness](#4-the-in-game-harness-od2_auto) (`OD2_AUTO*`, runs the real engine)
5. [`scripts/verify.sh`](#5-scriptsverifysh-and-scenarios) (the gate that runs 1-4 together)

What each layer is evidence for, feature by feature: [STATUS_MATRIX.md](STATUS_MATRIX.md).

## 1. Unit tests

```sh
go build ./... && go vet ./... && go test ./...
```

* `gofmt` must be clean for the files you touch. The repo has older unformatted upstream files that are deliberately
  not reformatted; CI gofmt-checks only the directories in `GOFMT_DIRS` (currently `d2game/d2autoscript`).
* The linker warning `ld: warning: ignoring duplicate libraries: '-lobjc'` is harmless.
* `go vet ./...` over the whole tree reports older findings (for example in `d2app`); CI vets only the pure packages.
* Style: table-driven tests; prefer pure packages with small interfaces over code that needs MPQs or a display.
  The pure rule packages (`d2combat`, `d2monster`, `d2skill`, `d2trade`, ...) are tested with fakes for their
  interfaces and a seeded `d2rand.Seed`, so results are reproducible.
* Randomised code: test with a fixed seed. The test vectors for `d2rand` come from an independent Python
  implementation, not from the Go code.

## 2. Real-data tests (environment variables)

These tests read your own copy of the game data. Set the variable, run the package, and compare against the output
you expect. Without the variable they skip (they never fail for missing data).

| Variable | What it points to | Used by |
|---|---|---|
| `D2_TABLES` | A folder of extracted game tables. The layout is whatever the individual test reads (for example `drlg/patch_d2/Levels.txt`, `drlg/bin/patch_d2/lvlprest.bin`, `itemstatcost.bin`, `armor.txt`, `sound/patch_d2/...`, `skills/...`, `monsters/...`); a test skips when the file it needs is missing. The author keeps it in `~/git/d2-tables`. | DRLG, `d2drop`, `d2sfx`, `d2monster`, `d2hero`, `d2calc`, `d2records`, `d2quest`, `d2statlist`, `d2equip`, `d2hireling`, `d2level`, `d2object`, `d2automap`, `d2s` (about 27 call sites) |
| `D2S_SAMPLE` | A real `.d2s` | header parser tests in `d2s` |
| `D2S_SAMPLE_JSON` | Expected parse of that save (nokka `d2s` JSON format) | item parser test in `d2s` |
| `D2S_SAMPLE_BODY` | A real `.d2s` that has a body (quests, waypoints, stats, skills, items) | body, writer, container, equipment and export tests (`d2s`, `d2hero`) |
| `D2S_SAMPLE_BODY_JSON` | Expected parse of `D2S_SAMPLE_BODY` from an independent reference parser | `d2s` body test |
| `D2S_SAMPLE_NEW` | A real new-character save from the game | `d2s` new-character test: every byte must match except the fields tagged as varying |
| `D2_DS1` | A folder with the real town DS1 files (`LutW.ds1`, `LutN.ds1`, `DockTown3.ds1`, `Fortress.ds1`, `townWest.ds1`) | `d2mapgen` `TestRealTownDS1` |
| `D2_DIFFICULTYLEVELS` | The `patch_d2` `DifficultyLevels.txt` | `d2difficulty` `TestRealTable` |
| `D2_DS1_ROOT` | Folder holding `patch_d2`, `d2exp`, `d2data`, each with `data/global/tiles` extracted | outdoor oracle test (`drlgoutdoor`) |
| `ORACLE_TILES_DIR` | A folder of full emulator tile dumps (`gen_tiles2.py` in the oracle, one level per `o_<seed>_<level>.json`; also needs `D2_TABLES` and `D2_DS1_ROOT`) | `drlgoutdoor` `TestTileDiffDir` (exact builder vs the emulator, per level), `TestTileSimDir` (old DS1 stamp path vs the emulator), `TestStaleCacheKeys`; results in [tile-diagnosis.md](tile-diagnosis.md) |
| `D2_INSTALL` | The Diablo II install folder | `default.key` tests (`d2key`, `d2player`) |
| `D2_DEFAULT_KEY` | A path to a real `default.key` | `d2key` |
| `D2_PL2` | A real act palette `.pl2` | `d2pl2` shade factors |
| `D2_STRING_TBL` | `data\local\lng\eng\string.tbl` | `d2quest` speech keys |
| `ORACLE_MAZE`, `ORACLE_WORLD`, `ORACLE_OUTDOOR`, `ORACLE_OUTDOOR45`, `ORACLE_TILES` | Override the committed golden JSON with another file (see the next section) | oracle tests |

Typical use:

```sh
D2_TABLES=$HOME/git/d2-tables \
D2S_SAMPLE_BODY=/path/to/real.d2s D2S_SAMPLE_BODY_JSON=/path/to/real.d2s.json \
go test ./d2common/... ./d2core/...
```

Notes on the data used by the author:

* The primary save oracle is a real level-94 1.14b Sorceress with its expected parse produced by the independent
  [nokka/d2s](https://github.com/nokka/d2s) (MIT). A second real save is used for the header and body.
  The README success log gives the headline numbers (byte-identical round trip, 60/60 items).
* Extract tables from your own install with an MPQ tool, for example `mpqcli extract <mpq> -f 'data\global\excel\X.txt' -o <dir>`.
  Load order matters: `patch_d2` overrides `d2exp` overrides `d2data`.

## 3. Oracle tests (the level generator ground truth)

The level generator (`d2common/d2drlg`) is only useful if it produces **the same levels as the original game for the
same seed**. Reading code cannot prove that, so the real `Game.exe` generator itself is used as the oracle.

### 3.1 What an oracle is here

The real 1.14b `Game.exe` DRLG functions (`DRLG_CreateDrlg`, `DRLG_GenerateLevel` and everything below them) are run
inside an x86-32 CPU emulator (Python, Unicorn engine) by a harness that:

* loads the PE image at its image base, stubs the Windows imports and the allocator, and serves the game's table and
  DS1 files from a folder of extracted data;
* calls the generator natively for a chosen game seed, difficulty and level id;
* walks the resulting level structure (level rectangle, seed, room list, preset info) and writes **numbers only** to JSON.

The Go port is then run on the same inputs and must agree exactly, including the final level seed (which also proves
the random-number draw order is right, since any extra or missing draw changes the seed).

### 3.2 The committed goldens

`d2common/d2drlg/testdata/` (numbers only; no game data inside):

| File | Content | Test |
|---|---|---|
| `world_act1_normal.json` | Act 1 world layout for 50 seeds | `drlgworld` `TestOracleWorld` |
| `maze_act1.json`, `maze_act23.json`, `maze_act45.json` | Maze levels: rooms (or a count plus a hash of the sorted room keys in the compact files), Def and file index, final level seed | `drlgmaze` `TestOracleMaze` (hard assertions) |
| `acts.json` | Act-level extra draws (Act 2 tomb choice, Act 3 flip) | `d2drlg` `acts_oracle_test.go` |
| `outdoor_act1.json` | Act 1 outdoor levels: stage numbers, room list, sha256 digests of the large grids; the first seeds are kept in full | `drlgoutdoor` `oracle_test.go` |
| `outdoor_act2.json`, `outdoor_act3.json` | Act 2 and 3 outdoor levels (41-46, 76-83) and the preset towns 40 and 75 | `drlgoutdoor` `TestOracleAct2Levels`, `TestOracleAct3Levels` |
| `preset_act1.json` | Act 1 preset levels | `drlgoutdoor` `TestOraclePresetAct1` |
| `tiles_act1.json`, `tiles_act23.json`, `tiles_act45.json` | Per-cell tile records (tile ids) of the rooms for Acts 1 to 5 | `drlgoutdoor` `TestOracleTiles` (override with `ORACLE_TILES`) |
| `outdoor_act45.json` | Acts 4/5 outdoor and preset levels: 12 seeds x 3 difficulties x 16 levels, numbers and digests (rect, vis, od.flags, neighbours, grids, room list, final seed) | `drlgoutdoor` `TestOracleAct45` |
| `gen_outdoor_compact.py`, `gen_tiles_compact.py` | The only generator scripts that live in the repo: shrink the big emulator goldens to the committed compact files | n/a |

The tests need `D2_TABLES` (and `D2_DS1_ROOT` for the outdoor and tile ones) because the Go port needs the same input tables
the original read. Proven coverage is stated in the package comments and the README status board; cite those rather
than this page for exact numbers.

### 3.3 Regenerating a golden (procedure)

The emulator harness and its regeneration scripts are **outside this repository** (they embed paths and extracted game
data on the author's machine). The procedure is:

1. Extract the game tables and DS1 files you need from your own MPQs into the harness's `gamefiles` folder
   (priority `patch_d2` > `d2exp` > `d2data`).
2. In the harness virtual environment (Python with `unicorn`, `capstone`, `pefile`), run the generator for the part you
   changed: `gen_world.py`, `gen_maze.py`, `gen_outdoor.py`, `gen_acts.py` or `gen_tiles.py`, each as
   `python gen_X.py <output.json> [number of seeds]` (the author's notes record this form for `gen_maze.py`; check the other scripts' headers). A 6-seed, 3-difficulty maze run takes about a minute; a 50-seed
   run takes about eight minutes per difficulty.
3. For outdoors, generate the large golden (30 seeds, about 10 MB) and compress it with
   `d2common/d2drlg/testdata/gen_outdoor_compact.py big.json out.json [nfull]` before committing.
4. Copy the small resulting JSON into `d2common/d2drlg/testdata/`. You can run a test against a different file first
   with `ORACLE_MAZE=/path/to/golden.json go test ./d2common/d2drlg/drlgmaze/` (same for `ORACLE_WORLD`, `ORACLE_OUTDOOR`).
5. Negative control: perturb a value in the golden and confirm the test fails. A test that cannot fail proves nothing.

Only numbers derived from the generator may be committed. Do not commit extracted tables, DS1/DT1 files or the
emulator itself.

### 3.4 Caveats stated honestly

* The DT1 tile library is not emulated by the oracle. Where the original picks a random tile, the emulator hook
  consumes one room-seed step; that is a model, not an observation. `drlgoutdoor` carries the same model
  (`RoomBuildOptions.PickTile`).
* Level 134 (Forgotten Sands) runs the Act 2 desert generator and is not ported, so it has no golden; the tile goldens
  cover the rooms the sampled games reach, and tile code paths no golden room reaches return an error.
* A golden is only as wide as its seeds and levels. Passing the committed seeds is strong evidence, not a proof for all
  2^32 seeds.

## 4. The in-game harness (`OD2_AUTO*`)

The engine can run scripted checks of itself so a change can be verified with no one clicking. It needs a GUI
session and your game data. All variables below were found by searching for `os.Getenv("OD2_` in the code
(`d2app`, `d2game`, `d2core`, `d2networking`). Ignore a variable here at your own risk if the code disagrees: the code wins.

### 4.1 Launch rules on macOS

* A plain shell (SSH, a background agent, `go run` from a non-GUI context) cannot start the engine: it fails with a
  Cocoa display error. The process must start inside the user's **GUI login session**.
* Build to a scratch path (`go build -o /tmp/od2 .`), put the environment in a small `.command` script
  (`#!/bin/zsh`, `export OD2_...`, `/tmp/od2 2>&1 | tee run.log`) and launch that.
* Prefer `launchctl asuser $(id -u) /bin/zsh script.command`. It starts the script in the GUI session **without
  opening a Terminal window**. `open script.command` also works but leaves a Terminal window behind every time;
  hundreds of them stop Terminal from working. `scripts/verify.sh` tries `$OD2_VERIFY_LAUNCH`, then `launchctl asuser`,
  then `open`, in that order.
* The launcher returns before the game starts. Wait for the game process to appear, then for it to exit, then read the
  log (strip ANSI colour codes first).
* Give each run its own port and scratch directory so parallel runs do not collide: `OD2_PORT=<n>` (server and client
  default port) and `OD2_CONFIG_DIR=<dir>` (config and `Saves`).
* Use `OD2_AUTOTEST_MUTE=1` so tests are silent and `OD2_AUTOEXIT=1` so they end by themselves.

### 4.2 Variables, grouped

Per-variable behaviour and log lines are in [macos-quickstart.md](macos-quickstart.md) (a table) and in the doc comment
of the file that reads the variable. Defaults below are from the code.

**Start and control**

| Variable | Meaning |
|---|---|
| `OD2_AUTOGAME=<file>` | Start that character directly (a `.d2s` is imported first). Required by most scenarios. |
| `OD2_AUTOEXIT` | Quit when the scenario finishes (script and quest runs exit 0 on PASS, 1 on FAIL). Some paths save the hero before quitting. |
| `OD2_AUTOTEST_MUTE` | No audio output; the sound engine still tracks voices. |
| `OD2_AUTOSCRIPT='step;step'` | Scripted hero actions: `wait:`, `move:x,y` / `move:npc=`, `cast:`, `panel:`, `say:`, `expect:log=`, `use:`, `waypoint:`, `expect:level=`, `automap:`, `exit`. Needs `OD2_AUTOGAME`. Parser: `d2game/d2autoscript`. Ends with `AUTOSCRIPT RESULT PASS` or `FAIL`. |
| `OD2_PORT` | Server/client port override. |
| `OD2_CONFIG_DIR` | Move `config.json` and `Saves` (for isolated runs). |
| `OD2_NO_SETUP`, `OD2_SETUP_AUTOPICK` | Skip the first-run dialogs; select what a scripted setup UI picks. |
| `OD2_D2S_DIR` | Folder of real `.d2s` characters to import into the character list (read only). |
| `OD2_D2S_WRITEBACK=<dir>` | Folder where exported `.d2s` files go (otherwise next to the `.od2` save). |
| `OD2_AUTOSCREEN=charselect` | Open the character select screen directly (used with `OD2_AUTOSHOT`). |
| `OD2_AUTOSPEED=<factor>` | Run game time (movement, fights, timers, script waits) that many times faster than real time, for long playthroughs; values above 1 only, capped. |
| `OD2_AUTOOPTIONS=1` | Drive the options menu rows (sound, video, automap pages) and check `config.json`. |
| `OD2_NOPERSIST=1` | Turn off level persistence (a revisited level is rebuilt). |
| `OD2_POPULATE=1\|0`, `OD2_NOPOPULATE` | Force the natural monster population on or off. By default it is on only in scripts that play (`walkto:` / `kill:` steps) and off in scenarios that test something else. |

**Worlds and levels**

| Variable | Meaning |
|---|---|
| `OD2_REALMAPS=0` | Opt out of the DRLG level providers (real towns, maze and outdoor levels, level changes, population), which are the default; the old placeholder map (Rogue Encampment only) is used instead. |
| `OD2_AUTOMAPSEED=<n>` | Play the maps of another game seed (for example `1` gives the other Lut Gholein variant). |
| `OD2_AUTOLEVEL=<id>` | Start directly in that level . |
| `OD2_AUTOMAP=<id>`, `OD2_AUTOMAP_DIFF`, `OD2_AUTOMAP_ASCII` | Generate that level from the hero's seed and log a summary; difficulty; log the room list and a walkability map. (Not the in-game automap panel; that is `OD2_AUTOSCRIPT` `automap:`.) |
| `OD2_AUTOTIME=<phase>[@degree]` | Force and freeze the day/night clock. |
| `OD2_LIGHTING=0` | Turn the light map off. |
| `OD2_AUTOSHOT=<file.png>`, `_DELAY`, `_SECONDS` | Save a frame after a delay in the world. |

**NPCs, menus, trade**

| Variable | Meaning |
|---|---|
| `OD2_AUTOTALK=Warriv,Akara` | Resolve and log each NPC's greeting. |
| `OD2_AUTOMENU=<npc>`, `OD2_AUTOMENU_CHOOSE`, `OD2_AUTOMENU_HOLD` | Open an NPC menu, choose an entry, keep it open. |
| `OD2_AUTOTRADE=Akara,Charsi`, `_SEED`, `_LEVEL` | Open vendor windows, log stock and prices, run a scripted buy/sell/repair. |
| `OD2_AUTOGAMBLE`, `OD2_AUTOIDENTIFY` | Gheed's gamble window; Cain's identify window. |
| `OD2_AUTOSOUND=<handle,...>` | Play Sounds.txt rows and log the voice decision. |

**Combat, monsters, skills**

| Variable | Meaning |
|---|---|
| `OD2_AUTOMONSTER=<id,count>`, `_SECONDS`, `_DIFF`, `_PASSIVE`, `_FAR`, `_AREA` | Spawn monsters near the hero and let the hero fight; difficulty 0..2; hero does not fight back; far ring for sound tests; map stands for an area. Logs `MONSTER ...` and `AUTOMONSTER summary`. |
| `OD2_AUTOCAST=<skill>,<count>`, `_LEVEL`, `_MANA` | The hero casts a skill at the nearest monster through the skill pipeline. Default 5 casts, grant level 10. |
| `OD2_AUTOCAST_CLASS=<Class>`, `OD2_AUTOCAST_CLVL=<n>` | Replace the save with a fresh hero of that class (character level default 18) so class skills can be cast. |
| `OD2_AUTOAI=<archetype\|monster>[,state=fear+confuse+...]`, `_SECONDS` | Spawn one monster of an AI archetype (such as Vulture or Summoner) or a monstats id, let its AI run, then force the states fear, confuse, attract, charm, blind, taunt through the `forcestate` console command; logs `MONSTER aistate ... from= to=`. |
| `OD2_AUTOBOSS=<names>` | Boss encounter triggers (Duriel tomb, Mephisto, Diablo seals, Baal throne waves) driven through `d2boss`. |
| `OD2_AUTODIFFICULTY=0\|1\|2`, `_FORCE` | Pick that difficulty on the difficulty screen; `_FORCE` also unlocks it in the save. |
| `OD2_AUTOMERC=1`, `_KILL`, `_HEROLEVEL`, `_EXP`, `_SECONDS` | Mercenary hire, fight, death/revive, follow. |
| `OD2_AUTOMARKDEAD=1` | Turn the `OD2_AUTOGAME` hero into a dead hardcore character, to test the load refusal. |
| `OD2_AUTODEATH=1`, `_MONSTER`, `_LEVEL`, `_HP`, `_HARDCORE` | Death, respawn, corpse recovery, penalties. |
| `OD2_AUTOAMBIENT=<level>`, `_PHASES`, `_SECONDS`, `_SPEED`, `OD2_AUTOSOUND_TRACE` | Sound environment and day phases; `SOUNDAT` positional-sound lines (the trace variable only switches those lines on). |

**Multiplayer** (two processes, see `96-multiplayer.sh` and `9d-party-trade.sh`)

| Variable | Meaning |
|---|---|
| `OD2_HOST=1` | Host a network game: TCP on `OD2_PORT`, bind address `OD2_BIND` (default 0.0.0.0 when hosting, loopback otherwise). |
| `OD2_JOIN=<host:port>`, `OD2_JOIN_RETRY=<seconds>` | Join a host; keep retrying while the host is still starting. |
| `OD2_PROTO=d2gs` | Use the Diablo II game protocol (`d2gs`, `d2gsnet`) instead of the default JSON packets; host and client must agree. |
| `OD2_AUTOPARTY=1` | Marks the party scenario: the monster director may spawn monsters while the hero is still in town. The party, trade and PvP steps themselves are `OD2_AUTOSCRIPT` steps. |

**Items, containers, objects, quests**

| Variable | Meaning |
|---|---|
| `OD2_AUTOGROUND=<seed>[,count]`, `_TC`, `_ILVL` (item level; default the hero's level), `_HOLD` | Drop items and gold from a treasure class around the hero. |
| `OD2_AUTOCHEST=<seed>[,id...]`, `OD2_AUTOCHEST_SEED` (base of the chest drop seeds) | Spawn chests and barrels and open them. |
| `OD2_AUTOPICKUP=1` | Pick up every ground item into the inventory. |
| `OD2_AUTOPANEL=stash,cube,belt,inventory`, `_HOLD` | Open container panels and log items with grid positions. |
| `OD2_AUTOSTASH`, `OD2_AUTOBELT` | Stash interaction; belt potion hotkeys. |
| `OD2_AUTOEQUIP=1`, `OD2_DURABILITY_CHANCE=<percent>` | Equip-rule scenario; override the durability-loss chance. |
| `OD2_AUTOOBJECT=<id\|name>[,count][;...]`, `_SHRINE`, `_LIFE`, `_MANA` | Spawn world objects, force a shrine type, set hero life/mana. |
| `OD2_AUTOQUEST=<quest>`, `_AREA`, `_DIFF`, `_REAL` | Scripted quest line; logs `AUTOQUEST RESULT PASS`/`FAIL`. |

**Saves and characters**

| Variable | Meaning |
|---|---|
| `OD2_AUTOSAVE=1` | Set a known gold value, save and exit; used to test the `.d2s` write-back. |
| `OD2_AUTONEWCHAR=<Class>[,hardcore][,classic][,ladder]`, `_NAME`, `_REF` | Create a new character, write its `.d2s`, optionally diff it against a real new-character file. |

**Performance**

| Variable | Meaning |
|---|---|
| `OD2_AUTOPERF=1`, `_SECONDS` (20), `_WARMUP` (6) | Meter update and render time per frame after a warm-up. |
| `OD2_PPROF_CPU=<file>`, `OD2_PPROF_HEAP=<file>` | CPU and heap profiles of the run. With the meter on, the CPU profile starts after the warm-up; without it, at start-up (stop with SIGTERM to get the file written). |

Every run logs `PERF mark <name> t_ms=` lines (renderer-created, assets-ready, first-frame, main-menu-ready, game-playable;
milliseconds since the process started) and `PERF span generate level=<id> ms=` for every level build (the local game
builds each level twice, server and client side). Headless benchmarks and a budget test on the real game data
(`D2_GAME_DIR` = install folder; skipped when unset; `scripts/verify.sh` finds the folder in the game config):

```sh
export D2_GAME_DIR="$HOME/.wine-d2classic/drive_c/Program Files (x86)/Diablo II"
go test ./d2core/d2map/d2mapgen -run LevelBudget -v        # every kind of level in every act, cold then warm, with a limit
go test ./d2core/d2map/d2mapgen -run xxx -bench Level -benchtime 3x   # warm level build, DRLG layout, table parsing
go test ./d2common/d2fileformats/d2mpq -bench Explode     # PKWare decoder against the old one, on d2data.mpq sectors
go test ./d2common/d2path -bench LongAStar                # path search against the old implementation
```

`scripts/verify.d/94-perf-real-levels.sh` checks start-up, the level build and 120 monsters under full-screen lighting on
Frigid Highlands with limits ten to twenty times the measured values.

Internal: `OD2_WATCH_PID` and `OD2_WATCH_LOG` are used by the macOS app bundle's crash watcher (`bundle_darwin.go`),
not by tests. `OD2_VERIFY_LAUNCH` and `OD2_VERIFY_SAVE` belong to `scripts/verify.sh` (below).

### 4.3 Reading the result

Every scenario logs lines with a fixed prefix (`AUTOSCRIPT`, `AUTOCAST`, `MONSTER`, `CAST`, `LEVEL CHANGE`, `D2S EXPORT`, ...)
and most end with a summary line. A scenario's verdict is whatever its `scenario_check` greps for. A run that logs
`[ERROR]`, `[WARNING]` or `panic` fails the gate unless the scenario opts out.

## 5. `scripts/verify.sh` and scenarios

`scripts/verify.sh` is the one-command gate. It needs zsh, a GUI session and a game install configured
(see [macos-quickstart.md](macos-quickstart.md)).

```sh
D2S_SAMPLE_BODY=/path/to/real.d2s D2S_SAMPLE_BODY_JSON=/path/to/expected.json \
D2_TABLES=/path/to/extracted/tables ./scripts/verify.sh
```

What it does, in order:

1. **build** the engine into a scratch directory.
2. **unit tests**: `go test ./...`, printing only failures.
3. if `D2S_SAMPLE_BODY` is set: the **real `.d2s` oracle tests** (`d2s` package), then the **save-back round trip**
   (`go test -run Export ./d2core/d2hero/`, which must include `TestExportUnchangedIsByteIdentical`).
4. if `D2S_SAMPLE_BODY` is set: **every scenario** `scripts/verify.d/*.sh` in file-name order. Each runs the game
   once on a copy of the save with `OD2_AUTOGAME`, `OD2_AUTOTEST_MUTE=1` and `OD2_AUTOEXIT=1` plus its own environment.
5. prints `ALL CHECKS PASSED` or `SOME CHECKS FAILED` (exit 1).

Environment: `D2_TABLES`, `D2S_SAMPLE_BODY`, `D2S_SAMPLE_BODY_JSON` as above; `OD2_VERIFY_SOUND=1` (play real audio in
every scenario instead of muting; a scenario can also set `scenario_unmuted=1`); `OD2_VERIFY_SAVE` (a `.d2s` to start in the game
instead of a copy of the sample); `OD2_VERIFY_LAUNCH` (command prefix used to start the generated `.command` file, for
example `launchctl asuser 501 /bin/zsh`). It picks a random free port for `OD2_PORT` per run.

### 5.1 Scenario file format

A scenario is one small zsh file in `scripts/verify.d/`, named `NN-title.sh` (the number orders the run). The runner
sources it and uses these names:

```zsh
scenario_name="human readable title"          # shown as the step heading
scenario_env() {                              # echo shell lines for the game's .command file
  echo 'export OD2_AUTOMONSTER="zombie1,2" OD2_AUTOCAST="Fire Bolt,4"'
}                                             # may use $save, $tmp, $OD2_PORT
scenario_check() {                            # inspect "$log.txt" (ANSI stripped); set fail=1 on problems
  grep -qE "AUTOCAST summary .* kills=[1-9]" $log.txt || { echo "FAIL: no kill"; fail=1; }
}
scenario_warnings_ok=1                        # optional: do not fail on [ERROR]/[WARNING] lines
```

The runner already exports `OD2_PORT`, `OD2_AUTOGAME`, `OD2_AUTOTEST_MUTE` (unless sound is requested) and `OD2_AUTOEXIT` for you.
Two more optional variables: `scenario_unmuted=1` (real audio for this scenario) and `scenario_warnings_ok=1`.

How a scenario runs, exactly (`scripts/verify.sh`): for each file in name order the runner unsets the previous
functions, sources the file, writes `$tmp/<NN-title>.command` (shebang, `OD2_PORT`, `OD2_AUTOGAME`, `OD2_AUTOEXIT`, mute
line, your `scenario_env` output, then `od2 | tee <log>`), launches it, waits for the game process to appear and exit,
strips ANSI colours into `<log>.txt`, runs `scenario_check`, and fails the scenario when the log holds `[ERROR]`,
`[WARNING]` or `panic` lines (except the known `skipping missing` and `KILL giving up for now` lines) unless
`scenario_warnings_ok=1`. A failed scenario is **run once more** (they are timing sensitive on a loaded machine); it is
red only if the second attempt fails too. Two-process scenarios (`96-multiplayer.sh`, `9d-party-trade.sh`) start the
joiner themselves and synchronise on `waitlog:` autoscript steps rather than on timing.

Existing scenarios (44 files in `scripts/verify.d/`, in run order): `10-menus-trade`, `20-script-walk`, `30-autosave`,
`40-monster-pack`, `50-containers`, `60-skill-cast`, `70-real-maps`, `71-real-outdoor`, `72-real-act45`,
`80-imported-hero-ui`, `80-waypoint-portal`, `81-charselect`, `81-waypoint-persist`, `82-maze-travel`,
`83-cave-chain-persist`, `84-mercenary`, `85-quests`, `85-quests-acts2-5`, `86-class-skills`, `87-gamble-identify`,
`88-death-newchar`, `89-hero-stats`, `8b-options`, `8c-hardcore`, `8d-hardcore-death`, `90-ambient-audio`,
`91-ambient-env`, `91-automap`, `92-objects2`, `93-equip-rules`, `95-perf`, `96-multiplayer`, `97-ai-states`,
`98-skillbar`, `99-act-travel`, `9a-difficulty`, `9b-act1-playthrough`, `9c-act1-reload`, `9c-bosses`,
`9d-act1-sample-hero`, `9d-party-trade`, `9e-act2-playthrough`, `9e-real-audio`, `9f-act2-lutn`. Which feature each
one is evidence for is in [STATUS_MATRIX.md](STATUS_MATRIX.md). Prefix order matters: `9c-act1-reload` loads the `.d2s`
that `9b-act1-playthrough` exported.

### 5.2 How to add a scenario

1. Make sure the behaviour logs a stable, greppable line (add one in the game code if needed, with a clear prefix).
2. Create `scripts/verify.d/NN-yourthing.sh` with the three pieces above. Pick `NN` so it runs after what it depends on.
3. Make the check **specific**: assert on values (`mana_paid=2.50`, `kills=[1-9]`), not only on "something was logged".
4. Run `./scripts/verify.sh` (the whole gate) at least once; for iteration, copy the generated `.command` approach by
   hand with your env and the launch rules in section 4.1.
5. Prove it can fail: break the behaviour (or the expected value) and confirm the scenario goes red.
6. No edits to `verify.sh` are needed. Update the table in `macos-quickstart.md` if you added a new `OD2_*` variable.

## 6. CI

`.github/workflows/ci.yml` runs on every push and pull request:

* **Linux (pure packages)**: gofmt on `GOFMT_DIRS`, `go build`, `go vet`, `go test` for the packages in `PURE_PKGS` (no
  display, no cgo, no game data). Add a new pure package to `PURE_PKGS` when you create one.
* **macOS arm64 (full build)**: gofmt, `go build ./...`, `go vet` on the pure packages and `go test ./...`. Tests that
  need game data skip themselves because the CI machine has none.

CI therefore checks that the code builds and the unit tests pass. The real-data, oracle (needs `D2_TABLES`) and in-game
checks run only on a machine with your game install, through `scripts/verify.sh`. That is why the merge rule in
[CONTRIBUTING.md](../CONTRIBUTING.md) requires the verify gate result, not only green CI.
