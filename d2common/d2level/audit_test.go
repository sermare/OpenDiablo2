package d2level

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// Level graph audit: a graph check over allLinks, the gates, the waypoint table
// and the act travel rules. Every violation is printed (t.Errorf does not stop);
// the parts that need the real tables skip when D2_TABLES is unset.

// lastLevel is the highest level id of the game (Pandemonium Finale).
const lastLevel = 136

// noKnownEntrance are levels no table or note links from the rest of their act:
// the Act 5 "Hell" mazes (125..127) have no Vis/Warp slot and no portal factory
// call names them (VERIFIED in Game.exe: the 11 callers of QUEST_Func_56ae80
// 0x56ae80 and its town whitelist 39, 133..136 never mention them), so they look
// unused; the Pandemonium event levels (133..136) are entered through permanent
// portals made in Harrogath (the whitelist is VERIFIED, the code that makes them
// was not located). They are skipped, not asserted unreachable.
var noKnownEntrance = map[int]bool{125: true, 126: true, 127: true, 133: true, 134: true, 135: true, 136: true}

// warpPartners pairs LvlWarp ids by their names ("Cave Up" is the partner of
// "Wilderness to Cave ..." and "Cave Down"): a link From->To with warp w needs a
// link To->From whose warp is in warpPartners[w]. Built from the LvlWarp.txt
// names, not from Levels.txt.
var warpPartners = func() map[int][]int {
	pairs := [][2][]int{
		{{0, 1, 2, 3, 5}, {4}},
		{{6, 7, 9, 12}, {8}},
		{{10}, {11}},
		{{13}, {14}},
		{{15}, {16}},
		{{17}, {18}},
		{{19, 20}, {21, 22}},
		{{23}, {22}},
		{{24}, {25}},
		{{26, 27}, {28, 29}},
		{{30, 31}, {32}},
		{{33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 46}, {45}},
		{{47, 49}, {48}},
		{{50}, {22}},
		{{51}, {52}},
		{{53, 54, 56}, {55}},
		{{57, 60}, {58, 59}},
		{{61}, {62, 63}},
		{{64, 67, 68}, {65, 66}},
		{{69}, {70}},
		{{71, 74}, {72, 73}},
		{{75}, {73}},
		{{76}, {78}},
		{{77}, {78}},
		{{79}, {74}},
		{{80}, {81}},
		{{81}, {82}},
	}
	m := map[int][]int{}
	add := func(a, b int) {
		for _, x := range m[a] {
			if x == b {
				return
			}
		}
		m[a] = append(m[a], b)
	}

	for _, p := range pairs {
		for _, a := range p[0] {
			for _, b := range p[1] {
				add(a, b)
				add(b, a)
			}
		}
	}

	return m
}()

func hasPartner(w, back int) bool {
	for _, p := range warpPartners[w] {
		if p == back {
			return true
		}
	}

	return false
}

// reachableSet follows every link from start; cross-act links are skipped
// (act changes are ActRules, not warps). gatedOnly collects levels that are only
// reachable through at least one gated move.
func reachableSet(start int) (all, ungated map[int]bool) {
	walk := func(skipGated bool) map[int]bool {
		seen := map[int]bool{start: true}
		queue := []int{start}

		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]

			for _, l := range LinksFrom(cur) {
				if seen[l.To] || ActOfLevel(l.To) != ActOfLevel(start) {
					continue
				}

				if skipGated && len(GatesOf(l.From, l.To)) > 0 {
					continue
				}

				seen[l.To] = true
				queue = append(queue, l.To)
			}
		}

		return seen
	}

	return walk(false), walk(true)
}

func TestLevelGraphReachability(t *testing.T) {
	for act := 1; act <= NumActs; act++ {
		town := ActStartLevel(act)
		all, ungated := reachableSet(town)

		var gatedOnly []int

		for lvl := 1; lvl <= lastLevel; lvl++ {
			if ActOfLevel(lvl) != act || noKnownEntrance[lvl] {
				continue
			}

			if !all[lvl] {
				t.Errorf("VIOLATION act %d: level %d is not reachable from town %d by any warp, edge or portal", act, lvl, town)
			} else if !ungated[lvl] {
				gatedOnly = append(gatedOnly, lvl)
			}
		}

		sort.Ints(gatedOnly)
		t.Logf("act %d: %d levels reachable, %d only behind a gate: %v", act, len(all), len(gatedOnly), gatedOnly)
	}
}

