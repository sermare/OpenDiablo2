# Gamepad and accessibility

Gamepads are optional. With none connected the game behaves exactly as before.
A controller can be plugged in or pulled out at any time (the log shows
`GAMEPAD connected` / `GAMEPAD disconnected`); several controllers work at once.

## Default layout

| Control | Action |
| --- | --- |
| Left stick | Walk toward the stick direction (the hero walks to a point 150 px away on the screen). While a panel or menu is open it moves the cursor instead. |
| Right stick | Move the on-screen cursor (menus, inventory, panels, aiming). The pad hands the cursor back to the mouse when the mouse moves. |
| A | Left click at the cursor |
| B | Right click at the cursor |
| Right trigger | Use the left skill (left mouse button) |
| Left trigger | Use the right skill (right mouse button) |
| Left / right bumper | Next left / next right skill |
| D-pad up, right, down, left | Potion belt slots 1, 2, 3, 4 |
| X, Y | Inventory, character panel |
| Left stick click | Skill tree |
| Right stick click | Quest log |
| Back | Automap |
| Start | Game menu (Escape) |

Panel and potion buttons press the key you bound in Esc -> Options -> Configure
Controls, so remapped keyboard controls keep working with the pad.

## Remapping

Esc -> Options -> Gamepad Controls lists every button; click a row to step
through the actions (click, right click, left skill, right skill, potions 1-4,
inventory, character, skill tree, quest log, party, automap, game menu, show
items, next/previous left/right skill, none). Choices are stored in
`config.json` (`Options`, keys `pad.a`, `pad.lb`, ...).

## Controller families

ebiten 2.0.2 offers no standard-layout API, only raw GLFW button and axis
indexes, so `d2core/d2input/d2gamepad/profile.go` carries one profile per family
(Xbox, PlayStation, Switch Pro; any other controller gets the Xbox layout). The
indexes are recalled from the SDL controller database for macOS and are
**unverified on hardware**. At start the game logs the raw layout of each
controller (`GAMEPAD raw layout ... buttons=N axes=M`). If a button is wrong,
add `gamepad-profiles.json` next to `config.json`:

```json
[{"name": "mine", "match": ["my pad name"],
  "buttons": {"A": {"button": 0}, "LT": {"axis": 4, "rest": -1}, "DUP": {"button": -4}},
  "axes": [0, 1, 2, 3]}]
```

A negative button index counts from the end of the list (the D-pad hat is
appended as up, right, down, left). `rest` is the trigger value at rest (-1 or 0).

## Testing without hardware

The autoscript harness has a synthetic controller: `pad:connect`,
`pad:disconnect`, `pad:press=<button>` (tap), `pad:hold=<button>`,
`pad:release=<button>`, `pad:stick=left|right,<x>,<y>` and `pad:state` (logs the
open panels, automap and active skills). See `scripts/verify.d/9e-gamepad.sh`.

## Accessibility (Esc -> Options -> Accessibility)

* Window scale 1X-3X: resizes the window at once (the 800x600 interface grows
  with it); the choice is used at the next start too.
* Color blind items: item names use the Okabe-Ito palette (magic sky blue, rare
  yellow, set bluish green, unique orange, crafted vermillion).
* Speech subtitle log: NPC speech and barks are also written to the log as
  `SUBTITLE <npc> [message <id>]: <text>` (text from the string table entry of
  the message id). `OD2_SUBTITLE_LOG=1` forces it on.
