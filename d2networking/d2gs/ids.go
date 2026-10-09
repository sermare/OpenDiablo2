package d2gs

import "fmt"

// Direction says which way a packet travels.
type Direction int

// Directions.
const (
	ServerToClient Direction = iota
	ClientToServer
)

func (d Direction) String() string {
	if d == ServerToClient {
		return "server->client"
	}
	return "client->server"
}

// Client -> server in-game packet ids. The names for 0x13, 0x16..0x2a and
// 0x2f..0x37 come from handler bodies (verified); the rest of the 0x01..0x66
// range is documented in Names with Verified=false.
const (
	C2SWalkToLocation    byte = 0x01 // verified: len 5, u16 x @1, u16 y @3
	C2SRunToLocation     byte = 0x03 // verified: same layout as 0x01
	C2SCastLeftLocation  byte = 0x05 // verified: len 5, u16 x @1, u16 y @3 (skill = selected left skill)
	C2SCastRightLocation byte = 0x0c // verified: same layout (selected right skill)
	C2SChat              byte = 0x15 // verified: [type][?] then NUL-terminated message @3 and recipient name
	C2SSelectSkill       byte = 0x3c // verified: len 9, u32 skill (bit 31 = right hand) @1, u32 item id @5
	C2SInteractUnit      byte = 0x13
	C2SPickUpUnit        byte = 0x16
	C2SDropCursorItem    byte = 0x17
	C2SItemToContainer   byte = 0x18
	C2SPickFromContainer byte = 0x19
	C2SItemToBody        byte = 0x1a
	C2SSwapTwoHandedBody byte = 0x1b
	C2SPickFromBody      byte = 0x1c
	C2SSwapBodyItem      byte = 0x1d
	C2SSwapWeaponHands   byte = 0x1e
	C2SSwapContainerItem byte = 0x1f
	C2SUseItem           byte = 0x20
	C2SStackItems        byte = 0x21
	C2SUnstackItem       byte = 0x22
	C2SItemToBelt        byte = 0x23
	C2SPickFromBelt      byte = 0x24
	C2SSwapBeltItem      byte = 0x25
	C2SUseBeltItem       byte = 0x26
	C2SIdentifyItem      byte = 0x27
	C2SInsertSocketItem  byte = 0x28
	C2SScrollToTome      byte = 0x29
	C2SItemToCube        byte = 0x2a
	C2SNpcInit           byte = 0x2f
	C2SNpcCancel         byte = 0x30
	C2SQuestMessage      byte = 0x31
	C2SBuyItem           byte = 0x32
	C2SSellItem          byte = 0x33
	C2SIdentifyItems     byte = 0x34
	C2SRepairItem        byte = 0x35
	C2SHireMercenary     byte = 0x36
	C2SGambleItem        byte = 0x37
)

// Pre-game control ids (client -> server lane 0, CCMD_ProcessLobbyControlPacket).
// Handler association is verified; sizes are not decoded in the notes.
const (
	CtlCreateGame     byte = 0x67
	CtlJoinGame       byte = 0x68
	CtlLeaveGame      byte = 0x69
	CtlGameClosing    byte = 0x6a // unverified meaning
	CtlJoinAct        byte = 0x6b
	CtlSaveFileUpload byte = 0x6c // len < 0x2000
	CtlPing           byte = 0x6d
	CtlUnknown6E      byte = 0x6e
	CtlUnknown70      byte = 0x70
)

// Admin lane ids (lane 2, replies sent with mode 2 and never compressed).
const (
	AdminServerInfo  byte = 0x01 // also used by the query path; 0x01 here is lane-2 only
	AdminConnect     byte = 0xfa
	AdminDisconnect  byte = 0xfb
	AdminPing        byte = 0xfc
	AdminServerInfo2 byte = 0xfd
)

// Server -> client packet ids that the notes tie to handler bodies.
const (
	S2CGameLoading    byte = 0x00
	S2CGameFlags      byte = 0x01
	S2CLoadSuccessful byte = 0x02
	S2CLoadAct        byte = 0x03
	S2CLoadComplete   byte = 0x04
	S2CUnloadComplete byte = 0x05
	S2CGameExit       byte = 0x06
	S2CPlayerHPMana   byte = 0x18 // handler special-cased, verified
	S2CItemAction     byte = 0x9c
	S2CItemActionExt  byte = 0x9d
	S2CSaveFileChunk  byte = 0xb3 // exempt from compression in mode 0, verified
	S2CClientAck      byte = 0xaf
	S2CLobbyCharInfo  byte = 0xb2
	S2CMetaAE         byte = 0xae
	S2CWalkVerify     byte = 0x96
	S2CPlayerLifeMana byte = 0x95
)

// Name describes a packet id.
type Name struct {
	Name string
	// Verified is true only when the name was confirmed against a handler body.
	// Everything else comes from public D2GS documentation and may be wrong.
	Verified bool
}

