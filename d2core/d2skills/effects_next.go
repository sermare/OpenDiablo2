package d2skills

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// unitState puts a state on the one monster the effect names (Taunt).
func (e *Engine) unitState(p *d2mapentity.Player, sk *d2skill.Skill, ef *d2skill.Effect) {
	mt, _ := ef.Target.(*monsterTarget)
	if mt == nil || !mt.m.Alive() {
		return
	}

	frames := ef.Frames
	if frames <= 0 {
		frames = defaultCurse // taunt has no auralencalc in the data
	}

	e.applyMonsterState(mt.m, d2state.Instance{Name: ef.State, Until: e.frame + frames, Mods: statMods(ef.Stats),
		Source: p.ID(), SkillID: sk.ID, Level: ef.Level})
	e.emit("state", "STATE apply skill=%q unit=%s state=%s frames=%d stats=%s", sk.Name, mt.m.Label(), ef.State, frames,
		describeMods(ef.Stats))
}

// Engine consumers of the effects the skills batch "next" produces (see
// d2common/d2skill/skills_next.go): a hit on one chosen unit (Psychic Hammer,
// Mind Blast) and the knockback.

// hitUnit rolls the effect's damage descriptor against the one target unit.
func (e *Engine) hitUnit(p *d2mapentity.Player, u *heroUnit, sk *d2skill.Skill, ef *d2skill.Effect) {
	mt, _ := ef.Target.(*monsterTarget)
	if mt == nil || !mt.m.Alive() || ef.Desc == nil {
		return
	}

	dmg := e.rollDesc(u, ef.Desc)

	e.Counters.AreaHits++
	e.target(mt.m)
	e.hurt(mt.m, p, &dmg, sk.Name)
}

// knockback pushes a surviving monster ef.Dist subtiles straight away from the
// hero, stopping at the first cell that cannot be walked on. U: the exe
// puts the victim into its knockback mode; the distance travelled there was
// not read.
func (e *Engine) knockback(p *d2mapentity.Player, u *heroUnit, sk *d2skill.Skill, ef *d2skill.Effect) {
	mt, _ := ef.Target.(*monsterTarget)
	if mt == nil || !mt.m.Alive() {
		return
	}

	hx, hy := u.Pos()
	x, y := mt.m.SubtilePos()
	nx, ny := knockDest(hx, hy, x, y, ef.Dist, e.pipe.Walkable)

	if nx == x && ny == y {
		return
	}

	mt.m.TeleportTo(nx, ny)
	e.emit("state", "KNOCKBACK skill=%q unit=%s from=(%d,%d) to=(%d,%d)", sk.Name, mt.m.Label(), x, y, nx, ny)
}

// knockDest walks away from (hx, hy) one subtile at a time, up to dist steps,
// while the next cell is walkable (nil: always).
func knockDest(hx, hy, x, y, dist int, walkable func(x, y int) bool) (int, int) {
	dx, dy := sign(x-hx), sign(y-hy)
	if dx == 0 && dy == 0 {
		return x, y
	}

	for i := 0; i < dist; i++ {
		nx, ny := x+dx, y+dy
		if walkable != nil && !walkable(nx, ny) {
			break
		}

		x, y = nx, ny
	}

	return x, y
}

// KnockClass implements d2skill.KnockClassed: a special boss (monstats boss)
// is the boss class (U: the exe's MONSTER_IsBossMonster may include uniques).
func (t *monsterTarget) KnockClass() d2skill.KnockClass {
	if t.m.Stat != nil && t.m.Stat.IsSpecialBoss {
		return d2skill.KnockBoss
	}

	return d2skill.KnockNormal
}

// ShieldDamage implements d2skill.ShieldDamager: the mindam / maxdam of the
// armor.txt row of the shield in the shield slot (Smite).
func (h *heroUnit) ShieldDamage() (min, max int, ok bool) {
	if h.merc != nil || h.p == nil || h.p.Equipment == nil || h.p.Equipment.Shield == nil || h.e.asset == nil {
		return 0, 0, false
	}

	rec := h.e.asset.Records.Item.Armors[h.p.Equipment.Shield.GetItemCode()]
	if rec == nil {
		return 0, 0, false
	}

	return rec.MinDamage, rec.MaxDamage, true
}

// redeemRoll is the per-corpse roll of Redemption (0x5cf2e0, VERIFIED): the
// 0..99 roll must be below calc1.
func redeemRoll(roll, chance int) bool { return roll < chance }
