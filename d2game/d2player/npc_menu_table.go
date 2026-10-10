package d2player

// NPCMenuAction is what happens when a row of an NPC menu is chosen.
type NPCMenuAction int

// The actions an NPC menu row can trigger.
const (
	NPCActionTalk NPCMenuAction = iota
	NPCActionTrade
	NPCActionTradeRepair
	NPCActionGamble
	NPCActionHire
	NPCActionIdentify
	NPCActionTravelWest // Warriv (act 2 stage): "go west"
	NPCActionSailWest   // Meshif (act 3 stage): "sail west"
	// NPCActionTravelEast and NPCActionSailEast are the rows Warriv (Act 1)
	// and Meshif (Act 2) get once the act's last quest is done; they are not
	// in the static table (UNVERIFIED how the client adds them).
	NPCActionTravelEast
	NPCActionSailEast
	NPCActionCancel
	// NPCActionTopic is a quest topic of the Talk submenu (a mode 2 message);
	// the message id is in NPCMenuRow.StringID and the text in Fallback.
	NPCActionTopic
	// NPCActionHireOffer and NPCActionReviveMerc are rows of the hire list
	// (not of the class table): one per offered mercenary, and the revive row.
	NPCActionHireOffer
	NPCActionReviveMerc
)

// String names the action for logs.
func (a NPCMenuAction) String() string {
	names := [...]string{
		"Talk", "Trade", "TradeRepair", "Gamble", "Hire", "Identify", "TravelWest", "SailWest", "TravelEast", "SailEast", "Cancel",
		"Topic", "HireOffer", "ReviveMerc",
	}

	if int(a) < 0 || int(a) >= len(names) {
		return "Unknown"
	}

	return names[a]
}

// NPCMenuRow is one visible row of an NPC menu.
type NPCMenuRow struct {
	// StringID is the string.tbl id the original game uses for the label.
	StringID int
	// Key is that string's key in string.tbl (checked against the 1.14b
	// string.tbl); OpenDiablo2 looks strings up by key, not by id.
	Key string
	// Fallback is the English text used when the string table has no match.
	Fallback string
	Action   NPCMenuAction
}

// String ids from the real game's menu table (Game.exe 1.14b, table at 0x725d60).
const (
	strIDTalk        = 0xd35
	strIDTrade       = 0xd44
	strIDTradeRepair = 0xd06
	strIDGamble      = 0xd46
	strIDHire        = 0xd45
	strIDIdentify    = 0xfb4
	strIDGoWest      = 0xd37
	strIDSailWest    = 0xd39
	// 0xd36 / 0xd38 follow from the string.tbl order WarrivMenu1b, 1c,
	// MeshifMenuEast, MeshifMenuWest (ids of 1c and West are verified)
	strIDGoEast   = 0xd36
	strIDSailEast = 0xd38
	strIDCancel   = 0xd48
)

//nolint:gochecknoglobals // static lookup data
var (
	rowTalk        = NPCMenuRow{strIDTalk, "TalkMenu", "Talk", NPCActionTalk}
	rowTrade       = NPCMenuRow{strIDTrade, "NPCMenuTrade", "Trade", NPCActionTrade}
	rowTradeRepair = NPCMenuRow{strIDTradeRepair, "NPCMenuTradeRepair", "Trade/Repair", NPCActionTradeRepair}
	rowGamble      = NPCMenuRow{strIDGamble, "gamble", "Gamble", NPCActionGamble}
	rowHire        = NPCMenuRow{strIDHire, "NPCMenuHire", "Hire", NPCActionHire}
	rowIdentify    = NPCMenuRow{strIDIdentify, "NPCIdentify1", "Identify Items", NPCActionIdentify}
	rowGoWest      = NPCMenuRow{strIDGoWest, "WarrivMenu1c", "Go West", NPCActionTravelWest}
	rowSailWest    = NPCMenuRow{strIDSailWest, "MeshifMenuWest", "Sail West", NPCActionSailWest}

	// RowGoEast and RowSailEast are the dynamic east-bound travel rows.
	RowGoEast   = NPCMenuRow{strIDGoEast, "WarrivMenu1b", "Go East", NPCActionTravelEast}
	RowSailEast = NPCMenuRow{strIDSailEast, "MeshifMenuEast", "Sail East", NPCActionSailEast}

	// RowCancel is the implicit last row of every NPC menu.
	RowCancel = NPCMenuRow{strIDCancel, "Back", "Cancel", NPCActionCancel}
)

