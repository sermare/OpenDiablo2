package d2level

import (
	"errors"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func TestCheckActThreeWarp(t *testing.T) {
	with := func(slot, bit int) *d2s.QuestRecord {
		q := &d2s.QuestRecord{}
		q.Set(slot, bit)

		return q
	}

	tests := []struct {
		name     string
		from, to int
		q        *d2s.QuestRecord
		want     error
	}{
		{"sealed without a record", LevelTravincal, LevelDurance1, nil, ErrDuranceSealed},
		{"sealed with a fresh record", LevelTravincal, LevelDurance1, &d2s.QuestRecord{}, ErrDuranceSealed},
		{"sealed when only the goal bit is set", LevelTravincal, LevelDurance1, with(slotKhalim, d2s.QuestBitProgressFirst), ErrDuranceSealed},
		{"open after the orb (reward pending)", LevelTravincal, LevelDurance1, with(slotKhalim, d2s.QuestBitRewardPending), nil},
		{"open after Cain's last line (done)", LevelTravincal, LevelDurance1, with(slotKhalim, d2s.QuestBitDone), nil},
		{"back up is never gated", LevelDurance1, LevelTravincal, nil, nil},
		{"durance 1 to 2", LevelDurance1, LevelDurance2, nil, nil},
		{"durance 2 to 3 (lair)", LevelDurance2, LevelDurance3, nil, nil},
		{"unrelated link", 76, 77, nil, nil},
	}

	for _, tc := range tests {
		if err := CheckActThreeWarp(tc.from, tc.to, tc.q); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", tc.name, err, tc.want)
		}
	}
}

func TestMephistoPortalOpen(t *testing.T) {
	slot := mustSlot(3, 6)
	done, pending := &d2s.QuestRecord{}, &d2s.QuestRecord{}
	done.Set(slot, d2s.QuestBitDone)
	pending.Set(slot, d2s.QuestBitRewardPending)

	for name, tc := range map[string]struct {
		q    *d2s.QuestRecord
		want bool
	}{"nil": {nil, false}, "alive": {&d2s.QuestRecord{}, false}, "done": {done, true}, "pending": {pending, true}} {
		if got := MephistoPortalOpen(tc.q); got != tc.want {
			t.Errorf("%s: %v, want %v", name, got, tc.want)
		}
	}
}

// The Act 3 chain must be linked in Levels.txt data: Travincal -> Durance 1 -> 2 -> 3 and back.
func TestDuranceLinksExist(t *testing.T) {
	for _, p := range [][2]int{{83, 100}, {100, 101}, {101, 102}, {100, 83}, {101, 100}, {102, 101}} {
		_, found := TileLinkTo(p[0], p[1])

		if !found {
			t.Errorf("no link %d -> %d", p[0], p[1])
		}
	}
}
