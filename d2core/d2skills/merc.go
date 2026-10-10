package d2skills

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
)

// Hireling skills. A mercenary casts the skills of its hireling.txt row
// through the same pipeline as the hero: the merc is a heroUnit whose p is
// its owner (so kills, experience and damage credit go to the owner) and
// whose merc field carries the hireling's level, skills and combat numbers.
// Simplifications (UNVERIFIED against the exe):
//
//   - a merc has no mana (mercMana is always enough) and no cooldown beyond
//     the pipeline's own;
//   - ammunition is assumed;
//   - its auras and buffs help only the owner and the merc itself (no party);
//   - self-moving and summoning effects (Leap, Teleport, golems) are skipped.

// mercMana is the mana a merc always has (8.8).
const mercMana = 1 << 24

// mercEffect lists the effect kinds a merc can run.
var mercEffect = map[string]bool{ //nolint:gochecknoglobals // static lookup
	"self_state": true, "area_state": true, "area_hit": true, "strikes": true, "aura": true, "area_damage": true,
}

// mercCtx is the merc-specific part of a heroUnit.
type mercCtx struct {
	id     string
	name   string
	c      d2monsters.MercCast
	skills map[int]int
}

// mercID is the unit id of a merc spawn.
func mercID(owner *d2mapentity.Player, token uint32) string {
	return fmt.Sprintf("merc:%s:%d", owner.ID(), token)
}

// mercCaster returns the caster of a merc, refreshed from the cast request. A
// new spawn (token) gets a new caster, so cooldowns and states do not leak
// from the previous one.
func (e *Engine) mercCaster(c d2monsters.MercCast) *heroUnit {
	id := mercID(c.Owner, c.Token)

	h := e.mercCasters[id]
	if h == nil {
		h = &heroUnit{e: e, p: c.Owner, seed: d2rand.New(e.opt.Seed ^ 0x6d657263 ^ c.Token), cooldowns: map[int]int{},
			merc: &mercCtx{id: id, name: id, skills: map[int]int{}}}
		e.mercCasters[id] = h
	}

	h.merc.c = c

	return h
}

// mercStat is a named stat of a merc: its Str/Dex plus what its states add.
func (h *heroUnit) mercStat(name string) int {
	v := 0

	switch name {
	case "strength":
		v = h.merc.c.Str
	case "dexterity":
		v = h.merc.c.Dex
	}

	return v + h.e.setOf(h.merc.id).Stat(h.e.frame, name)
}

// mercSupported reports whether the pipeline can run a skill by name.
func (e *Engine) mercSupported(name string) bool {
	sk := e.pipe.Skills.ByName(name)

	return sk != nil && d2skill.Implemented(sk)
}

// castMerc runs one hireling skill (both the start and the do function: the
// merc's own animation already played).
func (e *Engine) castMerc(c d2monsters.MercCast) bool {
	sk := e.pipe.Skills.ByName(c.Skill)
	if sk == nil || !d2skill.Implemented(sk) {
		return false
	}

	h := e.mercCaster(c)
	h.merc.skills[sk.ID] = c.SkillLevel
	h.merc.name = fmt.Sprintf("merc(%s)", c.Owner.Name())

	tg := d2skill.Target{X: c.X, Y: c.Y}
	if c.Target != nil {
		tx, ty := c.Target.SubtilePos()
		tg = d2skill.Target{X: tx, Y: ty, Unit: e.target(c.Target), UX: tx, UY: ty}
	}

	st := e.pipe.Start(h, sk.ID, tg)

	e.Counters.Casts++
	e.emit("cast", "CAST start merc=%s skill=%q id=%d level=%d aim=(%d,%d) ok=%v reason=%s", h.merc.name, sk.Name, sk.ID, st.Level,
		tg.X, tg.Y, st.OK, reasonOf(st.Reason))

	if !st.OK {
		e.Counters.Refused++

		return false
	}

	defer e.monsters.MercCredit(c.Owner)()

	return e.runDo(c.Owner, h, sk, tg)
}

// creditFor makes damage dealt for a merc's missile count as the merc's kill.
func (e *Engine) creditFor(ownerID string) func() {
	if h := e.mercCasters[ownerID]; h != nil {
		return e.monsters.MercCredit(h.p)
	}

	return func() {}
}

// mercDefense is the Director hook for a monster's hit on a merc: Thorns
// reflect part of a melee hit.
func (e *Engine) mercDefense(owner *d2mapentity.Player, attacker *d2mapentity.Monster, melee bool, dmg int) int {
	id := mercID(owner, e.monsters.MercToken(owner))

	if pct := e.setOf(id).ThornsPct(e.frame); pct > 0 && melee && dmg > 0 {
		back := dmg * pct / 100
		e.monsters.Damage(attacker, back, owner)
		e.emit("damage", "DEFENSE merc=%s thorns=%d dmg_through=%d", id, back, dmg)
	}

	return dmg
}

// mercStatHook is the Director hook for the stats a merc's states add.
func (e *Engine) mercStatHook(owner *d2mapentity.Player, name string) int {
	return e.setOf(mercID(owner, e.monsters.MercToken(owner))).Stat(e.frame, name)
}

func (e *Engine) installMercHooks() {
	e.monsters.SetMercSkills(d2monsters.MercSkillHooks{
		Supported: e.mercSupported, Cast: e.castMerc, Defense: e.mercDefense, Stat: e.mercStatHook,
	})
}
