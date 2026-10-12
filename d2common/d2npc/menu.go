// Package d2npc holds the pure rules of the town NPCs that Game.exe 1.14b keeps in its client UI code: which rows
// an NPC's menu shows (UI_BuildNpcMenu 0x4b1050 over the static table at 0x725d60), which line the Talk submenu
// plays for "introduction" and "gossip" (UI_InitNpcMenuEntry 0x4ade00, the callbacks 0x4b0a00 / 0x4b09e0), when the
// spoken greeting of UI_StartNpcInteraction (0x4b2f50) plays, and which NPCs restore the hero on talk
// (NPCSRV_RestoreVitalsOnTalk 0x576c80). No asset or UI types: everything is testable without the game files.
package d2npc

// Kind is what a menu row does.
type Kind int

// The kinds of NPC menu rows.
const (
	KindTalk Kind = iota
	KindTrade
	KindTradeRepair
	KindGamble
	KindHire
	KindIdentify
	KindGoWest
	KindSailWest
	KindRevive
	KindRespec
)

// String ids of string.tbl used by the menu table (0x725d60) and the rows added at run time.
const (
	StrTalk        = 0xd35
	StrTrade       = 0xd44
	StrTradeRepair = 0xd06
	StrGamble      = 0xd46
	StrHire        = 0xd45
	StrIdentify    = 0xfb4 // "Identify Items"
	StrIdentifyFee = 0xfb5 // "Identify Items: " followed by the fee
	StrGoWest      = 0xd37
	StrSailWest    = 0xd39
	StrRevive      = 0x1507 // the table text is "an evil force"; the row is drawn with the "Resurrect %s: %d" format (0x58a8)
	StrRespec      = 0x2ba0
)

// Option is one visible row of an NPC menu (the implicit last Cancel row is not listed).
type Option struct {
	StringID int
	Kind     Kind
	// Cost is the gold shown in the label (Identify fee, Resurrect cost); 0 = none shown.
	Cost int
}

func kindOf(strID int) Kind {
	switch strID {
	case StrTrade:
		return KindTrade
	case StrTradeRepair:
		return KindTradeRepair
	case StrGamble:
		return KindGamble
	case StrHire:
		return KindHire
	case StrIdentify:
		return KindIdentify
	case StrGoWest:
		return KindGoWest
	case StrSailWest:
		return KindSailWest
	case StrRevive:
		return KindRevive
	case StrRespec:
		return KindRespec
	default:
		return KindTalk
	}
}

// menuRow is one entry of the static table at 0x725d60 (48 entries of 0x27 bytes; the strings are the ids of
// the visible rows, the 5 handler pointers are not needed here).
type menuRow struct {
	class   int
	strings []int
	// reopen is the byte at +0x26: non-zero lets a click on the NPC reopen the menu (0x4b32c0).
	reopen bool
}

// monster class ids of the NPCs the rules name (monstats Id column).
const (
	ClassKashya     = 150 // 0x96
	ClassGreiz      = 198 // 0xc6
	ClassAsheara    = 252 // 0xfc
	ClassTyrael2    = 367 // 0x16f
	ClassQualKehk   = 515 // 0x203
	ClassAkara      = 148 // 0x94
	ClassJerhyn     = 201 // 0xc9
	ClassMalah      = 513 // 0x201
	ClassNihlathak  = 514 // 0x202
	ClassWarriv1    = 155
	ClassMeshif1    = 210
	ClassCain1      = 146
	ClassCain2      = 244
	ClassCain3      = 245
	ClassCain4      = 246
	ClassCain5      = 265
	ClassCain6      = 520
	ClassAtma       = 176 // 0xb0
	ClassFara       = 178 // 0xb2
	ClassJamella    = 405 // 0x195
	ClassOrmus      = 255 // 0xff
	identifyPerItem = 100
)

