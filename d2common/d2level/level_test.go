package d2level

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func TestActOfLevel(t *testing.T) {
	tests := []struct{ id, act int }{
		{0, 0}, {1, 1}, {2, 1}, {39, 1}, {40, 2}, {74, 2}, {75, 3}, {102, 3},
		{103, 4}, {108, 4}, {109, 5}, {132, 5}, {136, 5}, {-1, 0},
	}
	for _, tc := range tests {
		if got := ActOfLevel(tc.id); got != tc.act {
			t.Errorf("ActOfLevel(%d) = %d, want %d", tc.id, got, tc.act)
		}
	}
}

func TestActStartAndTown(t *testing.T) {
	want := []int{1, 40, 75, 103, 109}
	for act, lvl := range want {
		if got := ActStartLevel(act + 1); got != lvl {
			t.Errorf("ActStartLevel(%d) = %d, want %d", act+1, got, lvl)
		}

		if !IsTown(lvl) || ActOfLevel(lvl) != act+1 {
			t.Errorf("level %d should be the town of act %d", lvl, act+1)
		}
	}

	if ActStartLevel(0) != 0 || ActStartLevel(6) != 0 || IsTown(2) {
		t.Error("bad act/town handling outside the table")
	}
}

func TestWaypointStartType(t *testing.T) {
	tests := []struct {
		level int
		want  StartType
	}{
		{1, StartSpecial}, {40, StartSpecial}, {109, StartSpecial}, {0x2e, StartSpecial}, {0x4a, StartSpecial},
		{0x85, StartSpecial}, {0x88, StartSpecial}, {3, StartDefault}, {35, StartDefault},
	}
	for _, tc := range tests {
		if got := WaypointStartType(tc.level); got != tc.want {
			t.Errorf("WaypointStartType(%d) = %#x, want %#x", tc.level, got, tc.want)
		}
	}
}

func TestWaypointTable(t *testing.T) {
	seen := map[int]bool{}

	for bit := 0; bit < int(d2s.NumWaypoints); bit++ {
		lvl := WaypointLevel(bit)
		if lvl == 0 || seen[lvl] {
			t.Fatalf("bit %d: bad or duplicate level %d", bit, lvl)
		}

		seen[lvl] = true

		if b, ok := WaypointBit(lvl); !ok || b != bit {
			t.Errorf("WaypointBit(%d) = %d,%v, want %d", lvl, b, ok, bit)
		}

		// the waypoint bit ranges follow the acts, like the d2s package
		if got, want := ActOfLevel(lvl), d2s.Waypoint(bit).Act(); got != want {
			t.Errorf("bit %d level %d: act %d, d2s says %d", bit, lvl, got, want)
		}

		if DefaultLevelNames[lvl] == "" {
			t.Errorf("no default name for level %d", lvl)
		}
	}

	if _, ok := WaypointBit(2); ok {
		t.Error("Blood Moor has no waypoint")
	}

	if WaypointLevel(-1) != 0 || WaypointLevel(39) != 0 {
		t.Error("out of range bits must give 0")
	}

	if got := len(WaypointBitsOfAct(1)); got != 9 {
		t.Errorf("act 1 has %d waypoints, want 9", got)
	}

	if got := len(WaypointBitsOfAct(4)); got != 3 {
		t.Errorf("act 4 has %d waypoints, want 3", got)
	}
}

func TestWaypointList(t *testing.T) {
	var wp d2s.Waypoints

	wp.Set(0, d2s.WPRogueEncampment, true)
	wp.Set(0, d2s.WPColdPlains, true)
	wp.Set(1, d2s.WPStonyField, true) // other difficulty

	canLoad := func(l int) bool { return l == 1 }
	rows := WaypointList(wp, 0, 1, canLoad)

	if len(rows) != 9 {
		t.Fatalf("got %d rows, want 9", len(rows))
	}

	if rows[0].Level != 1 || !rows[0].Active || !rows[0].Loadable || !rows[0].Enabled() {
		t.Errorf("row 0 = %+v", rows[0])
	}

	if rows[1].Level != 3 || !rows[1].Active || rows[1].Loadable || rows[1].Enabled() {
		t.Errorf("row 1 should be active but greyed (not loadable): %+v", rows[1])
	}

	if rows[2].Active {
		t.Errorf("Stony Field is only active in nightmare: %+v", rows[2])
	}

	if all := WaypointList(wp, 0, 1, nil); !all[1].Loadable {
		t.Error("nil canLoad means everything loads")
	}

	if len(WaypointList(wp, 0, 9, nil)) != 0 {
		t.Error("act 9 has no waypoints")
	}
}

