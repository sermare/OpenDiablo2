package d2skills

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// The retaliation of the cold armors (wave 23, VERIFIED in the exe unless noted).
//
// SRVDO_FrozenArmorState (0x5c7540) registers the skill's auraevent1..3 on the caster with the event
// handler table at 0x72fd48. The three armors use:
//   - Frozen Armor, damagedinmelee -> handler 2 (0x5c8e20): the unit that damaged the wearer in melee
//     takes a damage struct with only a freeze length (calc1 of the skill, evaluated at the wearer's
//     level of the skill) and flag 0x20 (no further events). The handler returns at once when the
//     struct passed by the event has a physical part below 1.
//   - Chilling Armor, hitbymissile -> handler 1 (0x5c8d00): a hostile missile whose record has the
//     bits4 & 0x200 flag makes the wearer fire srvmissilea (chillingarmorbolt) at the missile's owner.
//     Wave 44 re-read 0x5c8d00: the flag is the missiles.txt ReturnFire column (bit 9 of the record flag word
//     filled by the loader 0x664230, VERIFIED in w37), the missile's owner (UNIT_FindOwnerUnitIfFlagged) must be
//     hostile to the wearer, a negative level argument becomes 1, and the bolt is created with kind 0x20 aimed
//     at the owner's position.
//   - Shiver Armor, attackedinmelee -> handler 3 (0x5c8f20): applies the skill's own elemental damage
//     to the attacker. The event fires from COMBAT_BuildAttackerDamage (0x5794e0) only when the
//     attacker is a player or a hireling, so ordinary monsters never trigger it; the monster path of
//     this port therefore does nothing for Shiver Armor (rivals and mercenaries: not wired, UNVERIFIED).
//
// The event ids are read from the dispatch sites: 0 fired at the victim of a missile contact
// (MISSILE_ProcessHitOrExpire), 1 at the victim of a melee hit (COMBAT_ApplyDamageToUnit with its
// resist flag 0), 3 from BuildAttackerDamage. The skills.txt names for the ids are inferred.

// frozenArmorRetaliation freezes the monster that hit the hero in melee for the physical part of the
// hit (phys, whole life) above zero. Called with the damage the hero actually takes.
func (e *Engine) frozenArmorRetaliation(p *d2mapentity.Player, attacker *d2mapentity.Monster, melee bool, phys int) string {
	if !melee || phys < 1 || attacker == nil {
		return ""
	}

	in := e.setOf(p.ID()).Get(e.frame, "frozenarmor")
	if in == nil {
		return ""
	}

	n := e.pipe.Calc1(e.hero(p), in.SkillID, maxInt(in.Level, 1))
	if n <= 0 {
		return ""
	}

	d := d2combat.Damage{FreezeLen: int32(n)}
	e.hurt(attacker, p, &d, "Frozen Armor")

	return fmt.Sprintf(" frozen_armor_freeze=%d", n)
}

// heroHitByMissile is the hitbymissile event of the hero: Chilling Armor fires its bolt back at the
// owner of a missile that has the ReturnFire flag.
func (e *Engine) heroHitByMissile(p *d2mapentity.Player, attacker *d2mapentity.Monster, missile string) {
	in := e.setOf(p.ID()).Get(e.frame, "chillingarmor")
	if in == nil || attacker == nil {
		return
	}

	ms := e.pipe.Missiles.ByName(missile)
	if ms == nil || !ms.ReturnFire {
		return
	}

	name := e.pipe.SrvMissileA(in.SkillID)
	if name == "" {
		return
	}

	h := e.hero(p)
	hx, hy := h.Pos()
	ax, ay := attacker.SubtilePos()

	e.pipe.CastTrap(h, in.SkillID, name, hx, hy, d2skill.Target{X: ax, Y: ay})
	e.emit("hit", "SKILL chilling_armor_bolt hero=%s at=(%d,%d) missile=%s", p.Name(), ax, ay, missile)
}
