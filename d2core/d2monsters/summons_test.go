package d2monsters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func TestPlanSummonTable(t *testing.T) {
	crow := &d2records.MonStatRecord{Key: "crownest1", SpawnKey: "foulcrow1", SpawnOffsetX: 0, SpawnOffsetY: 3,
		SpawnAnimationKey: "NU", AiKey: "FoulCrowNest"}
	gen := &d2records.MonStatRecord{Key: "generic", SpawnKey: "foulcrow1", AiKey: "HighPriest"}

	tests := []struct {
		name    string
		rec     *d2records.SkillRecord
		caster  *d2records.MonStatRecord
		ok      bool
		class   string
		cells   [][2]int
		atCast  bool
		max     int
		frames  int
		wantMod string
	}{
		{"Nest uses the caster spawn column and offset", &d2records.SkillRecord{Srvdofunc: 91}, crow, true,
			"foulcrow1", [][2]int{{0, 3}}, true, selfLimitedSummonCap, 0, "NU"},
		{"MinionSpawner", &d2records.SkillRecord{Srvdofunc: 135}, &d2records.MonStatRecord{SpawnKey: "minion1", SpawnOffsetY: 3, AiKey: "MinionSpawner"}, true,
			"minion1", [][2]int{{0, 3}}, true, selfLimitedSummonCap, 0, ""},
		// VERIFIED in the exe (0x5d0580, 0x5d0850): neither creates a unit
		{"Impregnate creates no unit", &d2records.SkillRecord{Srvdofunc: 133, Summon: "painworm1", Summode: "NU"}, gen, false,
			"", nil, false, 0, 0, ""},
		{"Overseer Whip creates no unit", &d2records.SkillRecord{Srvdofunc: 131, Summon: "suicideminion1", Summode: "S1", Sumumod: 33}, gen, false,
			"", nil, false, 0, 0, ""},
		{"Hydra: three, Param1 lifetime, petmax 99",
			&d2records.SkillRecord{Srvdofunc: 144, Summon: "hydra1", Pettype: "hydra", Petmax: d2calc.Compile("99", d2calc.KindSkill), Param1: 250},
			gen, true, "hydra1", [][2]int{{-1, -1}, {0, 0}, {1, -1}}, false, hydraSummonCap, 250, ""},
		// VERIFIED (0x5b1100, template 0x73a668): four units around the target
		{"DiabPrison none pettype", &d2records.SkillRecord{Srvdofunc: 104, Summon: "boneprison1", Pettype: "none"}, gen, true,
			"boneprison1", [][2]int{{1, 1}, {1, -1}, {-1, -1}, {-1, 1}}, false, prisonSummonCap, 0, ""},
		{"Nest of a generic AI (EvilHole stand-in) is held to the host cap", &d2records.SkillRecord{Srvdofunc: 91}, gen, true,
			"foulcrow1", [][2]int{{0, 0}}, true, nestSummonCap, 0, ""},
		{"a table petmax below the host cap wins", &d2records.SkillRecord{Srvdofunc: 144, Summon: "hydra1", Pettype: "hydra",
			Petmax: d2calc.Compile("2", d2calc.KindSkill)}, gen, true, "hydra1", [][2]int{{-1, -1}, {0, 0}, {1, -1}}, false, 2, 0, ""},
		{"Nest without spawn column summons nothing", &d2records.SkillRecord{Srvdofunc: 91}, &d2records.MonStatRecord{}, false, "", nil, false, 0, 0, ""},
		{"Resurrect is not a summon", &d2records.SkillRecord{Srvdofunc: 97}, crow, false, "", nil, false, 0, 0, ""},
		{"nil row", nil, crow, false, "", nil, false, 0, 0, ""},
	}

	for _, tc := range tests {
		p, ok := PlanSummon(tc.rec, tc.caster)
		if ok != tc.ok {
			t.Errorf("%s: ok=%v", tc.name, ok)

			continue
		}

		if !ok {
			continue
		}

		if p.Class != tc.class || p.AtCaster != tc.atCast || p.MaxAlive != tc.max || p.Frames != tc.frames ||
			len(p.Cells) != len(tc.cells) || (tc.wantMod != "" && p.Mode != tc.wantMod) {
			t.Errorf("%s: got %+v", tc.name, p)

			continue
		}

		for i := range p.Cells {
			if p.Cells[i] != tc.cells[i] {
				t.Errorf("%s: cell %d = %v", tc.name, i, p.Cells[i])
			}
		}
	}
}

func TestSummonRoomLimits(t *testing.T) {
	p := SummonPlan{Cells: make([][2]int, 3), MaxAlive: 5}

	for _, tc := range []struct{ alive, want int }{{0, 3}, {2, 3}, {3, 2}, {5, 0}, {7, 0}} {
		if got := p.summonRoom(tc.alive); got != tc.want {
			t.Errorf("alive %d: room %d, want %d", tc.alive, got, tc.want)
		}
	}

	if (SummonPlan{Cells: make([][2]int, 1)}).summonRoom(1000) != 1 {
		t.Error("no table limit: always the plan size")
	}
}

func TestLiveSummonsCounting(t *testing.T) {
	caster := petUnit(1, 0, 0, "MinionSpawner", "")
	a := petUnit(2, 1, 0, "Minion", "")
	b := petUnit(3, 2, 0, "Minion", "")
	other := petUnit(4, 3, 0, "Minion", "")

	for _, u := range []*unit{caster, a, b, other} {
		u.m.Stat = &d2records.MonStatRecord{Key: "minion1"}
	}

	a.summoner, b.summoner = 1, 1
	other.summoner = 9

	d := testDirector(caster, a, b, other)

	if n := d.liveSummons(1, "Minion1"); n != 2 {
		t.Errorf("two live summons, got %d", n)
	}

	caster.m.Stat = &d2records.MonStatRecord{Key: "minionspawner1", SpawnKey: "minion1"}

	if n := d.CountMinions(caster.b); n != 2 {
		t.Errorf("CountMinions %d", n)
	}

	if n := d.liveSummons(1, "other"); n != 0 {
		t.Errorf("other class counted: %d", n)
	}

	if n := d.liveSummons(0, ""); n != 0 {
		t.Errorf("natural monsters (summoner 0) are nobody's summons: %d", n)
	}
}

