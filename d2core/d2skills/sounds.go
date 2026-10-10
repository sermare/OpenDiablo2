package d2skills

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// SoundEvent is a Sounds.txt sound a skill or missile wants played. The engine
// only decides which handle and where (Skills.txt stsound/dosound columns and
// Missiles.txt TravelSound/HitSound columns); the owner plays it.
type SoundEvent struct {
	// Kind is skill-start, skill-do, missile-travel or missile-hit.
	Kind string
	// Handle is a Sounds.txt handle.
	Handle string
	// X, Y is the position in map subtiles.
	X, Y float64
	// Who names the skill or missile, for logs.
	Who string
	// Hero marks sounds made by the local hero (priority bonus).
	Hero bool
}

// sound asks the owner to play a sound; the returned stop function ends a
// looping one (nil when nothing is playing).
func (e *Engine) sound(ev SoundEvent) func() {
	if e.OnSound == nil || ev.Handle == "" {
		return nil
	}

	return e.OnSound(ev)
}

// skillSound plays a skill sound at the hero.
func (e *Engine) skillSound(p *d2mapentity.Player, kind, handle, who string) {
	if handle == "" {
		return
	}

	e.sound(SoundEvent{Kind: kind, Handle: handle, X: p.Position.X(), Y: p.Position.Y(), Who: who, Hero: true})
}

// missileSound plays the sounds of a missile event: the travel sound when it
// is created (stopped again when the missile ends) and the hit sound when it
// hits, hits a wall or explodes.
func (e *Engine) missileSound(ev d2missile.Event) {
	m := ev.Missile
	if m == nil || m.Spec == nil {
		return
	}

	rec := e.asset.Records.GetMissileByName(m.Spec.Name)
	if rec == nil {
		return
	}

	switch ev.Kind {
	case d2missile.EventCreate:
		if stop := e.sound(SoundEvent{Kind: "missile-travel", Handle: rec.TravelSound, X: m.X, Y: m.Y, Who: m.Spec.Name}); stop != nil {
			e.travel[m.ID] = stop
		}
	case d2missile.EventHit, d2missile.EventWall, d2missile.EventExplode:
		e.sound(SoundEvent{Kind: "missile-hit", Handle: rec.HitSound, X: m.X, Y: m.Y, Who: m.Spec.Name})
	}

	if m.Dead() {
		if stop := e.travel[m.ID]; stop != nil {
			stop()
			delete(e.travel, m.ID)
		}
	}
}
