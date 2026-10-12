package d2gs

import "fmt"

// Client -> server packets whose length was verified in this pass by reading
// the server handlers in Game.exe 1.14b: each rejects any other length with
// return code 3 (the dispatcher table is at 0x6e2010). The sender side is the
// same layout (the handlers are the only consumers). Addresses are the handler
// addresses.
//
// Ids that have a handler but are not framed because it ignores or always
// rejects them regardless of length (so their length cannot be read from it):
// 0x2c, 0x2d, 0x2e, 0x39, 0x42, 0x43, 0x45, 0x52, 0x66. 0x14 and 0x15 are the
// two variable chat packets.

// Ids.
const (
	C2SIgnored0B        byte = 0x0b // 0x547eb0, len 1
	C2SObjectMode       byte = 0x3d // 0x549d30, len 5
	C2SQuestItemUse     byte = 0x3e // 0x549d80, len 5
	C2SSetUnitMode      byte = 0x3f // 0x549ea0, len 3
	C2SRequestQuestLog  byte = 0x40 // 0x549ef0, len 1
	C2SRespawn          byte = 0x41 // 0x549f10, len 1
	C2SObjectItemUse    byte = 0x44 // 0x54a190, len 17
	C2SPetCommandUnit   byte = 0x46 // 0x54a2f0, len 13
	C2SPetCommandPoint  byte = 0x47 // 0x54a360, len 13
	C2SClearInteraction byte = 0x48 // 0x54a3d0, len 1
	C2STakeWaypoint     byte = 0x49 // 0x54a410, len 9
	C2SClaimUnit        byte = 0x4b // 0x54a510, len 9
	C2SFinishConvert    byte = 0x4c // 0x54a590, len 5
	C2SNpcIntroFlag     byte = 0x4d // 0x54a4d0, len 3
	C2SClickButton      byte = 0x4f // 0x54a5e0, len 7
	C2SDropGold         byte = 0x50 // 0x54a620, len 9
	C2SStartRunning     byte = 0x53 // 0x54a760, len 1
	C2SStopRunning      byte = 0x54 // 0x54a7a0, len 1
	C2SValidateIdle     byte = 0x60 // 0x54ac80, len 1
	C2SGiveItemToMerc   byte = 0x61 // 0x54b230, len 3
	C2SReviveMerc       byte = 0x62 // 0x549a30, len 5
	C2SPickItemToBelt   byte = 0x63 // 0x54b320, len 5
)

func init() {
	for id, n := range map[byte]int{
		C2SIgnored0B: 1, C2SObjectMode: 5, C2SQuestItemUse: 5, C2SSetUnitMode: 3, C2SRequestQuestLog: 1,
		C2SRespawn: 1, C2SObjectItemUse: 17, C2SPetCommandUnit: 13, C2SPetCommandPoint: 13,
		C2SClearInteraction: 1, C2STakeWaypoint: 9, C2SClaimUnit: 9, C2SFinishConvert: 5, C2SNpcIntroFlag: 3,
		C2SClickButton: 7, C2SDropGold: 9, C2SStartRunning: 1, C2SStopRunning: 1, C2SValidateIdle: 1,
		C2SGiveItemToMerc: 3, C2SReviveMerc: 5, C2SPickItemToBelt: 5,
	} {
		clientSizes[id] = n
	}

	for id, name := range map[byte]string{
		C2SObjectMode: "ObjectMode", C2SQuestItemUse: "QuestItemUse", C2SSetUnitMode: "SetUnitMode",
		C2SRequestQuestLog: "RequestQuestLog", C2SRespawn: "Respawn", C2SObjectItemUse: "ObjectItemUse",
		C2SPetCommandUnit: "PetCommandOnUnit", C2SPetCommandPoint: "PetCommandAtPoint",
		C2SClearInteraction: "ClearInteraction", C2STakeWaypoint: "TakeWaypoint", C2SClaimUnit: "ClaimUnitOwnership",
		C2SFinishConvert: "FinishItemConvert", C2SNpcIntroFlag: "SetNpcIntroFlag", C2SClickButton: "ClickButton",
		C2SDropGold: "DropGold", C2SStartRunning: "StartRunning", C2SStopRunning: "StopRunning",
		C2SValidateIdle: "ValidateIdleState", C2SGiveItemToMerc: "GiveItemToMercenary",
		C2SReviveMerc: "ReviveMercenary", C2SPickItemToBelt: "PickItemToBelt",
	} {
		clientNames[id] = Name{Name: name, Verified: true}
	}

	// server names confirmed from the client handlers (w46-names2.md and this pass)
	for id, name := range map[byte]string{
		S2CUnitActionBasic: "UnitActionBasic", S2CUnitStop: "UnitStop", S2CObjectState: "ObjectState",
		S2CMoveToEntity: "PlayerMoveToEntity", S2CAttrAddGoldByte: "AddGoldByte", S2CAttrAddExpByte: "AddExpByte",
		S2CAttrAddExpWord: "AddExpWord", S2CAttrSetExpDword: "SetExpDword", S2CAttrSetByte: "SetStatByte",
		S2CAttrSetWord: "SetStatWord", S2CAttrSetDword: "SetStatDword", S2CSetPlayerStat: "SetPlayerStat",
		S2CUpdateSkillLevel: "UpdateSkillLevel", S2CSetActiveSkill: "SetActiveSkill", S2CWaypointMenu: "WaypointMenu",
		S2CMissileCreate: "MissileCreate", S2CSkillSlot: "SkillSlot",
	} {
		serverNames[id] = Name{Name: name, Verified: true}
	}
}

