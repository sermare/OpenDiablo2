# Contributing to this fork

This is a fork of [OpenDiablo2](https://github.com/OpenDiablo2/OpenDiablo2) that aims at a native macOS Diablo II
playable with **your own game data**. The upstream contribution notes are in [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md);
this page describes how work is organised here.

Start with the three guides:

* [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): the packages and how data flows
* [docs/TESTING.md](docs/TESTING.md): unit, real-data, oracle and in-game tests; the verify gate
* [docs/REVERSE_ENGINEERING.md](docs/REVERSE_ENGINEERING.md): where the facts come from, the legal stance, the rules

## Never commit game files

No MPQs, extracted tables, string files, DS1/DT1/DC6/DCC assets, audio, executables or character saves, and no
decompiled Blizzard code. You need your own copy of Diablo II and Lord of Destruction. Tests that need real data read it
from environment variables (`D2_TABLES`, `D2S_SAMPLE_BODY`, ...) and skip when unset. Check `git status` before committing.

## Branches and merging

* Remotes: `origin` is upstream OpenDiablo2, `fork` is this fork. Push to `fork`.
* `integration` is the branch where everything merged and building lives. Start from it:
  `git checkout -b <your-branch> integration`.
* One feature or fix per branch, named by kind: `feat/<thing>`, `fix/<thing>`, `docs/<thing>`.
* Keep the change set to the packages your task concerns. Do not reformat files you do not own (older upstream files are
  intentionally not gofmt-clean); several branches are merged together, so drive-by edits cause conflicts.
* Merging into `integration` is done by the maintainer with a merge commit
  (`Merge remote-tracking branch 'fork/feat/<thing>' into integration`). Before it is merged, merge `integration` into
  your branch if it moved, and re-run the gate.
* The README status board is updated when something is **verified** (not assumed).

## The gate before a branch is ready

1. `gofmt -l <files you changed>` prints nothing; `go vet` is clean for the packages you touched.
2. `go build ./... && go test ./...` pass (the linker warning about duplicate `-lobjc` is harmless).
3. For gameplay changes, add or update a scenario in `scripts/verify.d/` and run `scripts/verify.sh` on a Mac with a game
   install (it needs a GUI session). See [docs/TESTING.md](docs/TESTING.md#5-scriptsverifysh-and-scenarios).
4. New pure packages are added to `PURE_PKGS` in `.github/workflows/ci.yml` so Linux CI covers them.
5. Honest evidence level on new behaviour: `VERIFIED`, `inferred` or `UNVERIFIED` in comments
   (see [docs/REVERSE_ENGINEERING.md](docs/REVERSE_ENGINEERING.md#4-evidence-levels)).

## Commits and pushes

* Messages say what changed and why. Agent-written commits end with the co-author trailer used in this fork's history.
* `git push -u fork <your-branch>`. Pull requests are not used for internal work.
* Table-driven tests, small interfaces, pure packages where possible.

## Code of conduct

See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). The project is GPLv3 and unaffiliated with Blizzard Entertainment.