//nolint:gochecknoglobals // static exe data
var menuTable = []menuRow{
	{148, []int{StrTalk, StrTrade}, true},
	{176, []int{StrTalk}, true},
	{146, []int{StrTalk}, false},
	{154, []int{StrTalk, StrTradeRepair}, true},
	{177, []int{StrTalk, StrTrade}, true},
	{199, []int{StrTalk, StrTrade, StrGamble}, true},
	{200, []int{StrTalk}, true},
	{147, []int{StrTalk, StrTrade, StrGamble}, true},
	{198, []int{StrTalk, StrHire}, true},
	{201, []int{StrTalk}, true},
	{202, []int{StrTalk, StrTrade}, true},
	{150, []int{StrTalk}, true},
	{178, []int{StrTalk, StrTradeRepair}, true},
	{155, []int{StrTalk}, true},
	{175, []int{StrTalk, StrGoWest}, true},
	{210, []int{StrTalk}, true},
	{244, []int{StrTalk, StrIdentify}, true},
	{265, []int{StrTalk, StrIdentify}, true},
	{245, []int{StrTalk, StrIdentify}, true},
	{246, []int{StrTalk, StrIdentify}, true},
	{251, []int{StrTalk}, false},
	{367, []int{StrTalk}, true},
	{257, []int{StrTradeRepair}, true}, // Halbu; the second 257 row (Talk only) further down is shadowed
	{405, []int{StrTrade, StrGamble}, true},
	{252, []int{StrTalk, StrHire, StrTrade}, true},
	{253, []int{StrTalk, StrTradeRepair}, true},
	{254, []int{StrTalk, StrTrade, StrGamble}, true},
	{255, []int{StrTalk, StrTrade}, true},
	{406, []int{StrTalk}, true},
	{257, []int{StrTalk}, true},
	{264, []int{StrTalk, StrSailWest}, true},
	{297, []int{StrTalk}, true},
	{266, []int{StrTalk}, true},
	{331, []int{StrTalk}, true},
	{408, []int{StrTalk}, true},
	{378, []int{StrTalk}, true},
	{377, []int{StrTalk}, true},
	{521, []int{StrTalk}, true},
	{520, []int{StrTalk, StrIdentify}, true},
	{511, []int{StrTalk, StrTradeRepair}, true},
	{513, []int{StrTalk, StrTrade}, true},
	{512, []int{StrTalk, StrTrade, StrGamble}, true},
	{514, []int{StrTalk, StrGamble}, true},
	{515, []int{StrTalk}, true},
	{527, []int{StrTalk}, false},
	{537, []int{StrTalk}, false},
	{538, []int{StrTalk}, false},
	{539, []int{StrTalk}, false},
}

func rowFor(class int) (menuRow, bool) {
	for _, r := range menuTable { // linear, first match wins (0x4af630)
		if r.class == class {
			return r, true
		}
	}

	return menuRow{}, false
}

// StaticRows is the static table of the exe for a class: the visible rows before any run time change. ok is
// false for a class that has no row (the exe then builds no menu at all; OpenDiablo2 falls back to Talk).
func StaticRows(class int) (rows []Option, ok bool) {
	r, ok := rowFor(class)
	if !ok {
		return nil, false
	}

	for _, s := range r.strings {
		rows = append(rows, Option{StringID: s, Kind: kindOf(s)})
	}

	return rows, true
}

// Reopens is the +0x26 byte of the class's table row (a click may reopen the menu).
func Reopens(class int) bool {
	r, _ := rowFor(class)

	return r.reopen
}

// TableSize is the number of rows of the static table (DAT_00724b8c = 48).
func TableSize() int { return len(menuTable) }

// IsMercHirer is the set of classes UI_BuildNpcMenu sends the hire-list request for (0xfc, 0xc6, 0x203, 0x96).
func IsMercHirer(class int) bool {
	return class == ClassAsheara || class == ClassGreiz || class == ClassQualKehk || class == ClassKashya
}

// IsMercReviver is the set of classes UI_FilterNpcMenuEntriesForExpansion (0x4b2ce0) adds the resurrect row to:
// the four hirers and Tyrael of the Pandemonium Fortress (0x16f).
func IsMercReviver(class int) bool { return IsMercHirer(class) || class == ClassTyrael2 }

// KashyaHireLevel is the hero level from which Kashya's menu gets its Hire row (UI_StartNpcInteraction:
// STATS_GetUnitStat(player, 0xc) > 7 calls UI_SetupMercHireMenuEntry 0x4b2cb0).
const KashyaHireLevel = 8

// QuestFlag reads one bit of a quest slot of the hero (QUESTREC_GetFlag).
type QuestFlag func(quest, bit int) bool

// Ctx is what the menu rules read from the game.
type Ctx struct {
	// HeroLevel is stat 0xc of the hero.
	HeroLevel int
	// Expansion: MAIN_CheckExpansionMpq (the resurrect row exists in Lord of Destruction only).
	Expansion bool
	// MercDead: the client knows of a dead mercenary to resurrect (DAT_007245a8 != -1; set by the server, the
	// packet is UNVERIFIED).
	MercDead bool
	// MercCost is the resurrect price (DAT_007b8e58), shown in the row.
	MercCost int
	// Unidentified is the number of unidentified items (TRADE_CountUnidentifiedItems).
	Unidentified int
	// Quest reads the hero's quest record; nil = every flag clear.
	Quest QuestFlag
	// RespecRowPresent: Akara's table row has the third (respec) string. Nothing in the exe's static code sets it
	// (the setter at 0x4b3630 has no caller in the analysis, UNVERIFIED how it runs), so the game decides.
	RespecRowPresent bool
	// RespecPanelFlag is UI_GetPanelFlagByte() == 2. VERIFIED (w47): the byte is the game difficulty (0x798694, written
	// by CLIENT_SetGameDifficulty 0x4491e0), so this is "the game is Hell"; RespecPanelFlagFor derives it.
	RespecPanelFlag bool
}

