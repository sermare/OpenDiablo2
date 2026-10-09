package d2gs

import (
	"encoding/binary"
	"fmt"
)

// Message is a typed client->server packet.
type Message interface {
	// PacketID returns the leading id byte.
	PacketID() byte
	// MarshalPacket returns the wire bytes (id included).
	MarshalPacket() []byte
}

type decodable interface {
	Message
	unmarshal(b []byte)
}

// All layouts below are little-endian (the handlers read raw x86 memory) and
// verified from handler bodies unless a comment says otherwise. Unused bytes
// (e.g. the 2 bytes after the u16 in 0x33/0x35) are written as zero and
// ignored on decode, so decode->encode is not byte-identical for them.

type wbuf []byte

func newW(id byte, size int) wbuf { b := make(wbuf, size); b[0] = id; return b }
func (w wbuf) u32(off int, v uint32) {
	binary.LittleEndian.PutUint32(w[off:], v)
}
func (w wbuf) u16(off int, v uint16) {
	binary.LittleEndian.PutUint16(w[off:], v)
}

type rbuf []byte

func (r rbuf) u32(off int) uint32 { return binary.LittleEndian.Uint32(r[off:]) }
func (r rbuf) u16(off int) uint16 { return binary.LittleEndian.Uint16(r[off:]) }

// InteractUnit (0x13, 9 bytes): +1 u32 unit type (handler rejects > 5 with
// return code 2), +5 u32 unit id.
type InteractUnit struct{ UnitType, UnitID uint32 }

// PickUpUnit (0x16, 13 bytes): +1 unit type (<=5), +5 unit id, +9 flag.
type PickUpUnit struct{ UnitType, UnitID, Flag uint32 }

// DropCursorItem (0x17, 5 bytes).
type DropCursorItem struct{ ItemID uint32 }

// ItemToContainer (0x18, 17 bytes). Page: 0 inventory, 2 trade, 3 cube, 4 stash.
type ItemToContainer struct{ ItemID, X, Y, Page uint32 }

// PickFromContainer (0x19, 5 bytes).
type PickFromContainer struct{ ItemID uint32 }

// ItemToBody (0x1a, 9 bytes). BodyLoc 1..10.
type ItemToBody struct{ ItemID, BodyLoc uint32 }

// SwapTwoHandedBody (0x1b, 9 bytes).
type SwapTwoHandedBody struct{ ItemID, BodyLoc uint32 }

// PickFromBody (0x1c, 3 bytes): +1 u16 body location.
type PickFromBody struct{ BodyLoc uint16 }

// SwapBodyItem (0x1d, 9 bytes).
type SwapBodyItem struct{ ItemID, BodyLoc uint32 }

// SwapWeaponHands (0x1e, 9 bytes). BodyLoc must be 4 or 5.
type SwapWeaponHands struct{ ItemID, BodyLoc uint32 }

// SwapContainerItem (0x1f, 17 bytes): cursor item, target item, cell x, y.
type SwapContainerItem struct{ CursorID, TargetID, X, Y uint32 }

// UseItem (0x20, 13 bytes).
type UseItem struct{ ItemID, X, Y uint32 }

// StackItems (0x21, 9 bytes).
type StackItems struct{ ItemID, TargetID uint32 }

// UnstackItem (0x22, 5 bytes).
type UnstackItem struct{ ItemID uint32 }

// ItemToBelt (0x23, 9 bytes).
type ItemToBelt struct{ ItemID, BeltSlot uint32 }

// PickFromBelt (0x24, 5 bytes).
type PickFromBelt struct{ ItemID uint32 }

// SwapBeltItem (0x25, 9 bytes): cursor item, belt item.
type SwapBeltItem struct{ CursorID, BeltID uint32 }

// UseBeltItem (0x26, 13 bytes): item id plus two u32 flag words (meaning
// unverified).
type UseBeltItem struct{ ItemID, Flags1, Flags2 uint32 }

