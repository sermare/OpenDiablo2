package d2herostats

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2animspeed"
)

// ticksByRule recomputes the animation length straight from the d2animspeed
// rate functions, independently of the table derivation helpers.
func ticksByRule(k Kind, frames, speed, stat int) int {
	var rate int

	switch k {
	case FHR:
		rate = d2animspeed.HitRate(speed, stat)
	case FBR:
		rate = d2animspeed.BlockRate(speed, stat, false)
	case FCR:
		rate = d2animspeed.CastRate(speed, stat)
	default:
		rate = d2animspeed.AttackRate(speed, 100, stat, 0, false)
	}

	t, _ := d2animspeed.Ticks(frames, rate)

	return t
}

// Every row is exactly the set of stat values where the rule's tick count
// drops, for every stat 0..600.
func TestEveryRowDerivesFromRule(t *testing.T) {
	for _, e := range Tables {
		var want []int

		prev := -1

		for s := 0; s <= 600; s++ {
			if tk := ticksByRule(e.Kind, e.Anim.Frames, e.Anim.Speed, s); tk != prev {
				want = append(want, s)
				prev = tk
			}
		}

		if !reflect.DeepEqual(e.Table, want) {
			t.Errorf("%s %s %q: table %v, rule gives %v", e.Class, e.Kind, e.Note, e.Table, want)
		}

		if e.Kind == FCR && !reflect.DeepEqual(e.Model.Table(), e.Table) {
			t.Errorf("%s FCR %q: cast formula %v != table %v", e.Class, e.Note, e.Model.Table(), e.Table)
		}
	}
}

// The exact rows, including the ones the old hand-copied tables got wrong.
func TestDerivedRowsGolden(t *testing.T) {
	sorcFHR := []int{0, 5, 9, 14, 20, 30, 42, 60, 86, 142, 280}
	row5 := []int{0, 7, 15, 27, 48, 86, 200}
	row3 := []int{0, 13, 32, 86, 600}

	for _, tc := range []struct {
		class string
		kind  Kind
		note  string
		want  []int
	}{
		{"Amazon", FHR, "", []int{0, 6, 13, 20, 32, 52, 86, 174, 600}},
		{"Sorceress", FHR, "", sorcFHR},
		{"Necromancer", FHR, "", []int{0, 5, 10, 16, 26, 39, 56, 86, 152, 377}}, // was Sorceress FCR
		{"Paladin", FHR, "", row5},
		{"Barbarian", FHR, "", row5}, // was the 4 frame row
		{"Druid", FHR, "human form, one-handed swing weapon", []int{0, 3, 7, 13, 19, 29, 42, 63, 99, 174, 456}},
		{"Druid", FHR, "human form, other weapon classes", []int{0, 5, 10, 16, 26, 39, 56, 86, 152, 377}},
		{"Druid", FHR, "werewolf", []int{0, 9, 20, 42, 86, 280}},
		{"Assassin", FHR, "", row5},

		{"Amazon", FCR, "", []int{0, 7, 14, 22, 32, 48, 68, 99, 152}},
		{"Sorceress", FCR, "", []int{0, 9, 20, 37, 63, 105, 200}},
		{"Necromancer", FCR, "", []int{0, 9, 18, 30, 48, 75, 125}},
		{"Paladin", FCR, "", []int{0, 9, 18, 30, 48, 75, 125}},
		{"Barbarian", FCR, "", []int{0, 9, 20, 37, 63, 105, 200}},
		{"Druid", FCR, "", []int{0, 4, 10, 19, 30, 46, 68, 99, 163}},
		{"Assassin", FCR, "", []int{0, 8, 16, 27, 42, 65, 102, 174}},
		{"Sorceress", FCR, "lightning/chain lightning", []int{0, 7, 15, 23, 35, 52, 78, 117, 194}},

		{"Amazon", FBR, "", row3},
		{"Sorceress", FBR, "", row5}, // was her FHR row
		{"Necromancer", FBR, "", []int{0, 6, 13, 20, 32, 52, 86, 174, 600}}, // was a 5 frame row
		{"Paladin", FBR, "with a shield", row3},
		{"Barbarian", FBR, "", []int{0, 9, 20, 42, 86, 280}},
		{"Druid", FBR, "human form", []int{0, 6, 13, 20, 32, 52, 86, 174, 600}},
		{"Assassin", FBR, "", row3},
	} {
		found := false

		for _, e := range Tables {
			if e.Class == tc.class && e.Kind == tc.kind && e.Note == tc.note {
				found = true

				if !reflect.DeepEqual(e.Table, tc.want) {
					t.Errorf("%s %s %q: %v, want %v", tc.class, tc.kind, tc.note, e.Table, tc.want)
				}
			}
		}

		if !found {
			t.Errorf("missing %s %s %q", tc.class, tc.kind, tc.note)
		}
	}
}