// npcMenuTable maps monstats class id (hcIdx) to the ordered visible rows,
// transcribed from d2-re-notes/ui-npc.md (the 48 entry table at 0x725d60).
// Not every talk-only NPC is listed (the notes end some lists with "...");
// NPCs without a row get NPCMenuFor's default of Talk only.
//
//nolint:gochecknoglobals // static lookup data
var npcMenuTable = map[int][]NPCMenuRow{
	// Talk only
	176: {rowTalk}, // Atma
	146: {rowTalk}, // Cain1
	200: {rowTalk}, // Geglash
	201: {rowTalk}, // Jerhyn
	155: {rowTalk}, // Warriv1
	210: {rowTalk}, // Meshif1
	251: {rowTalk}, // Tyrael1
	367: {rowTalk, rowHire}, // 0x16f Tyrael2: Hire opens only the merc revive (0x577a10 allow-list)
	297: {rowTalk}, // Natalya
	266: {rowTalk}, // Navi
	331: {rowTalk}, // 0x14b act 2 guard
	377: {rowTalk}, // 0x179 act 2 guard
	378: {rowTalk}, // 0x17a act 2 guard
	406: {rowTalk}, // Izual ghost
	408: {rowTalk}, // Malachai

	// Talk + Trade
	148: {rowTalk, rowTrade}, // Akara
	177: {rowTalk, rowTrade}, // Drognan
	202: {rowTalk, rowTrade}, // Lysander
	255: {rowTalk, rowTrade}, // Ormus
	513: {rowTalk, rowTrade}, // Malah

	// Talk + Trade/Repair
	154: {rowTalk, rowTradeRepair}, // Charsi
	178: {rowTalk, rowTradeRepair}, // Fara
	253: {rowTalk, rowTradeRepair}, // Hratli
	511: {rowTalk, rowTradeRepair}, // Larzuk

	// Talk + Trade + Gamble
	147: {rowTalk, rowTrade, rowGamble}, // Gheed
	199: {rowTalk, rowTrade, rowGamble}, // Elzix
	254: {rowTalk, rowTrade, rowGamble}, // Alkor
	512: {rowTalk, rowTrade, rowGamble}, // Drehya/Anya

	// Hire
	150: {rowTalk, rowHire},           // Kashya (the notes list Talk only; the hire row is needed for the Act 1 rogues)
	198: {rowTalk, rowHire},           // Greiz
	252: {rowTalk, rowHire, rowTrade}, // Asheara
	515: {rowTalk, rowHire},           // Qual-Kehk (quest-gated in HIRE_ProcessHireOffer)

	// No Talk row
	257: {rowTradeRepair},      // Halbu
	405: {rowTrade, rowGamble}, // Jamella

	// Talk + Identify (Cain 2..6)
	244: {rowTalk, rowIdentify},
	245: {rowTalk, rowIdentify},
	246: {rowTalk, rowIdentify},
	265: {rowTalk, rowIdentify},
	520: {rowTalk, rowIdentify},

	// Travel
	175: {rowTalk, rowGoWest},   // Warriv2
	264: {rowTalk, rowSailWest}, // Meshif2

	// GUESS: the notes say 0x202 lists Talk + Gamble but flag the
	// oddity as unverified (the price code treats it as a buy vendor).
	514: {rowTalk, rowGamble}, // Nihlathak (town)
}

// npcClassNames maps class ids to names, for logs and tests.
//
//nolint:gochecknoglobals // static lookup data
var npcClassNames = map[int]string{
	148: "Akara", 177: "Drognan", 202: "Lysander", 255: "Ormus", 513: "Malah",
	154: "Charsi", 178: "Fara", 253: "Hratli", 511: "Larzuk",
	147: "Gheed", 199: "Elzix", 254: "Alkor", 512: "Drehya",
	198: "Greiz", 252: "Asheara", 257: "Halbu", 405: "Jamella",
	244: "Cain2", 245: "Cain3", 246: "Cain4", 265: "Cain5", 520: "Cain6",
	175: "Warriv2", 264: "Meshif2", 155: "Warriv1", 210: "Meshif1",
	176: "Atma", 146: "Cain1", 200: "Geglash", 201: "Jerhyn", 150: "Kashya",
	251: "Tyrael1", 297: "Natalya", 266: "Navi",
}

// NPCMenuFor returns the visible rows (without the implicit Cancel) for a
// monster class id. ok is false when the class has no table row; the rows
// are then just Talk (notes section (c) point 6: NPCs without a row talk).
func NPCMenuFor(classID int) (rows []NPCMenuRow, ok bool) {
	rows, ok = npcMenuTable[classID]
	if !ok {
		return []NPCMenuRow{rowTalk}, false
	}

	return rows, true
}

// NPCClassName returns a known name for a class id, or "".
func NPCClassName(classID int) string {
	return npcClassNames[classID]
}