// IdentifyItem (0x27, 9 bytes): the two uids (scroll and target; order per notes).
type IdentifyItem struct{ ItemID, TargetID uint32 }

// InsertSocketItem (0x28, 9 bytes).
type InsertSocketItem struct{ ItemID, TargetID uint32 }

// ScrollToTome (0x29, 9 bytes).
type ScrollToTome struct{ ScrollID, TomeID uint32 }

// ItemToCube (0x2a, 9 bytes).
type ItemToCube struct{ ItemID, CubeID uint32 }

// NpcInit (0x2f, 9 bytes). Size verified; the two fields follow the public
// docs (unit type, unit id) and are UNVERIFIED.
type NpcInit struct{ UnitType, UnitID uint32 }

// NpcCancel (0x30, 9 bytes). Fields unverified, as NpcInit.
type NpcCancel struct{ UnitType, UnitID uint32 }

// QuestMessage (0x31, 9 bytes). Fields unverified: npc uid and message id
// per public docs.
type QuestMessage struct{ NpcID, MessageID uint32 }

// BuyItem (0x32, 17 bytes). Flags bit 31 = buy full stack, low 31 bits = tab
// or transaction type (0 normal, 2 gamble); Cost = client's expected price.
type BuyItem struct{ NpcID, ItemID, Flags, Cost uint32 }

// SellItem (0x33, 17 bytes): +9 is a u16 tab, +11..12 unused, +13 u32 cost
// (confirmed from 0x549960).
type SellItem struct {
	NpcID, ItemID uint32
	Tab           uint16
	Cost          uint32
}

// IdentifyItems (0x34, 5 bytes): identify-all at an NPC.
type IdentifyItems struct{ NpcID uint32 }

// RepairItem (0x35, 17 bytes): +9 u16 flags, +11..12 unused, +13 u32 cost
// (confirmed from disassembly of 0x5499a0). Per the notes bit 31 of the
// handler argument means repair-all; with a u16 that cannot be a bit of
// Flags, so the repair-all encoding is unresolved.
type RepairItem struct {
	NpcID, ItemID uint32
	Flags         uint16
	Cost          uint32
}

// HireMercenary (0x36, 9 bytes): +5 u16 offer index (bytes 7..8 unused).
type HireMercenary struct {
	NpcID uint32
	Offer uint16
}

// GambleItem (0x37, 5 bytes).
type GambleItem struct{ ItemID uint32 }

// ---- methods ----

func one(id byte, v uint32) []byte { w := newW(id, 5); w.u32(1, v); return w }
func two(id byte, a, b uint32) []byte {
	w := newW(id, 9)
	w.u32(1, a)
	w.u32(5, b)
	return w
}

func (InteractUnit) PacketID() byte          { return C2SInteractUnit }
func (m InteractUnit) MarshalPacket() []byte { return two(C2SInteractUnit, m.UnitType, m.UnitID) }
func (m *InteractUnit) unmarshal(b []byte)   { r := rbuf(b); m.UnitType, m.UnitID = r.u32(1), r.u32(5) }

func (PickUpUnit) PacketID() byte { return C2SPickUpUnit }
func (m PickUpUnit) MarshalPacket() []byte {
	w := newW(C2SPickUpUnit, 13)
	w.u32(1, m.UnitType)
	w.u32(5, m.UnitID)
	w.u32(9, m.Flag)
	return w
}
func (m *PickUpUnit) unmarshal(b []byte) {
	r := rbuf(b)
	m.UnitType, m.UnitID, m.Flag = r.u32(1), r.u32(5), r.u32(9)
}

func (DropCursorItem) PacketID() byte          { return C2SDropCursorItem }
func (m DropCursorItem) MarshalPacket() []byte { return one(C2SDropCursorItem, m.ItemID) }
func (m *DropCursorItem) unmarshal(b []byte)   { m.ItemID = rbuf(b).u32(1) }

