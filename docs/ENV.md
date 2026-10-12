# Environment variables (`OD2_*`)

Every `OD2_*` name that appears in the Go code (`grep -rhoE 'OD2_[A-Z0-9_]+' --include='*.go' .`, 183 names on
2026-10-12), one line each, plus the ones that only the `scripts/` read. Players need very few of these (the first
group); the rest drive the automated scenarios and are listed so nobody has to read the source. One line is a summary;
the doc comment of the file that reads the variable is the authority. Lines marked **(unverified)** were written from
a quick read of the code and not run. Test-only variables are not a supported interface and can change.

Related: [TESTING.md](TESTING.md) (how the harness uses them), [DISPLAY.md](DISPLAY.md), [macos-quickstart.md](macos-quickstart.md).

## For players

| Variable | Meaning |
|---|---|
| `OD2_DISPLAY=WxH` | Pin the logical screen size (e.g. `1920x1080`); also keeps the window instead of starting full screen. |
| `OD2_UI_SCALE=1..4` | Integer interface scale; wins over `config.json`. |
| `OD2_BAR_FILL=none\|black\|tile` | What is drawn beside the 800-px bottom bar on wide screens (default `none`). |
| `OD2_WINDOWED=1` | Start in a window instead of full screen (normal launches start full screen). |
| `OD2_CONFIG_DIR=<dir>` | Folder holding `config.json` and `Saves` instead of `~/Library/Application Support/OpenDiablo2`. |
| `OD2_D2S_DIR=<dir>` | Extra folder to import real `.d2s` characters from (read only). |
| `OD2_D2S_WRITEBACK=<dir>` | Folder where exported `.d2s` files are written (default: next to the engine's own `.od2` save). |
| `OD2_D2S_PROMPT=yes\|no` | Default answer to the damaged-save recovery question (scripted runs answer no). |
| `OD2_STASH_IMPORT=<file>` | A PlugY stash file (`.sss`/`.hss`/`.d2x`) whose items are added read-only to the stash of every imported hero. |
| `OD2_NO_SETUP=1` | Skip the first-run dialogs (game folder picker). |
| `OD2_LANGUAGE=<code>` | Language override (same as the `-lang` flag). |
| `OD2_PLAY_VIDEOS=1\|0` | Force cinematics on or off. Default: on in normal play (Options, Accessibility, Cinematics switches it), off in scripted runs. |
| `OD2_LIGHTING=0` | Turn the light map off. |
| `OD2_LIGHT_QUALITY=0\|1` | Turn off the occluded (line-of-sight) light painter. |
| `OD2_LIGHT_OBJ_LOS=0` | Turn off line-of-sight occlusion of object lights. |
| `OD2_BACKDROP=0` | Turn off the neighbour-level backdrop drawn beyond the map edge. |
| `OD2_PICKUP=cursor\|quick` | `cursor`: a clicked ground item goes to the cursor as in the original; `quick`: straight into the inventory, also under a script. |
| `OD2_NOPERSIST=1` | A revisited level is rebuilt instead of keeping its monsters and ground items. |
| `OD2_PREFETCH=0` | Turn off background asset decoding. |
| `OD2_EXE_MENU_LAYOUT=1` | Escape menu pentagrams at the original's positions. |
| `OD2_SUBTITLE_LOG=1` | Log speech subtitles (same as Options, Accessibility, Speech subtitle log). |
| `OD2_TXT_WARN=1` | Warn about wrong-case or missing column names when reading the game's `.txt` tables. |

## Multiplayer

| Variable | Meaning |
|---|---|
| `OD2_HOST=1` | Host a network game (TCP). |
| `OD2_JOIN=<host:port>` | Join a hosted game. |
| `OD2_JOIN_RETRY=<seconds>` | Keep retrying the connection that long. |
| `OD2_PORT=<n>` | Server and client port (default in `d2server/game_server.go`). |
| `OD2_BIND=<addr>` | Address the host listens on. |
| `OD2_PROTO=d2gs` | Speak the Diablo II game protocol (`d2gsnet`) instead of the engine's own packets. |
| `OD2_PLAYERS=0..8` | Force the player count used for monster scaling. |
| `OD2_IDLE_KICK=1` | Turn on the idle-player kick (opt-in; the exe's idle values are not understood). |
| `OD2_IDLE_KICK_MS=<ms>` | Test-only: idle kick after that many milliseconds. |
| `OD2_HITQUEUE=0` | Turn off the PvP hit-queue purge (on by default). |
| `OD2_REALM_DUMMIES=<n>` | Number of training dummies in the town of the realm scenario. |

## Game-logic switches and debug aids

| Variable | Meaning |
|---|---|
| `OD2_REALMAPS=0` | Opt out of the real (DRLG) level providers; the old placeholder map is used. |
| `OD2_POPULATE=1\|0`, `OD2_NOPOPULATE` | Force natural monster population on or off (default: on only in scripts that fight). |
| `OD2_POPULATE_DENSITY=<percent>` | Scale every level's monster density (test knob). |
| `OD2_NOMERC=1` | Leave the mercenary at home. |
| `OD2_NOREGEN=1` | Turn off natural regeneration (debug). |
| `OD2_DURABILITY_CHANCE=<percent>` | Test override of the durability loss chance. |
| `OD2_POTION_ROLL=off` | Turn off the doubled potion roll. |
| `OD2_BELTSTEAL_ROLL=<n>` | Test-only: fix the belt-steal roll. |
| `OD2_ENTRY=rule` | Outdoor entry placement by the rule instead of the default (debug). **(unverified)** |
| `OD2_MAZE_STAMP=1`, `OD2_TOWN_STAMP=1` | Debug: use the old stamped maze or town tiles instead of the exact generator. |
| `OD2_DEBUG_MARKERS` | Debug markers in the maze generator. **(unverified)** |
| `OD2_DRAWSTATS=1` | Log what each frame drew (`DRAWSTATS` line every two seconds). |
| `OD2_PREFETCH_LOG` | Log the prefetch decoder. |
| `OD2_PPROF_CPU=<file>`, `OD2_PPROF_HEAP=<file>` | Write a CPU or heap profile (`scripts` also read `OD2_PPROF_CPU_OUT`/`_HEAP_OUT`). |

## Test harness: start, speed, windows

| Variable | Meaning |
|---|---|
| `OD2_AUTOGAME=<file>` | Start that character directly (a `.d2s` is imported first). |
| `OD2_AUTOEXIT` | Quit when the scripted run ends (exit 0 on PASS). |
| `OD2_AUTOTEST_MUTE` | No audio output (the sound engine still tracks voices). |
| `OD2_AUTOSCRIPT='step;step'` | Scripted hero actions (`d2game/d2autoscript`); ends with `AUTOSCRIPT RESULT PASS` or `FAIL`. |
| `OD2_AUTOSCRIPT_HEAL=1` | Keep the scripted hero's life topped up (Hell scenarios). |
| `OD2_AUTOSPEED=<1..32>` | Run game time that many times faster than real time. |
| `OD2_TURBO` | Run many 25 Hz ticks per frame (long playthroughs). |
| `OD2_TURBO_BUDGET_MS`, `OD2_TURBO_DRAW_EVERY` | Tune turbo (defaults 12 ms and 10). |
| `OD2_QUIET` | Window behind the others, no focus, real mouse and keyboard ignored, no Dock icon. |
| `OD2_REAL_INPUT=1` | Let the real mouse through during `OD2_AUTOSCRIPT`. |
| `OD2_AUTOFLOW=<steps>` | Drive the front-end menus without clicking. |
| `OD2_AUTOMENU=<names>` | Open each NPC's menu and log its rows. |
| `OD2_AUTOMENU_CHOOSE=<row>` | Pick that menu row. |
| `OD2_AUTOMENU_HOLD=<s>` | Keep the last menu open that long. |
| `OD2_AUTOTALK=<names>` | Log and play each NPC's greeting. |
| `OD2_AUTOSCREEN=charselect\|cinematics` | Open that screen directly. |
| `OD2_AUTOSHOT=<file.png>` | Save a frame after a delay (never commit such files). |
| `OD2_AUTOSHOT_DELAY`, `OD2_AUTOSHOT_SECONDS` | Delay or run time before the frame is taken. |
| `OD2_AUTOLEVEL=<id>` | Start in that level. |
| `OD2_AUTOMAP=<id>` | Generate that level from the hero's seed and log a summary. |
| `OD2_AUTOMAP_DIFF`, `OD2_AUTOMAP_ASCII` | Difficulty for the above; print the room list and a walkability map. |
| `OD2_AUTOMAPSEED=<n>` | Play the maps of another game seed. |
| `OD2_AUTOTIME=<phase>[@deg]` | Force and freeze the day/night clock. |
| `OD2_AUTODIFFICULTY=0\|1\|2` | Start in that difficulty. |
| `OD2_AUTODIFFICULTY_FORCE=1` | Skip the difficulty unlock rule. |
| `OD2_AUTOMARKDEAD=1` | Mark the start hero dead (hardcore test aid). |
| `OD2_AUTOOPTIONS=1` | Drive the options menu rows and check `config.json`. |
| `OD2_AUTOSAVE=1` | Set gold, save, exit (save test). |
| `OD2_AUTONEWCHAR=<class>[,hardcore][,classic][,ladder]` | Create a new character and compare it with a reference. |
| `OD2_AUTONEWCHAR_NAME`, `OD2_AUTONEWCHAR_REF` | Its name; path of a real new-character file for a field-by-field diff. |
| `OD2_HERO=<class>` | Hero class for the long playthrough scenarios (`scripts/verify.d/lib/hero.sh`). |
| `OD2_SETUP_AUTOPICK=<path>` | Answer the first-run folder picker with this path. |
| `OD2_AUTO` | Prefix only: any `OD2_AUTO*` variable marks a scripted run (movies off, window kept). |

## Test harness: scenarios

| Variable | Meaning |
|---|---|
| `OD2_AUTOAI=<ai\|monster>[,state=...]` | AI archetype test; `OD2_AUTOAI_ALSO` adds a second subject, `OD2_AUTOAI_SECONDS` bounds it. |
| `OD2_AUTOAMBIENT` | Ambient sound test; `_PHASES` day phases, `_SECONDS` per phase, `_SPEED` timer multiplier. |
| `OD2_AUTOBELT=<cols>` | Drink the belt potion of each column. |
| `OD2_AUTOBOSS` | Marks a solo boss test (keeps the merc home). **(unverified)** |
| `OD2_AUTOBOSSFIGHT=<key,...>` | Fight each named boss in turn; `OD2_BOSSFIGHT_SECONDS`, `OD2_BOSSFIGHT_RING`, `OD2_BOSSFIGHT_HELPERS` tune it. |
| `OD2_AUTOCAST=<skills>` | Cast skills against monsters. Helpers: `_CLASS`, `_CLVL`, `_LEVEL`, `_MANA`, `_SKILLPOINTS`, `_CROWD`, `_SPREAD`. |
| `OD2_AUTOCHEST=<seed>[,ids]`, `OD2_AUTOCHEST_SEED` | Spawn chests and barrels and open them. |
| `OD2_AUTOCUBE=<names>` | Cube recipes test; `_LADDER=1` ladder recipes, `_SEED` recipe rolls. |
| `OD2_AUTODEATH` | Death and respawn test; `_HARDCORE`, `_HP`, `_LEVEL`, `_MONSTER` tune it. |
| `OD2_AUTOEQUIP=1` | Equip-rule scenario. |
| `OD2_AUTOGAMBLE=<names>`, `OD2_AUTOIDENTIFY=1`, `OD2_AUTOTRADE=<names>` | Vendor gamble, Cain identify, buy/sell tests; `OD2_AUTOTRADE_SEED`, `_LEVEL`, `_KEEP` tune trade. |
| `OD2_AUTOGROUND=<seed>[,count]` | Drop items and gold; `_TC`, `_ILVL`, `_HOLD` tune it. |
| `OD2_AUTOMERC` | Mercenary scenario; `_EXP`, `_HEROLEVEL`, `_HOME`, `_KILL`, `_LEVEL`, `_LEVELS`, `_MERCLEVEL`, `_SECONDS`, `_TYPES`, `_UI` tune it. |
| `OD2_AUTOMONSTER=<names>` | Spawn monsters and fight; `_ALWAYSHIT`, `_AREA`, `_DIFF`, `_FAR`, `_HOLD`, `_PASSIVE`, `_SECONDS`, `_SUMMONS` tune it. |
| `OD2_AUTOOBJECT=<id\|name>[,n]` | Spawn objects; `_FORCE`, `_GIVE`, `_GIVE_AFTER`, `_LIFE`, `_MANA`, `_MONSTER`, `_NEAR`, `_SHRINE` tune it. |
| `OD2_AUTOPANEL=<names>`, `OD2_AUTOSTASH=1`, `OD2_AUTOUILAYOUT` | Open panels, walk to the stash, log exact UI positions; `OD2_AUTOPANEL_HOLD=<s>` keeps the last one open. |
| `OD2_AUTOPARTY` | Party test marker (monsters ignore town). **(unverified)** |
| `OD2_AUTOPERF=1` | Frame-time meter; `_SECONDS` (default 20), `_WARMUP` (default 6). |
| `OD2_AUTOPICKUP=1` | Walk to and pick up every ground item. |
| `OD2_AUTOQUEST=<act\|stage>` | Quest scripts; `_AREA` level id, `_DIFF` difficulty, `_REAL=1` real monsters. |
| `OD2_AUTOSOUND=<handle,...>`, `OD2_AUTOSOUND_TRACE` | Play sounds; trace the voice rules. |
| `OD2_AUTOUBER=<rewards\|travel\|cube\|uber>` | Pandemonium scripts. |
| `OD2_SOUNDLOG=1` | Log every sound request and voice event. |
| `OD2_VERIFY_SOUND=1` | Play real audio in scenarios instead of muting (also named by the sound audit code). |
| `OD2_VIDEO_SECONDS=<n>` | End a movie after n seconds. |
| `OD2_VIDEO_TEST_FILE`, `_FRAMES`, `_PACKETS`, `OD2_VIDEO_REF_PCM` | Bink decoder tests on a `.bik` you extracted yourself (skipped when unset). |

## macOS app wrapper

| Variable | Meaning |
|---|---|
| `OD2_WATCH_PID`, `OD2_WATCH_LOG` | Set by the app launcher so the crash watcher can write into `~/Library/Logs/OpenDiablo2`. Not for users. |

## Read only by `scripts/`

| Variable | Meaning |
|---|---|
| `OD2_VERIFY_ONLY=<regex>` | Run only matching scenarios. |
| `OD2_VERIFY_CONFIG_DIR`, `OD2_VERIFY_SAVE`, `OD2_VERIFY_BIN`, `OD2_VERIFY_LAUNCH`, `OD2_VERIFY_KEEP`, `OD2_VERIFY_TMPFILE` | Private config, start save, prebuilt binary, launch prefix, keep scratch dir, write scratch path. See [TESTING.md](TESTING.md). |
| `OD2_VERIFY_EXTRA_ENV`, `OD2_VERIFY_MUTE_ENV`, `OD2_VERIFY_SPEED` | Extra game env, mute setting, game-clock factor (default 8; `1` = real time). |
| `OD2_VERIFY_FLAKE_DIR`, `OD2_FLAKE_TAG`, `OD2_FLAKE_BURNER` | `flake_loop.sh` capture folder, output tag, CPU burner. |
| `OD2_MAX_GAMES` | Machine-wide cap on game windows (`gameslot.sh`). |
| `OD2_VERIFY_NIGHTLY=1`, `OD2_CLASSES`, `OD2_CLASS_SCENARIOS`, `OD2_CLASS_MATRIX_DIR`, `_DOC`, `_FORCE`, `_VERIFY` | Slow class matrix (`verify_classes.sh`). |
| `OD2_PERF_CAMP_ZONE`, `_SECONDS`, `_WARMUP`, `_ENV`, `OD2_PERF_W41_MODE`, `OD2_PERF_W41_SECONDS` | Perf scenario settings. |
| `OD2_W36_BOSSES`, `OD2_W36_DIFF`, `OD2_W36_RING`, `OD2_W36_SECONDS` | Boss-fight scenario settings. |
| `OD2_HUNT_CLASS`, `OD2_HUNT_SKILL`, `OD2_KEEP_CAPTURES`, `OD2_QA_DIR`, `OD2_BAR_FILL_TEST` | Hunt and QA scenario helpers. **(unverified)** |

## Scenario helper variables, in full

The scenario rows above abbreviate their helpers (`_SECONDS` and so on). The same variables with their full names, so
they can be found with grep. All are test-only and take effect only together with their parent variable.

| Variable | Meaning |
|---|---|
| `OD2_AUTOAMBIENT_PHASES=2,4` | Day phases to visit (default day then night). |
| `OD2_AUTOAMBIENT_SECONDS` | Seconds per phase. |
| `OD2_AUTOAMBIENT_SPEED` | Multiplier on the ambient event timers. |
| `OD2_AUTOCAST_CLASS=<class>` | Cast test with a fresh temporary hero of that class (no real save touched). |
| `OD2_AUTOCAST_CLVL=<n>` | Character level of that fresh hero. |
| `OD2_AUTOCAST_LEVEL=<n>` | Skill level used for the casts. |
| `OD2_AUTOCAST_MANA=<n>` | Give the hero a full pool of n mana. |
| `OD2_AUTOCAST_SKILLPOINTS=<n>` | Hand the fresh hero n unspent skill points. |
| `OD2_AUTOCAST_CROWD=<n>[,<state>]` | Each cast waits until n neighbours (optionally in a state) stand around the target. |
| `OD2_AUTOCAST_SPREAD=<state>,<n>` | Extra casts for skills that must spread a state. |
| `OD2_AUTOCUBE_LADDER=1` | Make the cube test a ladder game. |
| `OD2_AUTOCUBE_SEED=<n>` | Seed the recipe rolls. |
| `OD2_AUTODEATH_HARDCORE=1` | Play a hardcore death (no respawn). |
| `OD2_AUTODEATH_HP=<n>` | Start the fight with n life. |
| `OD2_AUTODEATH_LEVEL=<n>` | Hero level for the experience-penalty test. |
| `OD2_AUTODEATH_MONSTER=<key>` | Monster that kills the hero. |
| `OD2_AUTOGROUND_HOLD=<s>` | Keep the game open that long when done. |
| `OD2_AUTOGROUND_ILVL=<n>` | Item level of the drops (default hero level). |
| `OD2_AUTOGROUND_TC=<name>` | Treasure class dropped from (default `Andariel`). |
| `OD2_AUTOMERC_EXP=<n>` | Experience given to the merc. |
| `OD2_AUTOMERC_HEROLEVEL=<n>` | Raise the hero's level so a low-level merc earns experience. |
| `OD2_AUTOMERC_HOME=<n>` | Level-change test: how many steps take the hero back to town (default 1). |
| `OD2_AUTOMERC_KILL=1` | Kill and revive the merc during the fight. |
| `OD2_AUTOMERC_LEVEL=<n>` | Merc level in the level-change test (default 29). |
| `OD2_AUTOMERC_LEVELS` | Run the merc through level changes. |
| `OD2_AUTOMERC_MERCLEVEL=<n>` | Merc level in the types test (default 40). |
| `OD2_AUTOMERC_SECONDS=<s>` | Length of the fight phase. |
| `OD2_AUTOMERC_TYPES=<ids>` | Hire each hireling id in turn and fight. |
| `OD2_AUTOMERC_UI=1` | Probe the hire lists, portrait and potion drop on it. |
| `OD2_AUTOMONSTER_ALWAYSHIT=1` | The test hero's attacks always hit. |
| `OD2_AUTOMONSTER_AREA=<id>` | Use that area (levels.txt id) for the monster rules. |
| `OD2_AUTOMONSTER_DIFF=<0..2>` | Difficulty of the monsters. |
| `OD2_AUTOMONSTER_FAR=<subtiles>` | Put every second monster on a far ring. |
| `OD2_AUTOMONSTER_HOLD=1` | Monsters are only scenery for an `OD2_AUTOSCRIPT` run. |
| `OD2_AUTOMONSTER_PASSIVE=1` | Leave the hero idle to watch monsters attack. |
| `OD2_AUTOMONSTER_SECONDS=<s>` | Run time before the summary (default 30). |
| `OD2_AUTOMONSTER_SUMMONS=1` | Log the live monster-cast summons. |
| `OD2_AUTOOBJECT_FORCE=lock,trap=<h>` | Force a container roll (locked, spawn handler 1-9). |
| `OD2_AUTOOBJECT_GIVE=<code>[,code]` | Put items (a key, gems) in the inventory first. |
| `OD2_AUTOOBJECT_GIVE_AFTER=<code>` | The same, right after the first object was operated. |
| `OD2_AUTOOBJECT_LIFE=<n>` | Set the hero's life first (to see wells and shrines work). |
| `OD2_AUTOOBJECT_MANA=<n>` | Set the hero's mana first. |
| `OD2_AUTOOBJECT_MONSTER=<key>[,n]` | Spawn n monsters (default 2) next to the objects first. |
| `OD2_AUTOOBJECT_NEAR=1` | Place each object 2 subtiles from the hero. |
| `OD2_AUTOOBJECT_SHRINE=<code\|all>` | Force the shrine type. |
| `OD2_AUTOPERF_SECONDS=<n>` | Measured seconds (default 20). |
| `OD2_AUTOPERF_WARMUP=<n>` | Seconds ignored first (default 6). |
| `OD2_AUTOQUEST_AREA=<id>` | levels.txt id the map stands for when the run starts. |
| `OD2_AUTOQUEST_DIFF=<0..2>` | Difficulty of the quest record. |
| `OD2_AUTOQUEST_REAL=1` | Den of Evil: spawn real monsters and let the hero kill them. |
| `OD2_AUTOTRADE_KEEP=1` | Keep the traded items instead of undoing the transactions. |
| `OD2_AUTOTRADE_LEVEL=<n>` | Hero level used for the vendor stock. |
| `OD2_VIDEO_TEST_FRAMES=<n>` | Bink test: frames to decode (default 600). |
| `OD2_VIDEO_TEST_PACKETS=<n>` | Bink test: audio packets to decode. |
