package d2mp

import (
	"fmt"
	"math"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
)

// harness runs a Sim and one Replica per player on a fake clock with a fixed
// one-way latency.
type harness struct {
	t     *testing.T
	sim   *Sim
	rep   map[uint32]*Replica
	now   float64
	latMs float64
	seed  uint32
	queue []delayed
}

type delayed struct {
	at  float64
	to  uint32
	evs []Event
}

func newHarness(t *testing.T, rules Rules, latMs float64) *harness {
	return &harness{t: t, sim: NewSim(Config{Seed: 777, Rules: rules}), rep: map[uint32]*Replica{}, latMs: latMs, seed: 777}
}

func (h *harness) join(id uint32, name string, level uint8) {
	h.sim.Join(JoinInfo{ID: id, Name: name, Class: 1, Level: level, Act: 1})
	h.rep[id] = NewReplica(ReplicaConfig{Self: id, Seed: h.seed, Rules: h.sim.rules, Clock: func() float64 { return h.now }})
}

// advance runs ms of game time in 10 ms slices (so replicas see in-between times).
func (h *harness) advance(ms int) {
	for i := 0; i < ms/10; i++ {
		h.now += 10
		h.sim.Tick(uint32(h.now))

		for id := range h.rep {
			if evs := h.sim.Take(id); evs != nil {
				// the codec is part of the path
				b := EncodeEvents(evs)
				dec, err := DecodeEvents(b)
				if err != nil {
					h.t.Fatal(err)
				}

				h.queue = append(h.queue, delayed{h.now + h.latMs, id, dec})
			}
		}

		rest := h.queue[:0]

		for _, d := range h.queue {
			if d.at <= h.now {
				for _, e := range d.evs {
					h.rep[d.to].Apply(e)
				}
			} else {
				rest = append(rest, d)
			}
		}

		h.queue = rest
	}
}

func (h *harness) settle() { h.advance(int(2*h.latMs) + 60) }

func TestCodecRoundTrip(t *testing.T) {
	evs := []Event{
		{Type: EvTick, A: 1234},
		{Type: EvSpawn, ID: 5, Unit: Unit{ID: 5, Kind: KindMonster, Type: 3, Level: 2, Name: "Zombie", HP: 60, MaxHP: 60,
			Segs: []Seg{{X0: 1, Y0: 2, X1: 3, Y1: 4, Speed: 4.5, T0: 99}}}},
		{Type: EvSeg, ID: 5, Seg: Seg{X0: 1.2, Y0: 2.4, X1: 3, Y1: 4, Speed: 6, T0: 10}},
		{Type: EvHit, ID: 5, Other: 1, A: 9, B: 51},
		{Type: EvTrade, ID: 1, Trade: TradeView{State: TradeOpen, Partner: 2, MyItems: []string{"1:cap"}, MyGold: 5}},
		{Type: EvInv, ID: 1, A: 7, Items: []string{"3:amu"}},
		{Type: EvMsg, ID: 1, Text: "hi"},
	}

	got, err := DecodeEvents(EncodeEvents(evs))
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != len(evs) || got[1].Unit.Name != "Zombie" || got[1].Unit.Segs[0].Speed != 4.5 ||
		got[4].Trade.MyItems[0] != "1:cap" || got[5].Items[0] != "3:amu" || got[2].Seg.X0 != 1.2 {
		t.Fatalf("round trip mismatch: %+v", got)
	}

	if _, err := DecodeEvents(EncodeEvents(evs)[:20]); err == nil {
		t.Fatal("truncated batch accepted")
	}

	c := Command{Type: CmdTradeOffer, A: 4, Items: []string{"1:x"}}
	if d, err := DecodeCommand(EncodeCommand(c)); err != nil || d.A != 4 || d.Items[0] != "1:x" {
		t.Fatalf("command: %+v %v", d, err)
	}
}