func (c Ctx) flag(quest, bit int) bool { return c.Quest != nil && c.Quest(quest, bit) }

// IdentifyFree reports the quest bits that make Cain's identify free: (4, 0) or (4, 1) of the quest record.
func IdentifyFree(c Ctx) bool { return c.flag(4, 0) || c.flag(4, 1) }

// IdentifyCost is the fee shown on Cain's row: 100 gold per unidentified item unless the quest bits are set.
func IdentifyCost(c Ctx) int {
	if IdentifyFree(c) {
		return 0
	}

	return c.Unidentified * identifyPerItem
}

// RespecPanelFlagFor is the value of Ctx.RespecPanelFlag for a difficulty (0 Normal, 1 Nightmare, 2 Hell).
func RespecPanelFlagFor(difficulty int) bool { return difficulty == 2 }

// HellRespecGrant is the server's Hell-only grant of the free respec (TRADE_ServerHandleNpcMenuAction 0x577b70, Akara
// 0x94): in Hell a hero with quest slot 1 bit 0 set (Den of Evil) and neither bit 0 nor bit 1 of slot 0x29 gets bits
// 0xd and 1 of slot 0x29 set (0x58daf0), which makes the reset that follows run. The client row shows in Hell on its own
// (RespecShown with RespecPanelFlag), the grant happens when the row's action reaches the server.
func HellRespecGrant(difficulty int, c Ctx) bool {
	return difficulty == 2 && c.flag(1, 0) && !c.flag(0x29, 0) && !c.flag(0x29, 1)
}

// RespecShown is the string-0x2ba0 branch of UI_BuildNpcMenu: skipped when quest slot 0x29 bit 0 is set;
// otherwise shown when its bit 1 is set, or when the panel byte is 2.
func RespecShown(c Ctx) bool {
	if c.flag(0x29, 0) {
		return false
	}

	return c.flag(0x29, 1) || c.RespecPanelFlag
}

// Options is the list of visible rows of an NPC's menu as the exe builds it: the static row, Kashya's Hire row
// from level 8, the resurrect row for the hirers and Tyrael, Cain's identify row only while the hero carries
// unidentified items (with the fee), Akara's respec row by its quest gate. ok is false for a class without a
// table row.
func Options(class int, c Ctx) (rows []Option, ok bool) {
	rows, ok = StaticRows(class)
	if !ok {
		return nil, false
	}

	if class == ClassKashya && c.HeroLevel >= KashyaHireLevel {
		rows = append(rows, Option{StringID: StrHire, Kind: KindHire})
	}

	if IsMercReviver(class) && c.Expansion {
		rows = withRevive(rows, c)
	}

	out := rows[:0:0]

	for _, r := range rows {
		switch r.Kind {
		case KindIdentify:
			if c.Unidentified == 0 {
				continue // the exe hides the row (and shortens the entry to two rows) when nothing is unidentified
			}

			r.Cost = IdentifyCost(c)
			if r.Cost > 0 {
				r.StringID = StrIdentifyFee
			}
		}

		out = append(out, r)
	}

	if class == ClassAkara && c.RespecRowPresent && RespecShown(c) {
		out = append(out, Option{StringID: StrRespec, Kind: KindRespec})
	}

	return out, true
}

// withRevive is UI_FilterNpcMenuEntriesForExpansion: the row goes in front of the Hire row when there is one,
// else after the last visible row; without a dead merc it is removed.
func withRevive(rows []Option, c Ctx) []Option {
	if !c.MercDead {
		return rows
	}

	row := Option{StringID: StrRevive, Kind: KindRevive, Cost: c.MercCost}

	for i, r := range rows {
		if r.Kind == KindHire {
			out := append([]Option{}, rows[:i]...)
			out = append(out, row)

			return append(out, rows[i:]...)
		}
	}

	return append(append([]Option{}, rows...), row)
}

// TableClasses lists the monster classes of the static table in table order (duplicates included: the exe's search is linear
// and the first row wins, 0x257 appears twice).
func TableClasses() []int {
	out := make([]int, len(menuTable))
	for i, r := range menuTable {
		out[i] = r.class
	}

	return out
}
