package d2gamescreen

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2skills"
)

func TestParseCastPart(t *testing.T) {
	for _, c := range []struct {
		in    string
		skill string
		count int
		must  bool
	}{
		{"Bash", "Bash", castTestDefaultCount, false},
		{" Bash,1 ", "Bash", 1, false},
		{"Bash,1!", "Bash", 1, true},
		{"Chain Lightning,2!", "Chain Lightning", 2, true},
		{"Chain Lightning !", "Chain Lightning", castTestDefaultCount, true},
		{"Fire Ball,x", "Fire Ball,x", castTestDefaultCount, false},
	} {
		skill, count, must := parseCastPart(c.in)
		if skill != c.skill || count != c.count || must != c.must {
			t.Errorf("%q: got (%q,%d,%v), want (%q,%d,%v)", c.in, skill, count, must, c.skill, c.count, c.must)
		}
	}
}

func TestCastSpreadEnv(t *testing.T) {
	for _, c := range []struct {
		in    string
		state string
		n     int
	}{
		{"", "", 0},
		{"rabies", "", 0},
		{"rabies,2", "rabies", 2},
		{" rabies , 3 ", "rabies", 3},
		{"rabies,0", "", 0},
		{",2", "", 0},
	} {
		t.Setenv("OD2_AUTOCAST_SPREAD", c.in)

		if state, n := castSpreadEnv(); state != c.state || n != c.n {
			t.Errorf("%q: got (%q,%d), want (%q,%d)", c.in, state, n, c.state, c.n)
		}
	}
}

func TestCastCrowdEnv(t *testing.T) {
	for _, c := range []struct {
		in    string
		n     int
		state string
	}{
		{"", 0, ""},
		{"3", 3, ""},
		{" 3 , rabies ", 3, "rabies"},
		{"0,rabies", 0, ""},
		{"x", 0, ""},
	} {
		t.Setenv("OD2_AUTOCAST_CROWD", c.in)

		if n, state := castCrowdEnv(); n != c.n || state != c.state {
			t.Errorf("%q: got (%d,%q), want (%d,%q)", c.in, n, state, c.n, c.state)
		}
	}
}

// A mustHit skill whose cast hit nobody (the hero's minions killed the target
// in the cast animation, or the swing missed) is cast again, a bounded number
// of times; other skills never are.
func TestMissedHit(t *testing.T) {
	start := d2skills.Counters{Hits: 7, Melee: 3}
	it := &castItem{mustHit: true, c0: start}

	if !it.missedHit(d2skills.Counters{Hits: 7, Melee: 4, Missiles: 2, Misses: 1}) {
		t.Error("a swing that missed or a missile that hit nothing must be repeated")
	}

	if it.missedHit(d2skills.Counters{Hits: 8, Melee: 4}) {
		t.Error("a skill that hit once is done")
	}

	if it.missedHit(d2skills.Counters{Hits: 7, Melee: 3, AreaHits: 1}) {
		t.Error("an aura pulse (Holy Fire) that hurt someone is a hit")
	}

	it.hitRetries = castItemMaxHitRetries
	if it.missedHit(start) {
		t.Error("retries are bounded")
	}

	if (&castItem{c0: start}).missedHit(start) {
		t.Error("a skill without the mustHit flag is never repeated")
	}
}
