package d2player

import (
	"image/color"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
)

// Cell tints and equipment slot highlights (w23-ui-b). Read from Game.exe 1.14b.
//
// The five tint colours are palette indices written at run time by
// UI_InitInventoryHighlightColors 0x47f8b0 into the byte table at 0x7b4b60 (that is
// the "filled at run time" writer; it runs from INV_LoadPanelLayouts). Each index is
// GFX_FindPaletteIndexForRGB of:
//
//	mode 0 (128, 0, 0)    bad: requirements not met, broken, busy, disabled
//	mode 1 (0, 128, 0)    hover / drop target that fits
//	mode 2 (0, 0, 128)    identified and usable (the normal tint of an item)
//	mode 3 (128, 128, 0)  swap target (cube)
//	mode 4 (128, 64, 64)  unidentified
//
// The belt has the same first four (UI_InitBeltHighlightColors 0x493710). A grid cell
// is painted by INV_DrawGridCellOverlay 0x47f920 (one rect per cell of the item
// footprint, each as big as a cell, drawn BEFORE the item picture), an equipment slot
// by INV_DrawEquipSlots 0x480510 and INV_DrawSlotHighlightRect 0x47f9d0 (which skips
// mode 2). The blend of GFX_DrawRectXYWH (last argument 0) is UNVERIFIED: the tint
// is drawn half transparent here.

// TintMode is the index into the exe's tint table.
type TintMode int

// The tint modes of the exe.
const (
	TintBad TintMode = iota
	TintHover
	TintUsable
	TintSwap
	TintUnidentified
	tintModes
)

// tintRGB are the exe's RGB triples; the exe stores the nearest palette index.
var tintRGB = [tintModes][3]uint8{
	TintBad:          {0x80, 0, 0},
	TintHover:        {0, 0x80, 0},
	TintUsable:       {0, 0, 0x80},
	TintSwap:         {0x80, 0x80, 0},
	TintUnidentified: {0x80, 0x40, 0x40},
}

// tintAlpha is the opacity the tints are drawn with (UNVERIFIED, see above).
const tintAlpha = 0x80

// TintColor is the colour of a mode as a premultiplied RGBA.
func TintColor(m TintMode) color.RGBA {
	if m < 0 || m >= tintModes {
		return color.RGBA{}
	}

	c := tintRGB[m]

	return color.RGBA{R: uint8(int(c[0]) * tintAlpha / 0xff), G: uint8(int(c[1]) * tintAlpha / 0xff), B: uint8(int(c[2]) * tintAlpha / 0xff), A: tintAlpha}
}

// CellState is what INV_DrawGridItems 0x47ff60 asks about an item that is not under the mouse.
type CellState struct {
	RequirementsMet bool // INV_CheckItemRequirements
	Disabled        bool // item flag 4
	Busy            bool // INV_IsBusyStateSet (waiting for the server)
	Broken          bool // not ITEM_IsNotBrokenAndEquippable
	Identified      bool // item flag 0x10
	QuestHighlight  bool // quest item whose code INV_IsQuestItemCodeHighlightable accepts
}

// GridCellMode is the tint of a grid cell item: 0 when the requirements fail or it is
// disabled, busy or broken; else 4 when not identified; else 2 (0 for a highlightable
// quest item).
func GridCellMode(s CellState) TintMode {
	switch {
	case !s.RequirementsMet || s.Disabled || s.Busy || s.Broken:
		return TintBad
	case !s.Identified:
		return TintUnidentified
	case s.QuestHighlight:
		return TintBad
	}

	return TintUsable
}

// HoverMode is the tint of the item under the mouse: green, or red while the identify
// cursor (cursor state 8) is over a non-transmogrify item.
func HoverMode(identifyCursor, transmogrify bool) TintMode {
	if identifyCursor && !transmogrify {
		return TintBad
	}

	return TintHover
}

// EquipSlotMode is the tint of a worn item's slot (INV_DrawEquipSlots): the same
// rule as the grid, but a usable identified item gets no rect at all. ok is false
// when nothing is drawn. Not ported: the two-hand / quiver specials of slots 4 and 5.
func EquipSlotMode(s CellState) (m TintMode, ok bool) {
	switch {
	case !s.RequirementsMet || s.Busy || s.Broken:
		return TintBad, true
	case !s.Identified:
		return TintUnidentified, true
	}

	return 0, false
}

// SlotHighlightMode is the colour of the body slot under a held item
// (INV_DrawEquipSlotHighlights 0x480410): green when the item fits the slot and the
// hero meets its requirements, or the item can be socketed into the worn one; else red.
func SlotHighlightMode(fits, requirementsMet, canSocket bool) TintMode {
	if (fits && requirementsMet) || canSocket {
		return TintHover
	}

	return TintBad
}

// drawTintRect paints one rectangle of a tint mode (the mode 2 rect is not drawn for slots).
func drawTintRect(target d2interface.Surface, x, y, w, h int, m TintMode) {
	if w <= 0 || h <= 0 {
		return
	}

	target.PushTranslation(x, y)
	target.DrawRect(w, h, TintColor(m))
	target.Pop()
}

// drawCellTint paints every cell of an item footprint (INV_DrawGridCellOverlay).
func (g *ItemGrid) drawCellTint(target d2interface.Surface, item InventoryItem) {
	if g.tint == nil {
		return
	}

	m := g.tint(item)
	if g.hovered != nil && g.hovered == item {
		m = HoverMode(false, false) // the identify cursor is not wired in this port
	}

	w, h := item.InventoryGridSize()
	sx, sy := item.InventoryGridSlot()

	for cy := 0; cy < h; cy++ {
		for cx := 0; cx < w; cx++ {
			x, y := g.SlotToScreen(sx+cx, sy+cy)
			drawTintRect(target, x, y, g.slotSize, g.slotSize, m)
		}
	}
}

// drawSlotTint paints the slot of a worn item (INV_DrawEquipSlots).
func (g *ItemGrid) drawSlotTint(target d2interface.Surface, eq EquipmentSlot) {
	if g.slotTint == nil {
		return
	}

	m, ok := g.slotTint(eq.item)
	if g.hovered != nil && g.hovered == eq.item {
		m, ok = HoverMode(false, false), true
	}

	if ok {
		drawTintRect(target, eq.x, eq.y-eq.height, eq.width, eq.height, m)
	}
}

// renderSlotHighlight paints the body slot under the mouse while an item is held
// (INV_DrawEquipSlotHighlights 0x480410): green when the item fits and its
// requirements are met, else red. The exe only does this on the right half of the
// screen below y = H - 0x27, which the panel position already guarantees here.
func (g *Inventory) renderSlotHighlight(target d2interface.Surface) {
	if g.cursor == nil || g.slotFits == nil {
		return
	}

	loc, ok := g.LocAt(g.lastMouseX, g.lastMouseY)
	if !ok {
		return
	}

	slot := g.grid.equipmentSlots[locSlots[loc]]
	m := SlotHighlightMode(g.slotFits(g.cursor, loc), true, false)
	drawTintRect(target, slot.x, slot.y-slot.height, slot.width, slot.height, m)
}
