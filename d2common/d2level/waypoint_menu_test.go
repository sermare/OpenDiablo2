package d2level

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func slots(done ...int) func(int) bool {
	return func(s int) bool {
		for _, d := range done {
			if d == s {
				return true
			}
		}

		return false
	}
}

func TestWaypointTabRanges(t *testing.T) {
	// the ranges of the tab table at 0x721820 must cover every waypoint level of their act and nothing else
	for tab := 0; tab < WaypointTabs; tab++ {
		first, last, ok := WaypointTabRange(tab)
		if !ok {
			t.Fatalf("tab %d missing", tab)
		}

		rows, _ := WaypointMenu(tab, 0, func(int) bool { return true })
		want := WaypointBitsOfAct(tab + 1)

		if len(rows) != len(want) {
			t.Errorf("tab %d: %d rows, act has %d waypoints", tab, len(rows), len(want))
		}

		for i, r := range rows {
			if r.Level < first || r.Level > last || r.No != want[i] {
				t.Errorf("tab %d row %d: %+v (want bit %d in %d..%d)", tab, i, r, want[i], first, last)
			}
		}
	}

	if _, _, ok := WaypointTabRange(5); ok {
		t.Error("tab 5 exists")
	}
}

func TestWaypointMenuStates(t *testing.T) {
	var wp d2s.Waypoints

	wp.Set(0, d2s.WPColdPlains, true)
	wp.Set(0, d2s.WPStonyField, true)

	active := func(no int) bool { return wp.Has(0, d2s.Waypoint(no)) }

	rows, sel := WaypointMenu(0, 3, active) // standing in the Cold Plains
	if len(rows) != 9 {
		t.Fatalf("act 1 lists %d rows, want 9", len(rows))
	}

	// the Rogue Encampment is on although the record does not carry it; Cold Plains is current; Stony Field selectable
	if !rows[0].Selectable || rows[0].Level != 1 {
		t.Errorf("row 0 %+v", rows[0])
	}

	if rows[1].Selectable || !rows[1].Current || rows[1].Level != 3 {
		t.Errorf("row 1 %+v", rows[1])
	}

	if !rows[2].Selectable || rows[3].Selectable {
		t.Errorf("rows 2,3 %+v %+v", rows[2], rows[3])
	}

	if sel != 2 {
		t.Errorf("selectable %d, want 2 (Rogue Encampment, Stony Field)", sel)
	}

	// alone in the camp with nothing else: the no-destination title
	_, sel = WaypointMenu(0, 1, func(int) bool { return false })
	if sel != 0 {
		t.Errorf("selectable %d, want 0", sel)
	}
}