func (ItemToContainer) PacketID() byte { return C2SItemToContainer }
func (m ItemToContainer) MarshalPacket() []byte {
	w := newW(C2SItemToContainer, 17)
	w.u32(1, m.ItemID)
	w.u32(5, m.X)
	w.u32(9, m.Y)
	w.u32(13, m.Page)
	return w
}
func (m *ItemToContainer) unmarshal(b []byte) {
	r := rbuf(b)
	m.ItemID, m.X, m.Y, m.Page = r.u32(1), r.u32(5), r.u32(9), r.u32(13)
}

func (PickFromContainer) PacketID() byte          { return C2SPickFromContainer }
func (m PickFromContainer) MarshalPacket() []byte { return one(C2SPickFromContainer, m.ItemID) }
func (m *PickFromContainer) unmarshal(b []byte)   { m.ItemID = rbuf(b).u32(1) }

func (ItemToBody) PacketID() byte          { return C2SItemToBody }
func (m ItemToBody) MarshalPacket() []byte { return two(C2SItemToBody, m.ItemID, m.BodyLoc) }
func (m *ItemToBody) unmarshal(b []byte)   { r := rbuf(b); m.ItemID, m.BodyLoc = r.u32(1), r.u32(5) }

func (SwapTwoHandedBody) PacketID() byte { return C2SSwapTwoHandedBody }
func (m SwapTwoHandedBody) MarshalPacket() []byte {
	return two(C2SSwapTwoHandedBody, m.ItemID, m.BodyLoc)
}
func (m *SwapTwoHandedBody) unmarshal(b []byte) {
	r := rbuf(b)
	m.ItemID, m.BodyLoc = r.u32(1), r.u32(5)
}

func (PickFromBody) PacketID() byte { return C2SPickFromBody }
func (m PickFromBody) MarshalPacket() []byte {
	w := newW(C2SPickFromBody, 3)
	w.u16(1, m.BodyLoc)
	return w
}
func (m *PickFromBody) unmarshal(b []byte) { m.BodyLoc = rbuf(b).u16(1) }

func (SwapBodyItem) PacketID() byte          { return C2SSwapBodyItem }
func (m SwapBodyItem) MarshalPacket() []byte { return two(C2SSwapBodyItem, m.ItemID, m.BodyLoc) }
func (m *SwapBodyItem) unmarshal(b []byte)   { r := rbuf(b); m.ItemID, m.BodyLoc = r.u32(1), r.u32(5) }

func (SwapWeaponHands) PacketID() byte          { return C2SSwapWeaponHands }
func (m SwapWeaponHands) MarshalPacket() []byte { return two(C2SSwapWeaponHands, m.ItemID, m.BodyLoc) }
func (m *SwapWeaponHands) unmarshal(b []byte)   { r := rbuf(b); m.ItemID, m.BodyLoc = r.u32(1), r.u32(5) }

func (SwapContainerItem) PacketID() byte { return C2SSwapContainerItem }
func (m SwapContainerItem) MarshalPacket() []byte {
	w := newW(C2SSwapContainerItem, 17)
	w.u32(1, m.CursorID)
	w.u32(5, m.TargetID)
	w.u32(9, m.X)
	w.u32(13, m.Y)
	return w
}
func (m *SwapContainerItem) unmarshal(b []byte) {
	r := rbuf(b)
	m.CursorID, m.TargetID, m.X, m.Y = r.u32(1), r.u32(5), r.u32(9), r.u32(13)
}

func (UseItem) PacketID() byte { return C2SUseItem }
func (m UseItem) MarshalPacket() []byte {
	w := newW(C2SUseItem, 13)
	w.u32(1, m.ItemID)
	w.u32(5, m.X)
	w.u32(9, m.Y)
	return w
}
func (m *UseItem) unmarshal(b []byte) {
	r := rbuf(b)
	m.ItemID, m.X, m.Y = r.u32(1), r.u32(5), r.u32(9)
}

