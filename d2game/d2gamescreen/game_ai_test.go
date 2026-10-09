package d2gamescreen

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
)

func TestParseAIAuto(t *testing.T) {
	for _, c := range []struct {
		spec   string
		ref    string
		states []d2monster.ForcedKind
		bad    bool
	}{
		{"Vulture", "Vulture", nil, false},
		{"skeleton1,state=fear", "skeleton1", []d2monster.ForcedKind{d2monster.ForcedFear}, false},
		{"Summoner,state=confuse+attract|charm", "Summoner",
			[]d2monster.ForcedKind{d2monster.ForcedConfuse, d2monster.ForcedAttract, d2monster.ForcedCharm}, false},
		{"", "", nil, true},
		{"x,state=bogus", "", nil, true},
		{"x,speed=2", "", nil, true},
	} {
		ref, states, err := parseAIAuto(c.spec)
		if (err != nil) != c.bad {
			t.Errorf("%q: err=%v", c.spec, err)

			continue
		}

		if c.bad {
			continue
		}

		if ref != c.ref || len(states) != len(c.states) {
			t.Errorf("%q: ref=%q states=%v", c.spec, ref, states)

			continue
		}

		for i := range states {
			if states[i] != c.states[i] {
				t.Errorf("%q: state %d = %v", c.spec, i, states[i])
			}
		}
	}
}