// BareCommand is a one-byte client packet (0x0b, 0x40, 0x41, 0x48, 0x53, 0x54,
// 0x60): the id alone is the request. 0x40 asks for the quest log states, 0x41
// respawns a dead player, 0x53 / 0x54 switch walking and running while moving,
// 0x48 clears the interaction flag at PlayerData+0x4c, 0x60 only validates the
// idle state (all from the handler names and bodies).
type BareCommand struct{ ID byte }

// PacketID implements Message.
func (m BareCommand) PacketID() byte { return m.ID }

// MarshalPacket implements Message.
func (m BareCommand) MarshalPacket() []byte { return []byte{m.ID} }

func (m *BareCommand) unmarshal([]byte) {}

// ObjectMode (0x3d): u32 object id @1; the server updates the object's mode by
// collision (OBJECT_ServerUpdateObjectModeByCollision) if the object is within
// range 10.
type ObjectMode struct{ ObjectID uint32 }

// PacketID implements Message.
func (ObjectMode) PacketID() byte { return C2SObjectMode }

// MarshalPacket implements Message.
func (m ObjectMode) MarshalPacket() []byte { return one(C2SObjectMode, m.ObjectID) }

func (m *ObjectMode) unmarshal(b []byte) { m.ObjectID = rbuf(b).u32(1) }

// QuestItemUse (0x3e): u32 item id @1; the item must be in the inventory or on
// the cursor, or close by in the same act.
type QuestItemUse struct{ ItemID uint32 }

// PacketID implements Message.
func (QuestItemUse) PacketID() byte { return C2SQuestItemUse }

// MarshalPacket implements Message.
func (m QuestItemUse) MarshalPacket() []byte { return one(C2SQuestItemUse, m.ItemID) }

func (m *QuestItemUse) unmarshal(b []byte) { m.ItemID = rbuf(b).u32(1) }

// SetUnitMode (0x3f, 3 bytes): u16 @1; values 0x19..0x20 set the player's mode
// (UNIT_SetOffsetAndRegisterInRoom), other values are accepted and ignored.
type SetUnitMode struct{ Mode uint16 }

// PacketID implements Message.
func (SetUnitMode) PacketID() byte { return C2SSetUnitMode }

// MarshalPacket implements Message.
func (m SetUnitMode) MarshalPacket() []byte {
	w := newW(C2SSetUnitMode, 3)
	w.u16(1, m.Mode)

	return w
}

func (m *SetUnitMode) unmarshal(b []byte) { m.Mode = rbuf(b).u16(1) }

