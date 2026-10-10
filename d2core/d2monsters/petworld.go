package d2monsters

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2summon"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// Wiring of d2summon (Roster, Compute, templates) and the NecroPet family of
// AIs (d2common/d2monster/ai_pet.go) into the Director.
//
// Summoner is engine independent: it talks to the world through PetSpawner,
// so tests drive it with a fake. The Director implements the world side
// (directorPets) and the d2monster.PetWorld methods (Owner and Teleport are
// shared with the mercenary code, OwnerEnemy is new).
//
// UNVERIFIED: which minions run the real pet AI. Only "minion" kinds whose
// monstats AI is NecroPet or Raven do; traps, totems and walls keep their
// skill-engine drivers (d2skills registerTrap / registerTotem).

// PetSpawner is the world a Summoner creates minions in.
type PetSpawner interface {
	// Frame is the current game frame.
	Frame() int
	// SpawnPet creates a minion of monstats id key with the computed stats
	// (nil: no template, use the order's plain modifiers) near (x, y) and
	// returns its id.
	SpawnPet(owner, key string, x, y int, o *d2skill.SummonOrder, st *d2summon.Stats) (uint32, error)
	// RemovePet takes a minion off the map (evicted or expired).
	RemovePet(id uint32)
	// PetAlive reports whether a minion still lives.
	PetAlive(id uint32) bool
}

// Summoner keeps one d2summon.Roster per owner.
type Summoner struct {
	world     PetSpawner
	templates *d2summon.Templates
	diff      d2summon.Difficulty
	rng       *d2rand.Seed
	rosters   map[string]*d2summon.Roster
}

// NewSummoner creates a Summoner. templates may be nil: minions are then
// spawned with the plain monstats numbers and the order's modifiers.
func NewSummoner(w PetSpawner, t *d2summon.Templates, diff d2summon.Difficulty, seed uint32) *Summoner {
	return &Summoner{world: w, templates: t, diff: diff, rng: d2rand.New(seed), rosters: map[string]*d2summon.Roster{}}
}

// Roster returns the roster of an owner (created on first use).
func (s *Summoner) Roster(owner string) *d2summon.Roster {
	r := s.rosters[owner]
	if r == nil {
		r = &d2summon.Roster{}
		s.rosters[owner] = r
	}

	return r
}

// Cast carries out a summon order: dead minions are dropped from the roster,
// Roster.Plan decides how many to create and whom to evict, and each new
// minion gets d2summon.Compute stats. place gives the position of the i-th
// of n new minions. It returns the ids created and the plan.
func (s *Summoner) Cast(owner, key string, o *d2skill.SummonOrder, passive []d2skill.StatMod,
	place func(i, n int) (x, y int)) ([]uint32, d2summon.Plan) {
	r := s.Roster(owner)

	for _, p := range r.Pets() {
		if !s.world.PetAlive(p.ID) {
			r.Remove(p.ID)
		}
	}

	plan := r.Plan(o)

	for _, id := range plan.Evict {
		s.world.RemovePet(id)
	}

	var st *d2summon.Stats

	if s.templates != nil {
		if t, ok := s.templates.ByID(key); ok {
			mods := d2summon.FromOrder(o, passive)
			if o.Level > 0 {
				if ac, th, ok := s.templates.LevelBonus(o.Level, s.diff); ok {
					mods.LevelAC, mods.LevelAR = ac, th
				}
			}

			c := d2summon.Compute(t, s.diff, mods, s.rng)
			st = &c
		}
	}

	ids := make([]uint32, 0, plan.Spawn)

	for i := 0; i < plan.Spawn; i++ {
		x, y := place(i, plan.Spawn)

		id, err := s.world.SpawnPet(owner, key, x, y, o, st)
		if err != nil {
			continue
		}

		ids = append(ids, id)
	}

	r.Apply(o, plan, ids, s.world.Frame())

	return ids, plan
}

