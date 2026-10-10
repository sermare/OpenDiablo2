package d2gamescreen

import (
	"os"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// alsoSubject is an extra OD2_AUTOAI_ALSO monster: OD2_AUTOAI_ALSO=<ai|monster>
// [+<ai|monster>...] spawns more monsters next to the hero beside the main
// subject, so one game window can show an in-game subject per ported AI family
// (scripts/verify.d/97-ai-states.sh). The extras only run their own AI; no
// state is injected into them.
type alsoSubject struct {
	ref     string
	m       *d2mapentity.Monster
	startX  int
	startY  int
	spawned bool
}

func alsoRefs() []string {
	spec := strings.TrimSpace(os.Getenv("OD2_AUTOAI_ALSO"))
	if spec == "" {
		return nil
	}

	return strings.FieldsFunc(spec, func(r rune) bool { return r == '+' || r == '|' || r == ' ' })
}

// spawnAlso places the extra subjects on a ring around the hero and logs one
// "AUTOAI also" line each (the AI it runs and whether it is implemented).
func (v *Game) spawnAlso(t *aiAutoTest) {
	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())

	for i, ref := range alsoRefs() {
		stat := v.monsters.FindStat(ref)
		if stat == nil {
			stat = v.monsters.FindArchetype(ref)
		}

		if stat == nil {
			v.Infof("AUTOAI also ref=%s: no such monster or AI archetype", ref)

			continue
		}

		// alternate sides of the hero, spreading out on a growing ring
		dx, dy := 6+2*(i/2), 3*(i%4)-4
		if i%2 == 1 {
			dx = -dx
		}

		m, err := v.monsters.SpawnNear(stat, hx+dx, hy+dy, 3)
		if err != nil {
			v.Infof("AUTOAI also ref=%s: spawn failed: %v", ref, err)

			continue
		}

		sx, sy := m.SubtilePos()
		t.also = append(t.also, &alsoSubject{ref: ref, m: m, startX: sx, startY: sy, spawned: true})

		b := m.Brain
		v.Infof("AUTOAI also ref=%s monster=%s id=%d ai=%s implemented=%v", ref, m.Label(), b.ID, b.Profile.AI,
			b.Def != nil && b.Def.Implemented)
	}
}

// summarizeAlso logs how far each extra subject moved and whether it lives.
func (v *Game) summarizeAlso(t *aiAutoTest) {
	for _, a := range t.also {
		x, y := a.m.SubtilePos()
		moved := absInt(x-a.startX) + absInt(y-a.startY)

		v.Infof("AUTOAI also-summary ref=%s ai=%s alive=%v moved=%d", a.ref, a.m.Brain.Profile.AI, a.m.Alive(), moved)
	}
}

func absInt(a int) int {
	if a < 0 {
		return -a
	}

	return a
}
