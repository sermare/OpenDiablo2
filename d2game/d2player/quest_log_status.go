package d2player

import (
	"fmt"

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

			frame := inProgresFrame

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
	st := s.questStatus[QuestLogKey(act, index)]

	switch {
	case st == d2enum.QuestStatusCompleted || st == d2enum.QuestStatusCompleting:
		return s.asset.TranslateString("qstsprevious")
	case st <= d2enum.QuestStatusNotStarted:
		return ""
	}

	key := fmt.Sprintf("qstsa%dq%d%d", act, index, st)
	if text := s.asset.TranslateString(key); text != key {
		return text
	}

	return ""
}
