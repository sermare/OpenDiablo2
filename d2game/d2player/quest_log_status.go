package d2player

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

// QuestLogKey is the key of the status map for the quest at (act 1..5, index
// 1..6); acts 1-3 have six quests, Act 4 has three.
func QuestLogKey(act, index int) int {
	key := (act-1)*d2enum.NormalActQuestsNumber + index - 1
	if act > d2enum.Act4 {
		key -= d2enum.HalfQuestsNumber
	}

	return key
}

// SetOnCompletionSeen registers a callback for the moment the player has seen
// a quest's completion animation in the log (the client then tells the server,
// packet 0x58, which sets the "update quest log" bit 12 of the quest).
func (s *QuestLog) SetOnCompletionSeen(cb func(act, index int)) { s.onSeen = cb }

func (s *QuestLog) markSeen(act, index int) {
	if s.onSeen != nil {
		s.onSeen(act, index)
	}
}

// Status returns the stored status of a quest (see d2enum.QuestStatus; values
// above InProgress are the description page of the quest).
func (s *QuestLog) Status(act, index int) int {
	return s.questStatus[QuestLogKey(act, index)]
}

// SetStatuses stores new quest statuses (keys from QuestLogKey) and refreshes
// the icons and buttons if the panel is loaded. A quest that goes from
// in progress to completed gets the completion animation on the next open.
func (s *QuestLog) SetStatuses(statuses map[int]int) {
	for key, st := range statuses {
		old := s.questStatus[key]
		if st == d2enum.QuestStatusCompleted && old > d2enum.QuestStatusNotStarted {
			st = d2enum.QuestStatusCompleting
		}

		s.questStatus[key] = st
	}

	for act := 1; act <= d2enum.ActsNumber; act++ {
		board := s.quests[act-1]
		if board == nil {
			continue
		}

		for n := range board.icons {
			st := s.questStatus[s.cordsToQuestID(act, n)]

			frame := inProgressIdleFrame
			if s.selectedTab == act-1 && s.selectedQuest == n+1 {
				frame = inProgresFrame
			}

			switch st {
			case d2enum.QuestStatusCompleted:
				frame = completedFrame
			case d2enum.QuestStatusCompleting:
				frame = 0
			case d2enum.QuestStatusNotStarted:
				frame = notStartedFrame
			}

			if board.icons[n] != nil {
				if err := board.icons[n].SetCurrentFrame(frame); err != nil {
					s.Error(err.Error())
				}
			}

			if n < len(board.buttons) {
				board.buttons[n].SetEnabled(st != d2enum.QuestStatusNotStarted)
			}
		}
	}

	if s.questName != nil && s.selectedQuest != d2enum.QuestNone {
		s.setQuestLabel()
	}
}

// Select opens the panel on the tab of an act and the quest of that index, like clicking its socket.
func (s *QuestLog) Select(act, index int) {
	s.Open()
	s.setTab(act - 1)
	s.onQuestClicked(index)
}

// Title returns the quest title of the log (the string qstsa<act>q<index>).
func (s *QuestLog) Title(act, index int) string {
	return s.asset.TranslateString(fmt.Sprintf("qstsa%dq%d", act, index))
}

// DescriptionText returns the description the log shows for a quest (used by
// logs and the autotest); "" when it has none.
func (s *QuestLog) DescriptionText(act, index int) string {
	if t, ok := s.texts[QuestLogKey(act, index)]; ok {
		return s.questText(t)
	}

	st := s.questStatus[QuestLogKey(act, index)]

	switch {
	case st == d2enum.QuestStatusCompleted || st == d2enum.QuestStatusCompleting:
		return s.asset.TranslateString("qstsprevious")
	case st <= d2enum.QuestStatusNotStarted:
		return ""
	}

	key := fmt.Sprintf("qstsa%dq%d%d", act, index, st)
	if text := s.asset.TranslateString(key); text != key {
		return fillCount(key, text)
	}

	return ""
}

// pageCounts are the numbers the pages with a "%d" show ("Rescue %d more Soldiers in the Frigid Highlands",
// "Break the remaining %d seals"): the quest system does not keep the counts yet, so the page shows the starting number
// (three groups of five soldiers; four seals after the first). OD2 STAND-IN: in the exe (UI_BuildQuestLogEntries 0x49e060)
// the "Rescue %d more Soldiers" page (string 0x5860 under header 0x585e) is formatted with DAT_007b7334, a count the server
// sends in a quest info packet (UI_HandleQuestInfoPacket 0x49f3a0, sub-id 0x20), and the page falls back to the generic
// string 0xe91 while that count is 0. The starting numbers here replace the missing server value. The seals page is NOT
// formatted by that function (only Den of Evil's 0xe82 pages 3/4 append DAT_007b732c), so the seal count is unverified.
// The panel showed the raw "%d" before.
var pageCounts = map[string]int{"qstsa5q22": 15, "qstsa4q23": 4}

func fillCount(key, text string) string {
	if !strings.Contains(text, "%d") {
		return text
	}

	return strings.ReplaceAll(text, "%d", strconv.Itoa(pageCounts[key]))
}

// QuestText is what the quest log of the original shows as the description of a quest: the string key of the page
// (d2quest.ClientLogView) and an optional number appended to it ("Monsters remaining: 5") or formatted into its "%d".
type QuestText struct {
	Key    string
	Count  int // < 0: none
	Append bool
}

// SetTexts stores the description texts of the quests (keys from QuestLogKey) and refreshes the label.
func (s *QuestLog) SetTexts(texts map[int]QuestText) {
	for k, t := range texts {
		s.texts[k] = t
	}

	if s.questName != nil && s.selectedQuest != d2enum.QuestNone {
		s.setQuestLabel()
	}
}

func (s *QuestLog) questText(t QuestText) string {
	if t.Key == "" {
		return ""
	}

	text := s.asset.TranslateString(t.Key)
	if text == t.Key {
		return ""
	}

	switch {
	case t.Count >= 0 && t.Append:
		text += strconv.Itoa(t.Count)
	case t.Count >= 0:
		text = strings.ReplaceAll(text, "%d", strconv.Itoa(t.Count))
	}

	return text
}

// refreshIconFrames puts the frame of the quest pictures of the open act: the selected quest with a page shows frame 25.
func (s *QuestLog) refreshIconFrames() {
	if s.selectedTab < 0 || s.selectedTab >= len(s.quests) || s.quests[s.selectedTab] == nil {
		return
	}

	act := s.selectedTab + 1

	for n, icon := range s.quests[s.selectedTab].icons {
		if icon == nil || s.questStatus[s.cordsToQuestID(act, n)] <= d2enum.QuestStatusNotStarted {
			continue
		}

		frame := inProgressIdleFrame
		if s.selectedQuest == n+1 {
			frame = inProgresFrame
		}

		if err := icon.SetCurrentFrame(frame); err != nil {
			s.Error(err.Error())
		}
	}
}
