package d2skills

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// heroUnit adapts a Player to d2skill.Unit.
type heroUnit struct {
	e         *Engine
	p         *d2mapentity.Player
	seed      *d2rand.Seed
	manaFrac  int // the 8.8 fraction of mana the integer Stats.Mana cannot hold
	cooldowns map[int]int

	passives     map[string]int
	passiveFrame int
	inPassive    bool
}

func (e *Engine) hero(p *d2mapentity.Player) *heroUnit {
	h := e.heroes[p.ID()]
	if h == nil {
		h = &heroUnit{e: e, p: p, seed: d2rand.New(e.opt.Seed ^ 0x736b696c), cooldowns: map[int]int{}}
		e.heroes[p.ID()] = h
	}

	return h
}

// Hero exposes the unit adapter of a player (scenarios set skills and mana).
func (e *Engine) Hero(p *d2mapentity.Player) *HeroHandle { return &HeroHandle{e.hero(p)} }

// HeroHandle is the public view of a hero unit.
type HeroHandle struct{ h *heroUnit }

// ManaString formats the current mana.
func (h *HeroHandle) ManaString() string { return h.h.manaString() }

func (h *heroUnit) manaString() string {
	return fmt.Sprintf("%.2f/%d", float64(h.Mana())/256, h.p.Stats.MaxMana)
}

func (h *heroUnit) ID() string     { return h.p.ID() }
func (h *heroUnit) IsPlayer() bool { return true }

// Gone reports a dead hero (death animation or corpse); see Owner.Gone.
func (h *heroUnit) Gone() bool { return h.p.IsDead() }
func (h *heroUnit) Level() int { return h.p.Stats.Level }

func (h *heroUnit) skill(id int) int {
	if s := h.p.Skills[id]; s != nil {
		return s.SkillPoints
	}

	return 0
}

// SkillLevel returns the skill points; +skills from items are not modelled.
func (h *heroUnit) SkillLevel(id int) int     { return h.skill(id) }
func (h *heroUnit) BaseSkillLevel(id int) int { return h.skill(id) }

// Stat is a base attribute plus what the hero's states (auras, buffs,
// charges) and passive skills add to it.
func (h *heroUnit) Stat(name string) int {
	v := 0

	switch name {
	case "strength":
		v = h.p.Stats.Strength
	case "dexterity":
		v = h.p.Stats.Dexterity
	case "energy":
		v = h.p.Stats.Energy
	case "vitality":
		v = h.p.Stats.Vitality
	}

	return v + h.e.setOf(h.p.ID()).Stat(h.e.frame, name) + h.passive(name)
}

// passive sums the passivestat columns of the hero's passive skills (Dodge,
// Claw Mastery, Warmth...), cached for the frame; the evaluation of a calc
// that reads a stat does not see the passives (no recursion).
func (h *heroUnit) passive(name string) int {
	if h.inPassive {
		return 0
	}

	if h.passiveFrame != h.e.frame+1 {
		h.inPassive = true
		h.passives = map[string]int{}

		for id, s := range h.p.Skills {
			if s == nil || s.SkillPoints < 1 {
				continue
			}

			for _, m := range h.e.pipe.PassiveStats(h, id) {
				h.passives[m.Stat] += m.Value
			}
		}

		h.inPassive = false
		h.passiveFrame = h.e.frame + 1
	}

	return h.passives[name]
}

func (h *heroUnit) Mana() int { return h.p.Stats.Mana<<8 | h.manaFrac }

func (h *heroUnit) SetMana(v int) {
	if v < 0 {
		v = 0
	}

	h.p.Stats.Mana, h.manaFrac = v>>8, v&0xff
}

func (h *heroUnit) Pos() (x, y int) { return int(h.p.Position.X()), int(h.p.Position.Y()) }
func (h *heroUnit) InTown() bool    { return h.p.IsInTown() }
func (h *heroUnit) Roller() d2combat.Roller {
	return h.seed
}

func (h *heroUnit) AttackRating() int {
	st := h.e.asset.Records.Character.Stats[h.p.Class]
	if st == nil {
		return d2combat.PlayerAttackRating(0, h.p.Stats.Dexterity, 0)
	}

	return d2combat.PlayerAttackRating(0, h.p.Stats.Dexterity, st.ToHitFactor)
}

// WeaponDamage is the right hand weapon's range, or 1-2 bare handed (the
// game's fallback min>=1, max>=2).
func (h *heroUnit) WeaponDamage() (min, max int) {
	min, max = 1, 2

	if h.p.Equipment != nil && h.p.Equipment.RightHand != nil {
		if rec := h.e.asset.Records.Item.Weapons[h.p.Equipment.RightHand.GetItemCode()]; rec != nil && rec.MaxDamage > 0 {
			min, max = rec.MinDamage, rec.MaxDamage
		}
	}

	return min, max
}

// RangedWeaponMissile is not derived from the weapon type yet (UNVERIFIED /
// not implemented): weapons are treated as melee.
func (h *heroUnit) RangedWeaponMissile() string { return "" }

