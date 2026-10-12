package d2level

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"

// The waypoint menu of the original, ported from Game.exe 1.14b (wave 24). VERIFIED = read in the decompile.
//
//   - UI_BuildWaypointList 0x498fc0 and the tab table at 0x721820 (11 byte records: built flag, first level, last level,
//     lowest and highest waypoint number): which rows a tab lists and which are selectable.
//   - UI_SelectWaypointTabForAct 0x498f30 and UI_DrawWaygatePanel 0x499190: which tab opens, which tabs are drawn.
//   - SERVER_ValidateWaypointTravel 0x547450, SERVER_HandleC2STakeWaypoint 0x54a410, SERVER_ExecuteWaypointTravel 0x582e30,
//     OBJECT_OperateActivateWaypoint 0x582d00: the server side of a trip.
//   - WAYPOINT_Activate / IsActivated / LoadSectionFromSave / AllocWaypointFlags 0x663590 ...: the record.
//   - TRADE_ServerHandleNpcMenuAction 0x577b70 and QUEST_ActivateHarrogathWaypoint 0x5b2c20: the waypoints that open by
//     talking.

// WaypointTabs is the number of tabs of the menu in the expansion (classic draws the first four).
const WaypointTabs = 5

// waypointTabLevels are the level id ranges the five tabs list (the table at 0x721820, VERIFIED): the tab lists every level
// of its range that has a waypoint number in Levels.txt. Act 1 1..38, Act 2 40..74, Act 3 75..102, Act 4 103..107,
// Act 5 109..136.
//
//nolint:gochecknoglobals // table of the exe
var waypointTabLevels = [WaypointTabs][2]int{{1, 38}, {40, 74}, {75, 102}, {103, 107}, {109, 136}}

// Quest slots (QUESTREC_GetFlag(record, slot, 0), bit 0 = QuestBitDone) that open the tabs. VERIFIED in
// UI_DrawWaygatePanel: tab 1 needs slot 7, tab 2 slot 15, tab 3 slot 23; tab 4 (Act V, expansion only) slot 26 (0x1a).
const (
	tabSlotAct2 = 7
	tabSlotAct3 = 15
	tabSlotAct4 = 23
	tabSlotAct5 = 26
	// tabSlotClampAct5 is the slot UI_SelectWaypointTabForAct tests for Act V (28, the act-finished slot of Act IV), not 26.
	tabSlotClampAct5 = 28
)

// WaypointTabRange returns the level range of tab 0..4.
func WaypointTabRange(tab int) (first, last int, ok bool) {
	if tab < 0 || tab >= WaypointTabs {
		return 0, 0, false
	}

	return waypointTabLevels[tab][0], waypointTabLevels[tab][1], true
}

// TabUnlocked reports whether the tab is drawn selectable (UI_DrawWaygatePanel): tab 0 always, the others once the quest
// slot is done; the Act V tab also needs the expansion. done reports bit 0 of a quest slot. The tab that is open is drawn
// whatever this says.
func TabUnlocked(tab int, expansion bool, done func(slot int) bool) bool {
	switch tab {
	case 0:
		return true
	case 1:
		return done(tabSlotAct2)
	case 2:
		return done(tabSlotAct3)
	case 3:
		return done(tabSlotAct4)
	case 4:
		return expansion && done(tabSlotAct5)
	}

	return false
}

// TabCount is how many tabs the panel draws: five with the expansion, four in classic.
func TabCount(expansion bool) int {
	if expansion {
		return WaypointTabs
	}

	return WaypointTabs - 1
}