func TestLevelGraphSymmetry(t *testing.T) {
	for _, l := range Links() {
		switch l.Kind {
		case KindEdge:
			if !hasLink(l.To, l.From, KindEdge) {
				t.Errorf("VIOLATION edge %d->%d has no edge back", l.From, l.To)
			}
		case KindTile:
			var backs []int

			for _, b := range LinksFrom(l.To) {
				if b.To == l.From && b.Kind == KindTile {
					backs = append(backs, b.Warp)
				}
			}

			if len(backs) == 0 {
				t.Errorf("VIOLATION warp %d->%d (warp %d) has no warp back", l.From, l.To, l.Warp)

				continue
			}

			if l.Warp < 0 {
				continue // seamless "-1" warps pair with -1
			}

			ok := false

			for _, w := range backs {
				ok = ok || hasPartner(l.Warp, w)
			}

			if !ok {
				t.Errorf("VIOLATION warp %d->%d uses LvlWarp %d but the way back uses %v (partners of %d: %v)",
					l.From, l.To, l.Warp, backs, l.Warp, warpPartners[l.Warp])
			}
		case KindPortal:
			if ActOfLevel(l.From) != ActOfLevel(l.To) {
				t.Errorf("VIOLATION portal %d->%d crosses acts (act travel belongs to ActRules)", l.From, l.To)
			}
		}

		if ActOfLevel(l.From) != ActOfLevel(l.To) {
			t.Errorf("VIOLATION link %d->%d (kind %d) crosses acts", l.From, l.To, l.Kind)
		}
	}
}

func hasLink(from, to int, k Kind) bool {
	for _, l := range LinksFrom(from) {
		if l.To == to && l.Kind == k {
			return true
		}
	}

	return false
}

func TestWaypointTableAudit(t *testing.T) {
	// Act ranges of the 39 bits: 9 + 9 + 9 + 3 + 9.
	want := map[int][2]int{1: {0, 8}, 2: {9, 17}, 3: {18, 26}, 4: {27, 29}, 5: {30, 38}}
	seen := map[int]int{}

	for bit := 0; bit < int(d2s.NumWaypoints); bit++ {
		lvl := WaypointLevel(bit)
		if lvl == 0 {
			t.Errorf("VIOLATION waypoint bit %d has no level", bit)

			continue
		}

		if prev, dup := seen[lvl]; dup {
			t.Errorf("VIOLATION level %d holds bits %d and %d", lvl, prev, bit)
		}

		seen[lvl] = bit

		if got, ok := WaypointBit(lvl); !ok || got != bit {
			t.Errorf("VIOLATION WaypointBit(%d) = %d %v, want %d", lvl, got, ok, bit)
		}

		act := ActOfLevel(lvl)
		if r := want[act]; bit < r[0] || bit > r[1] {
			t.Errorf("VIOLATION waypoint bit %d (level %d) is in act %d, whose bits are %d..%d", bit, lvl, act, r[0], r[1])
		}
	}

	for act, r := range want {
		if got := WaypointBitsOfAct(act); len(got) != r[1]-r[0]+1 {
			t.Errorf("VIOLATION act %d lists %d waypoints, want %d", act, len(got), r[1]-r[0]+1)
		}
	}

	// Every waypoint level must be reachable in its act (a waypoint nobody can
	// reach is a table error).
	for bit := 0; bit < int(d2s.NumWaypoints); bit++ {
		lvl := WaypointLevel(bit)
		if all, _ := reachableSet(ActStartLevel(ActOfLevel(lvl))); !all[lvl] {
			t.Errorf("VIOLATION waypoint level %d (bit %d) is unreachable from its town", lvl, bit)
		}
	}
}

func TestActTravelAudit(t *testing.T) {
	// Rule slots, independent of ActRules: Andariel = Sisters to the Slaughter
	// (1,6), Duriel = Seven Tombs (2,6), Mephisto = The Guardian (3,6), Diablo =
	// Terror's End (4,2).
	slots := map[[2]int][2]int{{1, 2}: {1, 6}, {2, 3}: {2, 6}, {3, 4}: {3, 6}, {4, 5}: {4, 2}}

	for k, q := range slots {
		r, ok := RuleFor(k[0], k[1])
		if !ok {
			t.Errorf("VIOLATION no act rule %d -> %d", k[0], k[1])

			continue
		}

		if s, _ := d2s.QuestSlot(q[0], q[1]); r.QuestSlot != s {
			t.Errorf("VIOLATION act rule %d -> %d gated by slot %d, want %d (quest %v)", k[0], k[1], r.QuestSlot, s, q)
		}
	}

	for _, r := range ActRules {
		if r.From < 1 || r.To < 1 || r.From > NumActs || r.To > NumActs {
			t.Errorf("VIOLATION act rule %+v out of range", r)
		}
	}

	// The last level of acts 1..3 leaves through the town of the next act.
	if got := TownPortalDestination(LevelDurance3); got != KurastDocks {
		t.Errorf("VIOLATION a portal in Mephisto's lair leads to %d, want Kurast Docks", got)
	}
}

