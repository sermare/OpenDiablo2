# Automap: what the original does and what this fork implements

Reverse-engineered from Game.exe 1.14b (Ghidra, UI\automap.cpp, D2Common LvlTbls.cpp) and checked against the extracted
tables. V = verified in the decompile, U = unverified / inferred. Code: `d2common/d2automap` (pure, tested) and
`d2game/d2player/automap*.go` (overlay). Console: `automap on|off|toggle|full|mini|stats`; script step `automap:<mode>`.

## Data
- `data\global\excel\automap.bin` (only in d2exp.mpq; patch_d2 has neither the bin nor the txt): u32 row count (0xcd6 = 3286), then
  44-byte rows: LevelName[16] @0, TileName[8] @0x10, Style u8 @0x18, StartSequence u8 @0x19, EndSequence u8 @0x1a, Cel1..Cel4 i32 @0x1c..0x28
  (0xFF in the byte columns = any, -1 cel = unused). Loader FUN_0061ff50. V. The "Expansion" line of the txt is not a row. The loader here
  prefers the bin and falls back to AutoMap.txt; the test compares both row by row.
- LevelName -> index through the string table at 0x6e9300: "None","1 Town","1 Wilderness","1 Cave",... = LvlTypes.txt Id (1..35). V.
- TileName -> index through the table at 0x6e9540: fl wl wr wtlr wtll wtr wbl wbr wld wrd wle wre co sh tr rf ld rd fd fi (0..19), the DT1
  tile orientation (V as a table, U that the engine compares it with the raw orientation).
- Rows are grouped per level; when a level occurs in several blocks the LAST block wins (all levels of the real table are one block). V.
- Lookup FUN_006202a0(levelType, tileName, style = DT1 main index, sequence = sub index): first row of the level whose tile name matches, style
  byte 0xFF or equal, start byte 0xFF or start <= sequence <= end; the cel is picked at random among the used cels (seed U). No row = no graphic.
- Frames: `data\global\ui\automap\MaxiMap.dc6` (1499 frames of 16x32) for the full map, `MaxiMapS.dc6` (8x16) for the mini map. Act2Map,
  Act4Map and ExTnMap (and S versions) are loaded too, but no AutoMap row uses a cel >= 1499 (U what they are for).
- Objects: objects.txt column `AutoMap` is an object's cel (shrines 310, wells 309, waypoints 307, stairs 693/694, ...). V.
- Registry options: AutoMapFade, AutoMap Centers, AutoMap Party, AutoMap Party Names, AutoMap Left.

## Cell store and projection
- Cell records {cel @+4, x @+6, y @+8} in AVL trees ordered by (y, x, cel); four trees: floors, walls, objects, extra; drawn in that order. V.
- Dedup (FUN_00453190): at the same position a new cel is dropped if it has no group in the 49-pair table at 0x710da8 or shares the group of
  the cel already there. V (table copied to model.go).
- Tile cell = FUN_00644770(tileX, tileY)/10 = ((tx-ty)*8, (tx+ty)*4); y += 24 when the tile's field +0x1c >= 16 (U what the field is). V.
- Object cell = (isoX/10+1, isoY/10-3) with isoX = (sx-sy)*16, isoY = (sx+sy)*8 on sub-tile coordinates. V.
- Screen = cell*10/scale - origin; scale 10 (full) or 20 (mini). Frames are bottom-left anchored.

## Reveal as you walk
FUN_004546b0 runs every client frame, also while the map is hidden. When the hero's iso position moved by more than 79 under the metric
(min*2 + max)/2 since the last reveal, the hero's room and the neighbour rooms of the same level are processed (FUN_004545d0): every floor and
wall tile not yet done and flagged "has been drawn" (0x20000) becomes a cell, and the room's objects are added (FUN_00454460). So the
radius is what the renderer has drawn, about one screen. V for trigger and flow, U for the exact meaning of 0x20000. Here: tiles inside the
screen diamond (screen/10 + margin) around the hero, same trigger.

## Display
- DAT_0079d1d8: 0 = full-screen overlay (scale 10), 1 = mini map (scale 20) in a 0x117 x 0xe1 box at x = W-0x119, y = 0x39 (right) or at the
  left (AutoMap Left), clipped to the box. Set by the options screen. Tab shows/hides (the flag was not located: U). V for sizes and box.
- Origin: full = (0x28 + hx - W/2 - panelShift, 0xf + hy - H/2) with panelShift -W/4 (right panel open) or +W/4 (left panel open); the hero is at
  (W/2-40, H/2-15) and its cross at +8,-8. Mini adds (W/3 - d264 - 0x10, H/3 - d260 - 0x10), d264 = 2W/3, d260 = 0x4e (right side). V.
- Fade option (AutoMapFade, AUTOMAP_SetFadeOption): the cell draw mode is 5 (normal) or, with fade on, 0/1/2 for cells inside a box (+-0x8c x -0x96..+0x82) around the screen centre, by distance (<50, <100, <150 on the (min*2+max)/2 metric); on the mini map flat mode 1. V that the values are passed, U that 0/1/2 are about 25/50/75 percent and the 50/100 steps. Implemented: d2automap.CellTransparency.
- Centre option (AutoMap Centers, AUTOMAP_RecalcOffsets): the mini map offsets d210/d214 are recomputed on a size change and, only if the option is on, when the panel layout changes. Implemented (MiniBoxOffsets, ComputeLayoutOffsets). The mini map moves to the left when a right panel is open.
- Party / names options (AUTOMAP_DrawUnitMarker): cross for party members only with Show Party; names (players, town NPCs, "Stash" text for object 267 instead of its black cross) with Show Names; dead players and hostile monsters are not drawn; permanent portal (60) hidden in levels 0x6f,0x70,0x75,0x7d-0x7f; mini cross offset (-1,+5). Implemented: d2automap.Classify.
- Text after the cells and markers: game name/password/IP, level name, "v 1.14b", clock; gold, right aligned, y = 0x18 + 16n. V.
- Drawn from the HUD pass before panels and text.

## Markers
Closed 12-point polyline (table at 0x6d7640: (0,-1)(2,-2)(4,-1)(2,0)(4,1)(2,2)(0,1)(-2,2)(-4,1)(-2,0)(-4,-1)(-2,-2), doubled; V) for players, friendly
NPCs, minions and portals; hostile monsters are NOT drawn (FUN_004552e0 returns 0; V). Colours are the palette entries nearest to: self
(0,0,255), hostile or other player (255,0,0), party (0,255,0), own minion (0x44,0x70,0x74), friendly NPC (0x48,0xa0,0x34), portal (0xf4,0xf4,0);
with the Act 1 palette these are (36,96,216), (252,44,0), (24,252,0), (68,112,116), (72,160,52), (244,192,76). Which exact unit class gets the
minion colour is only partly decoded (U).

## What this fork does not do
Saving the revealed map with the character (kept per level in memory for the session: d2automap.Store), roster markers for party members outside the view (RosterMarker only), minion classification beyond the model flag. Console/script: `automap fade|nofade|names|nonames|party|noparty|center|nocenter` override the options (verification aid). Scenarios: 91-automap-fidelity (town, Jail 1, return), 91-automap-outdoor (Blood Moor).