// ThrownMissile is a javelin when the scenario gives infinite ammunition (the
// weapon type is not mapped to missiles yet; UNVERIFIED), else none.
func (h *heroUnit) ThrownMissile() string {
	if h.e.opt.InfiniteAmmo {
		return "javelin"
	}

	return ""
}

func (h *heroUnit) HasAmmo() bool       { return h.e.opt.InfiniteAmmo }
func (h *heroUnit) ConsumeAmmo() bool   { return h.e.opt.InfiniteAmmo }
func (h *heroUnit) Cooldown(id int) int { return h.cooldowns[id] }
func (h *heroUnit) SetCooldown(id, until int) {
	h.cooldowns[id] = until
}

// ---- monsters as missile targets ----

type monsterTarget struct {
	e *Engine
	m *d2mapentity.Monster
}

func (e *Engine) target(m *d2mapentity.Monster) *monsterTarget {
	t := e.targets[m.ID()]
	if t == nil {
		t = &monsterTarget{e: e, m: m}
		e.targets[m.ID()] = t
	}

	return t
}

func (t *monsterTarget) ID() string     { return t.m.ID() }
func (t *monsterTarget) IsPlayer() bool { return false }
func (t *monsterTarget) Alive() bool    { return t.m.Alive() }
func (t *monsterTarget) Level() int     { return t.m.Vitals.Level }
func (t *monsterTarget) Defense(bool) int {
	set := t.e.setOf(t.m.ID())
	v := t.m.Vitals.Defense + set.Stat(t.e.frame, "armorclass")
	v += v * set.DefensePct(t.e.frame) / 100

	if v < 0 {
		v = 0
	}

	return v
}

// IsUndead and IsDemon implement d2missile.Kinded (monstats lUndead|hUndead
// and demon).
func (t *monsterTarget) IsUndead() bool {
	return t.m.Stat != nil && (t.m.Stat.IsUndeadLow || t.m.Stat.IsUndeadHigh)
}

func (t *monsterTarget) IsDemon() bool { return t.m.Stat != nil && t.m.Stat.IsDemon }

// Serial implements d2missile.Serial: the unit id Guided Arrow orders by.
func (t *monsterTarget) Serial() int { return int(t.e.monsters.UnitID(t.m)) }

// Size implements d2missile.Sized (monstats SizeX, subtracted from distances).
func (t *monsterTarget) Size() int {
	if t.m.StatEx != nil {
		return t.m.StatEx.SizeX
	}

	return 0
}

// SubPos implements d2missile.Positioned (homing, chain lightning).
func (t *monsterTarget) SubPos() (float64, float64) {
	x, y := t.m.SubtilePos()

	return float64(x) + 0.5, float64(y) + 0.5
}

// world adapts the engine to d2missile.World.
type world struct{ e *Engine }

func (w *world) Flags(x, y int) uint16 { return w.e.monsters.Grid().Flags(x, y) }
func (w *world) Frame() int            { return w.e.frame }

// IsEnemy: heroes' missiles hurt monsters (all of the director's monsters are
// hostile); monsters do not fire these missiles.
func (w *world) IsEnemy(o d2missile.Owner, t d2missile.Target) bool {
	return o.IsPlayer && !t.IsPlayer()
}

// Targets returns the living monsters whose footprint (a square of half size
// SizeX/2, at least 1) covers the subtile.
func (w *world) Targets(x, y int) []d2missile.Target {
	var out []d2missile.Target

	for _, m := range w.e.monsters.Monsters() {
		if !m.Alive() {
			continue
		}

		r := 1
		if m.StatEx != nil && m.StatEx.SizeX/2 > r {
			r = m.StatEx.SizeX / 2
		}

		mx, my := m.SubtilePos()
		if chebyshev(mx-x, my-y) <= r {
			out = append(out, w.e.target(m))
		}
	}

	return out
}

// EnemiesWithin implements d2missile.Finder (the scan 0x569510 with the filter
// 0x569100, verified): the living monsters of the owner's enemies whose
// subtile position is within radius subtiles (euclidean, squared compare) of
// (x, y), outside a town and in line of sight of the owner (wall bit 4 along
// the line owner -> candidate). Only hero owners are served: the engine has no
// monster-fired homing missiles and heroes are not enemy candidates (no PvP
// missiles). The listing is sorted by unit id; the sim keeps the lowest.
func (w *world) EnemiesWithin(o d2missile.Owner, x, y float64, radius int) []d2missile.Target {
	h := w.e.heroes[o.ID]
	if h == nil || !o.IsPlayer || (h.p.IsInTown() && !w.e.opt.IgnoreTown) {
		return nil
	}

	ox, oy := h.Pos()

	var cs []d2missile.Candidate

	for _, m := range w.e.monsters.Monsters() {
		mx, my := m.SubtilePos()
		cs = append(cs, d2missile.Candidate{Target: w.e.target(m), X: mx, Y: my})
	}

	return d2missile.Scan(w.e.monsters.Grid(), d2path.Point{X: ox, Y: oy}, x, y, radius, cs)
}

var (
	_ d2path.Grid      = (*world)(nil)
	_ d2missile.Finder = (*world)(nil)
)
