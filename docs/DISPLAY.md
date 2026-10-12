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
  bar. The bar's art is 800 wide; on a wider screen the original frame has no art for the sides, so by default the world shows there. `OD2_BAR_FILL=none` (the default) shows the world there (the look of D2 mods and Resurrected). `OD2_BAR_FILL=black` draws a black
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
`a2-menu-flow-1512x982` / `a2-menu-flow-1920x1080` go through the real menu flow (OD2_AUTOFLOW: main menu, character select,
hero creation, then in the game the escape menu, options and video options) and take a screenshot at every stop (`shot:<png>`,
`escmenu:<page>` flow steps). `scripts/shotcheck.go` checks them: window size, and for the menu screens that the non-black
content is exactly the centred 4:3 region (`d2display.MenuRegion` / `CheckMenuCentred`, unit tested). Menus are drawn at 800x600
and scaled to the window by the renderer, also with `OD2_DISPLAY` pinned; a menu screenshot is composed the same way
(`d2display.FrameInWindow`) so it shows what the player sees, with the bars. `a2-seed-sample-hero` and `a2-seed-new-hero` check
that a hero played from the menus logs `map seed of <name>: <seed>`, that the seed is the one of the .d2s header, and that the
town built (TownE1 for the sample hero) is the same on a second load; a new character gets its own non-zero seed at creation
(`d2hero.NewMapSeed`, written to the .od2 and the .d2s header), and a hero without one gets one when it is first played.
`a1-display-panels` opens the panels at 1920x1080 and `a1-display-menus` shows the character select (look at the
screenshots in the run directory). `9f-ui-layout` is pinned to `OD2_DISPLAY=800x600` (the golden of rectangles is in
column coordinates).

## Display QA wave 45 (a1-display-qa2-*)

Three scenarios take their shots into `OD2_QA_DIR` (outside the repo) for a look by eye; they check that the flow ran and that each shot has the window size (menu shots: centred 4:3).
`a1-display-qa2-menus` (`OD2_QA2_SIZE`, default 1920x1080): main menu, credits, cinematics, character select, hero creation, escape menu, options, video, sound, automap, accessibility pages.
`a1-display-qa2-game` (`OD2_QA2_SIZE`, `OD2_BAR_FILL`): town, character + inventory, skills + quests, automap, waypoint at 3440x1440 and 1440x1080 (4:3) with fill none, tile, black.
`a1-display-qa2-window` (nothing pinned): window resize (1280x720, 4:3, 1000x600, 800x600), full screen on and off, UI scale 2 in full screen.
New `OD2_AUTOFLOW` steps for them: `goto:credits|cinematics|mainmenu|charselect`, `resize:WxH`, `fullscreen:on|off|toggle`, `uiscale:N`, `pause:S` (they run on any screen, `AutoFlowTick`).
Found and fixed: the escape menu and its option pages stood in the bottom anchored column (low on a 1920x1080 screen); they are now centred on the screen like the original (`d2display.CenterShiftY`, mouse shifted to match).
Looked at, no fault: menus at 1920x1080, bar fill tile/black at 3440x1440 / 1440x1080, panels, resize, 4:3, full screen. Notes: the Video Options page still shows the row "800x600" whatever the window; the Cinematics screen shows the small trademark dot of the logo without the logo.

## Scenario scripts

`scripts/verify.sh` exports `OD2_DISPLAY=1512x982` for every scenario unless `OD2_DISPLAY` is set. `click:` and `hold:` steps take
`@x,y` (screen pixels), `@hero:dx,dy` (relative to the hero's screen position: resolution independent), `@ui:x,y`
(pixels of the 800x600 column, for the bar and panels) and `@monster`. Without a position the click lands just above the
screen centre.

## Approximate / not done

* Cinematics: `d2video` decodes Bink 1 revision `i` (the Diablo II videos) in pure Go and `VideoPlayer` plays them
  letterboxed with `VideoRect` (Esc/Space/Enter/click skips). Off by default: set `OD2_PLAY_VIDEOS=1` to play the intro
  before the main menu and the cinematics menu entries. The Bink audio track (RDFT or DCT, 44.1 kHz stereo) is decoded in pure Go (`d2video.BinkAudioDecoder`) and streamed to the sound device; the audio clock drives the picture (`d2video.Cinematic`). `OD2_VIDEO_SECONDS=n` ends a video after n seconds (with `OD2_AUTOEXIT` the game quits then).
* The tile range is a rectangle around the isometric diamond of the screen; `VisibleTileRows` cuts it per row to the
  tiles whose origin is within 400 px of the screen sides (and 200 above / 450 below), all four passes use it. At
  5120x1440 the rectangle has 2352 tiles and 1629 are drawn (69%); 1920x1080 960 -> 687. Measured at 5120x1440
  (`95c-perf-5120`, real level 111, 120 monsters): update 0.26-0.33 ms, render pass 1.6-1.7 ms CPU, frame
  ~16 ms wall (GPU present of the large target). Without the cull the drawn tiles were the whole rectangle (not timed).
* The map-engine test screen (`map_engine_testing.go`) is a developer tool; it now draws the world over the whole screen
  and picks in screen coordinates, its text overlay stays in the column.

* Wave 22 renderer: Act 4 ambient starts at 255 (the exe create value) and eases; hovered units get the doubled light byte and the 170%% highlight (monsters too); monsters and missiles cast light (MonStats2 Light/light-r/g/b, Missiles.txt Light/Red/Green/Blue, radius capped at 18 subtiles); draw mode 6 (PL2 table G) is an alpha-by-brightness blend; automap town folk are white and item-holding corpses magenta. Flash/darkening envelopes, shake call sites and Palshift.dat loader exist as tested pure code but are not wired to their triggers. Details and the UNVERIFIED list: d2-re-notes/w22-render.md.