func (StackItems) PacketID() byte          { return C2SStackItems }
func (m StackItems) MarshalPacket() []byte { return two(C2SStackItems, m.ItemID, m.TargetID) }
func (m *StackItems) unmarshal(b []byte)   { r := rbuf(b); m.ItemID, m.TargetID = r.u32(1), r.u32(5) }

func (UnstackItem) PacketID() byte          { return C2SUnstackItem }
func (m UnstackItem) MarshalPacket() []byte { return one(C2SUnstackItem, m.ItemID) }
func (m *UnstackItem) unmarshal(b []byte)   { m.ItemID = rbuf(b).u32(1) }

func (ItemToBelt) PacketID() byte          { return C2SItemToBelt }
func (m ItemToBelt) MarshalPacket() []byte { return two(C2SItemToBelt, m.ItemID, m.BeltSlot) }
func (m *ItemToBelt) unmarshal(b []byte)   { r := rbuf(b); m.ItemID, m.BeltSlot = r.u32(1), r.u32(5) }

func (PickFromBelt) PacketID() byte          { return C2SPickFromBelt }
func (m PickFromBelt) MarshalPacket() []byte { return one(C2SPickFromBelt, m.ItemID) }
func (m *PickFromBelt) unmarshal(b []byte)   { m.ItemID = rbuf(b).u32(1) }

func (SwapBeltItem) PacketID() byte          { return C2SSwapBeltItem }
func (m SwapBeltItem) MarshalPacket() []byte { return two(C2SSwapBeltItem, m.CursorID, m.BeltID) }
func (m *SwapBeltItem) unmarshal(b []byte)   { r := rbuf(b); m.CursorID, m.BeltID = r.u32(1), r.u32(5) }

func (UseBeltItem) PacketID() byte { return C2SUseBeltItem }
func (m UseBeltItem) MarshalPacket() []byte {
	w := newW(C2SUseBeltItem, 13)
	w.u32(1, m.ItemID)
	w.u32(5, m.Flags1)
	w.u32(9, m.Flags2)
	return w
}
func (m *UseBeltItem) unmarshal(b []byte) {
	r := rbuf(b)
	m.ItemID, m.Flags1, m.Flags2 = r.u32(1), r.u32(5), r.u32(9)
}

func (IdentifyItem) PacketID() byte          { return C2SIdentifyItem }
func (m IdentifyItem) MarshalPacket() []byte { return two(C2SIdentifyItem, m.ItemID, m.TargetID) }
func (m *IdentifyItem) unmarshal(b []byte)   { r := rbuf(b); m.ItemID, m.TargetID = r.u32(1), r.u32(5) }

func (InsertSocketItem) PacketID() byte { return C2SInsertSocketItem }
func (m InsertSocketItem) MarshalPacket() []byte {
	return two(C2SInsertSocketItem, m.ItemID, m.TargetID)
}
func (m *InsertSocketItem) unmarshal(b []byte) {
	r := rbuf(b)
	m.ItemID, m.TargetID = r.u32(1), r.u32(5)
}

func (ScrollToTome) PacketID() byte          { return C2SScrollToTome }
func (m ScrollToTome) MarshalPacket() []byte { return two(C2SScrollToTome, m.ScrollID, m.TomeID) }
func (m *ScrollToTome) unmarshal(b []byte)   { r := rbuf(b); m.ScrollID, m.TomeID = r.u32(1), r.u32(5) }

func (ItemToCube) PacketID() byte          { return C2SItemToCube }
func (m ItemToCube) MarshalPacket() []byte { return two(C2SItemToCube, m.ItemID, m.CubeID) }
func (m *ItemToCube) unmarshal(b []byte)   { r := rbuf(b); m.ItemID, m.CubeID = r.u32(1), r.u32(5) }

