package d2monster

import (
	"fmt"
	"strings"
	"testing"
)

// faWorld adds the optional host interfaces the ai_faithful_a.go ports use.
type faWorld struct {
	*fakeWorld
	stats      map[int]int
	casts      []string
	fightDone  bool
	fightLive  bool
	townRoom   bool
	killed     int
	minions    int
	cellsBlock bool
}

func newFA(dist int, inRange bool) *faWorld {
	return &faWorld{fakeWorld: newFake(dist, inRange), stats: map[int]int{}}
}

func (w *faWorld) UnitStat(_ *Brain, id int) int   { return w.stats[id] }
func (w *faWorld) SetUnitStat(_ *Brain, id, v int) { w.stats[id] = v }
func (w *faWorld) KillSelf(*Brain)                 { w.killed++ }
func (w *faWorld) InTown(*Brain) bool              { return w.townRoom }
func (w *faWorld) CountMinions(*Brain) int         { return w.minions }
func (w *faWorld) SpawnerCellsFree(*Brain) bool    { return !w.cellsBlock }
func (w *faWorld) AncientFightActive() bool        { return w.fightLive }
func (w *faWorld) FightDone() bool                 { return w.fightDone }
func (w *faWorld) PortalCount() int                { return 0 }
func (w *faWorld) SetUnitState(_ *Brain, s int, on bool) {
	w.casts = append(w.casts, fmt.Sprintf("state%x=%v", s, on))
}
func (w *faWorld) CastSkillAt(_ *Brain, skill int, m Mode, p Point) bool {
	w.casts = append(w.casts, fmt.Sprintf("skill%x/%d@%d,%d", skill, m, p.X, p.Y))

	return true
}

func faBrain(ai string, class int, skills []int, aip ...int) (*Brain, *Profile) {
	p := withSkills(profile(ai, aip...), skills...)
	b := NewBrain(7, class, Normal, p, testSeed)
	b.X, b.Y = 100, 100

	return b, p
}

// TestFaithfulARegistered: every ported name is implemented with the exe's
// verified target mode, and none of them is a generic stand-in think function.
func TestFaithfulARegistered(t *testing.T) {
	for _, n := range FaithfulA() {
		d, ok := Lookup(n)
		if !ok || !d.Implemented {
			t.Errorf("%s not registered", n)

			continue
		}

		if m, ok := AITargetMode(n); !ok || m != d.TargetMode {
			t.Errorf("%s: target mode %d, exe table %d", n, d.TargetMode, m)
		}

		if strings.ToLower(n)[0] > 'm' {
			t.Errorf("%s is not in A..M", n)
		}
	}
}

