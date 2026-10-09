package d2animspeed

import (
	"os"
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2animdata"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2herostats"
)

func TestDiminish(t *testing.T) {
	for _, tc := range []struct{ p, k, want int }{
		{0, KStandard, 0}, {20, KStandard, 17}, {60, KStandard, 40}, {120, KStandard, 60},
		{200, KStandard, 75}, {600, KStandard, 100},
		{150, KRunWalk, 75}, {50, KRunWalk, 37}, {-50, KStandard, -85},
		{-120, KStandard, -120}, // k+p == 0 must not divide by zero
	} {
		if got := Diminish(tc.p, tc.k); got != tc.want {
			t.Errorf("Diminish(%d,%d)=%d want %d", tc.p, tc.k, got, tc.want)
		}
	}
}

func TestRates(t *testing.T) {
	for _, tc := range []struct {
		name      string
		got, want int
	}{
		{"hit none", HitRate(256, 0), 128},
		{"hit 86 (eff 50)", HitRate(256, 86), 256},
		{"cast none", CastRate(256, 0), 256},
		{"cast capped at 175", CastRate(256, 100000), 256 * 175 / 100},
		{"block none", BlockRate(256, 0, false), 128},
		{"block state 0x65", BlockRate(256, 0, true), 256},
		{"block min 1", BlockRate(1, 0, false), 1},
		{"attack none", AttackRate(256, AttackBasePct, 0, 0, false), 256},
		{"attack floor 15", AttackRate(256, 0, 0, 0, false), 256 * 15 / 100},
		{"attack cap 175", AttackRate(256, AttackBasePct, 100000, 0, false), 256 * 175 / 100},
		{"attack kick -30", AttackRate(256, AttackBasePct, 0, 0, true), 256 * 70 / 100},
		{"attack ias 60", AttackRate(256, AttackBasePct, 60, 0, false), 256 * 140 / 100},
		{"walk none", WalkRate(256, 0, 0), 256},
		{"walk frw 150", WalkRate(256, 0, 150), 256 * 175 / 100},
		{"walk floor 25", WalkRate(256, -1000, 0), 256 * 25 / 100},
		{"other clamp lo", OtherRate(256, 0), 256 * 15 / 100},
		{"other clamp hi", OtherRate(256, 999), 256 * 175 / 100},
		{"rate cap", HitRate(0x7fff, 1000), MaxRate},
		{"zero base", HitRate(0, 10), 0},
		{"velocity", WalkVelocity(6<<8, 0, 150), (6 << 8) * 175 / 100},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: got %d want %d", tc.name, tc.got, tc.want)
		}
	}
}

func TestTicks(t *testing.T) {
	if _, err := Ticks(6, 0); err == nil {
		t.Error("expected error for zero rate")
	}

	if got, _ := Ticks(6, 256); got != 6 {
		t.Errorf("got %d", got)
	}

	if got, _ := ActionFrames(6, 128); got != 11 {
		t.Errorf("amazon hit recovery with no FHR: got %d want 11", got)
	}
}

func framesFn(kind d2herostats.Kind, m ClassModes) func(int) int {
	switch kind {
	case d2herostats.FHR:
		return func(p int) int { f, _ := ActionFrames(m.Hit.Frames, HitRate(m.Hit.Speed, p)); return f }
	case d2herostats.FBR:
		return func(p int) int { f, _ := ActionFrames(m.Block.Frames, BlockRate(m.Block.Speed, p, false)); return f }
	default:
		return func(p int) int { f, _ := ActionFrames(m.Cast.Frames, CastRate(m.Cast.Speed, p)); return f }
	}
}

// TestReproducesHeroStatsTables checks that the exe-derived rule plus the real
// AnimData frame counts reproduce the community breakpoint tables of
// d2herostats. The listed rows are the ones that agree.
func TestReproducesHeroStatsTables(t *testing.T) {
	agree := map[string]bool{
		"Amazon/FHR": true, "Sorceress/FHR": true, "Paladin/FHR": true, "Assassin/FHR": true,
		"Amazon/FCR": true, "Sorceress/FCR": true, "Necromancer/FCR": true, "Paladin/FCR": true,
		"Barbarian/FCR": true, "Druid/FCR": true, "Assassin/FCR": true,
		"Amazon/FBR": true, "Barbarian/FBR": true, "Assassin/FBR": true,
	}
	seen := 0

	for _, e := range d2herostats.Tables {
		key := e.Class + "/" + string(e.Kind)
		if !agree[key] || e.Note != "" {
			continue
		}

		seen++

		got := Breakpoints(framesFn(e.Kind, Classes[e.Class]), len(e.Table), 5000)
		if !reflect.DeepEqual(got, e.Table) {
			t.Errorf("%s: got %v want %v", key, got, e.Table)
		}
	}

	if seen != len(agree) {
		t.Errorf("checked %d rows, expected %d", seen, len(agree))
	}
}