func (NpcInit) PacketID() byte          { return C2SNpcInit }
func (m NpcInit) MarshalPacket() []byte { return two(C2SNpcInit, m.UnitType, m.UnitID) }
func (m *NpcInit) unmarshal(b []byte)   { r := rbuf(b); m.UnitType, m.UnitID = r.u32(1), r.u32(5) }

func (NpcCancel) PacketID() byte          { return C2SNpcCancel }
func (m NpcCancel) MarshalPacket() []byte { return two(C2SNpcCancel, m.UnitType, m.UnitID) }
func (m *NpcCancel) unmarshal(b []byte)   { r := rbuf(b); m.UnitType, m.UnitID = r.u32(1), r.u32(5) }

func (QuestMessage) PacketID() byte          { return C2SQuestMessage }
func (m QuestMessage) MarshalPacket() []byte { return two(C2SQuestMessage, m.NpcID, m.MessageID) }
func (m *QuestMessage) unmarshal(b []byte)   { r := rbuf(b); m.NpcID, m.MessageID = r.u32(1), r.u32(5) }

func (BuyItem) PacketID() byte { return C2SBuyItem }
func (m BuyItem) MarshalPacket() []byte {
	w := newW(C2SBuyItem, 17)
	w.u32(1, m.NpcID)
	w.u32(5, m.ItemID)
	w.u32(9, m.Flags)
	w.u32(13, m.Cost)
	return w
}
func (m *BuyItem) unmarshal(b []byte) {
	r := rbuf(b)
	m.NpcID, m.ItemID, m.Flags, m.Cost = r.u32(1), r.u32(5), r.u32(9), r.u32(13)
}

func (SellItem) PacketID() byte { return C2SSellItem }
func (m SellItem) MarshalPacket() []byte {
	w := newW(C2SSellItem, 17)
	w.u32(1, m.NpcID)
	w.u32(5, m.ItemID)
	w.u16(9, m.Tab)
	w.u32(13, m.Cost)
	return w
}
func (m *SellItem) unmarshal(b []byte) {
	r := rbuf(b)
	m.NpcID, m.ItemID, m.Tab, m.Cost = r.u32(1), r.u32(5), r.u16(9), r.u32(13)
}

func (IdentifyItems) PacketID() byte          { return C2SIdentifyItems }
func (m IdentifyItems) MarshalPacket() []byte { return one(C2SIdentifyItems, m.NpcID) }
func (m *IdentifyItems) unmarshal(b []byte)   { m.NpcID = rbuf(b).u32(1) }

func (RepairItem) PacketID() byte { return C2SRepairItem }
func (m RepairItem) MarshalPacket() []byte {
	w := newW(C2SRepairItem, 17)
	w.u32(1, m.NpcID)
	w.u32(5, m.ItemID)
	w.u16(9, m.Flags)
	w.u32(13, m.Cost)
	return w
}
func (m *RepairItem) unmarshal(b []byte) {
	r := rbuf(b)
	m.NpcID, m.ItemID, m.Flags, m.Cost = r.u32(1), r.u32(5), r.u16(9), r.u32(13)
}

func (HireMercenary) PacketID() byte { return C2SHireMercenary }
func (m HireMercenary) MarshalPacket() []byte {
	w := newW(C2SHireMercenary, 9)
	w.u32(1, m.NpcID)
	w.u16(5, m.Offer)
	return w
}
func (m *HireMercenary) unmarshal(b []byte) { r := rbuf(b); m.NpcID, m.Offer = r.u32(1), r.u16(5) }

func (GambleItem) PacketID() byte          { return C2SGambleItem }
func (m GambleItem) MarshalPacket() []byte { return one(C2SGambleItem, m.ItemID) }
func (m *GambleItem) unmarshal(b []byte)   { m.ItemID = rbuf(b).u32(1) }