// Step removes the minions whose lifetime ran out (call once per frame).
func (s *Summoner) Step() {
	f := s.world.Frame()

	for _, r := range s.rosters {
		for _, id := range r.Expire(f) {
			s.world.RemovePet(id)
		}
	}
}

// ---- Director side ----

// directorPets is the PetSpawner over a Director.
type directorPets struct {
	d      *Director
	owners func(id string) *d2mapentity.Player
}

func (p directorPets) Frame() int { return p.d.frame }

func (p directorPets) SpawnPet(owner, key string, x, y int, o *d2skill.SummonOrder, st *d2summon.Stats) (uint32, error) {
	stat := p.d.FindStat(key)
	if stat == nil {
		return 0, fmt.Errorf("unknown monster %q", key)
	}

	opt := MinionOptions{Owner: p.owners(owner), Kind: o.Kind, Frames: o.Frames, Tag: o.PetType, Stats: st, Level: o.Level}
	if opt.Tag == "" || opt.Tag == d2summon.Unlimited {
		opt.Tag = key
	}

	if st == nil { // no template: monstats numbers with the order's modifiers on top
		opt.HPPct, opt.HPFlat = o.HPPct, o.HPFlat

		for _, m := range o.Stats {
			switch m.Stat {
			case "damagepercent":
				opt.DamagePct += m.Value
			case "tohit":
				opt.ToHit += m.Value
			case "armorclass":
				opt.ArmorClass += m.Value
			}
		}
	}

	m, err := p.d.SpawnMinion(stat, x, y, opt)
	if err != nil {
		return 0, err
	}

	return p.d.byEntity[m.ID()].b.ID, nil
}

func (p directorPets) RemovePet(id uint32) {
	if u := p.d.units[id]; u != nil && u.ally != nil {
		p.d.expire(u)
	}
}

func (p directorPets) PetAlive(id uint32) bool {
	u := p.d.units[id]

	return u != nil && u.ally != nil && u.m.Alive()
}

// Summoner returns the Director's summoner, loading the monstats templates on
// first use.
func (d *Director) Summoner() *Summoner {
	if d.summoner != nil {
		return d.summoner
	}

	var tpl *d2summon.Templates

	if data, err := d.asset.LoadFile("/data/global/excel/monstats.txt"); err == nil {
		tpl, _ = d2summon.LoadTemplates(data)
	}

	if tpl != nil {
		if data, err := d.asset.LoadFile("/data/global/excel/monlvl.txt"); err == nil {
			_ = tpl.LoadMonLvl(data)
		}
	}

	pets := directorPets{d: d, owners: func(id string) *d2mapentity.Player {
		for _, p := range d.players() {
			if p.ID() == id {
				return p
			}
		}

		return nil
	}}

	d.summoner = NewSummoner(pets, tpl, d2summon.Difficulty(d.opt.Difficulty), d.opt.Seed)

	return d.summoner
}

// MinionByBrainID returns the minion a Summoner id names.
func (d *Director) MinionByBrainID(id uint32) *d2mapentity.Monster {
	if u := d.units[id]; u != nil && u.ally != nil {
		return u.m
	}

	return nil
}

// usesPetAI reports whether a minion is driven by the ported pet AI.
func usesPetAI(u *unit) bool {
	if u.ally == nil || u.ally.kind != "minion" || u.b.Def == nil || !u.b.Def.Implemented {
		return false
	}

	return strings.EqualFold(u.b.Def.Name, "NecroPet") || strings.EqualFold(u.b.Def.Name, "Raven")
}

// OwnerEnemy implements d2monster.PetWorld: the nearest hostile monster to
// the pet, if within radius.
func (d *Director) OwnerEnemy(b *d2monster.Brain, radius int) (d2monster.Target, int, bool) {
	t, dist, ok := d.nearestEnemy(b)

	return t, dist, ok && dist <= radius
}

var _ d2monster.PetWorld = (*Director)(nil)