// TestFaithfulAActions drives one think per row with aip columns of 0 or 100
// (so the percent rolls are certain) and checks the requested action.
func TestFaithfulAActions(t *testing.T) {
	type row struct {
		name    string
		ai      string
		class   int
		skills  []int
		aip     []int
		dist    int
		inRange bool
		pre     func(*Brain, *faWorld)
		log     string // exact last action, "" = none
		wake    int    // expected Wake when no action, -1 = skip
		check   func(*testing.T, *Brain, *faWorld)
	}

	rows := []row{
		{name: "larva out of range chases", ai: "MaggotLarva", aip: []int{0, 0, 100, 9}, dist: 6, log: "walk-target/1"},
		{name: "larva out of range stalls aip4", ai: "MaggotLarva", aip: []int{0, 0, 0, 9}, dist: 6, wake: 9},
		{name: "larva in range attacks", ai: "MaggotLarva", aip: []int{100, 5, 0, 9}, dist: 1, inRange: true, log: "attack4",
			check: func(t *testing.T, b *Brain, _ *faWorld) {
				if b.Scratch[0] != 1 {
					t.Errorf("attacked flag = %d", b.Scratch[0])
				}
			}},
		{name: "larva just attacked stalls aip2", ai: "MaggotLarva", aip: []int{100, 5, 0, 9}, dist: 1, inRange: true, wake: 5,
			pre: func(b *Brain, _ *faWorld) { b.Scratch[0] = 1 }},
		{name: "catapult fires without target", ai: "Catapult", aip: []int{100}, dist: 20, log: "attack4"},
		{name: "catapult waits 15", ai: "Catapult", aip: []int{0}, dist: 20, wake: 15},
		{name: "buffy sleeps 200", ai: "Buffy", wake: 200},
		{name: "death mauler attacks", ai: "DeathMauler", aip: []int{100}, dist: 1, inRange: true, log: "attack4"},
		{name: "death mauler casts at range", ai: "DeathMauler", skills: []int{0}, aip: []int{0, 0, 10, 100}, dist: 6, log: "cast0"},
		{name: "death mauler walks", ai: "DeathMauler", aip: []int{0, 100, 0, 0}, dist: 6, log: "walk-target/0"},
		{name: "death mauler waits 15", ai: "DeathMauler", aip: []int{0, 0, 0, 0}, dist: 6, wake: 15},
		{name: "scimitar approaches by range", ai: "FlyingScimitar", aip: []int{100, 0, 3, 0}, dist: 6, log: "walk-to(104,100)"},
		{name: "scimitar attacks", ai: "FlyingScimitar", aip: []int{0, 100, 3, 0}, dist: 1, inRange: true, log: "attack4"},
		{name: "scimitar stalls aip3", ai: "FlyingScimitar", aip: []int{0, 0, 3, 0}, dist: 1, inRange: true, wake: 3},
		{name: "blood lord approaches", ai: "BloodLord", aip: []int{0, 100, 0, 4}, dist: 6, log: "walk-target/0"},
		{name: "blood lord casts skill", ai: "BloodLord", skills: []int{0}, aip: []int{100, 0, 100, 4}, dist: 1, inRange: true, log: "cast0"},
		{name: "blood lord attacks", ai: "BloodLord", aip: []int{100, 0, 0, 4}, dist: 1, inRange: true, log: "attack4"},
		{name: "clawviper in range stalls aip5", ai: "ClawViper", aip: []int{0, 0, 0, 0, 6}, dist: 1, inRange: true, wake: 6},
		{name: "clawviper A1", ai: "ClawViper", aip: []int{0, 0, 100, 100, 6}, dist: 1, inRange: true, log: "attack4"},
		{name: "clawviper A2", ai: "ClawViper", aip: []int{0, 0, 100, 0, 6}, dist: 1, inRange: true, log: "attack5"},
		{name: "clawviper casts and buffs", ai: "ClawViper", skills: []int{0}, aip: []int{100, 10, 0, 0, 6, 1}, dist: 6, log: "cast0",
			check: func(t *testing.T, b *Brain, w *faWorld) {
				if b.Scratch[0] != 1 || len(w.casts) == 0 || w.casts[len(w.casts)-1] != "state5a=true" {
					t.Errorf("buff: scratch=%d states=%v", b.Scratch[0], w.casts)
				}
			}},
		{name: "clawviper-ex A2 in range", ai: "ClawViperEx", aip: []int{0, 0, 100, 0, 6}, dist: 1, inRange: true, log: "attack5"},
		{name: "clawviper-ex A1 with cooldown", ai: "ClawViperEx", aip: []int{0, 0, 0, 100, 6, 0, 10, 8}, dist: 6, log: "attack4", pre: func(b *Brain, w *faWorld) { w.frame = 1 },
			check: func(t *testing.T, b *Brain, _ *faWorld) {
				if b.Scratch[1] != 9 {
					t.Errorf("cooldown frame = %d", b.Scratch[1])
				}
			}},
		{name: "clawviper-ex A1 on cooldown stalls", ai: "ClawViperEx", aip: []int{0, 0, 0, 100, 6, 0, 10, 8}, dist: 6, wake: 7,
			pre: func(b *Brain, w *faWorld) { w.frame = 1; b.Scratch[1] = 50 }},
		{name: "corrupt lancer charges", ai: "CorruptLancer", aip: []int{0, 0, 0, 0, 10}, dist: 12, log: "run-target/0",
			check: func(t *testing.T, b *Brain, _ *faWorld) {
				if b.Scratch[0] != 1 {
					t.Errorf("charging = %d", b.Scratch[0])
				}
			}},
		{name: "corrupt lancer charge ends in a skill", ai: "CorruptLancer", skills: []int{0, 1, 2}, aip: []int{0, 0, 0, 0, 10, 100}, dist: 2, inRange: true, log: "cast0",
			pre: func(b *Brain, _ *faWorld) { b.Scratch[0] = 1 }},
		{name: "corrupt lancer third skill", ai: "CorruptLancer", skills: []int{0, 1, 2}, aip: []int{0, 0, 0, 0, 10, 0, 0, 100}, dist: 2, inRange: true, log: "cast2",
			pre: func(b *Brain, _ *faWorld) { b.Scratch[0] = 1 }},
		{name: "corrupt lancer plain A1", ai: "CorruptLancer", aip: []int{0, 0, 0, 0, 10}, dist: 2, inRange: true, log: "attack4",
			pre: func(b *Brain, _ *faWorld) { b.Scratch[0] = 1 }},
		{name: "corrupt lancer walks within 3", ai: "CorruptLancer", aip: []int{100, 0, 0, 0, 10}, dist: 5, log: "walk-target/3"},
		{name: "corrupt lancer stalls aip3", ai: "CorruptLancer", aip: []int{0, 0, 4, 0, 10}, dist: 5, wake: 4},
		{name: "hydra expires", ai: "Hydra", dist: 6, wake: -1, pre: func(b *Brain, w *faWorld) { b.Scratch[0] = 3; w.frame = 10 },
			check: func(t *testing.T, b *Brain, w *faWorld) {
				if w.killed != 1 {
					t.Errorf("not killed (%d)", w.killed)
				}
			}},
		{name: "bone wall waits 15", ai: "BoneWall", wake: 15},
		{name: "7T illusion first think attacks west", ai: "7TIllusion", dist: 6, log: "attack4",
			check: func(t *testing.T, b *Brain, _ *faWorld) {
				if b.Scratch[1] != 1 {
					t.Errorf("flag = %d", b.Scratch[1])
				}
			}},
		{name: "7T illusion second think dies", ai: "7TIllusion", dist: 6, wake: -1, pre: func(b *Brain, _ *faWorld) { b.Scratch[1] = 1 },
			check: func(t *testing.T, b *Brain, w *faWorld) {
				if w.killed != 1 {
					t.Errorf("not killed")
				}
			}},
		{name: "elemental beast starts exploding", ai: "ElementalBeast", aip: []int{0, 10, 7}, dist: 5, log: "attack8",
			check: func(t *testing.T, b *Brain, _ *faWorld) {
				if b.Scratch[0] != 1 {
					t.Errorf("phase = %d", b.Scratch[0])
				}
			}},
		{name: "elemental beast far stalls aip3", ai: "ElementalBeast", aip: []int{0, 3, 7}, dist: 5, wake: 7},
		{name: "elemental beast detonates", ai: "ElementalBeast", aip: []int{0, 3, 7}, dist: 1, inRange: true, log: "attack8",
			pre: func(b *Brain, _ *faWorld) { b.Scratch[0] = 1 },
			check: func(t *testing.T, b *Brain, w *faWorld) {
				if b.Scratch[0] != 2 || b.Wake != w.frame+1 {
					t.Errorf("phase=%d wake=%d", b.Scratch[0], b.Wake)
				}
			}},
		{name: "elemental beast dies in phase 2", ai: "ElementalBeast", aip: []int{0, 3, 7}, dist: 1, wake: -1,
			pre: func(b *Brain, _ *faWorld) { b.Scratch[0] = 2 },
			check: func(t *testing.T, b *Brain, w *faWorld) {
				if w.killed != 1 {
					t.Errorf("not killed")
				}
			}},
		{name: "maggot egg hatches", ai: "MaggotEgg", skills: []int{0}, aip: []int{9, 100}, dist: 3, log: "cast0",
			check: func(t *testing.T, b *Brain, _ *faWorld) {
				if b.Scratch[0] != 1 || b.Wake != 9 {
					t.Errorf("phase=%d wake=%d", b.Scratch[0], b.Wake)
				}
			}},
		{name: "maggot egg removed after hatching", ai: "MaggotEgg", skills: []int{0}, aip: []int{9, 100}, dist: 3, wake: -1,
			pre: func(b *Brain, _ *faWorld) { b.Scratch[0] = 1 },
			check: func(t *testing.T, b *Brain, w *faWorld) {
				if w.killed != 1 || b.Wake != 9 {
					t.Errorf("killed=%d wake=%d", w.killed, b.Wake)
				}
			}},
		{name: "mosquito nest dies past its limit", ai: "MosquitoNest", aip: []int{2, 30, 5}, dist: 3, wake: -1,
			pre: func(b *Brain, _ *faWorld) { b.preRan = true; b.Scratch[1] = 3 },
			check: func(t *testing.T, b *Brain, w *faWorld) {
				if w.killed != 1 {
					t.Errorf("not killed")
				}
			}},
		{name: "mosquito nest spawns", ai: "MosquitoNest", skills: []int{0}, aip: []int{2, 30, 5}, dist: 3, log: "cast0",
			pre: func(b *Brain, w *faWorld) { w.frame = 40; b.preRan = true; b.Scratch[0] = 0 },
			check: func(t *testing.T, b *Brain, w *faWorld) {
				if b.Scratch[1] != 1 || b.Scratch[0] != 45 {
					t.Errorf("count=%d next=%d", b.Scratch[1], b.Scratch[0])
				}
			}},
		{name: "mosquito nest blocked cells wait 25", ai: "MosquitoNest", skills: []int{0}, aip: []int{2, 30, 5}, dist: 3, wake: 25,
			pre: func(b *Brain, w *faWorld) { w.frame = 0; w.cellsBlock = true; b.Scratch[0] = 0 }},
		{name: "foul crow nest far waits 25", ai: "FoulCrowNest", skills: []int{0}, aip: []int{50, 0, 3}, dist: 30, wake: 25},
		{name: "foul crow nest retires at its limit", ai: "FoulCrowNest", skills: []int{0}, aip: []int{50, 0, 3}, dist: 10, wake: -1,
			pre: func(b *Brain, _ *faWorld) { b.preRan = true; b.Scratch[1] = 3 },
			check: func(t *testing.T, b *Brain, w *faWorld) {
				if w.killed != 1 {
					t.Errorf("not killed")
				}
			}},
		{name: "minion spawner casts", ai: "MinionSpawner", skills: []int{0}, aip: []int{3, 4, 60, 30}, dist: 10, log: "cast0",
			check: func(t *testing.T, b *Brain, _ *faWorld) {
				if b.Scratch[1] != 1 || b.Scratch[0] != 60 {
					t.Errorf("casts=%d next=%d", b.Scratch[1], b.Scratch[0])
				}
			}},
		{name: "minion spawner at the cap stalls aip2", ai: "MinionSpawner", skills: []int{0}, aip: []int{3, 4, 60, 30}, dist: 10, wake: 4,
			pre: func(b *Brain, w *faWorld) { w.minions = 4 }},
		{name: "minion spawner too far stalls aip2", ai: "MinionSpawner", skills: []int{0}, aip: []int{3, 4, 60, 30}, dist: 31, wake: 4},
		{name: "hell meteor stalls aip2", ai: "HellMeteor", skills: []int{0}, aip: []int{0, 12, 5}, dist: 20, wake: 12},
		{name: "gargoyle fires within an axis", ai: "GargoyleTrap", skills: []int{0}, aip: []int{20, 100, 7, 3}, dist: 6, log: "cast0",
			check: func(t *testing.T, b *Brain, _ *faWorld) {
				if b.Scratch[0] != 7 {
					t.Errorf("pause = %d", b.Scratch[0])
				}
			}},
		{name: "gargoyle pause", ai: "GargoyleTrap", skills: []int{0}, aip: []int{20, 100, 7, 3}, dist: 6, wake: 7,
			pre: func(b *Brain, _ *faWorld) { b.Scratch[0] = 7 }},
		{name: "gargoyle stalls aip4", ai: "GargoyleTrap", skills: []int{0}, aip: []int{20, 0, 7, 3}, dist: 6, wake: 3},
		{name: "frozen horror casts", ai: "FrozenHorror", skills: []int{0}, aip: []int{0, 0, 100, 5}, dist: 6, log: "cast0",
			pre: func(b *Brain, _ *faWorld) { b.Profile.Skills[0].Level = 3 }},
		{name: "frozen horror drops its aura and attacks", ai: "FrozenHorror", aip: []int{100, 0, 0, 5}, dist: 1, inRange: true, log: "attack4",
			pre: func(b *Brain, w *faWorld) { w.states[0xc] = true },
			check: func(t *testing.T, b *Brain, w *faWorld) {
				if len(w.casts) != 1 || w.casts[0] != "statec=false" {
					t.Errorf("states = %v", w.casts)
				}
			}},
		{name: "ancient C melee", class: ancientC, ai: "Ancient", aip: []int{0, 0, 100}, dist: 1, inRange: true, log: "attack"},
		{name: "ancient A skill past the target", class: ancientA, ai: "Ancient", skills: []int{0}, aip: []int{10, 100, 0, 5}, dist: 6, log: "cast0"},
		{name: "ancient B kites", class: ancientB, ai: "Ancient", aip: []int{10, 0, 0, 100, 6}, dist: 1, inRange: true, log: "walk-to(94,100)",
			check: func(t *testing.T, b *Brain, _ *faWorld) {
				if b.Wake != 25 {
					t.Errorf("wake = %d", b.Wake)
				}
			}},
		{name: "ancient waits during the quest fight", class: ancientC, ai: "Ancient", aip: []int{0, 0, 100}, dist: 1, inRange: true, wake: 25,
			pre: func(b *Brain, w *faWorld) { w.fightLive = true }},
		{name: "arach wounded backs off", ai: "Arach", skills: []int{}, aip: []int{0, 0, 0, 50, 0}, dist: 1, inRange: true, log: "walk-to(96,100)",
			pre: func(b *Brain, _ *faWorld) { b.Scratch[0] = 1; b.HPPercent = 40 }},
		{name: "arach recovered charges", ai: "Arach", aip: []int{0, 0, 100, 50, 0}, dist: 1, inRange: true, log: "walk-target/0",
			pre: func(b *Brain, _ *faWorld) { b.Scratch[0] = 1; b.HPPercent = 90 }},
		{name: "arach strikes", ai: "Arach", aip: []int{100, 0, 0, 0, 0}, dist: 1, inRange: true, log: "attack4"},
		{name: "arach retreats to cast", ai: "Arach", skills: []int{0}, aip: []int{0, 0, 0, 0, 50}, dist: 1, inRange: true, log: "cast0",
			pre: func(b *Brain, _ *faWorld) { b.HPPercent = 20 }},
		{name: "blood hawk dives", ai: "BloodHawk", aip: []int{100, 0, 0, 0, 40}, dist: 8, log: "walk-target/0"},
		{name: "blood hawk follows through", ai: "BloodHawk", aip: []int{0, 0, 0, 0, 40}, dist: 1, inRange: true, log: "attack4",
			pre: func(b *Brain, _ *faWorld) { b.Scratch[0] = 1 }},
		{name: "arcane tower first volley shot", ai: "ArcaneTower", skills: []int{0}, aip: []int{2, 6, 30, 3, 40, 8}, dist: 5, log: "cast0",
			check: func(t *testing.T, b *Brain, _ *faWorld) {
				if b.Scratch != [3]int{0, 1, 6} {
					t.Errorf("scratch = %v", b.Scratch)
				}
			}},
		{name: "arcane tower volley ends", ai: "ArcaneTower", skills: []int{0}, aip: []int{2, 6, 30, 3, 40, 8}, dist: 5, log: "cast0",
			pre: func(b *Brain, _ *faWorld) { b.Scratch = [3]int{0, 1, 0} },
			check: func(t *testing.T, b *Brain, _ *faWorld) {
				if b.Scratch != [3]int{1, 3, 30} {
					t.Errorf("scratch = %v", b.Scratch)
				}
			}},
		{name: "arcane tower attack volley", ai: "ArcaneTower", aip: []int{2, 6, 30, 3, 40, 8}, dist: 5, log: "attack4",
			check: func(t *testing.T, b *Brain, _ *faWorld) {
				if b.Scratch != [3]int{0, 1, 8} {
					t.Errorf("scratch = %v", b.Scratch)
				}
			}},
		{name: "arcane tower waits for its timer", ai: "ArcaneTower", aip: []int{2, 6, 30, 3, 40, 8}, dist: 5, wake: 10,
			pre: func(b *Brain, _ *faWorld) { b.Scratch = [3]int{0, 1, 99} }},
		{name: "mosquito attacks", ai: "Mosquito", aip: []int{0, 0, 100, 0, 3}, dist: 1, inRange: true, log: "attack4"},
		{name: "mosquito casts", ai: "Mosquito", skills: []int{0}, aip: []int{0, 0, 100, 100, 3}, dist: 1, inRange: true, log: "cast0"},
		{name: "mosquito chases", ai: "Mosquito", aip: []int{0, 0, 0, 0, 3}, dist: 6, log: "walk-target/0"},
		{name: "mosquito retreats", ai: "Mosquito", aip: []int{0, 0, 0, 0, 3}, dist: 6, log: "walk-to(90,100)",
			pre: func(b *Brain, _ *faWorld) { b.Scratch = [3]int{2, 0, 0} },
			check: func(t *testing.T, b *Brain, _ *faWorld) {
				if b.Scratch[0] != 1 {
					t.Errorf("phase = %d", b.Scratch[0])
				}
			}},
		{name: "finger mage flees when alone and hurt", ai: "FingerMage", aip: []int{0, 0, 0, 50, 8, 5, 12, 6}, dist: 6, log: "walk-to(91,100)",
			pre: func(b *Brain, _ *faWorld) { b.HPPercent = 20 },
			check: func(t *testing.T, b *Brain, _ *faWorld) {
				if b.Scratch[0] != 1 {
					t.Errorf("phase = %d", b.Scratch[0])
				}
			}},
		{name: "finger mage melee", ai: "FingerMage", aip: []int{100, 0, 0, 0, 8, 5, 12, 6}, dist: 1, inRange: true, log: "attack4"},
		{name: "finger mage approaches", ai: "FingerMage", aip: []int{0, 0, 0, 0, 4, 5, 12, 6}, dist: 9, log: "walk-target/7"},
		{name: "good npc ranged waits when not idle", ai: "GoodNpcRanged", wake: 5,
			pre: func(b *Brain, _ *faWorld) { b.Mode = ModeWalk }},
		{name: "ancient statue idles without the quest", ai: "AncientStatue", wake: 25},
	}

	for _, r := range rows {
		w := newFA(r.dist, r.inRange)

		class := r.class
		if class == 0 {
			class = 1
		}

		b, _ := faBrain(r.ai, class, r.skills, r.aip...)
		def, _ := Lookup(r.ai)
		b.SetAI(def)

		if r.pre != nil {
			r.pre(b, w)
		}

		Tick(w, b)

		got := w.last()
		switch {
		case r.log == "attack": // A1 or A2
			if got != "attack4" && got != "attack5" {
				t.Errorf("%s: log %v", r.name, w.log)
			}
		case got != r.log:
			t.Errorf("%s: last action %q, want %q (all %v)", r.name, got, r.log, w.log)
		}

		if r.log == "" && r.wake > 0 && b.Wake != r.wake {
			t.Errorf("%s: wake %d, want %d", r.name, b.Wake, r.wake)
		}

		if r.check != nil {
			r.check(t, b, w)
		}
	}
}

