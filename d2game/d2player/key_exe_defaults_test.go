package d2player

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

func TestExeDefaultKeysShape(t *testing.T) {
	seen := map[uint32]bool{}

	for _, k := range ExeDefaultKeys {
		if seen[k.Command] || k.Command > 56 {
			t.Errorf("command %d repeated or out of range", k.Command)
		}

		seen[k.Command] = true
	}

	if len(seen) != 57 {
		t.Fatalf("%d commands, want 57", len(seen))
	}

	// every key of the configuration panel belongs to a real command
	for _, g := range ExeKeyConfigGroups {
		for _, c := range g {
			if !seen[c] {
				t.Errorf("config group names unknown command %d", c)
			}
		}
	}
}

// the keys the task names, straight from the exe table
func TestExeDefaultKeysNamed(t *testing.T) {
	by := map[uint32]ExeDefaultKey{}
	for _, k := range ExeDefaultKeys {
		by[k.Command] = k
	}

	want := map[uint32]struct{ p, s uint16 }{
		0: {'A', 'C'}, 1: {'B', 'I'}, 2: {'P', exeUnbound}, 4: {'Q', exeUnbound}, 7: {0x09, 0x100},
		12: {'T', exeUnbound}, 14: {0x70, exeUnbound}, 21: {0x77, exeUnbound}, 23: {'1', exeUnbound},
		26: {'4', exeUnbound}, 34: {0x11, exeUnbound}, 35: {'R', 0x102}, 36: {0x10, exeUnbound},
		37: {0x12, 0x101}, 56: {0x1b, exeUnbound},
	}
	for c, w := range want {
		if by[c].Primary != w.p || by[c].Secondary != w.s {
			t.Errorf("command %d = %+v", c, by[c])
		}
	}
}

// knownKeyDifferences are the places where the Go defaults and the exe table
// disagree on purpose or for lack of an input source: the three secondaries are
// mouse buttons 3, 4 and 5, which the input manager does not report as keys
// (so a binding there would never fire), and F9..F12 are bound in the exe to
// commands 8..11 (the automap group, UNVERIFIED which is which) while Go binds
// Home, F11 and F12 to its own automap events.
var knownKeyDifferences = map[d2enum.GameEvent]string{
	d2enum.ToggleAutomap:        "secondary mouse button 3",
	d2enum.ToggleRunWalk:        "secondary mouse button 5",
	d2enum.HoldShowGroundItems:  "secondary mouse button 4",
	d2enum.TogglePartyOnAutomap: "not in the exe table by this name",
	// OD2 layout asked for by the owner: number keys pick skills, Q drinks the potions in order, L is the quest log
	d2enum.UseSkill1: "OD2 number key", d2enum.UseSkill2: "OD2 number key", d2enum.UseSkill3: "OD2 number key",
	d2enum.UseSkill4: "OD2 number key", d2enum.UseBeltSlot1: "OD2: Q drinks", d2enum.UseBeltSlot2: "OD2: Q drinks",
	d2enum.UseBeltSlot3: "OD2: Q drinks", d2enum.UseBeltSlot4: "OD2: Q drinks", d2enum.ToggleQuestLog: "OD2: L (Q drinks)",
}

func TestGoDefaultsMatchExeTable(t *testing.T) {
	km := newTestKeyMap()

	for _, row := range ExeDefaultKeys {
		ev, ok := realCommandEvents[row.Command]
		if !ok || row.Command == 56 { // 56 is Escape, handled by the screen itself (SetPrimaryBinding refuses it)
			continue
		}

		got := km.GetKeysForGameEvent(ev)
		if got == nil {
			t.Errorf("command %d (event %d) has no Go binding", row.Command, ev)

			continue
		}

		wantP, okP := vkToKey(row.Primary)
		if !okP {
			wantP = -1
		}

		wantS, okS := vkToKey(row.Secondary)
		if !okS {
			wantS = -1
		}

		if _, known := knownKeyDifferences[ev]; got.Primary != wantP && !known {
			t.Errorf("command %d primary: Go %d, exe %d", row.Command, got.Primary, wantP)
		}

		if got.Secondary != wantS {
			if _, known := knownKeyDifferences[ev]; !known {
				t.Errorf("command %d secondary: Go %d, exe %d", row.Command, got.Secondary, wantS)
			}
		}
	}
}
