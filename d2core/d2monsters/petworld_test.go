package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2summon"
)

type fakePets struct {
	frame   int
	next    uint32
	alive   map[uint32]bool
	removed []uint32
	stats   map[uint32]*d2summon.Stats
}

func newFakePets() *fakePets {
	return &fakePets{alive: map[uint32]bool{}, stats: map[uint32]*d2summon.Stats{}}
}

func (f *fakePets) Frame() int { return f.frame }

func (f *fakePets) SpawnPet(_, _ string, _, _ int, _ *d2skill.SummonOrder, st *d2summon.Stats) (uint32, error) {
	f.next++
	f.alive[f.next], f.stats[f.next] = true, st

	return f.next, nil
}

func (f *fakePets) RemovePet(id uint32)     { f.removed = append(f.removed, id); delete(f.alive, id) }
func (f *fakePets) PetAlive(id uint32) bool { return f.alive[id] }

const fakeMonstats = "Id\thcIdx\tAI\tVelocity\tRun\tminHP\tmaxHP\tAC\tA1MinD\tA1MaxD\tA1TH\n" +
	"necroskeleton\t1\tNecroPet\t6\t6\t20\t20\t10\t2\t4\t30\n"

func TestSummonerRosterAndStats(t *testing.T) {
	tpl, err := d2summon.LoadTemplates([]byte(fakeMonstats))
	if err != nil {
		t.Fatal(err)
	}

	w := newFakePets()
	s := NewSummoner(w, tpl, d2summon.Normal, 1)
	place := func(i, n int) (int, int) { return 10 + i, 20 }

	o := &d2skill.SummonOrder{Key: "necroskeleton", PetType: "skeleton", Max: 2, Count: 1, Kind: "minion", HPPct: 50,
		Stats: []d2skill.StatMod{{Stat: "damagepercent", Value: 100}}}

	ids, _ := s.Cast("hero", "necroskeleton", o, nil, place)
	if len(ids) != 1 {
		t.Fatalf("ids = %v", ids)
	}

	if st := w.stats[ids[0]]; st == nil || st.MaxHP != 30 || st.DmgMax != 8 || st.DmgMin != 4 {
		t.Errorf("computed stats = %+v", st)
	}

	s.Cast("hero", "necroskeleton", o, nil, place)

	// third cast at the limit evicts the oldest
	ids3, plan := s.Cast("hero", "necroskeleton", o, nil, place)
	if len(ids3) != 1 || len(plan.Evict) != 1 || plan.Evict[0] != 1 || len(w.removed) != 1 || w.removed[0] != 1 {
		t.Errorf("eviction: ids=%v plan=%+v removed=%v", ids3, plan, w.removed)
	}

	if got := s.Roster("hero").Count("skeleton"); got != 2 {
		t.Errorf("count = %d", got)
	}

	// a minion that died is dropped before planning: no eviction needed
	delete(w.alive, 2)

	_, plan = s.Cast("hero", "necroskeleton", o, nil, place)
	if len(plan.Evict) != 0 || s.Roster("hero").Count("skeleton") != 2 {
		t.Errorf("dead pet: plan=%+v count=%d", plan, s.Roster("hero").Count("skeleton"))
	}

	// another owner has its own roster
	if ids, _ := s.Cast("other", "necroskeleton", o, nil, place); len(ids) != 1 || s.Roster("other").Total() != 1 {
		t.Error("rosters must be per owner")
	}
}

func TestSummonerMultiAndExpiry(t *testing.T) {
	w := newFakePets()
	s := NewSummoner(w, nil, d2summon.Normal, 1) // no templates: stats stay nil
	place := func(i, n int) (int, int) { return i, 0 }

	raven := &d2skill.SummonOrder{Key: "raven", PetType: "raven", Max: 4, Count: 4, Kind: "minion", Frames: 100}

	ids, _ := s.Cast("h", "raven", raven, nil, place)
	if len(ids) != 4 || w.stats[ids[0]] != nil {
		t.Fatalf("ids=%v stats=%v", ids, w.stats[ids[0]])
	}

	if ids, _ = s.Cast("h", "raven", raven, nil, place); len(ids) != 0 {
		t.Errorf("full group must create nothing, got %v", ids)
	}

	w.frame = 99
	s.Step()

	if len(w.removed) != 0 {
		t.Error("expired too early")
	}

	w.frame = 100
	s.Step()

	if len(w.removed) != 4 || s.Roster("h").Total() != 0 {
		t.Errorf("removed=%v total=%d", w.removed, s.Roster("h").Total())
	}

	// unlimited pettype (walls): every piece is created
	wall := &d2skill.SummonOrder{Key: "bonewall", PetType: "none", Max: 64, Count: 8, Kind: "wall"}
	if ids, _ = s.Cast("h", "bonewall", wall, nil, place); len(ids) != 8 {
		t.Errorf("wall pieces = %d", len(ids))
	}
}