// TestFaithfulARollOrder checks that the draws come in the exe's order and
// number: the brain's generator must equal a shadow generator advanced by the
// same rolls.
func TestFaithfulARollOrder(t *testing.T) {
	t.Run("HellMeteor x then y", func(t *testing.T) {
		w := newFA(30, false)
		b, _ := faBrain("HellMeteor", 1, []int{0}, 100, 5, 4)
		def, _ := Lookup("HellMeteor")
		b.SetAI(def)

		sh := shadow(b)
		sh.Roll(100)
		x := 100 + int(sh.Roll(8)) - 4
		y := 100 + int(sh.Roll(8)) - 4

		var gotX, gotY int
		// MoveTo-free: Cast logs only the slot, so rebuild the point from a world hook
		hw := &pointWorld{faWorld: w}
		Tick(hw, b)

		gotX, gotY = hw.p.X, hw.p.Y
		if gotX != x || gotY != y {
			t.Errorf("target (%d,%d), want (%d,%d)", gotX, gotY, x, y)
		}

		if *b.Seed != *sh {
			t.Errorf("rng state differs from the three expected draws")
		}
	})

	t.Run("Baboon rage draws", func(t *testing.T) {
		for id := uint32(1); id < 40; id++ {
			w := newFA(1, true)
			w.stats[0x4a] = 80
			p := profile("Baboon", 50, 0, 0, 100, 8)
			p.Walk, p.Run = 100, 150
			b := NewBrain(id, 1, Normal, p, testSeed)
			b.X, b.Y = 100, 100
			b.HPPercent = 10
			b.Aggressive = true
			def, _ := Lookup("Baboon")
			b.SetAI(def)

			sh := shadow(b)
			enters := sh.Roll(100) < 50

			Tick(w, b)

			if enters {
				ticks := int(sh.Roll(5)) + 2
				if b.Scratch[0] != ticks {
					t.Fatalf("id %d: rage ticks %d, want %d", id, b.Scratch[0], ticks)
				}

				bonus := (8 * 80) >> 3
				if b.Scratch[2] != bonus || w.stats[0x4a] != 80+bonus {
					t.Fatalf("id %d: bonus %d stat %d", id, b.Scratch[2], w.stats[0x4a])
				}
			} else if b.Scratch[0] != 0 {
				t.Fatalf("id %d: raged without the 50%% roll", id)
			}

			if *b.Seed != *sh && enters {
				t.Fatalf("id %d: rng state differs", id)
			}
		}
	})

	t.Run("Baboon rage ends and restores the stat", func(t *testing.T) {
		w := newFA(1, true)
		w.stats[0x4a] = 90
		b, _ := faBrain("Baboon", 1, nil, 50, 0, 0, 100, 8)
		def, _ := Lookup("Baboon")
		b.SetAI(def)
		b.Scratch = [3]int{1, 0, 10}
		b.HPPercent = 80

		Tick(w, b)

		if b.Scratch[0] != 0 || w.stats[0x4a] != 80 {
			t.Errorf("scratch=%v stat=%d", b.Scratch, w.stats[0x4a])
		}
	})

	t.Run("FoulCrowNest stall is 20 + step mod 10", func(t *testing.T) {
		w := newFA(10, false)
		b, _ := faBrain("FoulCrowNest", 1, nil, 50, 0, 3)
		def, _ := Lookup("FoulCrowNest")
		b.SetAI(def)
		sh := shadow(b)
		want := 20 + int(sh.Step()%10)

		Tick(w, b)

		if b.Wake != want {
			t.Errorf("wake %d, want %d", b.Wake, want)
		}
	})

	t.Run("InvisoPet scatters round the leader", func(t *testing.T) {
		w := newFA(10, false)
		pw := &pointWorld{faWorld: w}
		b, _ := faBrain("InvisoPet", 1, nil, 6, 20)
		def, _ := Lookup("InvisoPet")
		b.SetAI(def)

		l, _ := faBrain("Skeleton", 2, nil)
		l.X, l.Y = 140, 160
		l.AddMinion(b)

		w.frame = 5

		sh := shadow(b)
		x := int(sh.Roll(12)) - 6 + 140
		y := int(sh.Roll(12)) - 6 + 160

		Tick(pw, b)

		if len(w.casts) != 1 || w.casts[0] != fmt.Sprintf("skill124/9@%d,%d", x, y) {
			t.Errorf("casts %v, want skill124 at %d,%d", w.casts, x, y)
		}

		if b.Scratch[0] != 25 {
			t.Errorf("next cast frame %d", b.Scratch[0])
		}
	})

	t.Run("GoodNpcRanged rolls 30 then 30", func(t *testing.T) {
		for id := uint32(1); id < 30; id++ {
			w := newFA(6, false)
			p := profile("GoodNpcRanged")
			b := NewBrain(id, 5, Normal, p, testSeed)
			b.X, b.Y = 100, 100
			def, _ := Lookup("GoodNpcRanged")
			b.SetAI(def)

			sh := shadow(b)
			first := sh.Roll(100) < 30

			Tick(w, b)

			switch {
			case first:
				if w.last() != "attack4" {
					t.Fatalf("id %d: want attack, got %v", id, w.log)
				}
			default:
				second := sh.Roll(100) < 30
				if second && len(w.log) == 0 {
					t.Fatalf("id %d: want circle, got none", id)
				}
			}
		}
	})

	t.Run("Hydra fires on a 60% roll after expiry check", func(t *testing.T) {
		for id := uint32(1); id < 30; id++ {
			w := newFA(6, false)
			p := withSkills(profile("Hydra"), 0)
			b := NewBrain(id, 1, Normal, p, testSeed)
			b.X, b.Y = 100, 100
			def, _ := Lookup("Hydra")
			b.SetAI(def)

			sh := shadow(b)
			fires := sh.Roll(100) < 60

			Tick(w, b)

			if fires != (w.last() == "cast0") {
				t.Fatalf("id %d: fires=%v log=%v", id, fires, w.log)
			}

			if !fires && b.Wake != 10 {
				t.Fatalf("id %d: wake %d", id, b.Wake)
			}
		}
	})

	t.Run("BloodHawk wanders with speed override", func(t *testing.T) {
		w := newFA(8, false)
		b, _ := faBrain("BloodHawk", 1, nil, 0, 0, 0, 0, 40)
		def, _ := Lookup("BloodHawk")
		b.SetAI(def)

		sh := shadow(b)
		sh.Roll(100) // aip1 dive %
		sh.Roll(100) // aip2 wander split (0: always the short wander)
		sh.Step()    // Wander: axis step
		sh.Roll(3)
		sh.Step()
		sh.Step()

		Tick(w, b)

		if len(w.log) != 1 || !strings.HasPrefix(w.log[0], "walk-to(") {
			t.Fatalf("log %v", w.log)
		}

		if *b.Seed != *sh {
			t.Errorf("rng state differs: wander draws are 4 steps after the two rolls")
		}
	})

	t.Run("Ancient C picks A1 or A2 from roll(2)", func(t *testing.T) {
		for id := uint32(1); id < 20; id++ {
			w := newFA(1, true)
			b := NewBrain(id, ancientC, Normal, profile("Ancient", 0, 0, 100), testSeed)
			b.X, b.Y = 100, 100
			def, _ := Lookup("Ancient")
			b.SetAI(def)

			sh := shadow(b)
			sh.Roll(100)
			want := "attack5"
			if sh.Roll(2) != 0 {
				want = "attack4"
			}

			Tick(w, b)

			if w.last() != want {
				t.Fatalf("id %d: %s, want %s", id, w.last(), want)
			}
		}
	})
}