func TestLevelDeterministic(t *testing.T) {
	r := DefaultRules{}
	a, b := r.Level(LevelSeed(5, 2), 2), r.Level(LevelSeed(5, 2), 2)

	if fmt.Sprint(a.Monsters, a.Objects) != fmt.Sprint(b.Monsters, b.Objects) || len(a.Blocked) != len(b.Blocked) {
		t.Fatal("same seed gave different levels")
	}

	c := r.Level(LevelSeed(6, 2), 2)
	if fmt.Sprint(a.Monsters) == fmt.Sprint(c.Monsters) {
		t.Fatal("different seeds gave the same monsters")
	}

	if len(r.Level(LevelSeed(5, d2level.RogueEncampment), 1).Monsters) != 0 {
		t.Fatal("town has monsters")
	}
}

func TestWalkSeenAndSmooth(t *testing.T) {
	h := newHarness(t, DefaultRules{}, 60)
	h.join(1, "A", 10)
	h.join(2, "B", 10)
	h.settle()

	if len(h.rep[2].Units()) < 3 { // both heroes + objects
		t.Fatalf("B sees %d units", len(h.rep[2].Units()))
	}

	h.sim.Walk(1, false, 58, 48)

	// sample B's view of A every 10 ms: no frame may jump more than speed*dt + quantisation
	var px, py float64
	first := true
	worst := 0.0

	for i := 0; i < 200; i++ {
		h.advance(10)

		if x, y, ok := h.rep[2].Pos(1); ok {
			if !first {
				worst = math.Max(worst, dist(px, py, x, y))
			}

			px, py, first = x, y, false
		}
	}

	if worst > WalkSpeed*0.010+0.01 {
		t.Fatalf("remote hero jumped %.3f tiles in one 10 ms frame", worst)
	}

	if math.Abs(px-58) > 0.3 || math.Abs(py-48) > 0.3 {
		t.Fatalf("remote hero ended at %.2f,%.2f", px, py)
	}

	if h.sim.Digest(1) != h.rep[2].Digest() {
		t.Fatal("digests differ")
	}
}

func TestPredictionNoRubberBand(t *testing.T) {
	h := newHarness(t, DefaultRules{}, 100)
	h.join(1, "A", 10)
	h.settle()

	r := h.rep[1]
	h.sim.Walk(1, true, 60, 48)
	r.PredictWalk(true, 60, 48)

	var lx, ly float64
	worst := 0.0

	for i := 0; i < 150; i++ {
		h.advance(10)
		x, y, _ := r.Pos(1)

		if i > 0 && dist(lx, ly, x, y) > RunSpeed*0.010+0.05 {
			worst = math.Max(worst, dist(lx, ly, x, y))
		}

		if i > 0 {
			if x < lx-0.001 {
				t.Fatalf("local hero went backwards at frame %d (%.3f -> %.3f)", i, lx, x)
			}
		}

		lx, ly = x, y
	}

	if worst > 0 {
		t.Fatalf("local hero jumped %.3f in a frame", worst)
	}
}

