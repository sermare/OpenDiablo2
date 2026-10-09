# Status matrix

One row per feature, with the three states that matter and the evidence for each. Read it with the
status board in the [README](../README.md) (what the maintainers consider done) and
[TESTING.md](TESTING.md) (how each kind of evidence is produced). Where this page and a package comment
disagree, the package comment (`doc.go`, `VERIFIED` / `UNVERIFIED` tags) is the detail and the code is the truth.

## Legend

| State | Meaning here |
|---|---|
| **implemented** | The code exists, is wired into the engine (or is a pure package with unit tests) and is exercised by a unit test or a `scripts/verify.d` scenario. |
| **verified-against-original** | The output was compared with the real game: (a) the emulator oracle (the real `Game.exe` generator run in an x86 emulator, numbers committed as goldens in `d2common/d2drlg/testdata`); (b) a real `.d2s` save, byte-identical round trip or equality with an independent reference parser; (c) a real game file or table (`default.key`, `Sounds.txt`, `DifficultyLevels.txt`, `lvlprest.bin`). **Unit tests that encode a reading of the decompiled binary do not count.** |
| **approximate** | Ported from reverse-engineering notes (binary-read, may be tagged `UNVERIFIED` in the code), modelled on visible behaviour, or an engine choice. It runs and is tested, but nobody compared it with the running original. |

A row can be implemented and verified, or implemented and approximate. "no evidence found" means the code and tests
were searched and nothing compares that part with the original.

**How evidence is gated.** Unit tests run everywhere (`go test ./...`). Tests that need game data skip themselves
unless `D2_TABLES` (extracted tables), `D2_DS1_ROOT` (extracted DS1/DT1 tiles), `D2_DS1`, `D2S_SAMPLE*`, `D2_INSTALL`,
`D2_PL2`, `D2_STRING_TBL`, `D2_DIFFICULTYLEVELS` are set; so a green CI run proves the
logic builds and passes its synthetic tests, not the oracle claims. Scenario evidence (`scripts/verify.d/NN-*.sh`) needs a
GUI session, a Diablo II install and a real `.d2s` (`D2S_SAMPLE_BODY`), and runs only through `scripts/verify.sh`. Scenarios show
that the pieces work together in the running engine; they never prove fidelity to the original on their own.

## 1. File formats and saves

| Feature | Implemented | Verified against original | Approximate / gaps | Evidence |
|---|---|---|---|---|
| MPQ, DC6, DCC, DS1, DT1, COF, PL2, TBL, TXT, DAT, animdata, font parsers | yes (upstream, extended) | no comparison beyond loading the real 1.14b data | `d2compression` WAV decoding has no byte comparison | tests in each `d2fileformats/*` package; `D2_PL2` test in `d2pl2`; real data boot |
| `.d2s` read: header, quests, waypoints, NPC flags, stats, skills, items, corpse, mercenary | yes | yes: matches the independent parser nokka/d2s on a real level-94 save (checksum, 16 attributes, 30 skills, 60/60 items) and on a second real save | fields tagged "varying" in the new-character test; some rarely used header bytes kept verbatim | `d2s`: `TestRealSave`, `TestRealBody`, `TestParseHeader`, `TestParseBodySynthetic` (real ones need `D2S_SAMPLE*`) |
| `.d2s` write | yes | yes: an unchanged parse writes back byte for byte | edited saves are checked by re-parse and checksum, not by loading in the real game | `d2s` write tests; `d2hero` `TestExportUnchangedIsByteIdentical`, `TestRealSampleContainersRoundTripThroughWriter`; `verify.sh` save-back step |
| New character `.d2s` | yes | yes for the bytes the test compares with a real new-character file | varying fields (timestamps, checksum) excluded | `d2s` new-character test (`D2S_SAMPLE_NEW`); scenario `88-death-newchar.sh` (`OD2_AUTONEWCHAR_REF`) |
| Import into the engine (`HeroState`) and export back, with gear | yes | partly: the real sample's totals equal the stored values | the `.od2` JSON is the engine's own format | `d2hero` `TestRealSaveTotalsMatchStoredCurrentValues`, `TestStoredFromD2S`; scenarios `80-imported-hero-ui.sh`, `30-autosave.sh`, `81-waypoint-persist.sh` |
| `default.key` key bindings | parser only | yes: layout matches the real 1146-byte file | the two flag fields and mouse codes 0x100-0x102 unverified; not applied to the in-game bindings | `d2key` `TestRealDefaultKey` (`D2_DEFAULT_KEY` or `D2_INSTALL`) |
| Diablo II random generator and level-seed hierarchy | yes | yes: independent Python vectors; README states it was checked instruction by instruction | none known | `d2rand` `TestStepSequence`, `TestRollRules`, `TestSeedHierarchy`; also every DRLG oracle (a wrong draw changes the final seed) |

