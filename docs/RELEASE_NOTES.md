# Release notes

**There has been no release of this fork yet.** The only tag in the repository (`v0.1-pre-alpha`, 2020) is from the
upstream OpenDiablo2 project, and `scripts/release.sh` derives its version from it. These notes therefore describe what
a player would see in a build of the current tree (fork/integration 20a31960, landed as `land/w38`) compared with the
build of waves 20 to 21 (integration 694b5da0, 2026-10-11), the last state described in [STATUS_LOG.md](STATUS_LOG.md)
before waves 36 to 43. Source: the commit messages of that range, not a hand play. Nobody has played this build by
hand for long; items are "scenario-checked" (a scripted hero and a log line) unless stated. Unverified items are marked.

Checks behind this tree: full verify run 813, 291 scenarios in 6 jobs, all passed
([TESTING.md](TESTING.md#5b-parallel-verify-and-flake-tooling)). Known gaps: [KNOWN_GAPS.md](KNOWN_GAPS.md).

## Starting the game

* The game **starts full screen** in normal launches (Cmd+Enter or the FULL SCREEN option for a window;
  `OD2_WINDOWED=1` to start windowed). The Cmd+Enter double toggle is fixed, Cmd+F works, the option persists.
* **Cinematics** play by default: the opening movie on first launch, movies at act changes and for Baal, with a window-wide
  letterbox. The Cinematics option (Options, Accessibility) switches them off. Decoding runs off the main thread.
  Audio and video sync were checked by clock only.
* **New heroes start with the class kit** (worn weapon and shield, belt potions, scrolls) instead of placeholder gear, and
  worn items persist in the `.d2s`.
* **Classic (non-LoD) game mode** is wired into monsters, loot, vendors, the cube and the Act V gate. **unverified by hand**
* The crash log keeps the last crash (`OpenDiablo2.log`, previous run in `OpenDiablo2.previous.log`).

## Saves

* One validated, locked, atomic `.d2s` writer with rotating backups (`.d2s.bak1` to `.bak3`), an offer to recover a damaged
  save, no write of a partly loaded hero, last-writer protection and an opt-in repair of placeholder items. A bit-flip test
  over a real save shows every modelled field survives write(parse(x)).
* Your original files are still never modified; items found in this game live in the engine's own save.

## Interface

* Mouse wheel scrolling and keyboard item moving; keyboard control of NPC lists and the main menu pages; gamepad D-pad and A
  in list menus; keys can be rebound ([CONTROLS.md](CONTROLS.md)). Clicks while casting no longer queue a walk.
* Vendor window split into the four tabs by item type.
* Item tooltips compare with the worn item (inventory, stash, cube, vendor, cursor, ground), with a colour-blind palette.
* Character sheet: the four hand skill boxes (skill, Damage, Attack Rating) are filled; Shift spends all points at once;
  experience bar and tooltip; item-granted skills marked on the skill tree.
* Mercenary: hire list text in the original dialog form, HUD portrait with life bar, potion and gear drop on it.
* Belt: real-click behaviour for 4, 8, 12 and 16-box belts. Party screen tooltips use the original's strings.
* Light: occluded (line-of-sight) light painter ported from the exe, plus a quality ladder. The neighbouring level is drawn
  beyond the map edge (no black void on arrival). Look not compared side by side. **unverified**

## Combat, monsters, skills

* Boss and monster missile skills now hit the hero: Diablo lightning and wall, Andariel spray, Mephisto bolt and poison nova,
  Baal tentacles, nova rings, Duriel Holy Freeze, boss curses.
* Melee range gate is on: the hero walks into reach before swinging; the hero never hurts its own units.
* Monster AI: 141 of 148 archetypes ported faithfully (was 134); many more pinned against the exe with tests. Mercenary,
  pet and summon thinkers are ported from the exe (wolves, bear, ravens follow the leader like the original).
* Skills: Telekinesis picks up scrolls, gold, potions and keys from afar; Enchant on an ally; Charged Strike bolts; Multiple
  Shot, Guided Arrow and Strafe pick their missile by weapon class; Charge, Conversion fixes; Druid forms get correct mana.
* Nightmare and Hell: scaling checked against the real tables in play scenarios (resist penalty, drop levels, quest rewards).

## Quests, levels, towns

* Rite of Passage altar and gate ported from the exe (right objects, statue delay, shared quest state for party members).
* Acts 2 to 5 audits: log pages, area-entry pages, Rescue counts, vendor stock. Act 1 outdoor waypoints and sub-theme
  objects (shrines, wells, stones) are placed. Lut Gholein has a Jerhyn placeholder object. The Act 5 Anya and Nihlathak
  boss work lives on the unlanded `feat/w43-fullgame` branch.

## Multiplayer

* Join-time sync, player trade rules (town, cooldown, equipment lock), shared gold drop owner, quest sharing, robustness
  against unknown senders, dropped connections, join races and forged XP or gold. Still direct TCP between copies of this
  game, not Battle.net.

## Sound

* Exe-verified sounds for objects, portals, doors, forms and item use; hero hit and speech gates; area-entry hero voice;
  quest music plans and music resume. Not confirmed by a human listen.

## Performance and platform

* Indexed tile lookup, cached shade grids and a pre-loaded automap sprite (frame-time scenario in the verify set).
* Windows: experimental cross-compiled build, one full screen toggle per Alt+Enter; never run on Windows. A newer Ebiten is
  on `feat/w36-ebiten-upgrade`, not landed.
* Apple Silicon only, macOS 14 or newer, not notarised (right-click, Open).

## Not in this tree

`feat/w43-fullgame` (Normal to Hell chain harness, live Baal encounter, Anya in Harrogath, Nihlathak boss, altar and gate
clicks) broke Nightmare scenarios and was reverted out of the landed tree; it stays on its branch.
