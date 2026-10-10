# Progress targets: what each bar measures

Each bar is `done / total`. **done** counts only what is on the `integration` branch and has passed the full
`scripts/verify.sh`. The dashed segment (**pending**) is work that exists on branches but has not passed a full
verify yet; it is not counted in the percentage. Numbers marked `~` are estimates.

| Bar | Denominator | Numerator / how to count it |
|---|---|---|
| Game.exe functions named in Ghidra | all functions Ghidra finds in `Game.exe` (11,257) | `GET :8089/find_functions?has_custom_name=true` count; the last 22 are the anti-cheat module, skipped on purpose |
| Hedged exe names re-verified against the decompile | ~1,633 names flagged unverified in `d2-re-notes/naming-*.md` (estimate) | names reviewed in `names-verified-logic.md` and `names-verified-ui-net.md` (confirmed or corrected) |
| Original source files indexed | 277 assert-string source paths | paths mapped to modules and functions |
| Source files studied in depth | 277 | files with notes in `d2-re-notes` |
| Level generator proven equal to the real game | records in the goldens | sum of oracle golden records that match (mazes, outdoors, presets, tiles) |
| Real .d2s save: read and write verified | 10 file sections | sections parsed and rewritten byte for byte |
| NPC interaction features | 8 | click-walk, greeting, menu, return greeting, vendors, gamble, identify, mercenary hire |
| Acts playable end to end (scripted play) | 5 | acts whose scripted playthrough scenario passes (`9b`, `9e`, `9g`, `9h`, `9i`) |
| Levels walked by a verify scenario | 136 levels in Levels.txt | distinct level ids a scenario enters (`walkto:exit=`, `expect:level=`, `from= to=` log checks); a lower bound |
| Main quests with logic implemented | 27 (6+6+6+3+6) | quests whose states, triggers and rewards are in `d2quest` and tested |
| Player skills with engine behaviour | 210 (7 classes x 30) | skills the `d2skills` engine runs (~130 now) |
| Monster AI archetypes implemented | 148 `monai.txt` entries | AI names with a Go think function (145; Tentacle, TentacleHead, FrogDemon missing) |
| Monster AI archetypes ported faithfully | 148 | ~85: ported from the exe think functions, not stand-ins |
| In-game UI panels matching the original layout | 14 panels | panels whose logged rectangles equal the golden from the exe in a real game run (0 so far) |
| Item generation gaps closed vs the exe | 12 gaps in the itemgen notes | gaps closed and checked against the emulator |

How to update: edit `docs/progress.json` (`done`, `pending`, `total`, `note`), run `python3 scripts/make_progress_svg.py`,
commit. A bar moves from pending to done only when its branch is merged into `integration` and `verify.sh` passed on it.

## Game v1 complete (the headline bar)

**Definition of v1:** a person double-clicks the Mac app, makes or loads a character and plays Acts 1-5 single player
(quests, items, skills, monsters, saves, sound) without hitting a blocker; multiplayer through the game screen and
the Mac packaging count too. The percentage is the weighted average of the bars below (weights sum to 100, kept in
`docs/progress.json` under `v1.weights`). Reverse-engineering bars (names in Ghidra, files studied) are deliberately
not in it: they support the game but are not the product. The dashed part is work on branches that has not passed the
full verify yet.

| Weight | Bar |
|---|---|
| 14 | Acts playable end to end (scripted play) |
| 14 | Acts played start to finish by a human tester (0 until a person plays them) |
| 6 | Levels walked by a verify scenario |
| 8 | Main quests with logic implemented |
| 10 | Player skills with engine behaviour |
| 6 + 2 | Monster AI archetypes ported faithfully + implemented |
| 6 | Item generation gaps closed vs the exe |
| 8 | In-game UI panels matching the original layout |
| 6 | Sound and music confirmed by a human listen |
| 4 | Multiplayer playable through the game screen |
| 4 | Mac app: build, Finder launch with real maps, disk image |
| 4 + 4 + 4 | Level generator proven, .d2s save verified, NPC interaction features |

New bars (all start honest): *Sound and music confirmed by a human listen* (0/5: only a person can confirm
menu music, NPC voices, footsteps, spell sounds, ambient), *Multiplayer playable through the game screen* (0/4),
*Mac app* (2/3), *Acts played start to finish by a human tester* (0/5: the real completeness gauge).
