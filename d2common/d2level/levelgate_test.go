package d2level

import (
	"errors"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func TestCheckLevelGate(t *testing.T) {
	with := func(slot, bit int) *d2s.QuestRecord {
		q := &d2s.QuestRecord{}
		q.Set(slot, bit)

		return q
	}

	rite, _ := d2s.QuestSlot(5, 5)
	summoner, _ := d2s.QuestSlot(2, 5)

	tests := []struct {
		name     string
		from, to int
		via      GateVia
		q        *d2s.QuestRecord
		gated    bool
		slot     int
	}{
		{"summit to keep, no quest", 120, 128, GateWarp, &d2s.QuestRecord{}, true, rite},
		{"summit to keep, nil record", 120, 128, GateWarp, nil, true, rite},
		{"summit to keep, rite done", 120, 128, GateWarp, with(rite, d2s.QuestBitDone), false, 0},
		{"summit to keep, rite reward pending", 120, 128, GateWarp, with(rite, d2s.QuestBitRewardPending), false, 0},
		{"summit to keep, only started", 120, 128, GateWarp, with(rite, d2s.QuestBitProgressFirst), true, rite},
		{"keep 1 to 2 is free", 128, 129, GateWarp, nil, false, 0},
		{"throne to chamber is free", 131, 132, GateWarp, nil, false, 0},
		{"waypoint to keep 2", 118, 129, GateWaypoint, nil, true, rite},
		{"town portal to throne", 109, 131, GatePortal, &d2s.QuestRecord{}, true, rite},
		{"town portal to throne, done", 109, 131, GatePortal, with(rite, d2s.QuestBitDone), false, 0},
		{"portal from inside the keep", 130, 128, GatePortal, nil, false, 0},
		{"town portal to canyon", 40, 46, GatePortal, &d2s.QuestRecord{}, true, summoner},
		{"warp to canyon is not table gated", 45, 46, GateWarp, &d2s.QuestRecord{}, false, 0},
		{"ungated level", 109, 113, GatePortal, nil, false, 0},
	}

	for _, tc := range tests {
		err := CheckLevelGate(tc.from, tc.to, tc.via, true, tc.q)
		if (err != nil) != tc.gated || (err != nil && !errors.Is(err, ErrLevelGated)) {
			t.Errorf("%s: err = %v, gated want %v", tc.name, err, tc.gated)
			continue
		}

		var ge *LevelGateError
		if tc.gated && (!errors.As(err, &ge) || ge.Slot != tc.slot || ge.To != tc.to) {
			t.Errorf("%s: err = %#v, want slot %d", tc.name, err, tc.slot)
		}
	}
}

// With D2_TABLES every level the real Levels.txt flags is gated against its own column, and nothing else is.
func TestLevelGateMatchesLevelsTxt(t *testing.T) {
	lt, _ := loadLevels(t)

	for _, expansion := range []bool{false, true} {
		col := "QuestFlag"
		if expansion {
			col = "QuestFlagEx"
		}

		for id := 1; id <= 136; id++ {
			want := lt.num(id, col)

			if got := LevelQuestFlag(id, expansion); got != want {
				t.Errorf("level %d %s: table %d, Levels.txt %d", id, col, got, want)
			}

			err := CheckLevelGate(0, id, GatePortal, expansion, &d2s.QuestRecord{})
			if (err != nil) != (want != 0) {
				t.Errorf("level %d %s=%d: portal err %v", id, col, want, err)
			}

			if want != 0 {
				q := &d2s.QuestRecord{}
				q.Set(want, d2s.QuestBitDone)

				if err := CheckLevelGate(0, id, GatePortal, expansion, q); err != nil {
					t.Errorf("level %d: still gated with slot %d done: %v", id, want, err)
				}
			}
		}
	}
}
