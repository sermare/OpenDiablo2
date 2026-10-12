package d2quest

import (
	_ "embed" // the page tables are embedded
	"encoding/csv"
	"strconv"
	"strings"
	"sync"
)

// The quest log of the original has two halves. The server side picks a page byte per quest (the getter at node+0xE8,
// or QUEST_GetDefaultLogState 0x541e30, see Game.LogPage); the client turns the page byte and the hero's quest record
// into a text, a picture and an icon (UI_BuildQuestLogEntries 0x49e060, UI_DrawQuestLogPanel 0x49fc00). This file is
// the client half: the page table (which string each page shows; read from the tables at 0x722aa8..0x7231a7 of
// Game.exe 1.14b, 41 slots of 14 pages) and the selection rules. Everything is VERIFIED against the binary unless a
// comment says UNVERIFIED.
//
// The page table is NOT "qstsa<act>q<index><page>": page 10 is the reward page again, 11 "completed in a previous
// game", 12 "another player completed it first", 13 "Quest completed", and the pages 3, 4 ... of several quests point
// at the strings of other numbers (the Horadric Staff shows qstsa2q25 on page 4 and qstsa2q24 on pages 5 and 6).

//go:embed data/logpages.csv
var logPagesCSV string

//go:embed data/logslots.csv
var logSlotsCSV string

// LogCategory is the icon class of a quest in the log (the switch on node+0x266 in UI_DrawQuestLogPanel).
type LogCategory int

// The icon classes of the quest log.
const (
	// LogCatAnimating: finished, the player has not looked yet (bit 12 clear): the completion animation (frames 0..24)
	// plays once and the client then sends packet 0x58.
	LogCatAnimating LogCategory = iota
	// LogCatDone: finished and seen: the "done" picture (frame 24).
	LogCatDone
	// LogCatLocked: nothing to show yet (the quest is not available): frame 26, the button is disabled.
	LogCatLocked
	// LogCatActive: a page of text is shown: the quest picture (frame 25 when selected, frame 0 otherwise).
	LogCatActive
)

// LogSlot is the placement of a quest in the log (the 16 byte table at 0x723230).
type LogSlot struct {
	Slot int
	// RewardPageMinusOne is the second word of the quest table: the page shown while the reward is pending (+1); -1
	// when the page byte of the server decides (Tools of the Trade, Cain, Forgotten Tower, ...).
	RewardPageMinusOne int
	Act                int // 1..5
	Socket             int // 0..5: position in the 3x2 grid of the act page
	Picture            int // global picture index (a<act>q<n>.dc6 of the act is Picture - first of the act + 1)
}

type logTables struct {
	pages map[[2]int]string
	slots map[int]LogSlot
}

//nolint:gochecknoglobals // lazily parsed embedded tables
var (
	logOnce sync.Once
	logTbl  logTables
)

func loadLogTables() *logTables {
	logOnce.Do(func() {
		logTbl.pages = map[[2]int]string{}
		logTbl.slots = map[int]LogSlot{}

		rows, err := csv.NewReader(strings.NewReader(logPagesCSV)).ReadAll()
		if err != nil {
			panic("logpages.csv: " + err.Error())
		}

		for _, r := range rows[1:] {
			slot, _ := strconv.Atoi(r[0])
			page, _ := strconv.Atoi(r[1])
			logTbl.pages[[2]int{slot, page}] = r[2]
		}

		rows, err = csv.NewReader(strings.NewReader(logSlotsCSV)).ReadAll()
		if err != nil {
			panic("logslots.csv: " + err.Error())
		}

		for _, r := range rows[1:] {
			var v [5]int
			for i := range v {
				v[i], _ = strconv.Atoi(r[i])
			}

			logTbl.slots[v[0]] = LogSlot{Slot: v[0], RewardPageMinusOne: v[1], Act: v[2], Socket: v[3], Picture: v[4]}
		}
	})

	return &logTbl
}

// LogSlotFor is the record slot of the quest with the title qstsa<act>q<index> (act 1..5, index 1..6; 1..3 in Act 4).
func LogSlotFor(act, index int) int {
	base := [6]int{0, 0, 8, 16, 24, 34}
	if act < 1 || act > 5 {
		return 0
	}

	return base[act] + index
}