// The Druid human FHR discrepancy is explained by the speed of the 1HS record.
func TestDruidFHRExplained(t *testing.T) {
	oldCommunity := []int{0, 3, 7, 13, 19, 29, 42, 63, 99, 174, 456}

	if got := DeriveBreakpoints(FHR, 7, 248); !reflect.DeepEqual(got, oldCommunity) {
		t.Errorf("7 frames at 248: %v", got)
	}

	if e, _ := Lookup("Druid", FHR); !reflect.DeepEqual(e.Table, oldCommunity) {
		t.Errorf("default Druid FHR changed: %v", e.Table)
	}
}

func TestUnverifiedRowsFlagged(t *testing.T) {
	for _, e := range Tables {
		flagged := e.Unverified != ""
		forced := e.Note == "werewolf" || e.Note == "werebear" || e.Note == "lightning/chain lightning" ||
			e.Kind == IAS || e.Note == "with a shield"

		if forced && !flagged {
			t.Errorf("%s %s %q must be flagged unverified", e.Class, e.Kind, e.Note)
		}

		if !forced && flagged {
			t.Errorf("%s %s %q flagged unexpectedly", e.Class, e.Kind, e.Note)
		}
	}
}

func TestNextBreakpoint(t *testing.T) {
	e, _ := Lookup("Sorceress", FCR) // 14 frames: 14, 13, ... ticks

	next, now, then, ok := e.NextBreakpoint(0)
	if !ok || next != 9 || now != 14 || then != 13 {
		t.Errorf("from 0: %d %d %d %v", next, now, then, ok)
	}

	next, now, then, ok = e.NextBreakpoint(9)
	if !ok || next != 20 || now != 13 || then != 12 {
		t.Errorf("from 9: %d %d %d %v", next, now, then, ok)
	}

	if _, _, _, ok = e.NextBreakpoint(200); ok {
		t.Error("200 is the last breakpoint")
	}

	// Hit recovery without FHR is half speed: 8 frames take 16 ticks.
	h, _ := Lookup("Sorceress", FHR)
	if _, now, then, _ = h.NextBreakpoint(0); now != 16 || then != 15 {
		t.Errorf("sorc FHR ticks %d -> %d", now, then)
	}
}

// The embedded per-class numbers agree with d2animspeed.Classes (hand-to-hand
// records), the other embedded copy of the AnimData constants.
func TestAnimConstantsAgreeWithClasses(t *testing.T) {
	for _, e := range Tables {
		m, ok := d2animspeed.Classes[e.Class]
		if !ok || e.Note != "" {
			continue
		}

		var want d2animspeed.Mode

		switch e.Kind {
		case FHR:
			want = m.Hit
		case FBR:
			want = m.Block
		case FCR:
			want = m.Cast
		default:
			continue
		}

		if e.Anim.Frames != want.Frames || e.Anim.Speed != want.Speed {
			t.Errorf("%s %s: %+v vs Classes %+v", e.Class, e.Kind, e.Anim, want)
		}
	}
}

func TestDeriveCustomAnimation(t *testing.T) {
	// reusable for any record: a 5 frame hit animation at 256
	if got := DeriveBreakpoints(FHR, 5, 256); !reflect.DeepEqual(got, []int{0, 7, 15, 27, 48, 86, 200}) {
		t.Errorf("%v", got)
	}

	// claws attack: 11 frames at 208 is slower than bare hands at 256
	claws, _ := Lookup("Assassin", IAS)
	if claws.Note != "martial arts, bare hands" {
		t.Fatalf("default IAS entry %q", claws.Note)
	}

	if d2animspeed.ActionTicks(d2animspeed.ActionAttack, 11, 208, 0) <= d2animspeed.ActionTicks(d2animspeed.ActionAttack, 11, 256, 0) {
		t.Error("claws should be slower at 0 IAS")
	}
}