// SelectWaypointTab is UI_SelectWaypointTabForAct: the tab that opens for the act index (0..4) of the level the hero stands
// in. It never opens a tab beyond the acts the hero has finished: Act V falls back to Act IV unless slot 28 is done, Act IV
// to III unless slot 23, then to II unless slot 15, then to I unless slot 7. An act index of 5 or more gives tab 0.
func SelectWaypointTab(actIndex int, done func(slot int) bool) int {
	a := actIndex
	if a >= WaypointTabs || a < 0 {
		a = 0
	}

	switch a {
	case 0:
		return 0
	case 1:
		// straight to the slot 7 test
	case 2:
		if !done(tabSlotAct3) {
			a = 1
		}
	case 3:
		if !done(tabSlotAct4) {
			a = 2
		}

		if !done(tabSlotAct3) {
			a = 1
		}
	case 4:
		if !done(tabSlotClampAct5) {
			a = 3
		}

		if !done(tabSlotAct4) {
			a = 2
		}

		if !done(tabSlotAct3) {
			a = 1
		}
	}

	if !done(tabSlotAct2) {
		return 0
	}

	return a
}

// WaypointMenuRow is one row of the menu.
type WaypointMenuRow struct {
	Level int
	No    int // the waypoint number (bit)
	// Selectable is true when the waypoint is activated and is not the level the hero stands in. The other rows are drawn
	// dimmed (not activated) or highlighted but dead (the current level).
	Selectable bool
	Current    bool
}

// WaypointMenu builds the rows of a tab like UI_BuildWaypointList: every waypoint number between the lowest and the
// highest of the tab's level range that belongs to a level, in number order. active reports whether the record has the
// number. Bit 0 (the Rogue Encampment) is always on in a loaded record. Selectable counts the rows the title text says
// "choose" for (string 0xf96) instead of the no-destination text (0xf97).
func WaypointMenu(tab, currentLevel int, active func(no int) bool) (rows []WaypointMenuRow, selectable int) {
	first, last, ok := WaypointTabRange(tab)
	if !ok {
		return nil, 0
	}

	lo, hi := -1, -1

	for lvl := first; lvl <= last; lvl++ {
		no, has := WaypointBit(lvl)
		if !has {
			continue
		}

		if lo < 0 || no < lo {
			lo = no
		}

		if no > hi {
			hi = no
		}
	}

	for no := lo; lo >= 0 && no <= hi; no++ {
		lvl := WaypointLevel(no)
		if lvl == 0 || lvl < first || lvl > last {
			continue
		}

		on := no == 0 || active(no)
		row := WaypointMenuRow{Level: lvl, No: no, Current: lvl == currentLevel}
		row.Selectable = on && !row.Current

		if row.Selectable {
			selectable++
		}

		rows = append(rows, row)
	}

	return rows, selectable
}

// NormalizeWaypoints applies what loading a record does (WAYPOINT_LoadSectionFromSave and AllocWaypointFlags): bit 0 is set
// in every difficulty.
func NormalizeWaypoints(wp *d2s.Waypoints) {
	for d := 0; d < len(wp); d++ {
		wp.Set(d, d2s.Waypoint(0), true)
	}
}

// TravelCode is the result of SERVER_ValidateWaypointTravel.
type TravelCode int

// The codes (VERIFIED returns). A hero too far from the object gets the raw return of UNIT_IsFarFromPoint (0x546de0),
// which is 1 (VERIFIED: its far exit loads EAX=1), so far shares code 1 with a missing object.
const (
	TravelOK         TravelCode = 0
	TravelNoObject   TravelCode = 1 // the waypoint object is not in the game, or the hero is too far from it (UNIT_IsFarFromPoint returns 1)
	TravelNotActive  TravelCode = 2 // another act than the object's, or the waypoint is not activated
	TravelBadRequest TravelCode = 3 // a level number out of range, a level without waypoint, packet size
)

// TravelRequest is what the checks read.
type TravelRequest struct {
	ObjectFound bool
	PlayerAct   int
	ObjectAct   int
	Far         bool // the hero is far from the waypoint object
	Target      int  // 0: just close the menu
	LevelCount  int  // levels.txt row count (nLevels)
	Active      func(no int) bool
}

