# ui-panels-next2: party rows, mercenary panel, help screen, gold line (Game.exe 1.14b)

Numbers and observations only. Pictures are placed by their bottom-left corner. "verified" = read in the decompiler, disassembly or raw memory. Panel offset (e0,e4) = (80,-60) at 800x600 (screen y = panel y + 60).

## Party screen (resolves the open row-y question)
- Row y: UI_RebuildPartyList 0x496ac0 keeps EDI = 0x5a, adds 0x26 per listed player, and passes it (register) to UI_AppendPartyListEntry 0x496670 -> UI_InitPartyRowWidgets 0x496540 (verified in disassembly). Row bottoms are 0x5a + 0x26*i panel relative; UI_DrawPartyScreenMemberRows 0x498a40 uses the same 0x5a/0x26 and draws at most 8 rows. Own roster entry is skipped.
- Widgets (bottom y): invite/accept/leave x 0xbe on the row (hover 0x35 wide, 0x14 high); hostile toggle (relation bit 08) x 0x0e at row+7; bit 01 toggle x 0xf5; bit 02 toggle x 0x10a; bit 04 (mute) x 0x11f, all on the row, hover 0x14 wide and 0x14 high (inclusive ranges). The bit 01 toggle is only drawn/tooltipped when game-info flag 4 is set (UI_IsGameInfoFlag4Set): this answers the "fourth toggle at 325" question. It exists at x 0xf5+80 = 325 but only in that game mode; the Go panel still has no widget for it.
- Row name hover: x 0x24..0x8c, y [rowY+0xd-0x1a, rowY+0xd]. Header "party membership" (string 0x1038) hover x 0xe3..0x118, y 0x18..0x23; header picture at x 0xe3. Button captions are centred around panel x + (0xbe | 0xaf | 0x9b) + 0x35 (0xaf / 0x9b are the leave and joined variants; strings 0x1034/0x1035/0x1036).
- Name text is cut at 0x88 px (0x60 when the button is shown); the location line at x+0xc, 0x8c px. The Go panel puts name and class on two lines per row, the original has name, level and status on one baseline (not changed).
- Go change: widget tops now 130 (invite, listen, mute) and 137 (hostile) + 38 per row at 800x600 (were 143/136/146). The close button (449) was already right.

## Mercenary panel
- Name header 0x4868f0 (asm 0x4869a0): left anchor x = e0 + 0xf, y 0xd6 - e4; cut at 0xa0 px. (Go centred it at x offset+80.)
- Label table 0x6dab08 is 12 records of 14 bytes {x1, x2, y (ints), string id (u16)}; the earlier Go table read it one field off. Records: experience (15,0,236,4058), level (145,0,236,4057), next level (200,0,236,4059), strength (15,0,281,4060), dexterity (15,0,305,4062), damage (15,0,329,4061), defense (15,0,353,4064), fire/cold/lightning/poison (180,245, y 281/305/329/353, 4071..4074), life (180,0,214,4068). x2 != 0: drawn by UI_DrawTextLine between x1 and x2; x2 == 0: left-anchored at x1. Quirk: the exe adds e0 to x2 before the zero test, so at 800x600 the x2 branch is always taken with range [x1+80, 80]; effect unknown (unverified), not reproduced.
- Value table 0x721648 (20 bytes {flag, x1, y, x2, stat}) was already right. The anchor of flag == 0 rows (left or right edge) is still unverified (register argument).
- Go: label rects now carry W = x2 - x1; the panel centres captions with W > 0 and left-anchors the others; golden regenerated.

## Help screen (0x492b00; no panel offset)
- 800x600 border (800helpborder, 7 frames) bottoms: f0 (0,256) f1 (0,512) f2 (66,20) f3 (322,20) f4 (535,256) f5 (780,256) f6 (780,512). 640x480: eight frames, (0,256) (256,256) (0,H-48) (256,H-48) (320,256) (576,256) (320,H-48) (576,H-48).
- Eight yellow bullets at x 104, bottoms 73 + 20*i. Eleven white dots (800) with two vertical 1-px lines at dot x+3 and x+4 from y1 to y2 (table in help_layout.go); nine in 640. The dots differ from the earlier hand-tuned Go ones by up to 5 px.
- Close button 32x32 bottom-left (0x2ad, 0x38) at 800, (0x20d, 0x38) at 640; click accepts x in [x, x+0x20], y in [0x18, 0x38].
- Text: only the y arguments are visible (title 0x11, subtitle 0x49, list rows 0x4e + 0x14*i for 8 rows, callouts 0x177..0x209); the x values are lost registers (unverified). Go labels were left alone; Go bullets moved to the legend positions with the text x unchanged, so the bullet/text gap may need a look in a run.
- Go change: fixed frame positions (no width arithmetic), close y 24, dots/lines per table, bullets per table.

## Gold line (UI_DrawGoldAmountAndTooltip 0x484250)
- Three variants (stash gold stat 15, carried gold stat 14, third = stat 15 with label string 0xcf3); numbers in gold_layout.go. This is the amount + coin button, not a dialog. The original move/drop gold dialog was NOT located: strings only name the coin pictures (Panel\goldcoinbtn, Panel\inv_goldbtn); the click path in INV_HandleMouseDown 0x48d6c0 is unread. Go MoveGoldPanel stays unverified.

## NPC menu
- No new work. The dialog geometry in npc_menu.go (UI_CreateDialog / 0x4b4d40 sizing) was already from the exe. UI_CreateNpcMenuDialog 0x49c860 is a different (list-box) dialog at (W-0x14, H-0x82); its relation to the Talk/Trade menu is not established. Unchecked.

## Not done / next
Move-gold dialog, NPC menu golden, help text x anchors and the 640 text, party name/level text x, merc value-row anchors, a Go widget for the bit 01 toggle. No game run this session.
