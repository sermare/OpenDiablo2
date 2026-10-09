package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func TestDivisorsFor(t *testing.T) {
	recs := d2records.DifficultyLevels{
		d2enum.DifficultyNormal:    {MonsterColdDivisor: 1, MonsterFreezeDivisor: 1},
		d2enum.DifficultyNightmare: {MonsterColdDivisor: 2, MonsterFreezeDivisor: 3},
	}

	for _, c := range []struct{ diff, chill, freeze int }{{0, 1, 1}, {1, 2, 3}, {2, 0, 0}} {
		if ch, fr := divisorsFor(recs, c.diff); ch != c.chill || fr != c.freeze {
			t.Errorf("diff %d: got %d,%d want %d,%d", c.diff, ch, fr, c.chill, c.freeze)
		}
	}

	if ch, fr := divisorsFor(nil, 1); ch != 0 || fr != 0 {
		t.Errorf("nil table: %d,%d", ch, fr)
	}
}

// The divisors shorten the cold and freeze lengths (chill minimum 1).
func TestDivisorsReachSet(t *testing.T) {
	s := d2state.New()
	s.ApplyHit(0, d2state.Hit{ColdLen: 3, FreezeLen: 40, HasColdEffect: true, ColdEffect: -50, ChillDiv: 4, FreezeDiv: 4})

	if in := s.Get(0, d2state.Chill); in == nil || in.Until != 1 {
		t.Errorf("chill %+v want until 1", in)
	}

	if in := s.Get(0, d2state.Freeze); in == nil || in.Until != 10 {
		t.Errorf("freeze %+v want until 10", in)
	}
}

func TestHeroDiedClearsStatesAndStreams(t *testing.T) {
	e := &Engine{sets: map[string]*d2state.Set{}, defs: d2state.Defs{}}
	e.setOf("hero").Apply(0, d2state.Instance{Name: "x", Until: 100})
	e.setOf("hero").AddStream(0, "poison", 256, 50, "", 0)
	e.HeroDied("hero")

	if e.HasState("hero", "x") || len(e.setOf("hero").Streams(1)) != 0 {
		t.Error("hero death should clear states and streams")
	}
}

func TestHeroDiedKeepsPlrStayDeathOnly(t *testing.T) {
	defs := d2state.Defs{"stay": {PlrStayDeath: true}, "gone": {}}
	e := &Engine{sets: map[string]*d2state.Set{}, defs: defs}

	for _, id := range []string{"dead", "alive"} {
		set := e.setOf(id)
		set.Apply(0, d2state.Instance{Name: "stay", Until: 100})
		set.Apply(0, d2state.Instance{Name: "gone", Until: 100})
		set.AddStream(0, "poison", 256, 50, "", 0)
	}

	e.HeroDied("dead")

	if e.HasState("dead", "gone") || len(e.setOf("dead").Streams(1)) != 0 {
		t.Error("dead hero keeps a state or stream")
	}

	if !e.HasState("dead", "stay") {
		t.Error("plrstaydeath state must survive death")
	}

	if !e.HasState("alive", "gone") || !e.HasState("alive", "stay") || len(e.setOf("alive").Streams(1)) == 0 {
		t.Error("a hero who did not die was affected")
	}
}