// ValidateTravel ports 0x547450 in order. NOTE: the target level's act is NOT compared; the compared acts are the hero's
// and the waypoint object's. So a waypoint takes the hero to any activated waypoint of any act.
func ValidateTravel(r TravelRequest) TravelCode {
	if !r.ObjectFound {
		return TravelNoObject
	}

	if r.PlayerAct != r.ObjectAct {
		return TravelNotActive
	}

	if r.Far {
		return TravelNoObject
	}

	if r.Target == 0 {
		return TravelOK
	}

	if r.Target < r.LevelCount {
		if no, ok := WaypointBit(r.Target); ok {
			if no == 0 || r.Active(no) {
				return TravelOK
			}

			return TravelNotActive
		}
	}

	return TravelBadRequest
}

// TakeWaypointCooldownMs is the wait between the last portal or waypoint use and the next waypoint request
// (SERVER_HandleC2STakeWaypoint: 10000 ms against the tick the relation record keeps; the portal uses 5000).
const TakeWaypointCooldownMs = 10000

// TakeWaypointPacketSize is the size of the C2S take-waypoint packet; any other size gives result 3.
const TakeWaypointPacketSize = 9

// TakeWaypoint is the order of the handler: packet size, cooldown (result 1, the hero plays the refusing mode 0x13 and the
// target is cleared), then the validation. A success runs the travel (SERVER_ExecuteWaypointTravel).
func TakeWaypoint(size int, sinceLastMs int64, v TravelCode) TravelCode {
	if size != TakeWaypointPacketSize {
		return TravelBadRequest
	}

	if sinceLastMs >= 0 && sinceLastMs < TakeWaypointCooldownMs {
		return TravelNoObject
	}

	return v
}

// ExecuteTravelOK is the check SERVER_ExecuteWaypointTravel repeats before it moves the hero: the object must be a
// waypoint (objects.txt operate function), the target must differ from the level the object stands in, be non-zero and
// activated. The arrival uses WaypointStartType.
func ExecuteTravelOK(isWaypointObject bool, objectLevel, target int, active func(no int) bool) bool {
	if !isWaypointObject || target == 0 || target == objectLevel {
		return false
	}

	no, ok := WaypointBit(target)

	return ok && (no == 0 || active(no))
}

// NpcWaypoint is a waypoint that opens when an NPC takes the hero to its town (TRADE_ServerHandleNpcMenuAction).
type NpcWaypoint struct {
	Class int // monster class of the NPC
	Level int // the town the trip ends in; its waypoint is activated
	Slot  int // the quest slot whose bit 0 the menu entry needs
	// Extra is an UNVERIFIED extra condition (Tyrael: the game record's seed field must be non-zero).
	Extra string
}

// NpcWaypoints lists the three travel rows that activate the arrival town's waypoint (VERIFIED): Warriv (155) to Lut
// Gholein (0x28) after slot 6, Meshif (210) to the Kurast Docks (0x4b) after slot 14, Tyrael (367) to Harrogath (0x6d)
// after slot 26. The rows of the other NPCs (175, 264, 520, 0xaf, 0x108 to levels 1, 0x28, 0x67) travel without it.
//
//nolint:gochecknoglobals // table of the exe
var NpcWaypoints = []NpcWaypoint{
	{Class: 155, Level: 0x28, Slot: 6},
	{Class: 210, Level: 0x4b, Slot: 0xe},
	{Class: 367, Level: 0x6d, Slot: 0x1a, Extra: "game record seed field non-zero (dwGameSeed), meaning UNVERIFIED"},
}

// NpcWaypointFor finds the row of an NPC class.
func NpcWaypointFor(class int) (NpcWaypoint, bool) {
	for _, n := range NpcWaypoints {
		if n.Class == class {
			return n, true
		}
	}

	return NpcWaypoint{}, false
}

// HarrogathLevel is the level QUEST_ActivateHarrogathWaypoint activates (called from the Act IV to Act V travel handler).
const HarrogathLevel = 0x6d