// TestRealMonsterSummonSlots walks every monstats skill slot and checks that
// each one whose skills.txt row is a summoning srvdofunc plans the class the
// row (or the caster's spawn column) names, and that every kind in the 1.14b
// tables is reached. Needs $D2_TABLES.
func TestRealMonsterSummonSlots(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	rm, err := d2records.NewRecordManager(d2util.LogLevelNone)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range []struct{ path, file string }{
		{d2resource.MonStats, "monstats.txt"},
		{d2resource.Skills, "skills.txt"},
	} {
		buf, err := os.ReadFile(filepath.Join(root, "monsters", "patch_d2", f.file))
		if err != nil {
			t.Skip(err)
		}

		if err = rm.Load(f.path, d2txt.LoadDataDictionary(buf)); err != nil {
			t.Fatalf("%s: %v", f.file, err)
		}
	}

	byClass := map[string]*d2records.MonStatRecord{}
	for _, st := range rm.Monster.Stats {
		byClass[strings.ToLower(st.Key)] = st
	}

	seen := map[SummonKind]int{}
	slots := 0

	for _, st := range rm.Monster.Stats {
		for _, name := range []string{st.SkillId1, st.SkillId2, st.SkillId3, st.SkillId4, st.SkillId5, st.SkillId6,
			st.SkillId7, st.SkillId8} {
			rec := rm.GetSkillByName(name)
			if rec == nil || SummonKindOf(rec.Srvdofunc) == SummonNone {
				continue
			}

			slots++

			want := rec.Summon
			if want == "" {
				want = st.SpawnKey
			}

			p, ok := PlanSummon(rec, st)
			if want == "" {
				if ok {
					t.Errorf("%s/%s: plan without a class", st.Key, name)
				}

				continue
			}

			if !ok || !strings.EqualFold(p.Class, want) {
				t.Errorf("%s/%s: plan %+v ok=%v, want class %s", st.Key, name, p, ok, want)

				continue
			}

			if byClass[strings.ToLower(p.Class)] == nil {
				t.Errorf("%s/%s: class %q is not in monstats", st.Key, name, p.Class)
			}

			seen[p.Kind]++

			// the host cap: every summoning slot is bounded, and a self-limited AI's own live
			// limit (any difficulty) never reaches the safety net
			if p.MaxAlive <= 0 || p.MaxAlive > selfLimitedSummonCap {
				t.Errorf("%s/%s: unbounded or oversized cap %d", st.Key, name, p.MaxAlive)
			}

			if !selfLimitedSummoners[strings.ToLower(st.AiKey)] && p.MaxAlive > nestSummonCap && p.Kind != SummonHydra {
				t.Errorf("%s/%s (%s): generic AI cap %d", st.Key, name, st.AiKey, p.MaxAlive)
			}

			own := 0

			switch strings.ToLower(st.AiKey) {
			case "foulcrownest", "sarcophagus":
				own = max3(st.AiParameterNormal3, st.AiParameterNightmare3, st.AiParameterHell3)
			case "mosquitonest":
				own = max3(st.AiParameterNormal1, st.AiParameterNightmare1, st.AiParameterHell1)
			case "minionspawner", "vilemother":
				own = max3(st.AiParameterNormal2, st.AiParameterNightmare2, st.AiParameterHell2)
			}

			if own > p.MaxAlive {
				t.Errorf("%s/%s: the AI's own limit %d is above the host cap %d", st.Key, name, own, p.MaxAlive)
			}
		}
	}

	for _, k := range []SummonKind{SummonNest, SummonSpawner, SummonHydra, SummonPrison} {
		if seen[k] == 0 {
			t.Errorf("summon kind %d never planned from the real tables", k)
		}
	}

	t.Logf("%d summoning slots, by kind %v", slots, seen)
}

func max3(a, b, c int) int {
	if b > a {
		a = b
	}

	if c > a {
		a = c
	}

	return a
}

func TestDiabPrisonClassesAndNumbering(t *testing.T) {
	p, ok := PlanSummon(&d2records.SkillRecord{Srvdofunc: 104, Summon: "boneprison1", Pettype: "none"}, &d2records.MonStatRecord{})
	if !ok {
		t.Fatal("DiabPrison not planned")
	}

	want := []string{"boneprison1", "boneprison2", "boneprison3", "boneprison4"}
	if len(p.Classes) != len(want) {
		t.Fatalf("classes %v", p.Classes)
	}

	for i := range want {
		if p.Classes[i] != want[i] {
			t.Errorf("class %d = %q, want %q", i, p.Classes[i], want[i])
		}
	}

	for _, tc := range []struct {
		base string
		n    int
		want []string
	}{
		{"foo9", 2, []string{"foo9", "foo10"}},
		{"noNumber", 3, nil},
		{"", 1, nil},
	} {
		got := numberedClasses(tc.base, tc.n)
		if len(got) != len(tc.want) {
			t.Errorf("numberedClasses(%q,%d) = %v", tc.base, tc.n, got)

			continue
		}

		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("numberedClasses(%q,%d)[%d] = %q", tc.base, tc.n, i, got[i])
			}
		}
	}
}