// DecodeClient parses one complete client->server packet into its typed
// form (a pointer to the struct, e.g. *BuyItem). ids without a typed struct return ErrUnknownSize/ErrInvalidID-style
// errors wrapped with the id; callers can still use the raw Packet.
func DecodeClient(b []byte) (Message, error) {
	if len(b) == 0 {
		return nil, ErrShort
	}
	var m decodable
	switch b[0] {
	case C2SWalkToLocation, C2SRunToLocation:
		m = &MoveToLocation{Run: b[0] == C2SRunToLocation}
	case C2SCastLeftLocation, C2SCastRightLocation:
		m = &CastOnLocation{Right: b[0] == C2SCastRightLocation}
	case C2SSelectSkill:
		m = &SelectSkill{}
	case C2SInteractUnit:
		m = &InteractUnit{}
	case C2SPickUpUnit:
		m = &PickUpUnit{}
	case C2SDropCursorItem:
		m = &DropCursorItem{}
	case C2SItemToContainer:
		m = &ItemToContainer{}
	case C2SPickFromContainer:
		m = &PickFromContainer{}
	case C2SItemToBody:
		m = &ItemToBody{}
	case C2SSwapTwoHandedBody:
		m = &SwapTwoHandedBody{}
	case C2SPickFromBody:
		m = &PickFromBody{}
	case C2SSwapBodyItem:
		m = &SwapBodyItem{}
	case C2SSwapWeaponHands:
		m = &SwapWeaponHands{}
	case C2SSwapContainerItem:
		m = &SwapContainerItem{}
	case C2SUseItem:
		m = &UseItem{}
	case C2SStackItems:
		m = &StackItems{}
	case C2SUnstackItem:
		m = &UnstackItem{}
	case C2SItemToBelt:
		m = &ItemToBelt{}
	case C2SPickFromBelt:
		m = &PickFromBelt{}
	case C2SSwapBeltItem:
		m = &SwapBeltItem{}
	case C2SUseBeltItem:
		m = &UseBeltItem{}
	case C2SIdentifyItem:
		m = &IdentifyItem{}
	case C2SInsertSocketItem:
		m = &InsertSocketItem{}
	case C2SScrollToTome:
		m = &ScrollToTome{}
	case C2SItemToCube:
		m = &ItemToCube{}
	case C2SNpcInit:
		m = &NpcInit{}
	case C2SNpcCancel:
		m = &NpcCancel{}
	case C2SQuestMessage:
		m = &QuestMessage{}
	case C2SBuyItem:
		m = &BuyItem{}
	case C2SSellItem:
		m = &SellItem{}
	case C2SIdentifyItems:
		m = &IdentifyItems{}
	case C2SRepairItem:
		m = &RepairItem{}
	case C2SHireMercenary:
		m = &HireMercenary{}
	case C2SGambleItem:
		m = &GambleItem{}
	case 0x02, 0x04, 0x06, 0x07, 0x09, 0x0a, 0x0d, 0x0e, 0x10, 0x11:
		m = &UnitOrder{ID: b[0]}
	case 0x08, 0x0f:
		m = &LocationOrder{ID: b[0]}
	case C2SToggleState12:
		m = &ToggleState12{}
	case C2SAllocateStat:
		m = &AllocateStat{}
	case C2SAddSkillPoint:
		m = &AddSkillPoint{}
	case C2SNpcTrade:
		m = &NpcTrade{}
	case C2SSetHotkey:
		m = &SetHotkey{}
	case C2SPartyRelation:
		m = &PartyRelation{}
	case C2SPartyRequest:
		m = &PartyRequest{}
	default:
		return nil, fmt.Errorf("%w: no typed decoder for client id %#x", ErrUnknownSize, b[0])
	}
	want, _ := ExpectedSize(b[0], ClientToServer)
	if len(b) != want {
		return nil, fmt.Errorf("%w: id %#x wants %d bytes, got %d", ErrWrongSize, b[0], want, len(b))
	}
	m.unmarshal(b)
	return m, nil
}
