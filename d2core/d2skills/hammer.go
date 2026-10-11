package d2skills

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"

// ConcentrationDamagePct implements d2skill.ConcentrationHolder: the damage
// percent of the hero's Concentration state alone (0x647550).
func (h *heroUnit) ConcentrationDamagePct() int {
	in := h.e.setOf(h.p.ID()).Get(h.e.frame, "concentration")
	if in == nil {
		return 0
	}

	v := 0

	for _, m := range in.Mods {
		if m.Stat == "damagepercent" {
			v += m.Value
		}
	}

	return v
}

var _ d2skill.ConcentrationHolder = (*heroUnit)(nil)