// TestHeroStatsDiscrepancies pins the rows where the d2herostats literal
// community tables disagree with the exe rule applied to the real AnimData
// (these look like wrong rows in d2herostats; not edited here). The derived
// values are the community tables for the same class and kind.
func TestHeroStatsDiscrepancies(t *testing.T) {
	for _, tc := range []struct {
		class string
		kind  d2herostats.Kind
		want  []int
	}{
		{"Necromancer", d2herostats.FHR, []int{0, 5, 10, 16, 26, 39, 56, 86, 152, 377}},
		{"Barbarian", d2herostats.FHR, []int{0, 7, 15, 27, 48, 86, 200}},
		{"Sorceress", d2herostats.FBR, []int{0, 7, 15, 27, 48, 86, 200}},
		{"Necromancer", d2herostats.FBR, []int{0, 6, 13, 20, 32, 52, 86, 174, 600}},
	} {
		got := Breakpoints(framesFn(tc.kind, Classes[tc.class]), len(tc.want), 5000)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s %s: got %v want %v", tc.class, tc.kind, got, tc.want)
		}

		e, _ := d2herostats.Lookup(tc.class, tc.kind)
		if reflect.DeepEqual(e.Table, tc.want) {
			t.Errorf("%s %s: d2herostats now agrees; move the row to the agreeing list", tc.class, tc.kind)
		}
	}
}

// TestNoModifierRates is the regression baseline: with no speed modifiers
// walk, run, attack and cast play at the AnimData speed (what the engine does
// today); hit and block recovery play at half of it (the engine has no such
// animations yet).
func TestNoModifierRates(t *testing.T) {
	for name, c := range Classes {
		for label, m := range map[string]Mode{"walk": c.Walk, "run": c.Run, "attack": c.Attack1, "cast": c.Cast} {
			var got int

			switch label {
			case "walk", "run":
				got = WalkRate(m.Speed, 0, 0)
			case "attack":
				got = AttackRate(m.Speed, AttackBasePct, 0, 0, false)
			default:
				got = CastRate(m.Speed, 0)
			}

			if got != m.Speed {
				t.Errorf("%s %s: rate %d want %d", name, label, got, m.Speed)
			}
		}

		if got := HitRate(c.Hit.Speed, 0); got != c.Hit.Speed/2 {
			t.Errorf("%s hit: %d", name, got)
		}

		if got := BlockRate(c.Block.Speed, 0, false); got != c.Block.Speed/2 {
			t.Errorf("%s block: %d", name, got)
		}
	}
}

// TestClassesMatchAnimData verifies the embedded numbers against the real
// expansion animdata.d2 (D2_ANIMDATA=path); skipped when unset.
func TestClassesMatchAnimData(t *testing.T) {
	path := os.Getenv("D2_ANIMDATA")
	if path == "" {
		t.Skip("D2_ANIMDATA not set")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	ad, err := d2animdata.Load(b)
	if err != nil {
		t.Fatal(err)
	}

	tokens := map[string]string{
		"Amazon": "AM", "Sorceress": "SO", "Necromancer": "NE", "Paladin": "PA",
		"Barbarian": "BA", "Druid": "DZ", "Assassin": "AI",
	}

	for class, c := range Classes {
		tok := tokens[class]

		for mode, want := range map[string]Mode{"WL": c.Walk, "RN": c.Run, "GH": c.Hit, "BL": c.Block, "A1": c.Attack1, "SC": c.Cast} {
			recs := ad.GetRecords(tok + mode + "HTH")
			if len(recs) == 0 {
				t.Errorf("%s%sHTH missing", tok, mode)
				continue
			}

			if got := (Mode{recs[0].FramesPerDirection(), recs[0].Speed()}); got != want {
				t.Errorf("%s%sHTH: animdata %v, table %v", tok, mode, got, want)
			}
		}
	}
}
