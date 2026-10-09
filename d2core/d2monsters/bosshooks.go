package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// BossHooks connect the boss AIs (Baal's throne, the crab walking to the
// Worldstone portal) to the encounter logic (d2boss). All fields are optional.
type BossHooks struct {
	// Throne runs one step of the throne statue's think function.
	Throne func(b *d2monster.Brain, step d2monster.ThroneStep, wave int) bool
	// WaveCleared reports the last throne wave dead.
	WaveCleared func() bool
	// NearestObject finds an object of an objects.txt id around the monster.
	NearestObject func(b *d2monster.Brain, class, radius int) (d2monster.Point, int, bool)
	// Left is called after a monster entered a portal and left the level.
	Left func(b *d2monster.Brain)
}

// SetBossHooks installs the hooks.
func (d *Director) SetBossHooks(h BossHooks) { d.boss = h }

// Throne implements d2monster.ThroneHost.
func (d *Director) Throne(b *d2monster.Brain, step d2monster.ThroneStep, wave int) bool {
	if d.boss.Throne == nil {
		return true
	}

	return d.boss.Throne(b, step, wave)
}

// WaveCleared implements d2monster.WaveGate (true without hooks).
func (d *Director) WaveCleared(*d2monster.Brain) bool {
	if d.boss.WaveCleared == nil {
		return true
	}

	return d.boss.WaveCleared()
}

// NearestObject implements d2monster.ObjectFinder.
func (d *Director) NearestObject(b *d2monster.Brain, class, radius int) (d2monster.Point, int, bool) {
	if d.boss.NearestObject == nil {
		return d2monster.Point{}, 0, false
	}

	return d.boss.NearestObject(b, class, radius)
}

// LeaveLevel implements d2monster.Leaver: the monster enters a portal and is
// gone without dying (no kill event, no loot).
func (d *Director) LeaveLevel(b *d2monster.Brain) {
	u := d.units[b.ID]
	if u == nil {
		return
	}

	d.emit("boss", "MONSTER leave name=%s id=%d", u.m.Label(), b.ID)
	d.engine.RemoveEntity(u.m)
	d.forget(u)

	if d.boss.Left != nil {
		d.boss.Left(b)
	}
}

// Dismiss implements d2monster.Dismisser: the monster vanishes (a tentacle
// whose time is up, a clone whose leader died).
func (d *Director) Dismiss(b *d2monster.Brain) {
	u := d.units[b.ID]
	if u == nil {
		return
	}

	d.emit("boss", "MONSTER dismiss name=%s id=%d", u.m.Label(), b.ID)
	d.engine.RemoveEntity(u.m)
	d.forget(u)
}

// KillMonster kills a monster as the hero's blow would (loot, experience, the
// OnKill hook). It returns false for a dead or unknown monster.
func (d *Director) KillMonster(m *d2mapentity.Monster) bool {
	u := d.byEntity[m.ID()]
	if u == nil || !u.m.Alive() {
		return false
	}

	var hero *d2mapentity.Player
	if list := d.players(); len(list) > 0 {
		hero = list[0]
	}

	u.m.Vitals.HP = 0
	d.kill(u, hero)

	return true
}

// BrainOf returns the AI brain of a monster (nil for unknown monsters).
func (d *Director) BrainOf(m *d2mapentity.Monster) *d2monster.Brain {
	if u := d.byEntity[m.ID()]; u != nil {
		return u.b
	}

	return nil
}
