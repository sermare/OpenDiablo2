package d2mapentity

import (
	"math"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
)

func TestSpeedOverrideAddsToSlow(t *testing.T) {
	m := &Monster{walkSpeed: 4, runSpeed: 8}
	m.mode = d2monster.ModeWalk

	cases := []struct {
		slow, over int
		want       float64
	}{
		{0, 0, 4}, {0, 20, 4.8}, {-50, 50, 4}, {0, 500, 4 * 2.26}, {0, -500, 4 * 0.05}, {-100, -100, 4 * 0.05},
	}

	for _, c := range cases {
		m.slowPct, m.overridePct = c.slow, 0
		m.SetSpeedOverride(c.over)

		if got := m.slowed(m.walkSpeed); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("slow %d over %d: speed %v, want %v", c.slow, c.over, got, c.want)
		}
	}

	m.SetSpeedOverride(MaxSpeedOverride + 40)
	if m.SpeedOverride() != MaxSpeedOverride {
		t.Errorf("not clamped: %d", m.SpeedOverride())
	}

	m.SetSpeedOverride(0)
	if m.SpeedOverride() != 0 {
		t.Error("0 must clear")
	}
}