## 2. Level generation (DRLG) and maps

| Feature | Implemented | Verified against original | Approximate / gaps | Evidence |
|---|---|---|---|---|
| Table loading (`Levels`, `LvlMaze`, `LvlPrest`, `LvlTypes`, `LvlSub`) | yes | yes: compiled `lvlprest.bin` equals the txt for the real tables | none known | `d2drlg` `TestRealTablesBinMatchesTxt` (`D2_TABLES`) |
| Act 1 world layout | yes | yes: 50 seeds | which rectangle `ValidateCluster2` inflates is `UNVERIFIED` (see package comment) | `drlgworld` `TestOracleWorld`; golden `world_act1_normal.json` |
| Maze levels, Acts 1-5 (caves, crypts, sewers, tombs, lairs, Arcane Sanctuary, dungeons, Durance, Act 4/5 mazes) | yes | yes: rooms, def and file index and final level seed against the emulator | DS1 object gates only for the preset files that have any; level 107 and the Act 4/5 mazes live in `drlgmaze` | `drlgmaze` `TestOracleMaze`; goldens `maze_act1.json`, `maze_act23.json`, `maze_act45.json`; `acts.json` via `d2drlg` `TestOracleActExtras` |
| Act 1 outdoor levels 2-7, 0x11, 0x27 (up to the room grids) | yes | yes: stage numbers, room list and digests of the large grids | other Act 1 level ids are not covered by the golden | `drlgoutdoor` `TestOracleOutdoor`; golden `outdoor_act1.json` (needs `D2_TABLES`, `D2_DS1_ROOT`) |
| Act 1 preset levels | yes | yes | only the golden's seeds | `drlgoutdoor` `TestOraclePresetAct1`; golden `preset_act1.json` |
| Act 2 and Act 3 outdoor (levels 41-46, 76-83, towns 40 and 75) | yes | yes: creation draws and room data for the sampled games | DT1 tile pick, preset-room flags and DS1 object gates for Act 3 files the sampled games never drew are unverified | `drlgoutdoor` `TestOracleAct2Levels`, `TestOracleAct3Levels`, `TestAct23World`, `TestTowns`; goldens `outdoor_act2.json`, `outdoor_act3.json` |
| Act 4 and Act 5 outdoor and preset levels | yes | yes: 12 seeds x 3 difficulties x 16 levels | level 134 (Forgotten Sands, Act 2 desert style generator) is not ported | `drlgoutdoor` `TestOracleAct45`; golden `outdoor_act45.json` |
| Per-cell tile records (tile ids) for Acts 1-5 | yes | yes against the tile goldens | the DT1 library is not emulated: the random tile pick is modelled as one room-seed step (`RoomBuildOptions.PickTile`); tile paths no golden room reaches return an error | `drlgoutdoor` `TestOracleTiles`; goldens `tiles_act1.json`, `tiles_act23.json`, `tiles_act45.json`; unit tests in `tiles_test.go` |
| Map provider wiring (maze and outdoor levels in the engine, stamps, rooms) | yes, behind `OD2_REALMAPS=1` | no: only the pure generators are compared | the engine renders and walks the result; fidelity of rendering is not compared | scenarios `70-real-maps.sh`, `71-real-outdoor.sh`, `72-real-act45.sh`, `82-maze-travel.sh`, `83-cave-chain-persist.sh`; `d2mapgen` `TestProviderOrder`, `TestAct2WorldBordersForEverySeed` |
| Act towns 1-5 | yes | partly: the start markers and NPC counts of the real town DS1s | which Lut Gholein variant is chosen by seed is from notes | `d2mapgen` `TestRealTownDS1` (`D2_DS1`), `TestChooseTownStart`, `TestTownFileIndex`; scenarios `99-act-travel.sh`, `9f-act2-lutn.sh` |
| Level graph, waypoints, portals, act travel rules | yes | no | binary-read from `SERVER_ChangePlayerLevel`; the special waypoint arrival start type is `UNVERIFIED` | `d2level` `TestCheckActTravel`, `TestRuleForNPC`, `TestCaveEntranceDestination`, waypoint tests; scenarios `80-waypoint-portal.sh`, `81-waypoint-persist.sh`, `99-act-travel.sh` |
| Level persistence (a revisited level keeps its state) | yes (engine) | no | an engine choice; `OD2_NOPERSIST=1` turns it off | scenario `83-cave-chain-persist.sh` |
| Collision grid and click-to-move pathfinder | yes | no | binary-read (straight line, short A* with costs 2/3) | `d2path` tests (`AStarOptimalCost`, `PartialWhenGoalSealed`, ...), `d2mapengine` path tests |
| Automap (table, tile to cell lookup, projection, reveal) | yes | no | binary-read; the orientation read from the DT1 is "strongly suggested" | `d2automap` `TestBinMatchesTxt` and the lookup tests (real-table ones skip without `D2_TABLES`); scenario `91-automap.sh` |
| Day/night clock and ambient colour | yes | no | colour blend and phase advance are modelled (`UNVERIFIED`) | `d2daynight` tests; `d2maprenderer` `TestTintUsesShadeRow`; scenario `91-ambient-env.sh` |
| Light map (48x48 subtile lights) | yes | no | line-of-sight painter and reciprocal table not implemented or assumed | `d2lightmap` tests; `d2maprenderer` `TestAdvanceLightingRebuildsEachTick`; `OD2_LIGHTING=0` disables |

