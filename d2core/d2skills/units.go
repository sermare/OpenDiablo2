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
func (h *heroUnit) Level() int     { return h.p.Stats.Level }

func (h *heroUnit) skill(id int) int {
	if s := h.p.Skills[id]; s != nil {
		return s.SkillPoints
	}

	return 0
}

// SkillLevel returns the skill points; +skills from items are not modelled.
func (h *heroUnit) SkillLevel(id int) int     { return h.skill(id) }
func (h *heroUnit) BaseSkillLevel(id int) int { return h.skill(id) }

func (h *heroUnit) Stat(name string) int {
	switch name {
	case "strength":
		return h.p.Stats.Strength
	case "dexterity":
		return h.p.Stats.Dexterity
	case "energy":
		return h.p.Stats.Energy
	case "vitality":
		return h.p.Stats.Vitality
	}

	return 0
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
func (h *heroUnit) ThrownMissile() string       { return "" }

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
	return t.m.Vitals.Defense
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

var _ d2path.Grid = (*world)(nil)
