# Display: resolutions, widescreen and the 800x600 interface column

The original Diablo II draws a fixed 800x600 (or 640x480) screen. This fork keeps the original interface exactly
as it was and lets the world use any screen size: more map is visible on a bigger or wider screen, the interface
keeps its native pixel size. Package `d2common/d2display` is the single display model; everything below is derived
from it.

## Try it

| Want | How |
| --- | --- |
| 1512x982 (MacBook), the default | start the app; nothing to set |
| 1920x1080 | `OD2_DISPLAY=1920x1080`, or `"Display": {"Width": 1920, "Height": 1080}` in `config.json` |
| ultrawide | `OD2_DISPLAY=3440x1440`, `OD2_DISPLAY=5120x1440`, `OD2_DISPLAY=2560x1440` |
| any monitor, full screen | Cmd+Enter (or `"FullScreen": true`): the logical size follows the monitor |
| the legacy original | `OD2_DISPLAY=800x600` |
| UI at 2x (4K, `OD2_UI_SCALE`) | `OD2_UI_SCALE=2` (or Esc, Options, 2X): window pixels / 2 = logical size, so a 3840x2160 screen shows the 1920x1080 layout with every pixel doubled |

Games are only started from the app or from `scripts/verify.sh`; for a scenario `OD2_DISPLAY=1920x1080 scripts/verify.sh`
runs everything at that size (the default of `verify.sh` is `1512x982`).

Size rules (`d2display.Resolve`, `d2display.Logical`):

* `OD2_DISPLAY=WxH` pins the logical size. A window of another size shows the picture scaled to fit (letterboxed). The
  size is never below 800x600.
* Without it the logical size follows the window: window (or full screen) size divided by the integer UI scale.
  Resizing the window shows more or less map. The start size is `Display.Width/Height` of the config, else
  1512x982, limited to the monitor (minus 80 px for the menu bar and the title bar).
* `OD2_UI_SCALE` (1..4) beats the `uiscale` option, which beats `WindowScale` of the config.
* A window smaller than 800x600 (for example 640x480) uses an 800x600 logical screen.

## Layout rules

```
+--------------------------------------------------------------+  W x H logical screen
|   the world fills all of it (map, camera, culling, picking)  |
|              +---------- 800 x 600 column ---------+         |
|              |  left half   |   right half         |         |
|              |  character   |   inventory          |         |
|              |  party       |   skill tree         |         |
|              |  quests      |   stash, cube, trade |         |
|              |  waypoints (centred)                |         |
|              |  [orb][skills][belt][mini][skills][orb]       |
+--------------+---------------------------------------+-------+
```

* **World.** The map renderer viewport is the whole `W x H`: the tile range that is drawn
  (`d2maprenderer.TileRange`), the camera centre, the visibility test and `ScreenToWorld` use the real size. At
  800x600 the tile range is exactly the old one (unit test `TestTileRangeLegacy800`). When a panel is open the
  hero is moved 200 px away from it (original: the other half of the screen), centred on `W/2 +- 200`.
  Entities are never culled by a screen radius (all entities of the level are indexed per tile and drawn where their
  tile is in range) and the monster AI works on distances in subtiles, so nothing has to be populated for a bigger view.
* **Interface column.** The original 800x600 interface is drawn at its native pixel size in a column
  (`d2display.Origin`). In the game the column is centred horizontally and stands on the bottom edge
  (`AnchorBottom`: origin `((W-800)/2, H-600)`). So the bottom bar is centred at the bottom, left panels sit
  in the left half of the column and right panels in the right half, exactly as in the original relative to the
  bar. The bar's art is 800 wide; on a wider screen the world is visible on both sides of it by default (the original
  frame has no art for the sides, it is the look of D2 mods and Resurrected). `OD2_BAR_FILL=black` draws a black
  strip of the bar's height on both sides; `OD2_BAR_FILL=tile` repeats the outermost 8 columns of the bar art (the
  left edge of the left globe holder frame on the left, the right edge of the right globe holder on the right).
  Layout: `d2display.BarFillSpans` (pure, tested), drawn by `HUD.renderBarFill`.
