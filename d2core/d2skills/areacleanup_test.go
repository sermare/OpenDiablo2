package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func auraEngine(defs d2state.Defs) *Engine {
	return &Engine{sets: map[string]*d2state.Set{}, defs: defs, auras: map[string]*auraRun{},
		heroes: map[string]*heroUnit{}, targets: map[string]*monsterTarget{}, pets: map[string][]*d2mapentity.Monster{},
		visuals: map[uint32]*d2mapentity.Missile{}, fx: map[*d2mapentity.Missile]int{}, dots: map[string]dotTotal{}}
}

func TestHeroDiedStopsAuraWhoseStateWasCleared(t *testing.T) {
	e := auraEngine(d2state.Defs{"holyfire": {}, "keeper": {PlrStayDeath: true}})

	for _, id := range []string{"dead", "alive"} {
		e.setOf(id).Apply(0, d2state.Instance{Name: "holyfire", Until: 100})
		e.auras[id] = &auraRun{ef: d2skill.Effect{State: "holyfire"}}
	}

	e.HeroDied("dead")

	if e.auras["dead"] != nil {
		t.Error("the dead hero's aura keeps pulsing and would give the state back")
	}

	if e.auras["alive"] == nil || !e.HasState("alive", "holyfire") {
		t.Error("a hero who did not die lost his aura")
	}

	// an aura state with plrstaydeath survives death, so the aura stays on
	e.setOf("k").Apply(0, d2state.Instance{Name: "keeper", Until: 100})
	e.auras["k"] = &auraRun{ef: d2skill.Effect{State: "keeper"}}
	e.HeroDied("k")

	if e.auras["k"] == nil {
		t.Error("plrstaydeath aura was stopped")
	}
}

func TestAreaChangedDropsAreaStateKeepsHero(t *testing.T) {
	e := auraEngine(d2state.Defs{"holyfire": {}})
	e.heroes["hero"] = &heroUnit{cooldowns: map[int]int{5: 99}}
	e.setOf("hero").Apply(0, d2state.Instance{Name: "holyfire", Until: 1000})
	e.auras["hero"] = &auraRun{ef: d2skill.Effect{State: "holyfire"}}
	e.setOf("m1").Apply(0, d2state.Instance{Name: "holyfire", Until: 1000})
	e.dots["m1"] = dotTotal{poison: 3}
	e.targets["m1"] = &monsterTarget{}
	e.pets["hero"] = []*d2mapentity.Monster{{}}
	e.storms = []*stormRun{{}}
	e.traps = []*trapRun{{}}
	e.watches = []*watch{{}}
	e.after(10, func() {})

	e.AreaChanged(nil)

	if len(e.storms)+len(e.traps)+len(e.watches)+len(e.timers) != 0 || len(e.pets) != 0 || len(e.targets) != 0 ||
		len(e.dots) != 0 {
		t.Error("area bookkeeping survived the area change")
	}

	if e.HasState("m1", "holyfire") {
		t.Error("a monster's states survived (ids are reused by the next area)")
	}

	if !e.HasState("hero", "holyfire") || e.auras["hero"] == nil || e.heroes["hero"].cooldowns[5] != 99 {
		t.Error("the hero lost his aura, states or cooldowns")
	}
}

// A hero who neither dies nor changes area is not touched: another hero's
// death leaves his aura and states alone.
func TestHeroDiedLeavesOtherHeroesAura(t *testing.T) {
	e := auraEngine(d2state.Defs{"holyfire": {}})
	e.setOf("a").Apply(0, d2state.Instance{Name: "holyfire", Until: 100})
	e.auras["a"] = &auraRun{ef: d2skill.Effect{State: "holyfire"}}
	e.HeroDied("b")

	if e.auras["a"] == nil || !e.HasState("a", "holyfire") {
		t.Error("another hero's death changed this hero's aura")
	}
}