// ObjectItemUse (0x44, 17 bytes): u32 object id @5, u32 item id @9, u16 action
// @13 (2, or 3 = with the cursor item; anything else returns code 2). Bytes
// @1..4 and @15..16 are not read.
type ObjectItemUse struct {
	ObjectID, ItemID uint32
	Action           uint16
}

// PacketID implements Message.
func (ObjectItemUse) PacketID() byte { return C2SObjectItemUse }

// MarshalPacket implements Message.
func (m ObjectItemUse) MarshalPacket() []byte {
	w := newW(C2SObjectItemUse, 17)
	w.u32(5, m.ObjectID)
	w.u32(9, m.ItemID)
	w.u16(13, m.Action)

	return w
}

func (m *ObjectItemUse) unmarshal(b []byte) {
	r := rbuf(b)
	m.ObjectID, m.ItemID, m.Action = r.u32(5), r.u32(9), r.u16(13)
}

// PetCommandUnit (0x46, 13 bytes): u32 target unit id @5, u32 command @9
// (above 5 returns code 2). Bytes @1..4 are not read.
type PetCommandUnit struct{ UnitID, Command uint32 }

// PacketID implements Message.
func (PetCommandUnit) PacketID() byte { return C2SPetCommandUnit }

// MarshalPacket implements Message.
func (m PetCommandUnit) MarshalPacket() []byte {
	w := newW(C2SPetCommandUnit, 13)
	w.u32(5, m.UnitID)
	w.u32(9, m.Command)

	return w
}

func (m *PetCommandUnit) unmarshal(b []byte) { r := rbuf(b); m.UnitID, m.Command = r.u32(5), r.u32(9) }

// PetCommandPoint (0x47, 13 bytes): u16 x @5, u16 y @9 (the handler reads the
// two words from those offsets; the bytes between are not read).
type PetCommandPoint struct{ X, Y uint16 }

// PacketID implements Message.
func (PetCommandPoint) PacketID() byte { return C2SPetCommandPoint }

// MarshalPacket implements Message.
func (m PetCommandPoint) MarshalPacket() []byte {
	w := newW(C2SPetCommandPoint, 13)
	w.u16(5, m.X)
	w.u16(9, m.Y)

	return w
}

func (m *PetCommandPoint) unmarshal(b []byte) { r := rbuf(b); m.X, m.Y = r.u16(5), r.u16(9) }

// TakeWaypoint (0x49, 9 bytes): u32 waypoint object id @1, u16 destination
// waypoint index @5; the server validates the travel and executes it (it also
// refuses within 10 s of the last relation change).
type TakeWaypoint struct {
	ObjectID uint32
	Dest     uint16
}

// PacketID implements Message.
func (TakeWaypoint) PacketID() byte { return C2STakeWaypoint }

// MarshalPacket implements Message.
func (m TakeWaypoint) MarshalPacket() []byte {
	w := newW(C2STakeWaypoint, 9)
	w.u32(1, m.ObjectID)
	w.u16(5, m.Dest)

	return w
}

func (m *TakeWaypoint) unmarshal(b []byte) { r := rbuf(b); m.ObjectID, m.Dest = r.u32(1), r.u16(5) }

// ClaimUnit (0x4b, 9 bytes): u32 unit type @1 (above 5 returns code 2), u32
// unit id @5; a player may only claim itself.
type ClaimUnit struct{ UnitType, UnitID uint32 }

// PacketID implements Message.
func (ClaimUnit) PacketID() byte { return C2SClaimUnit }

// MarshalPacket implements Message.
func (m ClaimUnit) MarshalPacket() []byte { return two(C2SClaimUnit, m.UnitType, m.UnitID) }

func (m *ClaimUnit) unmarshal(b []byte) { r := rbuf(b); m.UnitType, m.UnitID = r.u32(1), r.u32(5) }

// FinishConvert (0x4c, 5 bytes): u32 item id @1 (0xffffffff = none).
type FinishConvert struct{ ItemID uint32 }

// PacketID implements Message.
func (FinishConvert) PacketID() byte { return C2SFinishConvert }

// MarshalPacket implements Message.
func (m FinishConvert) MarshalPacket() []byte { return one(C2SFinishConvert, m.ItemID) }