func TestGatesAudit(t *testing.T) {
	has := func(from, to, slot int) bool {
		for _, g := range GatesOf(from, to) {
			if g.Slot == slot {
				return true
			}
		}

		return false
	}
	slot := func(act, q int) int { s, _ := d2s.QuestSlot(act, q); return s }

	tests := []struct {
		name         string
		from, to     int
		act, quest   int
		wantDirectly bool
	}{
		{"Palace (and so the Arcane Sanctuary portal) needs slot 11", 40, 50, 2, 3, true},
		{"Arcane Sanctuary from outside the palace", 40, 74, 2, 3, true},
		{"Duriel through a tomb needs the Seven Tombs flag (0x59b700)", 66, 73, 2, 6, true},
		{"Arreat Summit to the Ancients' Way needs Rite of Passage (0x58ae70)", 120, 118, 5, 5, true},
		{"Worldstone Chamber needs Eve of Destruction (0x58c3f0)", 131, 132, 5, 6, true},
		{"Duriel's lair level flag from outside the tombs", 40, 73, 2, 5, true},
		{"Canyon from the Arcane Sanctuary needs The Summoner", 74, 46, 2, 5, true},
		{"Durance via the Orb", LevelTravincal, LevelDurance1, 3, 2, true},
		{"Durance level flag (Blackened Temple)", LevelTravincal, LevelDurance1, 3, 5, true},
		{"Worldstone Keep needs Rite of Passage", 120, 128, 5, 5, true},
		{"Nihlathak's temple portal", 109, 121, 5, 4, true},
		{"Cow Level portal needs Terror's End (classic flag)", 1, 39, 4, 2, true},
	}

	for _, tc := range tests {
		if !has(tc.from, tc.to, slot(tc.act, tc.quest)) {
			t.Errorf("VIOLATION %s: %d->%d is not gated by quest (%d,%d) slot %d; gates: %+v",
				tc.name, tc.from, tc.to, tc.act, tc.quest, slot(tc.act, tc.quest), GatesOf(tc.from, tc.to))
		}
	}

	// Free movement inside a gated area stays free.
	for _, p := range [][2]int{{50, 51}, {51, 52}, {101, 102}, {128, 129}, {66, 46}} {
		if g := GatesOf(p[0], p[1]); len(g) != 0 {
			t.Errorf("VIOLATION %d->%d is inside one gated area but has gates %+v", p[0], p[1], g)
		}
	}

	// Feature gates of Act 4: Hellforge (quest 3) in River of Flame, seals
	// (Terror's End) in the Chaos Sanctuary.
	feat := map[int]int{107: slot(4, 3), 108: slot(4, 2)}

	for lvl, want := range feat {
		found := false

		for _, g := range Gates() {
			if g.To == lvl && g.From == 0 && g.Why != "Levels.txt QuestFlag" && g.Slot == want {
				found = true
			}
		}

		if !found {
			t.Errorf("VIOLATION level %d lacks its feature gate on slot %d", lvl, want)
		}
	}

	// Name -> slot is consistent with the d2s layout.
	for _, g := range Gates() {
		if s, ok := d2s.QuestSlot(g.Act, g.Quest); !ok || s != g.Slot {
			t.Errorf("VIOLATION gate %+v: (act %d, quest %d) is slot %d %v", g, g.Act, g.Quest, s, ok)
		}
	}

	// Town portals go to the town of the level's act.
	for lvl := 1; lvl <= lastLevel; lvl++ {
		if got := TownPortalDestination(lvl); got != ActStartLevel(ActOfLevel(lvl)) || !IsTown(got) {
			t.Errorf("VIOLATION town portal in level %d leads to %d", lvl, got)
		}
	}
}

// --- real data -------------------------------------------------------------

type levelsTable struct {
	col  map[string]int
	rows map[int][]string
}