func TestValidateWaypoint(t *testing.T) {
	var wp d2s.Waypoints

	wp.Set(0, d2s.WPRogueEncampment, true)
	wp.Set(0, d2s.WPColdPlains, true)

	base := WaypointRequest{PlayerAct: 1, ObjectAct: 1, Distance: 1, Target: 3, Waypoints: wp}

	tests := []struct {
		name string
		mod  func(*WaypointRequest)
		want error
	}{
		{"ok", func(*WaypointRequest) {}, nil},
		{"close menu", func(r *WaypointRequest) { r.Target = 0; r.Distance = 99 }, nil},
		{"wrong act object", func(r *WaypointRequest) { r.ObjectAct = 2 }, ErrWaypointWrongAct},
		{"too far", func(r *WaypointRequest) { r.Distance = 2.5 }, ErrWaypointOutOfRange},
		{"custom range", func(r *WaypointRequest) { r.Distance = 2.5; r.Range = 3 }, nil},
		{"no waypoint at level", func(r *WaypointRequest) { r.Target = 2 }, ErrWaypointInvalidLevel},
		{"other act target", func(r *WaypointRequest) { r.Target = 40 }, ErrWaypointNotInAct},
		{"not activated", func(r *WaypointRequest) { r.Target = 4 }, ErrWaypointNotActive},
		{"wrong difficulty", func(r *WaypointRequest) { r.Diff = 1 }, ErrWaypointNotActive},
	}
	for _, tc := range tests {
		r := base
		tc.mod(&r)

		if got := ValidateWaypoint(r); !errors.Is(got, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestActivateWaypoint(t *testing.T) {
	var wp d2s.Waypoints

	if !ActivateWaypoint(&wp, 2, 5) || !wp.Has(2, d2s.WPDarkWood) {
		t.Fatal("Dark Wood should activate in hell")
	}

	if ActivateWaypoint(&wp, 2, 5) {
		t.Error("second activation must report false")
	}

	if wp.Has(0, d2s.WPDarkWood) {
		t.Error("other difficulties must be untouched")
	}

	if ActivateWaypoint(&wp, 0, 2) {
		t.Error("Blood Moor has no waypoint")
	}
}

func TestCooldown(t *testing.T) {
	var c Cooldown

	if !c.Ready(0, WaypointCooldownSeconds) {
		t.Error("ready before the first change")
	}

	c.Mark(100)

	tests := []struct {
		now, window float64
		want        bool
	}{
		{105, PortalCooldownSeconds, true}, {104.9, PortalCooldownSeconds, false},
		{109.9, WaypointCooldownSeconds, false}, {110, WaypointCooldownSeconds, true},
	}
	for _, tc := range tests {
		if got := c.Ready(tc.now, tc.window); got != tc.want {
			t.Errorf("Ready(%v,%v) = %v, want %v", tc.now, tc.window, got, tc.want)
		}
	}
}

func TestValidatePortal(t *testing.T) {
	has := func(f int) bool { return f == 7 }

	tests := []struct {
		name string
		r    PortalRequest
		want error
	}{
		{"ok", PortalRequest{Dest: 1}, nil},
		{"no dest", PortalRequest{}, ErrPortalNoDest},
		{"owner", PortalRequest{Dest: 1, Restricted: true, Owner: "a", Player: "a"}, nil},
		{"stranger", PortalRequest{Dest: 1, Restricted: true, Owner: "a", Player: "b"}, ErrPortalNotOwner},
		{"party", PortalRequest{Dest: 1, Restricted: true, Owner: "a", Player: "b", InParty: true}, nil},
		{"unrestricted", PortalRequest{Dest: 1, Owner: "a", Player: "b"}, nil},
		{"quest flag set", PortalRequest{Dest: 74, QuestFlag: 7, HasQuestFlag: has}, nil},
		{"quest flag missing", PortalRequest{Dest: 74, QuestFlag: 9, HasQuestFlag: has}, ErrPortalQuestFlag},
		{"quest flag no record", PortalRequest{Dest: 74, QuestFlag: 9}, ErrPortalQuestFlag},
	}
	for _, tc := range tests {
		if got := ValidatePortal(tc.r); !errors.Is(got, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}

	if PortalStateFrames != 0x4b || PortalStateID != 0x66 {
		t.Error("portal state constants changed")
	}
}

func TestLoadActRoundTrip(t *testing.T) {
	p := LoadAct{Act: 3, Seed: 0xDEADBEEF, StartLevel: 75, Aux: 0x01020304}
	b := p.Encode()

	want := []byte{0x03, 0x02, 0xEF, 0xBE, 0xAD, 0xDE, 75, 0, 0x04, 0x03, 0x02, 0x01}
	if !reflect.DeepEqual(b, want) {
		t.Fatalf("encode = % x, want % x", b, want)
	}

	got, err := DecodeLoadAct(b)
	if err != nil || got != p {
		t.Fatalf("decode = %+v, %v", got, err)
	}

	for _, bad := range [][]byte{nil, b[:11], append([]byte{0x04}, b[1:]...), {0x03, 9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}} {
		if _, err := DecodeLoadAct(bad); err == nil {
			t.Errorf("decode of % x should fail", bad)
		}
	}
}

func TestPlanTransition(t *testing.T) {
	p, err := PlanTransition(3, 35, StartDefault, 1, 2)
	if err != nil || p.ActChange || p.LoadAct != nil || p.TownTransition || p.FromAct != 1 || p.ToAct != 1 {
		t.Errorf("same act: %+v, %v", p, err)
	}

	p, err = PlanTransition(35, 1, WaypointStartType(1), 1, 2)
	if err != nil || !p.TownTransition || p.StartType != StartSpecial || p.ActChange {
		t.Errorf("to town: %+v, %v", p, err)
	}

	p, err = PlanTransition(1, 40, StartDefault, 0xABCD, 0x1234)
	if err != nil || !p.ActChange || p.LoadAct == nil {
		t.Fatalf("act change: %+v, %v", p, err)
	}

	if want := (LoadAct{Act: 2, Seed: 0xABCD, StartLevel: 40, Aux: 0x1234}); *p.LoadAct != want {
		t.Errorf("LoadAct = %+v, want %+v", *p.LoadAct, want)
	}

	if _, err := PlanTransition(1, 0, 0, 0, 0); !errors.Is(err, ErrBadLevel) {
		t.Errorf("level 0: %v", err)
	}

	if _, err := PlanTransition(1, 200, 0, 0, 0); !errors.Is(err, ErrBadLevel) {
		t.Errorf("level 200: %v", err)
	}
}

func TestActWorlds(t *testing.T) {
	var a ActWorlds

	if !a.Ensure(2) || a.Ensure(2) || !a.Built(2) || a.Built(1) {
		t.Error("Ensure must build once")
	}

	if a.Ensure(0) || a.Ensure(6) || a.Built(9) {
		t.Error("invalid acts are never built")
	}
}

func TestClassifyObject(t *testing.T) {
	tests := []struct {
		door    bool
		sub, op int
		want    ObjectKind
		name    string
	}{
		{true, 0, 8, ObjectDoor, "door"},
		{true, 0, 29, ObjectDoor, "act2 door"},
		{false, 64, 23, ObjectWaypoint, "waypoint"},
		{false, 4, 15, ObjectPortal, "portal"},
		{false, 1, 1, ObjectShrine, "shrine"},
		{false, 8, 40, ObjectContainer, "chest"},
		{false, 128, 18, ObjectDoor, "jail cell door"},
		{false, 128, 30, ObjectOther, "trap object"},
		{false, 0, 0, ObjectOther, "nothing"},
	}
	for _, tc := range tests {
		if got := ClassifyObject(tc.door, tc.sub, tc.op); got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestLinks(t *testing.T) {
	// every act 1 level that has a link is reachable from the Rogue Encampment
	for lvl := 2; lvl <= 37; lvl++ {
		if !Reachable(RogueEncampment, lvl) {
			t.Errorf("level %d not reachable from the town", lvl)
		}
	}

	// Cold Plains (3) has the cave entrance to level 9 with LvlWarp 4 (Levels.txt)
	if dst, ok := Destination(3, 99); ok {
		t.Errorf("warp 99 must not exist, got %d", dst)
	}

	if l, ok := TileLinkTo(3, 9); !ok || l.Source != SourceLevelsTxt || l.Kind != KindTile {
		t.Errorf("Cold Plains -> Cave 2: %+v %v", l, ok)
	}

	// the verified world neighbours are symmetric edges
	for _, e := range drlgEdges {
		var fwd, back bool

		for _, l := range LinksFrom(e[0]) {
			fwd = fwd || (l.To == e[1] && l.Kind == KindEdge && l.Source == SourceDRLG)
		}

		for _, l := range LinksFrom(e[1]) {
			back = back || (l.To == e[0] && l.Kind == KindEdge)
		}

		if !fwd || !back {
			t.Errorf("edge %v not symmetric", e)
		}
	}

	if _, ok := TileLinkTo(1, 2); ok {
		t.Error("town to Blood Moor is an edge, not a tile")
	}

	if Reachable(RogueEncampment, LutGholein) {
		t.Error("acts are joined by waypoints/NPCs, not by Levels.txt links")
	}

	if len(Links()) != len(levelsTxtLinks)+2*len(drlgEdges)+len(portalLinks) {
		t.Error("Links() size mismatch")
	}
}

func TestParseLevelLinks(t *testing.T) {
	txt := "Name\tId\tVis0\tVis1\tVis2\tVis3\tVis4\tVis5\tVis6\tVis7\tWarp0\tWarp1\tWarp2\tWarp3\tWarp4\tWarp5\tWarp6\tWarp7\r\n" +
		"a\t3\t9\t9\t0\t0\t0\t0\t0\t0\t4\t4\t-1\t-1\t-1\t-1\t-1\t-1\r\n" +
		"b\t200\t1\t0\t0\t0\t0\t0\t0\t0\t1\t0\t0\t0\t0\t0\t0\t0\r\n" +
		"c\t10\t4\t5\t14\t0\t0\t0\t0\t0\t4\t4\t5\t0\t0\t0\t0\t0\n"

	got, err := ParseLevelLinks([]byte(txt))
	if err != nil {
		t.Fatal(err)
	}

	want := [][3]int{{3, 9, 4}, {10, 4, 4}, {10, 5, 4}, {10, 14, 5}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if _, err := ParseLevelLinks([]byte("Name\tId\n")); err == nil {
		t.Error("missing columns must fail")
	}

	if _, err := ParseLevelLinks(nil); err == nil {
		t.Error("empty input must fail")
	}
}

// TestEmbeddedLinksMatchLevelsTxt compares the embedded table with a real
// Levels.txt when D2_TABLES points at the extracted tables.
func TestEmbeddedLinksMatchLevelsTxt(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	raw, err := os.ReadFile(filepath.Join(dir, "drlg", "patch_d2", "Levels.txt"))
	if err != nil {
		t.Skip(err)
	}

	got, err := ParseLevelLinks(raw)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, levelsTxtLinks) {
		t.Errorf("embedded links differ from Levels.txt (%d vs %d entries)", len(levelsTxtLinks), len(got))
	}
}

// TestWaypointColumnMatchesLevelsTxt checks the waypoint table against the
// Waypoint column of a real Levels.txt.
func TestWaypointColumnMatchesLevelsTxt(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	raw, err := os.ReadFile(filepath.Join(dir, "drlg", "patch_d2", "Levels.txt"))
	if err != nil {
		t.Skip(err)
	}

	byBit, err := ParseWaypointColumn(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(byBit) != int(d2s.NumWaypoints) {
		t.Fatalf("Levels.txt has %d waypoint levels, want %d", len(byBit), d2s.NumWaypoints)
	}

	for bit, lvl := range byBit {
		if WaypointLevel(bit) != lvl {
			t.Errorf("bit %d: table says level %d, Levels.txt says %d", bit, WaypointLevel(bit), lvl)
		}
	}
}

func TestTileDestinationCaveEntrances(t *testing.T) {
	tests := []struct {
		name         string
		level, style int
		want         int
		ok           bool
	}{
		{"Blood Moor cave entrance tile is Cave Down", 2, 5, 8, true},
		{"Cold Plains cave entrance", 3, 5, 9, true},
		{"Den of Evil entry stairs go up", 8, 0, 2, true},
		{"Den of Evil has no way down", 8, 4, 0, false},
		{"cave 1 entry stairs go up", 9, 0, 3, true},
		{"cave 1 style 4 goes down", 9, 4, 13, true},
		{"cave 1 of Stony Field: two up exits and one down", 10, 1, 5, true},
		{"cave 1 of Stony Field down", 10, 4, 14, true},
		{"Black Marsh hole", 11, 4, 15, true},
		{"crypt level 1 entry stairs go up", 18, 0, 17, true},
		{"crypt level 2 next stairs go down", 21, 1, 22, true},
		{"crypt level 2 entry stairs go up", 21, 0, 20, true},
		{"jail level 1 next stairs go down", 29, 1, 30, true},
		{"catacombs level 1 next stairs go down", 34, 1, 35, true},
		{"Burial Grounds crypt keeps its LvlWarp id", 17, 6, 18, true},
		{"Burial Grounds mausoleum", 17, 7, 19, true},
		{"treasure cave 2 exit tile (style 1) goes back up", 13, 1, 9, true},
		{"pit level 2 exit tile", 16, 1, 12, true},
		{"catacombs level 4 exit tile", 37, 0, 36, true},
		{"catacombs level 3 next stairs lead to level 4", 36, 1, 37, true},
		{"treasure cave 2 has no other exit", 13, 4, 0, false},
		{"a cottage tile leads nowhere", 2, 8, 0, false},
		{"unknown style", 2, 99, 0, false},
	}

	for _, tc := range tests {
		got, ok := TileDestination(tc.level, tc.style)
		if ok != tc.ok || got != tc.want {
			t.Errorf("%s: got %d %v, want %d %v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}