## 3. Monsters and AI

| Feature | Implemented | Verified against original | Approximate / gaps | Evidence |
|---|---|---|---|---|
| Monster AI framework (brain, per-unit RNG, think envelope) | yes | no | binary-read; per-function tags in `d2monster` comments | `d2monster` (about 100 tests: `Mummy`, `ScarabGroupAlert`, `BruteAip3Twice`, ...) |
| Think functions: the classic archetypes, Andariel, Smith, Blood Raven, Vulture, Summoner, Duriel, Mephisto, Diablo, Izual, Baal minions, Fallen Shaman | yes | no | `MONAI_PostTargetChecks` hooks (wounded teleport, Summoner wake-up, threat re-targeting), Tentacle, FrogDemon and the other remaining names are stand-ins in `ai_rest.go` | `d2monster` tests; scenario `40-monster-pack.sh`, `97-ai-states.sh` |
| Forced states (fear, blind, taunt, confuse, attract, charm) | yes | no | binary-read from the alternate AI table | `d2monster` forced-state tests; scenario `97-ai-states.sh` (`OD2_AUTOAI`) |
| Engine glue: spawning from DS1, collision cells, melee at the animation halfway frame, ranged shots | yes | no | one cell per unit; unit blocking mask `UNVERIFIED`; champion/unique modifiers (monumod) and monster-vs-monster targeting (except forced states) not ported; hero defence uses dexterity only; corpse lifetime is an engine choice | `d2monsters` tests (`ComputeVitalsPerDifficulty`, `AttackPlans`, ...); scenarios `40-monster-pack.sh`, `95-perf.sh`, `90-ambient-audio.sh` |
| Monster stats per difficulty | yes | partly: scaling formula and `DifficultyLevels.txt` rows read from the real tables | the progression byte of the header is `UNVERIFIED` | `d2difficulty` `TestRealTable` (`D2_DIFFICULTYLEVELS`), `TestScaleStat`, `TestMonsterStat`; `d2monsters` `TestComputeVitalsPerDifficulty`; scenario `9a-difficulty.sh` |
| Difficulty selection and per-difficulty progress | yes | no | stored in the `.d2s` active difficulty byte | `d2hero` `TestRealSaveDifficultyChoice`; scenario `9a-difficulty.sh` |
| Boss encounters (Duriel tomb, Mephisto, Diablo seals, Baal waves) | yes | no | ids and the wave cycle binary-read; delays, positions and portal objects `UNVERIFIED` | `d2boss` tests (`DurielTomb`, `SealsAndDiablo`, `ThroneWaves`); scenario `9c-bosses.sh` (`OD2_AUTOBOSS`) |

