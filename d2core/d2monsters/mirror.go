package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Mirror monsters: in a game played through the realm (package d2realm) the
// server decides where monsters are, when they are hit and when they die. The
// engine only draws them. A mirror monster is a normal monster entity (sprites,
// sounds, death animation) whose AI never runs and whose life never changes
// here: a blow that would hurt it is reported to OnMirrorHit instead, and the
// realm's Hit / Death / Seg events arrive through MirrorHP / MirrorKill /
// MirrorMove.

// SpawnMirror places a mirror monster on a free subtile near the position.
func (d *Director) SpawnMirror(stat *d2records.MonStatRecord, subX, subY int) (*d2mapentity.Monster, error) {
	m, err := d.SpawnNear(stat, subX, subY, 2)
	if err != nil {
		return nil, err
	}

	if u := d.byEntity[m.ID()]; u != nil {
		u.mirror = true
	}

	return m, nil
}

// IsMirror reports whether a monster is a mirror of a realm unit.
func (d *Director) IsMirror(m *d2mapentity.Monster) bool {
	u := d.byEntity[m.ID()]

	return u != nil && u.mirror
}

// MirrorHP sets the life the realm reports (hp of maxHP).
func (d *Director) MirrorHP(m *d2mapentity.Monster, hp, maxHP int) {
	u := d.byEntity[m.ID()]
	if u == nil || !u.mirror || maxHP <= 0 || !u.m.Alive() {
		return
	}

	u.m.Vitals.HP = u.m.Vitals.MaxHP * hp / maxHP
	if u.m.Vitals.HP <= 0 {
		u.m.Vitals.HP = 1 // only MirrorKill kills
	}

	u.m.StopMoving()
	u.m.SetMode(d2monster.ModeGetHit)
}

// MirrorKill kills a mirror monster as the realm decided: the death animation
// and sound play, there is no local experience and no local loot (the realm
// owns both). It returns false for a dead or unknown monster.
func (d *Director) MirrorKill(m *d2mapentity.Monster) bool {
	u := d.byEntity[m.ID()]
	if u == nil || !u.mirror || !u.m.Alive() {
		return false
	}

	u.m.Vitals.HP = 0
	u.m.Die()
	d.fp.Remove(u.b.ID)
	d.Counters.Deaths++
	d.playPlans(u, deathPlans(d.soundRecord(u)))
	d.emit("death", "MONSTER death name=%s id=%d by=realm xp=0", u.m.Label(), u.b.ID)

	if d.OnKill != nil {
		d.OnKill(KillEvent{Monster: u.m, Class: u.b.Class, Label: u.m.Label(), ByHero: false})
	}

	return true
}

// MirrorMove puts a mirror monster on the subtile the realm reports.
func (d *Director) MirrorMove(m *d2mapentity.Monster, subX, subY int) {
	if u := d.byEntity[m.ID()]; u != nil && u.mirror && u.m.Alive() {
		u.m.TeleportTo(subX, subY)
	}
}