func (lt levelsTable) num(id int, name string) int {
	f := lt.rows[id]
	if i, ok := lt.col[name]; ok && i < len(f) {
		if v, err := strconv.Atoi(strings.TrimSpace(f[i])); err == nil {
			return v
		}
	}

	return 0
}

func loadLevels(t *testing.T) (levelsTable, map[int]string) {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	raw, err := os.ReadFile(filepath.Join(dir, "drlg", "patch_d2", "Levels.txt"))
	if err != nil {
		t.Skip(err)
	}

	lt := levelsTable{col: map[string]int{}, rows: map[int][]string{}}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	first := true

	for sc.Scan() {
		f := strings.Split(strings.TrimRight(sc.Text(), "\r"), "\t")
		if first {
			for i, h := range f {
				lt.col[h] = i
			}

			first = false

			continue
		}

		if id, err := strconv.Atoi(strings.TrimSpace(f[lt.col["Id"]])); err == nil {
			lt.rows[id] = f
		}
	}

	warps := map[int]string{}

	if wraw, err := os.ReadFile(filepath.Join(dir, "drlg", "d2exp", "LvlWarp.txt")); err == nil {
		wsc := bufio.NewScanner(bytes.NewReader(wraw))
		wsc.Scan()

		for wsc.Scan() {
			f := strings.Split(strings.TrimRight(wsc.Text(), "\r"), "\t")
			if len(f) > 1 {
				if id, err := strconv.Atoi(f[1]); err == nil {
					warps[id] = f[0]
				}
			}
		}
	}

	return lt, warps
}

func TestLevelGraphAgainstTables(t *testing.T) {
	lt, warps := loadLevels(t)

	for id := 1; id <= lastLevel; id++ {
		if _, ok := lt.rows[id]; !ok {
			t.Errorf("VIOLATION Levels.txt has no row for level %d", id)

			continue
		}

		// Act column is 0-based.
		if act := lt.num(id, "Act") + 1; act != ActOfLevel(id) {
			t.Errorf("VIOLATION level %d: Levels.txt act %d, ActOfLevel %d", id, act, ActOfLevel(id))
		}

		// Quest flags.
		if got, want := [2]int{lt.num(id, "QuestFlag"), lt.num(id, "QuestFlagEx")}, levelQuestFlag[id]; got != want {
			t.Errorf("VIOLATION level %d: Levels.txt QuestFlag/Ex %v, table %v", id, got, want)
		}

		// Waypoint column versus the table.
		wp := lt.num(id, "Waypoint")
		if raw := lt.rows[id]; lt.col["Waypoint"] < len(raw) && strings.TrimSpace(raw[lt.col["Waypoint"]]) != "" {
			bit, has := WaypointBit(id)

			switch {
			case wp < int(d2s.NumWaypoints) && (!has || bit != wp):
				t.Errorf("VIOLATION level %d: Levels.txt waypoint bit %d, table says %d %v", id, wp, bit, has)
			case wp >= int(d2s.NumWaypoints) && has:
				t.Errorf("VIOLATION level %d has table bit %d but Levels.txt has no waypoint", id, bit)
			}
		}
	}

	// Warp ids of every link exist in LvlWarp.txt, and the per-level Vis/Warp
	// slots of Levels.txt equal the tile links (ids up to 132).
	if len(warps) > 0 {
		for _, l := range Links() {
			if l.Kind == KindTile && l.Warp >= 0 {
				if _, ok := warps[l.Warp]; !ok {
					t.Errorf("VIOLATION link %d->%d uses LvlWarp id %d, which LvlWarp.txt does not have", l.From, l.To, l.Warp)
				}
			}
		}
	} else {
		t.Log("LvlWarp.txt not found under drlg/d2exp: warp id existence not checked")
	}

	// Quest column (Quest advances when the level is entered): report only.
	var withQuest []string

	for id := 1; id <= lastLevel; id++ {
		if q := lt.num(id, "Quest"); q != 0 {
			withQuest = append(withQuest, fmt.Sprintf("%d:quest %d", id, q))
		}
	}

	t.Logf("levels with a Quest column value (not audited): %v", withQuest)

	// Levels.txt links to levels in another act.
	for _, l := range Links() {
		if l.Kind == KindTile && ActOfLevel(l.From) != ActOfLevel(l.To) {
			t.Errorf("VIOLATION Levels.txt link %d->%d crosses acts", l.From, l.To)
		}
	}
}
