package d2asset

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2animspeed"
)

// TestEngineRateMatchesOracleWithoutModifiers is the differential regression
// test: for a character with no speed modifiers the engine's frame time
// (1/(speed*speedUnit), as set in createMode/SetAnimSpeed) turns back into
// exactly the AnimData speed, which is what the exe-derived oracle produces for
// walk, run, attack and cast. Hit and block recovery are deliberately absent:
// the oracle says they play at half the AnimData speed and the engine does not
// play them at all, so they cannot be wired as identical.
func TestEngineRateMatchesOracleWithoutModifiers(t *testing.T) {
	for speed := 1; speed <= 1024; speed++ {
		engineSeconds := 1.0 / (float64(speed) * speedUnit)
		engineRate := rateFromFrameSeconds(engineSeconds)

		for name, oracle := range map[string]int{
			"walk/run": d2animspeed.WalkRate(speed, 0, 0),
			"attack":   d2animspeed.AttackRate(speed, d2animspeed.AttackBasePct, 0, 0, false),
			"cast":     d2animspeed.CastRate(speed, 0),
		} {
			if engineRate != oracle {
				t.Fatalf("speed %d %s: engine rate %d, oracle %d", speed, name, engineRate, oracle)
			}
		}
	}
}
