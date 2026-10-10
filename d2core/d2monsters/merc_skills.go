package d2monsters

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// MercCast is one cast of a hireling skill handed to the skill engine
// (d2skills): the engine runs the skills.txt pipeline with the merc as the
// caster, so missiles, auras and melee effects come from the real tables
// instead of the merc's own damage numbers.
type MercCast struct {
	Owner *d2mapentity.Player
	// Token identifies this spawn of the merc (its unit id): an aura started
	// by an earlier spawn ends when the token no longer matches.
	Token      uint32
	Skill      string
	SkillLevel int
	Level      int // merc level
	// X, Y is the merc's subtile; Target the monster aimed at (nil for a
	// buff or aura).
	X, Y   int
	Target *d2mapentity.Monster
	// the merc's own combat stats (hireling.txt at its level)
	AR, DmgMin, DmgMax, Str, Dex int
}

// MercSkillHooks connect the director to the skill engine. All are optional:
// without them the merc keeps the old behaviour (own damage for attack modes
// and a timed flag for buffs).
type MercSkillHooks struct {
	// Supported reports whether the engine can run a skill by name.
	Supported func(name string) bool
	// Cast runs the skill; false means the cast was refused (line of sight,
	// no target...).
	Cast func(c MercCast) bool
	// Defense is called when a monster's attack hits the merc; it returns
	// the damage that gets through (Thorns reflects part of it).
	Defense func(owner *d2mapentity.Player, attacker *d2mapentity.Monster, melee bool, dmg int) int
	// Stat returns a stat the merc's states (auras, Frozen Armor...) add,
	// by ItemStatCost name (armorclass, item_armor_percent, damagepercent).
	Stat func(owner *d2mapentity.Player, name string) int
}

// SetMercSkills installs the skill engine hooks.
func (d *Director) SetMercSkills(h MercSkillHooks) { d.mercHooks = h }

// MercToken is the unit id of the owner's current merc spawn (0 if none).
func (d *Director) MercToken(owner *d2mapentity.Player) uint32 {
	if u := d.mercs[owner]; u != nil {
		return u.b.ID
	}

	return 0
}

// MercAlive reports whether the merc spawn with that token is alive.
func (d *Director) MercAlive(owner *d2mapentity.Player, token uint32) bool {
	u := d.mercs[owner]

	return u != nil && u.b.ID == token && u.m.Alive()
}

// MercCredit makes kills resolved until the returned function is called count
// as the merc's (experience for the merc); the owner gets the usual share.
func (d *Director) MercCredit(owner *d2mapentity.Player) func() {
	prev := d.killer
	d.killer = d.mercs[owner]

	return func() { d.killer = prev }
}

// mercStat is a stat the merc's states add (0 without the hook).
func (d *Director) mercStat(mu *mercUnit, name string) int {
	if d.mercHooks.Stat == nil {
		return 0
	}

	return d.mercHooks.Stat(mu.owner, name)
}

// mercCastFor fills the cast description of a merc.
func (d *Director) mercCastFor(u *unit, skill string, level int, target *d2mapentity.Monster) MercCast {
	mu := u.merc
	x, y := u.m.SubtilePos()

	return MercCast{
		Owner: mu.owner, Token: u.b.ID, Skill: skill, SkillLevel: level, Level: mu.level, X: x, Y: y, Target: target,
		AR: mu.stats.AR, DmgMin: mu.stats.DmgMin, DmgMax: mu.stats.DmgMax, Str: mu.stats.Str, Dex: mu.stats.Dex,
	}
}

// ProbeMercSkills casts every skill of the merc's hireling.txt row once through
// the skill engine at the target, whatever the AI would choose (scenario
// helper: buffs and rare skills would otherwise be seen only by chance). It
// returns "skill=ok|refused|unsupported" per used slot.
func (d *Director) ProbeMercSkills(owner *d2mapentity.Player, target *d2mapentity.Monster) []string {
	u := d.mercs[owner]
	if u == nil || !u.m.Alive() {
		return nil
	}

	var out []string

	for i, s := range u.merc.rec.Skills {
		if !s.Used() {
			break
		}

		res := "unsupported"

		if d.mercHooks.Supported != nil && d.mercHooks.Supported(s.Name) {
			tgt := target
			if _, instant := tableMode(s.Mode); instant {
				tgt = nil
			}

			res = "refused"
			if d.mercHooks.Cast(d.mercCastFor(u, s.Name, u.merc.rec.SkillLevel(i, u.merc.level), tgt)) {
				res = "ok"
				u.merc.buffs[s.Name] = 1 << 30
			}
		}

		out = append(out, fmt.Sprintf("%s=%s", s.Name, res))
	}

	return out
}

// ClearHostiles removes every hostile monster and corpse (scenario helper:
// the next phase starts with a clean field).
func (d *Director) ClearHostiles() {
	for _, u := range d.sortedUnits() {
		if u.friendly() || u.b.Allied {
			continue
		}

		d.engine.RemoveEntity(u.m)
		d.forget(u)
	}
}