## 4. Skills, combat and stats

| Feature | Implemented | Verified against original | Approximate / gaps | Evidence |
|---|---|---|---|---|
| Calc language of `skills.txt` / `missiles.txt` | yes (`d2calc`) | partly: every row of the real tables compiles and evaluates | `rand` modulus, `sklvl` argument order unverified | `d2calc` `TestRealTablesCompile`, `TestEvalRealSkillRows`, `TestLanguageQuirks` (real ones need `D2_TABLES`) |
| Combat math (to-hit, defence, block, crit, damage, mana cost) | yes | no | binary-read; functions say "inferred" where not read directly | `d2combat` tests (24) |
| Skill cast pipeline (start: validity, mana, cooldown; do: missiles, melee, effects) | yes | no | per-skill behaviour inferred from table columns is marked `U` in handlers; Telekinesis and Find Item not implemented | `d2skill` tests (31); `d2skills` glue; scenario `60-skill-cast.sh` |
| Class skills, all seven classes (about 130 skills) | yes | no | the README states several formulas were spot-checked in the binary and others are unverified | `d2skill` class tests (`Curses`, `Auras`, `Summons`, `TeleportAndWalls`, ...); `TestRealClassSkillsAreImplemented` (real tables); scenario `86-class-skills.sh` |
| Missiles (move, collision, pierce, sub-missiles) | yes | no | no homing, area hit functions, periodic effects; direction is exact rather than the original 64 steps | `d2missile` tests (19); `TestRealMissileSpecs` (real tables) |
| Timed states, auras, poison and burn | yes | partly: poison total equals the in-game tooltip arithmetic | stacking of several poisons and the exact slow percents unverified | `d2state` `TestPoisonTotalMatchesTooltip`, `TestApplyExpireAndRefresh` |
| Stat lists (vitality to life, defence, attack rating, resist caps, block) | yes | partly: totals of a real save equal its stored values | speed breakpoints etc. binary-read | `d2statlist` tests; `d2hero` `TestRealSaveTotalsMatchStoredCurrentValues`; scenario `89-hero-stats.sh` |
| Skill bar, hotkeys, skill points with prerequisites | yes | no | saved in the `.d2s` header; round trip checked by re-parse | scenario `98-skillbar.sh` |

## 5. Items, loot, vendors, trade

| Feature | Implemented | Verified against original | Approximate / gaps | Evidence |
|---|---|---|---|---|
| Treasure classes, quality roll, affix selection | yes | no: real-table tests check that the tables load and that frequencies are plausible, not that the original produced the same item | parts derived from branch structure are `UNVERIFIED` in the package comment | `d2drop` `TestRealTablesLoad`, `TestRealRollDeterministic`, `TestRealQualityFrequencies`, `TestQuality*` |
| Ground drops, chests, gold | yes | no | item level of object drops is the hero level (the original uses the area level) | `d2ground` tests; scenarios `50-containers.sh`; `OD2_AUTOGROUND`, `OD2_AUTOCHEST` |
| Equip rules, requirements, durability loss, broken pieces | yes | no | binary-read (formulas and tables at fixed addresses in the notes) | `d2equip` tests; scenario `93-equip-rules.sh` |
| Inventory, stash, cube (storage), belt | yes | partly: containers of a real save round-trip | cube recipes: `cubemain.txt` is loaded as records (`d2records`) but no transmute logic was found in the code, so crafting is not implemented | `d2inventory` tests (10); `d2hero` container round-trip tests; scenario `50-containers.sh` |
| Vendor stock and gamble stock | yes | no | binary-read from `TRADE_GenerateVendorStock` | `d2vendor` tests (12); scenarios `10-menus-trade.sh`, `87-gamble-identify.sh` |
| Buy, sell, repair, gamble and identify prices | yes | no | 32-bit integer maths as in the original; no comparison of printed prices with the running game | `d2trade` tests (`ItemPrice`, `GamblePrice`, `IdentifyCost`) |
| NPC menus and greeting logic | yes | partly: menu rows and greeting rows come from the real tables | not checked visually in the board's words | scenario `10-menus-trade.sh` (`OD2_AUTOTALK`, `OD2_AUTOMENU`) |
| Mercenaries (hire offers, stats, skills, experience, revive) | yes | partly: stat computation and level checked against a real save | the offer table is a model of the per-game roll | `d2hireling` `TestRealNokkaMercLevel`, `TestRealStatsHandComputed`, `TestRealOffers`; scenario `84-mercenary.sh` |