var serverNames = map[byte]Name{
	0x00: {"GameLoading", false}, 0x01: {"GameFlags", false}, 0x02: {"LoadSuccessful", false},
	0x03: {"LoadAct", false}, 0x04: {"LoadComplete", false}, 0x05: {"UnloadComplete", false},
	0x06: {"GameExit", false}, 0x07: {"MapReveal", false}, 0x08: {"MapHide", false},
	0x09: {"AssignLevelWarp", false}, 0x0a: {"RemoveObject", false}, 0x0b: {"GameHandshake", false},
	0x0c: {"NPCGetHit", false}, 0x0d: {"PlayerStop", false}, 0x0e: {"ObjectState", false},
	0x0f: {"PlayerMove", false}, 0x10: {"CharToObject", false}, 0x11: {"ReportKill", false},
	0x15: {"ReassignPlayer", false}, 0x16: {"Unknown16", false},
	0x18: {"PlayerHPMana", true}, 0x19: {"SetByteAttr", false}, 0x1a: {"AddByteAttr", false},
	0x1b: {"SetWordAttr", false}, 0x1c: {"AddWordAttr", false}, 0x1d: {"SetDwordAttr", false},
	0x1e: {"AddDwordAttr", false}, 0x1f: {"AddDwordAttr2", false},
	0x20: {"UpdateItemStats", false}, 0x21: {"UpdateSkill", false}, 0x26: {"Chat", false},
	0x27: {"NPCInfo", false}, 0x28: {"UpdateQuestInfo", false}, 0x29: {"GameQuestInfo", false},
	0x2a: {"NPCTransaction", false}, 0x3e: {"UpdateItemStats", false},
	0x4c: {"NPCSkill", false}, 0x4d: {"UnitMove", false}, 0x51: {"WorldObject", false},
	0x52: {"AssignMerc", false}, 0x59: {"PlayerInGame", false}, 0x5a: {"EventMessage", false},
	0x5b: {"PlayerJoin", false}, 0x5c: {"PlayerLeave", false}, 0x5d: {"QuestItemState", false},
	0x5e: {"GameMessage", false}, 0x63: {"WaypointMenu", false}, 0x82: {"PortalOwnership", false},
	0x89: {"ActEnd", false}, 0x95: {"PlayerLifeMana", true}, 0x96: {"WalkVerify", true},
	0x9c: {"ItemAction", false}, 0x9d: {"ItemActionExt", false},
	0xae: {"MetaAE", false}, 0xaf: {"ClientAck", true}, 0xb2: {"LobbyCharInfo", true},
	0xb3: {"SaveFileChunk", true},
}

var clientNames = map[byte]Name{
	0x01: {"WalkToLocation", false}, 0x02: {"WalkToEntity", false}, 0x03: {"RunToLocation", false},
	0x04: {"RunToEntity", false}, 0x05: {"CastLeftOnLocation", false}, 0x06: {"CastLeftOnEntity", false},
	0x13: {"InteractUnit", true},
	0x14: {"OverheadChat", false}, 0x15: {"Chat", false},
	0x16: {"PickUpUnit", true}, 0x17: {"DropCursorItem", true}, 0x18: {"ItemToContainer", true},
	0x19: {"PickFromContainer", true}, 0x1a: {"ItemToBody", true}, 0x1b: {"SwapTwoHandedBody", true},
	0x1c: {"PickFromBody", true}, 0x1d: {"SwapBodyItem", true}, 0x1e: {"SwapWeaponHands", true},
	0x1f: {"SwapContainerItem", true}, 0x20: {"UseItem", true}, 0x21: {"StackItems", true},
	0x22: {"UnstackItem", true}, 0x23: {"ItemToBelt", true}, 0x24: {"PickFromBelt", true},
	0x25: {"SwapBeltItem", true}, 0x26: {"UseBeltItem", true}, 0x27: {"IdentifyItem", true},
	0x28: {"InsertSocketItem", true}, 0x29: {"ScrollToTome", true}, 0x2a: {"ItemToCube", true},
	0x2f: {"NpcInit", true}, 0x30: {"NpcCancel", true}, 0x31: {"QuestMessage", true},
	0x32: {"BuyItem", true}, 0x33: {"SellItem", true}, 0x34: {"IdentifyItems", true},
	0x35: {"RepairItem", true}, 0x36: {"HireMercenary", true}, 0x37: {"GambleItem", true},
	0x38: {"Trade", false}, 0x3c: {"SelectSkill", false}, 0x40: {"QuestRequest", false},
	0x5e: {"Waypoint", false}, 0x5f: {"Stash", false}, 0x66: {"Phrase", false},
	0x67: {"CreateGame", true}, 0x68: {"JoinGame", true}, 0x69: {"LeaveGame", true},
	0x6a: {"GameClosing", false}, 0x6b: {"JoinAct", true}, 0x6c: {"SaveFileUpload", true},
	0x6d: {"Ping", false}, 0x6e: {"Control6E", false}, 0x70: {"Control70", false},
	0xfa: {"AdminConnect", false}, 0xfb: {"AdminDisconnect", false}, 0xfc: {"AdminPing", false},
	0xfd: {"AdminServerInfo", false},
}

// PacketName returns the best known name for an id and whether it was
// verified. Unknown ids yield a hex placeholder.
func PacketName(id byte, dir Direction) (string, bool) {
	m := serverNames
	if dir == ClientToServer {
		m = clientNames
	}
	if n, ok := m[id]; ok {
		return n.Name, n.Verified
	}
	return fmt.Sprintf("Unknown%02X", id), false
}
