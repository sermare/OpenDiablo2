package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// The prison doors of Rescue on Mount Arreat (AI "Idle", killable) are props the hero can destroy, not enemies.
func TestIsDestructibleProp(t *testing.T) {
	prop := func(key string, mod func(*d2records.MonStatRecord)) *d2records.MonStatRecord {
		st := &d2records.MonStatRecord{Key: key, Enabled: true, IsKillable: true, AiKey: "Idle"}
		if mod != nil {
			mod(st)
		}

		return st
	}

	cases := []struct {
		name string
		st   *d2records.MonStatRecord
		want bool
	}{
		{"prison door", prop("prisondoor", nil), true},
		{"barricade door", prop("barricadedoor2", nil), true},
		{"barricade tower", prop("barricadetower", nil), true},
		{"not killable", prop("prisondoor", func(s *d2records.MonStatRecord) { s.IsKillable = false }), false},
		{"an npc", prop("prisondoor", func(s *d2records.MonStatRecord) { s.IsNpc = true }), false},
		{"another idle monster", prop("boneprison1", nil), false},
		{"nil", nil, false},
	}

	for _, c := range cases {
		if got := IsDestructibleProp(c.st); got != c.want {
			t.Errorf("%s: IsDestructibleProp = %v, want %v", c.name, got, c.want)
		}
	}

	if IsHostile(prop("prisondoor", nil)) {
		t.Error("a prison door is not an enemy (IsHostile must stay false: it never attacks)")
	}
}