## 6. Quests and world objects

| Feature | Implemented | Verified against original | Approximate / gaps | Evidence |
|---|---|---|---|---|
| Quest state machine, Act 1 and Radament's Lair | yes | partly: the speech tables were compared byte for byte with the notes' reading of the binary; quest bits go into a real `.d2s` | clean-room D2MOO logic, a single player instead of party loops | `d2quest` tests (41); scenarios `85-quests.sh`, `9b-act1-playthrough.sh`, `9c-act1-reload.sh` |
| Quests of Acts 2-5 | yes | no | Act 3-5 triggers are table driven and `UNVERIFIED` (`generic.go`) | `d2quest` `TestAct2To5MessagesHaveNPCsAndSounds`, `TestEveryQuestHasItsSlot`; scenario `85-quests-acts2-5.sh`, `9e-act2-playthrough.sh` |
| Shrines, wells, weapon racks and other operate functions | yes | no | binary-read dispatch table and shrine states; chest/barrel drop details approximate | `d2object` tests (`TestRealShrines` needs `D2_TABLES`); scenario `92-objects2.sh` |
| Waypoint activation persisted to `.d2s` | yes | no | writes the save-file bits | scenario `81-waypoint-persist.sh` |

## 7. Death, hardcore, options, audio

| Feature | Implemented | Verified against original | Approximate / gaps | Evidence |
|---|---|---|---|---|
| Death, respawn, corpse, penalties | yes | no | engine choices for what the notes do not cover | scenario `88-death-newchar.sh` |
| Hardcore death and load refusal | yes | no | flags written into the `.d2s` | scenarios `8c-hardcore.sh`, `8d-hardcore-death.sh` |
| Sound voice allocation (priority, groups, variants) | yes | partly: Sounds.txt rows read from the real file | binary-read rules; some tagged inferred | `d2sfx` tests, `TestRealSoundsTxt` (`D2_TABLES`); scenario `90-ambient-audio.sh` |
| Ambient sound, music by day/night, positional sound | yes | no | rate of the day cycle `UNVERIFIED` | `d2audio` tests; scenario `91-ambient-env.sh`; real audio run `9e-real-audio.sh` |
| Options menu | yes | no | saves to `config.json` | scenario `8b-options.sh` |
| First run discovery, `.app` bundle | yes | not applicable | | `d2setup` tests (5); `scripts/make-app.sh` |
| Frame time | measured | not applicable | | scenario `95-perf.sh` |

## 8. Networking and multiplayer

| Feature | Implemented | Verified against original | Approximate / gaps | Evidence |
|---|---|---|---|---|
| D2GS packet size tables, Huffman coder, framing | yes (`d2gs`) | server and client size tables read from the binary and pinned by tests; Huffman table read from the binary | client to server blob framing assumed equal to server to client | `d2gs` `TestExpectedSizeServer`, `TestExpectedSizeClient`, `TestHuffmanTable`, `TestBlobFraming` |
| Engine over D2GS (`OD2_PROTO=d2gs`) | yes (`d2gsnet`) | partly: walk/run, skill select and cast are verified layouts | join layout unverified; hero state, saves and waypoints ride in a tunnel in the variable packets | `d2gsnet` `TestJoinHandshake`, `TestMoveCastChatLeave`, `TestBigStateSurvivesTheTunnel`; scenario `96-multiplayer.sh` |
| Default JSON protocol, local and TCP clients | yes (upstream, extended) | no | UDP connection type is unused | `d2netpacket` round-trip tests; scenario `96-multiplayer.sh` |
| Party, roster, shared experience, hostility | yes | partly: the party list and kill sharing rules are binary-read | invitation flow and the level 9 requirement `UNVERIFIED` | `d2party` tests; `d2server` `TestServerPartyAndPvP`, `TestServerPartyXP`; scenario `9d-party-trade.sh` |
| Player to player trade | yes | no | session rules modelled on visible behaviour (`PlrTrade.cpp` not analysed) | `d2playertrade` tests; `d2server` `TestServerTrade`; scenario `9d-party-trade.sh` |
