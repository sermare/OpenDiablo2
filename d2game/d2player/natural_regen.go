package d2player

import (
	"os"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2herostats"
)

// noNaturalRegen is OD2_NOREGEN=1: the natural regeneration is off (a debug
// switch to compare scripted runs with and without it).
var noNaturalRegen = os.Getenv("OD2_NOREGEN") == "1"

// advanceNaturalRegen runs the exe's per-frame vitals routines on the hero
// (d2herostats/regen.go): mana comes back on its own with the class ManaRegen
// (charstats, 120 s to full) raised by the regenerate-mana percent of the
// items, and life only comes back with replenish life. The potions' over time
// part is separate (advancePotions). A dead hero does not regenerate.
func (g *GameControls) advanceNaturalRegen(elapsed float64) {
	st := g.hero.Stats
	if noNaturalRegen || st == nil || st.Health <= 0 {
		return
	}

	in := d2herostats.RegenInput{MaxLife: st.MaxHealth, MaxMana: st.MaxMana}

	if g.asset != nil {
		if cs := g.asset.Records.Character.Stats[g.hero.Class]; cs != nil {
			in.ManaRegenSeconds = cs.ManaRegen
		}
	}

	if t := st.Totals; t != nil {
		in.ManaBonusPct, in.ManaFlatRaw, in.LifeRegenRaw = t.ManaRecoveryPct, t.ManaRecoveryRaw, t.LifeRegen
	}

	g.vitals.Advance(elapsed, in, &st.Health, &st.Mana)
}