* **Menus and full screen dialogs** (main menu, character select, credits, cinematics) centre the column on both axes
  (`AnchorCenter`) on black. Loading screens, level-change fades, the Help dim and other black overlays cover the
  whole screen. Cinematics are scaled as before (video code unchanged).
* **Mouse.** Input handlers (all panels, all widgets, all menus) receive the cursor in *column space*
  (`d2display.ToColumn`). Only world code converts back with `d2display.ToScreen` (`ScreenToWorld`, hover
  highlight of monsters/items/NPCs, name labels, item labels, the NPC dialog anchor). A gamepad cursor and the
  hero position use the real size (`d2gamepad.Controller.SetScreen`).
* **Tooltips** clamp to the real screen, which in column space is `d2display.ScreenRectInColumn`.
* **Automap** covers the whole screen (full size and the mini map in the top right corner of the screen); the
  hero stands at the screen centre, with the panel shift of a quarter of the 800 column (200 px, `PanelShift.Pixels`),
  the same distance the world camera moves the hero beside an open panel.
* **Zone name text** and **notices** ("quest log updated", `NoticePlacement`) stand in the upper part of the screen,
  centred on the screen, not on the column. NPC subtitles stand just above the bar (bottom of the column), centred.
* **Cinematics**: `d2display.VideoRect` is the letterbox of a video of any size on the screen (largest size at the
  native aspect, centred, black surround). The Bink decoder of this fork parses headers only and nothing draws video
  frames yet, so there is no frame to place; the cinematic and intro screens use the centred-column layout on black.

Screens opt in with `ScreenAnchorHandler.DisplayAnchor()`; only the game returns `AnchorBottom`. The app translates the
screen and the widgets by the column origin and renders the world, the loading screen, the cursor and the console in
screen space (`d2app.App.render`).

## Pure layout functions and tests

`d2display`: `Clamp`, `Logical`, `Origin`, `Parse`, `Resolve`, `ToColumn`, `ToScreen`, `ScreenRectInColumn`.
Golden tests: 800x600, 1512x982, 1920x1080, 2560x1440, 3440x1440, 5120x1440, 1980x1800, 640x480 (clamped to 800x600)
in `d2common/d2display/display_test.go`; `d2core/d2map/d2maprenderer/widescreen_test.go` checks that the tile range
covers every screen corner at all those sizes and for panel-shifted views.

Scenarios (`scripts/verify.d/`): `a1-display-1512x982`, `a1-display-1920x1080`, `a1-display-3440x1440` boot the game at that
size, check the logged display model (column origin, bar span `hud_x=`), the screenshot size, and click far right / far left
of the world: the picked world point must agree with the camera at the screen edge and the hero must walk to that side.
`a1-display-panels` opens the panels at 1920x1080 and `a1-display-menus` shows the character select (look at the
screenshots in the run directory). `9f-ui-layout` is pinned to `OD2_DISPLAY=800x600` (the golden of rectangles is in
column coordinates).

## Scenario scripts

`scripts/verify.sh` exports `OD2_DISPLAY=1512x982` for every scenario unless `OD2_DISPLAY` is set. `click:` and `hold:` steps take
`@x,y` (screen pixels), `@hero:dx,dy` (relative to the hero's screen position: resolution independent), `@ui:x,y`
(pixels of the 800x600 column, for the bar and panels) and `@monster`. Without a position the click lands just above the
screen centre.

## Approximate / not done

* Cinematics and the intro videos: the layout function exists (`VideoRect`), the decoder does not output frames yet.
* The tile range is a rectangle around the isometric diamond of the screen; `VisibleTileRows` cuts it per row to the
  tiles whose origin is within 400 px of the screen sides (and 200 above / 450 below), all four passes use it. At
  5120x1440 the rectangle has 2352 tiles and 1629 are drawn (69%); 1920x1080 960 -> 687. Measured at 5120x1440
  (`95c-perf-5120`, real level 111, 120 monsters): update 0.26-0.33 ms, render pass 1.6-1.7 ms CPU, frame
  ~16 ms wall (GPU present of the large target). Without the cull the drawn tiles were the whole rectangle (not timed).
* The map-engine test screen (`map_engine_testing.go`) is a developer tool; it now draws the world over the whole screen
  and picks in screen coordinates, its text overlay stays in the column.