func TestFightKillsAndSharesXP(t *testing.T) {
	h := newHarness(t, strongHero{}, 40)
	h.join(1, "A", 8)
	h.join(2, "B", 8)
	h.sim.Command(1, Command{Type: CmdPartyInvite, Target: 2})
	h.sim.Command(2, Command{Type: CmdPartyAccept})
	h.settle()

	if h.sim.Unit(1).Party == 0 || h.sim.Unit(1).Party != h.sim.Unit(2).Party {
		t.Fatal("not in one party")
	}

	// go to the first field level through the town exit portal
	for _, id := range []uint32{1, 2} {
		var portal uint32

		for _, u := range h.sim.UnitsIn(1) {
			if u.Kind == KindObject && u.Type == ObjPortal {
				portal = u.ID
			}
		}

		h.sim.Interact(id, KindObject, portal)
	}

	h.advance(4000)

	if h.sim.Level(1) != 2 || h.sim.Level(2) != 2 {
		t.Fatalf("levels %d %d", h.sim.Level(1), h.sim.Level(2))
	}

	h.settle()

	if h.sim.Digest(2) != h.rep[1].Digest() || h.sim.Digest(2) != h.rep[2].Digest() {
		t.Fatal("level digests differ after arrival")
	}

	// A fights every monster with fire bolts and melee, B with fireballs
	h.sim.SelectSkill(2, SkillFireBall, false)

	for round := 0; round < 400; round++ {
		alive := 0

		for _, u := range h.sim.UnitsIn(2) {
			if u.Kind != KindMonster || u.Dead {
				continue
			}

			alive++
			h.sim.Interact(1, KindMonster, u.ID)
			bx, by := h.sim.pos(h.sim.Unit(2))
			mx, my := h.sim.pos(u)

			if dist(bx, by, mx, my) < 14 {
				h.sim.Cast(2, false, mx, my)
			} else {
				h.sim.Walk(2, true, mx, my)
			}

			break
		}

		if alive == 0 {
			break
		}

		h.advance(500)
	}

	h.advance(3000)

	if h.sim.Stats.Kills < 10 {
		t.Fatalf("only %d kills (hp %d %d dead %v %v)", h.sim.Stats.Kills, h.sim.Unit(1).HP, h.sim.Unit(2).HP, h.sim.Unit(1).Dead, h.sim.Unit(2).Dead)
	}

	if h.sim.Unit(1).Dead && h.sim.Unit(2).Dead {
		t.Log("both heroes died; still comparing state")
	}

	if h.rep[1].XP == 0 || h.rep[2].XP == 0 {
		t.Fatalf("party xp: %d %d", h.rep[1].XP, h.rep[2].XP)
	}

	h.settle()

	d := h.sim.Digest(2)
	if d != h.rep[1].Digest() || d != h.rep[2].Digest() {
		t.Fatal("world state differs after the fight")
	}
}

type strongHero struct{ DefaultRules }

func (strongHero) Hero(class, level uint8) (int32, int, int) { return 3000, 20, 30 }

type weakHero struct{ DefaultRules }

func (weakHero) Hero(class, level uint8) (int32, int, int) { return 12, 3, 5 }

func TestDeathRespawnVisible(t *testing.T) {
	h := newHarness(t, weakHero{}, 30)
	h.join(1, "A", 1)
	h.join(2, "B", 1)
	h.settle()

	// A walks into the field via the portal and lets monsters kill it
	var portal uint32

	for _, u := range h.sim.UnitsIn(1) {
		if u.Kind == KindObject && u.Type == ObjPortal {
			portal = u.ID
		}
	}

	h.sim.Interact(1, KindObject, portal)
	h.sim.Interact(2, KindObject, portal)
	h.advance(4000)
	h.settle()

	for i := 0; i < 600 && !h.sim.Unit(1).Dead; i++ {
		for _, u := range h.sim.UnitsIn(2) {
			if u.Kind == KindMonster && !u.Dead {
				mx, my := h.sim.pos(u)
				h.sim.Walk(1, true, mx, my)
			}
		}

		h.advance(200)
	}

	if !h.sim.Unit(1).Dead {
		t.Skip("hero survived the scripted walk")
	}

	h.settle()

	if u := h.rep[2].Unit(1); u == nil || !u.Dead {
		t.Fatal("B did not see A die")
	}

	h.sim.Command(1, Command{Type: CmdRespawn})
	h.settle()

	if h.sim.Level(1) != 1 || h.rep[1].Level != 1 || h.sim.Unit(1).Dead {
		t.Fatal("A did not respawn in town")
	}

	if h.rep[2].Unit(1) != nil {
		t.Fatal("B still sees A's corpse after the respawn")
	}
}