// TestAncientStatueCastsAfterTheFight covers the quest-gated cast.
func TestAncientStatueCastsAfterTheFight(t *testing.T) {
	w := newFA(5, false)
	w.fightDone = true
	pw := &pointWorld{faWorld: w}
	b, _ := faBrain("AncientStatue", 1, nil)
	def, _ := Lookup("AncientStatue")
	b.SetAI(def)

	Tick(pw, b)

	if len(w.casts) != 1 || w.casts[0] != "skill12e/4@100,100" {
		t.Errorf("casts = %v", w.casts)
	}

	// still guarded by state 0x92
	w2 := newFA(5, false)
	w2.fightDone = true
	w2.states[0x92] = true
	b2, _ := faBrain("AncientStatue", 1, nil)
	b2.SetAI(def)
	Tick(&pointWorld{faWorld: w2}, b2)

	if len(w2.casts) != 0 || b2.Wake != 25 {
		t.Errorf("guarded statue: casts=%v wake=%d", w2.casts, b2.Wake)
	}
}

// pointWorld records the ground point of Cast.
type pointWorld struct {
	*faWorld
	p Point
}

func (w *pointWorld) Cast(b *Brain, slot int, t Target) bool {
	w.p = Point{t.X, t.Y}

	return w.faWorld.Cast(b, slot, t)
}

// TestSpeedBoost covers the run/walk velocity excess helper.
func TestSpeedBoost(t *testing.T) {
	for _, c := range []struct{ walk, run, want int }{
		{0, 100, 0}, {100, 100, 0}, {100, 90, 0}, {100, 150, 50}, {100, 400, 0x78}, {50, 75, 50},
	} {
		p := &Profile{Walk: c.walk, Run: c.run}
		if got := p.speedBoost(); got != c.want {
			t.Errorf("walk %d run %d: %d, want %d", c.walk, c.run, got, c.want)
		}
	}
}