func (m *FinishConvert) unmarshal(b []byte) { m.ItemID = rbuf(b).u32(1) }

// NpcIntroFlag (0x4d, 3 bytes): u16 monster class @1 (must be below the
// monstats count); marks that NPC's intro as heard.
type NpcIntroFlag struct{ MonsterClass uint16 }

// PacketID implements Message.
func (NpcIntroFlag) PacketID() byte { return C2SNpcIntroFlag }

// MarshalPacket implements Message.
func (m NpcIntroFlag) MarshalPacket() []byte {
	w := newW(C2SNpcIntroFlag, 3)
	w.u16(1, m.MonsterClass)

	return w
}

func (m *NpcIntroFlag) unmarshal(b []byte) { m.MonsterClass = rbuf(b).u16(1) }

// ClickButton (0x4f, 7 bytes): three u16 @1, @3, @5 handed to
// SERVER_ProcessButtonClick (UNVERIFIED: which are the button id and its
// argument).
type ClickButton struct{ Button, Arg1, Arg2 uint16 }

// PacketID implements Message.
func (ClickButton) PacketID() byte { return C2SClickButton }

// MarshalPacket implements Message.
func (m ClickButton) MarshalPacket() []byte {
	w := newW(C2SClickButton, 7)
	w.u16(1, m.Button)
	w.u16(3, m.Arg1)
	w.u16(5, m.Arg2)

	return w
}

func (m *ClickButton) unmarshal(b []byte) {
	r := rbuf(b)
	m.Button, m.Arg1, m.Arg2 = r.u16(1), r.u16(3), r.u16(5)
}

// DropGold (0x50, 9 bytes): two dwords @1 and @5 passed to SERVER_DropPlayerGold.
// UNVERIFIED: which is the player id and which the amount (the amount is
// assumed to be the second).
type DropGold struct{ PlayerID, Amount uint32 }

// PacketID implements Message.
func (DropGold) PacketID() byte { return C2SDropGold }

// MarshalPacket implements Message.
func (m DropGold) MarshalPacket() []byte { return two(C2SDropGold, m.PlayerID, m.Amount) }

func (m *DropGold) unmarshal(b []byte) { r := rbuf(b); m.PlayerID, m.Amount = r.u32(1), r.u32(5) }

// GiveItemToMerc (0x61, 3 bytes): u16 body location @1 of the item to equip on
// the mercenary (0 with an item on the cursor means give the cursor item).
type GiveItemToMerc struct{ BodyLoc uint16 }

// PacketID implements Message.
func (GiveItemToMerc) PacketID() byte { return C2SGiveItemToMerc }

// MarshalPacket implements Message.
func (m GiveItemToMerc) MarshalPacket() []byte {
	w := newW(C2SGiveItemToMerc, 3)
	w.u16(1, m.BodyLoc)

	return w
}

func (m *GiveItemToMerc) unmarshal(b []byte) { m.BodyLoc = rbuf(b).u16(1) }

// ReviveMerc (0x62, 5 bytes): u32 @1 handed to HIRE_ServerHandleReviveMercenary
// (UNVERIFIED: the mercenary unit id).
type ReviveMerc struct{ MercID uint32 }

// PacketID implements Message.
func (ReviveMerc) PacketID() byte { return C2SReviveMerc }

// MarshalPacket implements Message.
func (m ReviveMerc) MarshalPacket() []byte { return one(C2SReviveMerc, m.MercID) }

func (m *ReviveMerc) unmarshal(b []byte) { m.MercID = rbuf(b).u32(1) }

// PickItemToBelt (0x63, 5 bytes): u32 item id @1; the item moves to a free belt
// slot.
type PickItemToBelt struct{ ItemID uint32 }

// PacketID implements Message.
func (PickItemToBelt) PacketID() byte { return C2SPickItemToBelt }

// MarshalPacket implements Message.
func (m PickItemToBelt) MarshalPacket() []byte { return one(C2SPickItemToBelt, m.ItemID) }

func (m *PickItemToBelt) unmarshal(b []byte) { m.ItemID = rbuf(b).u32(1) }