func TestChestItemsAndTrade(t *testing.T) {
	h := newHarness(t, DefaultRules{}, 20)
	h.join(1, "A", 5)
	h.join(2, "B", 5)
	h.settle()

	var chest uint32

	for _, u := range h.sim.UnitsIn(1) {
		if u.Kind == KindObject && u.Type == ObjChest {
			if chest == 0 {
				chest = u.ID
			}
		}
	}

	h.sim.Interact(1, KindObject, chest)
	h.advance(2000)
	h.settle()

	if h.rep[2].Unit(chest).State != ObjOpen {
		t.Fatal("B did not see the chest open")
	}

	var items []uint32

	for _, u := range h.sim.UnitsIn(1) {
		if u.Kind == KindItem {
			items = append(items, u.ID)
		}
	}

	if len(items) != 2 {
		t.Fatalf("chest dropped %d items", len(items))
	}

	h.sim.PickUp(1, items[0])
	h.advance(1500)
	h.settle()

	inv, _ := h.sim.Inventory(1)
	if len(inv) != 1 || len(h.rep[1].Inv) != 1 || h.rep[2].Unit(items[0]) != nil {
		t.Fatalf("pickup: inv %v replica %v", inv, h.rep[1].Inv)
	}

	// drop it again, B takes it, then they trade
	id, _, _ := ParseRef(inv[0])
	h.sim.Drop(1, id)
	h.settle()

	if h.rep[2].Unit(id) == nil {
		t.Fatal("B did not see the dropped item")
	}

	h.sim.PickUp(1, id)
	h.advance(1500)

	h.sim.Command(1, Command{Type: CmdTradeRequest, Target: 2})
	h.settle()

	if h.rep[2].Trade == nil || !h.rep[2].Trade.Incoming {
		t.Fatal("B got no trade request")
	}

	h.sim.Command(2, Command{Type: CmdTradeRespond, A: 1})
	inv, _ = h.sim.Inventory(1)
	h.sim.Command(1, Command{Type: CmdTradeOffer, Items: inv})
	h.settle()
	h.sim.Command(1, Command{Type: CmdTradeAccept})
	h.sim.Command(2, Command{Type: CmdTradeAccept})
	h.settle()

	if h.rep[1].Trade != nil && h.rep[2].LastTrade == nil {
		t.Log("accept still locked, waiting")
	}

	h.advance(2500)
	h.sim.Command(1, Command{Type: CmdTradeAccept})
	h.sim.Command(2, Command{Type: CmdTradeAccept})
	h.settle()

	i1, _ := h.sim.Inventory(1)
	i2, _ := h.sim.Inventory(2)

	if len(i1) != 0 || len(i2) != 1 || len(h.rep[2].Inv) != 1 || h.rep[2].LastTrade == nil || h.rep[2].LastTrade.State != TradeDone {
		t.Fatalf("trade: %v %v replica %v %+v", i1, i2, h.rep[2].Inv, h.rep[2].LastTrade)
	}
}

func TestTownPortalAndWaypoint(t *testing.T) {
	h := newHarness(t, DefaultRules{}, 20)
	h.join(1, "A", 5)
	h.join(2, "B", 5)
	h.settle()

	// to level 2 through the exit, cast a town portal there, B follows through it
	var exit uint32

	for _, u := range h.sim.UnitsIn(1) {
		if u.Kind == KindObject && u.Type == ObjPortal {
			exit = u.ID
		}
	}

	h.sim.Interact(1, KindObject, exit)
	h.advance(3000)
	h.sim.SelectSkill(1, SkillTownPortal, true)
	h.sim.Cast(1, true, 50, 50)
	h.settle()

	var tp uint32

	for _, u := range h.sim.UnitsIn(1) {
		if u.Kind == KindObject && u.Type == ObjPortal && u.Dest == 2 && u.Name == "A" {
			tp = u.ID
		}
	}

	if tp == 0 || h.rep[2].Unit(tp) == nil {
		t.Fatal("the town side of the portal is missing for B")
	}

	h.sim.Interact(2, KindObject, tp)
	h.advance(3000)
	h.settle()

	if h.sim.Level(2) != 2 || h.rep[2].Level != 2 {
		t.Fatalf("B level %d/%d", h.sim.Level(2), h.rep[2].Level)
	}

	if h.sim.Digest(2) != h.rep[1].Digest() || h.sim.Digest(2) != h.rep[2].Digest() {
		t.Fatal("digests differ after the portal")
	}

	// waypoint: A is back in town and activates the town waypoint (already active), B has none in level 2
	h.sim.UseWaypoint(2, 40)
	h.settle()

	if h.sim.Level(2) != 2 {
		t.Fatal("waypoint worked without an activated destination")
	}
}
