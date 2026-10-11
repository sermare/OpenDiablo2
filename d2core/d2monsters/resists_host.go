package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// Resists implements d2monster.ResistFinder: the fire, cold and lightning resistance (stats 0x27, 0x2b, 0x29)
// of a target. The Summoner picks his element from the first two, and Diablo's target weights read all three
// (0x5f5c10 family). A hero gives the shown values from her equipment, a monster the monstats columns of its
// difficulty; a mercenary or an unknown target counts as 0.
func (d *Director) Resists(t d2monster.Target) (fire, cold, lightning int) {
	if t.IsPlayer || t.ID < unitTargetBase {
		if p := d.playerFor(t.ID); p != nil && p.Stats != nil && p.Stats.Totals != nil {
			r := p.Stats.Totals.ResistShown

			return r[d2statlist.ResFire], r[d2statlist.ResCold], r[d2statlist.ResLight]
		}

		return 0, 0, 0
	}

	if tu := d.units[t.ID-unitTargetBase]; tu != nil && tu.m != nil && tu.m.Stat != nil {
		r := MonsterResists(tu.m.Stat, tu.m.Vitals.Difficulty)

		return r[2], r[4], r[3]
	}

	return 0, 0, 0
}