// LogPlacement is where the log puts the quest with the title number index of an act: the socket of the 3x2 grid (0..5)
// and the number n of the picture a<act>q<n>.dc6. They differ from the title number in Act 1 (Tools of the Trade stands in
// socket 4, the Search for Cain in 2, the Forgotten Tower in 3), Act 3 (the sockets run backwards over the first four
// quests) and Act 4 (the Hellforge stands in socket 1 with picture 2, Terror's End in socket 2 with picture 3).
func LogPlacement(act, index int) (socket, picture int, ok bool) {
	info, found := LogSlotInfo(LogSlotFor(act, index))
	if !found || info.Act != act {
		return 0, 0, false
	}

	first := [6]int{0, 0, 6, 12, 18, 21}

	return info.Socket, info.Picture - first[act] + 1, true
}

// LogPageKey is the string key a page of the quest in a record slot shows ("" for a page the table leaves empty, i.e. the
// "Invalid State" marker).
func LogPageKey(slot, page int) string {
	return loadLogTables().pages[[2]int{slot, page}]
}

// LogSlotInfo returns the placement of a quest slot; ok is false for slots without a log entry (prologues, travel talks).
func LogSlotInfo(slot int) (LogSlot, bool) {
	s, ok := loadLogTables().slots[slot]

	return s, ok
}

// Slots of the record the client rules name directly.
const (
	slotArcane    = 12
	slotTombs     = 14
	slotTemple    = 21
	slotPrison    = 37
	slotBetray    = 38
	slotHellforge = 27
	slotSiege     = 35
	slotRescue    = 36
	slotDen       = 1
	slotFallen    = 25
	slotCain      = 4
)

// The strings the client rules use directly (string ids 0xe8e, 0xe90, 0xe91, 0xe92, 0xe9b, 0x5860).
const (
	keyComplete   = "qstsComplete"
	keyPrevious   = "qstsprevious"
	keyOther      = "qstsother"
	keyNoGame     = "qstsThankYouComeAgain"
	keyNoGameMany = "qstsThankYouComeAgainMulti"
	keyOneLeft    = "qstsa1q140"
	keyRescueN    = "qstsa5q22"
	keyTempleAsk  = "qstsa3q53"
)

// LogCtx is what the client rules read besides the page byte.
type LogCtx struct {
	// Flag reads a bit of the hero's quest record (the record of the game difficulty).
	Flag func(slot, bit int) bool
	// Global reads the game-wide record (UI_GetNpcInteractionQuestRecord2: what any player of the game has done); nil
	// when the game keeps none.
	Global func(slot, bit int) bool
	// Multi is the "joined or hosted a multi-player game" global (0x798698): it picks the wording of "cannot complete".
	Multi bool
	// DenLeft is the number of monsters left in the Den of Evil (0x7b732c, pushed by the server; -1: not known), Soldiers
	// the number of soldiers still to rescue (0x7b7334).
	DenLeft, Soldiers int
}

// LogView is what the log shows for one quest.
type LogView struct {
	Cat  LogCategory
	Page int    // the page the text belongs to (the page byte after the client rules)
	Key  string // string key of the text; "" for none
	// Count is appended to the text (Den of Evil pages 3 and 4: "Monsters remaining: 5") or formatted into its "%d"
	// (qstsa5q22: "Rescue %d more Soldiers"); -1: none.
	Count  int
	Append bool
}

// Quest record bits as the client names them.
const (
	bitRG   = 0
	bitRP   = 1
	bitUpd  = 12
	bitPGD  = 13
	bitCN   = 14
	bitCB   = 15
	nopeInt = -1
)