// decodeClientExtra handles the ids of this file for DecodeClient.
func decodeClientExtra(id byte) decodable {
	switch id {
	case C2SIgnored0B, C2SRequestQuestLog, C2SRespawn, C2SClearInteraction, C2SStartRunning, C2SStopRunning, C2SValidateIdle:
		return &BareCommand{ID: id}
	case C2SObjectMode:
		return &ObjectMode{}
	case C2SQuestItemUse:
		return &QuestItemUse{}
	case C2SSetUnitMode:
		return &SetUnitMode{}
	case C2SObjectItemUse:
		return &ObjectItemUse{}
	case C2SPetCommandUnit:
		return &PetCommandUnit{}
	case C2SPetCommandPoint:
		return &PetCommandPoint{}
	case C2STakeWaypoint:
		return &TakeWaypoint{}
	case C2SClaimUnit:
		return &ClaimUnit{}
	case C2SFinishConvert:
		return &FinishConvert{}
	case C2SNpcIntroFlag:
		return &NpcIntroFlag{}
	case C2SClickButton:
		return &ClickButton{}
	case C2SDropGold:
		return &DropGold{}
	case C2SGiveItemToMerc:
		return &GiveItemToMerc{}
	case C2SReviveMerc:
		return &ReviveMerc{}
	case C2SPickItemToBelt:
		return &PickItemToBelt{}
	}

	return nil
}

// ServerMessage is a typed server -> client packet.
type ServerMessage interface{ Marshal() []byte }

// DecodeServer parses a complete server -> client packet of the unit-action,
// attribute, skill and world families of this package into its typed value
// (not a pointer). Other ids return ErrUnknownSize; the older typed parsers
// (ParseGameFlags, ParsePlayerMove, ...) are still called directly.
func DecodeServer(b []byte) (ServerMessage, error) {
	if len(b) == 0 {
		return nil, ErrShort
	}

	var (
		m   ServerMessage
		err error
	)

	switch b[0] {
	case S2CUnitActionBasic:
		m, err = ParseUnitActionBasic(b)
	case S2CUnitStop:
		m, err = ParseUnitStop(b)
	case S2CObjectState:
		m, err = ParseObjectState(b)
	case S2CMoveToEntity:
		m, err = ParseMoveToEntity(b)
	case S2CUnitSkillOnTarget, S2CUnitSkillLocal:
		m, err = ParseUnitSkillOnTarget(b)
	case S2CNpcMoveToLoc:
		m, err = ParseNpcMoveToLocation(b)
	case S2CNpcMoveToTarget:
		m, err = ParseNpcMoveToTarget(b)
	case S2CNpcStateAtLoc:
		m, err = ParseNpcStateAtLocation(b)
	case S2CNpcStateOnTarget:
		m, err = ParseNpcStateOnTarget(b)
	case S2CNpcAttackLoc:
		m, err = ParseNpcAttackLocation(b)
	case S2CNpcAttackTarget:
		m, err = ParseNpcAttackTarget(b)
	case S2CNpcStop:
		m, err = ParseNpcStop(b)
	case S2CAttrAddGoldByte, S2CAttrAddExpByte, S2CAttrAddExpWord, S2CAttrSetExpDword, S2CAttrSetByte, S2CAttrSetWord, S2CAttrSetDword:
		m, err = ParseAttrUpdate(b)
	case S2CSetPlayerStat:
		m, err = ParseSetPlayerStat(b)
	case S2CUpdateSkillLevel:
		m, err = ParseUpdateSkillLevel(b)
	case S2CSetActiveSkill:
		m, err = ParseSetActiveSkill(b)
	case S2CSkillSlot:
		m, err = ParseSkillSlot(b)
	case S2CWaypointMenu:
		m, err = ParseWaypointMenu(b)
	case S2CMissileCreate:
		m, err = ParseMissileCreate(b)
	default:
		return nil, fmt.Errorf("%w: no typed server decoder for id %#x", ErrUnknownSize, b[0])
	}

	if err != nil {
		return nil, err
	}

	return m, nil
}