func TestSelectWaypointTab(t *testing.T) {
	const a1, a2, a3, a4 = 7, 15, 23, 28

	tests := []struct {
		name string
		act  int
		done []int
		want int
	}{
		{"act 1 always", 0, nil, 0},
		{"act 2 unfinished act 1", 1, nil, 0},
		{"act 2", 1, []int{a1}, 1},
		{"act 3 without act 2 done falls to act 2", 2, []int{a1}, 1},
		{"act 3", 2, []int{a1, a2}, 2},
		{"act 4 falls to act 3", 3, []int{a1, a2}, 2},
		{"act 4", 3, []int{a1, a2, a3}, 3},
		{"act 5 falls to act 4", 4, []int{a1, a2, a3}, 3},
		{"act 5", 4, []int{a1, a2, a3, a4}, 4},
		{"act 5 slot 28 alone does not skip the chain", 4, []int{a4}, 0},
		{"bad index", 9, []int{a1, a2, a3, a4}, 0},
	}

	for _, tc := range tests {
		if got := SelectWaypointTab(tc.act, slots(tc.done...)); got != tc.want {
			t.Errorf("%s: tab %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestTabUnlocked(t *testing.T) {
	done := slots(7, 15, 23, 26)

	for tab, want := range []bool{true, true, true, true, true} {
		if got := TabUnlocked(tab, true, done); got != want {
			t.Errorf("tab %d: %v", tab, got)
		}
	}

	if TabUnlocked(4, false, done) {
		t.Error("act 5 tab in classic")
	}

	if TabUnlocked(1, true, slots()) || TabUnlocked(4, true, slots(7, 15, 23)) {
		t.Error("locked tabs open")
	}

	if TabCount(true) != 5 || TabCount(false) != 4 {
		t.Error("tab counts")
	}
}

func TestValidateTravel(t *testing.T) {
	active := func(no int) bool { return no == 1 }

	base := TravelRequest{ObjectFound: true, PlayerAct: 1, ObjectAct: 1, Target: 3, LevelCount: 137, Active: active}

	tests := []struct {
		name string
		mod  func(*TravelRequest)
		want TravelCode
	}{
		{"ok", func(*TravelRequest) {}, TravelOK},
		{"object missing", func(r *TravelRequest) { r.ObjectFound = false }, TravelNoObject},
		{"other act object", func(r *TravelRequest) { r.ObjectAct = 2 }, TravelNotActive},
		{"far", func(r *TravelRequest) { r.Far = true }, TravelNoObject},
		{"far beats close menu", func(r *TravelRequest) { r.Far, r.Target = true, 0 }, TravelNoObject},
		{"close menu", func(r *TravelRequest) { r.Target = 0 }, TravelOK},
		{"not activated", func(r *TravelRequest) { r.Target = 4 }, TravelNotActive},
		{"level without waypoint", func(r *TravelRequest) { r.Target = 2 }, TravelBadRequest},
		{"level out of range", func(r *TravelRequest) { r.Target = 200 }, TravelBadRequest},
		{"other act target, activated", func(r *TravelRequest) { r.Target = 40; r.Active = func(int) bool { return true } }, TravelOK},
		{"town of act 1 always", func(r *TravelRequest) { r.Target = 1; r.Active = func(int) bool { return false } }, TravelOK},
	}

	for _, tc := range tests {
		r := base
		tc.mod(&r)

		if got := ValidateTravel(r); got != tc.want {
			t.Errorf("%s: %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestTakeWaypointOrder(t *testing.T) {
	if TakeWaypoint(8, -1, TravelOK) != TravelBadRequest {
		t.Error("short packet")
	}

	if TakeWaypoint(9, 9999, TravelOK) != TravelNoObject {
		t.Error("cooldown 9.999 s")
	}

	if TakeWaypoint(9, 10000, TravelOK) != TravelOK || TakeWaypoint(9, -1, TravelNotActive) != TravelNotActive {
		t.Error("pass through")
	}
}

func TestExecuteTravelOK(t *testing.T) {
	act := func(no int) bool { return no == 2 } // bit 2 = level 4, the Stony Field

	tests := []struct {
		name     string
		isWP     bool
		here, to int
		want     bool
	}{
		{"ok", true, 1, 4, true},
		{"not a waypoint object", false, 1, 4, false},
		{"same level", true, 4, 4, false},
		{"zero", true, 1, 0, false},
		{"inactive", true, 1, 5, false},
		{"rogue encampment", true, 4, 1, true},
	}

	for _, tc := range tests {
		if got := ExecuteTravelOK(tc.isWP, tc.here, tc.to, act); got != tc.want {
			t.Errorf("%s: %v", tc.name, got)
		}
	}
}

func TestNormalizeWaypoints(t *testing.T) {
	var wp d2s.Waypoints

	NormalizeWaypoints(&wp)

	for d := 0; d < 3; d++ {
		if !wp.Has(d, d2s.WPRogueEncampment) || wp.Has(d, d2s.WPColdPlains) {
			t.Errorf("difficulty %d: %#x", d, wp[d])
		}
	}
}

func TestNpcWaypoints(t *testing.T) {
	// the slots and levels must agree with the act travel rules that use the same NPCs
	for _, n := range NpcWaypoints {
		r, ok := RuleForNPC(n.Class)
		if !ok {
			t.Fatalf("npc %d has no travel rule", n.Class)
		}

		if ActStartLevel(r.To) != n.Level {
			t.Errorf("npc %d: town %d, rule goes to %d", n.Class, n.Level, ActStartLevel(r.To))
		}

		if _, has := WaypointBit(n.Level); !has {
			t.Errorf("npc %d: level %d has no waypoint", n.Class, n.Level)
		}
	}

	if n, ok := NpcWaypointFor(155); !ok || n.Level != 40 || n.Slot != slotAndarielForTest() {
		t.Errorf("Warriv: %+v", n)
	}

	if _, ok := NpcWaypointFor(175); ok {
		t.Error("the return Warriv row opens no waypoint")
	}
}

// Warriv's menu entry needs the slot of Sisters to the Slaughter's reward: slot 6 (Act I quest 6 is slot 6 + the prologue).
func slotAndarielForTest() int { return 6 }

func TestWaypointReach(t *testing.T) {
	if WaypointReachTiles(0) != 2 || WaypointReachTiles(1) != 4.4 || WaypointReachTiles(6) != 2 {
		t.Fatal("reach by class")
	}

	for _, c := range []struct {
		dx, dy, r int
		far       bool
	}{{10, 10, 10, false}, {11, 0, 10, true}, {-11, 0, 10, true}, {0, -10, 10, false}, {8, 8, 10, false}, {22, 22, 22, false}, {0, 23, 22, true}} {
		if FarFromWaypoint(c.dx, c.dy, c.r) != c.far {
			t.Fatalf("%+v", c)
		}
	}
}