// ClientLogView is UI_BuildQuestLogEntries for the quest in record slot slot with the server page byte page. The
// category is the icon class of UI_DrawQuestLogPanel.
//
//nolint:gocyclo,funlen // a line by line transcription of the client function
func ClientLogView(slot, page int, c LogCtx) LogView {
	locked := LogView{Cat: LogCatLocked, Page: page, Count: nopeInt}

	if _, ok := LogSlotInfo(slot); !ok {
		return locked
	}

	flag := func(s, b int) bool { return c.Flag != nil && c.Flag(s, b) }
	global := func(s, b int) bool { return c.Global != nil && c.Global(s, b) }

	rg, pgd, rp := flag(slot, bitRG), flag(slot, bitPGD), flag(slot, bitRP)
	show := func(p int, cat LogCategory) LogView {
		return LogView{Cat: cat, Page: p, Key: LogPageKey(slot, p), Count: nopeInt}
	}

	seen := func() LogCategory {
		if flag(slot, bitUpd) {
			return LogCatDone
		}

		return LogCatAnimating
	}

	info, _ := LogSlotInfo(slot)

	reachedTail := false

	switch {
	case slot == slotSiege && rp:
		// the sockets quest: reward pending always shows a page (page byte 0 is read as 4)
		if page == 0 {
			page = 4
		}

		reachedTail = true
	case slot != slotTombs || rg:
		switch {
		case !pgd && rg: // finished in an earlier game (the record of a loaded hero has no bit 13)
			return LogView{Cat: seen(), Page: 11, Key: keyPrevious, Count: nopeInt}
		case pgd && rg:
			return LogView{Cat: seen(), Page: 13, Key: keyComplete, Count: nopeInt}
		case pgd: // goal done, reward not taken: E2C8
			if v, ok := rewardView(slot, info, rp, show); ok {
				return v
			}
		}
	default: // the Seven Tombs, reward not granted
		if pgd {
			if v, ok := rewardView(slot, info, rp, show); ok {
				return v
			}
		} else {
			switch {
			case (flag(slotArcane, bitRG) || flag(slotArcane, bitPGD)) && page == 1:
				return show(7, LogCatActive)
			case flag(slotTombs, 3):
				return show(5, LogCatActive)
			case flag(slotTombs, 4):
				return show(6, LogCatActive)
			}
		}
	}

	if !reachedTail {
		// E3E2: reward pending with the "CB" bit
		if rp && flag(slot, bitCB) {
			p := 10

			switch slot {
			case slotPrison:
				switch {
				case !flag(slotPrison, 8) || flag(slotPrison, 9):
					if !flag(slotPrison, 8) {
						p = 5
					}
				default:
					p = 6
				}
			case slotBetray:
				p = 4
				if flag(slotBetray, 4) {
					p = 5
				}
			}

			v := show(p, LogCatActive)
			if v.Key == "" {
				return locked
			}

			if slot == slotHellforge {
				v.Cat = LogCatDone
			}

			return v
		}

		// the page byte 0: the quest has nothing to show, unless the game-wide record says it ran out
		if page == 0 {
			if !global(slot, bitPGD) {
				if global(slot, bitCB) && !pgd && !rp {
					v := show(0, LogCatActive)
					v.Key = noGameFor(c)

					return v
				}
			} else {
				if slot == slotTemple && !flag(slotTemple, bitRG) && flag(slotTemple, 4) {
					v := show(0, LogCatActive)
					v.Key = keyTempleAsk

					return v
				}

				if !pgd && !rp {
					v := show(0, LogCatActive)
					v.Key = noGameFor(c)

					return v
				}
			}

			return locked
		}
	}

	// the generic tail: page > 0
	if slot == slotDen {
		v := show(page, LogCatActive)
		if page == 3 || page == 4 {
			switch {
			case c.DenLeft < 0: // the count is not known: the plain text
			case c.DenLeft < 2:
				v.Key = keyOneLeft
			default:
				v.Count, v.Append = c.DenLeft, true
			}
		}

		if page == 13 {
			v.Cat = seen()
		}

		return v
	}

	v := show(page, LogCatActive)

	if (!rg && !pgd && !rp && global(slot, bitPGD) && slot != 0x13) || flag(slot, bitCN) {
		v.Key = noGameFor(c)
	} else if v.Key == keyOther && !c.Multi {
		v.Key = keyNoGame
	}

	if slot == slotRescue && v.Key == keyRescueN {
		if c.Soldiers == 0 {
			v.Key = keyNoGame
		} else {
			v.Count = c.Soldiers
		}
	}

	if page == 13 {
		v.Cat = seen()
	}

	return v
}

func noGameFor(c LogCtx) string {
	if c.Multi {
		return keyNoGameMany
	}

	return keyNoGame
}

// rewardView is label E2C8: a quest with a fixed reward page shows it while the reward is pending (the Fallen Angel
// leaves it to the page byte).
func rewardView(slot int, info LogSlot, rp bool, show func(int, LogCategory) LogView) (LogView, bool) {
	if info.RewardPageMinusOne == -1 || !rp || slot == slotFallen {
		return LogView{}, false
	}

	return show(info.RewardPageMinusOne+1, LogCatActive), true
}
