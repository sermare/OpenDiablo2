package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
)

func rabiesEngine() *Engine {
	return &Engine{Logger: d2util.NewLogger(), targets: map[string]*monsterTarget{}, sets: map[string]*d2state.Set{},
		heroes: map[string]*heroUnit{}, monSeed: d2rand.New(1), frame: 100}
}

// A monster is an infection carrier: it reports its states by name (the
// contagion skips the infected) and owns the plague missile, which follows it,
// ends with it and hands on the frame its infection expires.
func TestMonsterTargetCarriesTheRabiesPlague(t *testing.T) {
	e := rabiesEngine()
	m := monAt(12, 7)
	tg := e.target(m)

	var st d2missile.Stateful = tg
	if st.HasStateNamed("rabies") || st.HasStateNamed("") {
		t.Fatal("a clean monster reports a state")
	}

	e.setOf(m.ID()).Apply(e.frame, d2state.Instance{Name: "rabies", Until: e.frame + 290})

	if !st.HasStateNamed("rabies") || st.HasStateNamed("amplify damage") {
		t.Error("HasStateNamed does not follow the state set")
	}

	var ow d2missile.Ownable = tg

	o := ow.AsOwner()
	if o.ID != m.ID() || o.IsPlayer || o.Roller == nil || o.Gone == nil || o.Pos == nil || o.StateExpire == nil {
		t.Fatalf("owner %+v", o)
	}

	if x, y := o.Pos(); x != 12 || y != 7 {
		t.Errorf("owner position %v,%v", x, y)
	}

	if f, ok := o.StateExpire("rabies"); !ok || f != 390 {
		t.Errorf("expiry %d %v, want 390 true", f, ok)
	}

	if _, ok := o.StateExpire("poison"); ok {
		t.Error("expiry of a state it does not have")
	}

	if o.Gone() {
		t.Error("living monster is gone")
	}
}
