package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
)

// A curse cast at a spot where the target already died (minions kill the
// 10 life zombies of the scenario during the cast animation: the "affected=0"
// lines of 86-class-skills) forces no AI state. It must be counted so the
// scripted cast is repeated instead of counted as done.
func TestCurseOnEmptySpotIsWasted(t *testing.T) {
	e := newTestEngine()
	e.monsters = &d2monsters.Director{}

	p := &d2mapentity.Player{}
	u := &heroUnit{p: p}
	sk := &d2skill.Skill{Name: "Dim Vision"}
	ef := &d2skill.Effect{Kind: "area_state", Origin: "aim", X: 155, Y: 86, Radius: 13, State: "dimvision", Frames: 625}

	before := e.Counters.Wasted()
	e.areaState(p, u, sk, ef)

	if got := e.Counters.Wasted() - before; got != 1 || e.Counters.EmptyArea != 1 {
		t.Fatalf("empty curse: wasted +%d, EmptyArea %d, want 1 and 1", got, e.Counters.EmptyArea)
	}

	if e.Counters.Refused != 0 {
		t.Errorf("an empty curse is not a refusal: %d", e.Counters.Refused)
	}
}
