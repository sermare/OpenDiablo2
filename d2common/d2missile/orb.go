package d2missile

import "math"

// Frozen Orb (missiles.txt frozenorb: SrvDoFunc 15, SrvHitFunc 29, Param1 1,
// Param2 19, sHitPar1 4; SubMissile1 frozenorbbolt, HitSubMissile1
// frozenorbnova). The exe bodies (0x5acba0, 0x5a9640) were not read, so the
// following is a STAND-IN built from the table columns (UNVERIFIED):
//   - every Param1 frames the orb creates SubMissile1 at its position; the
//     bolt's direction is a multiple of 360/orbDirections degrees that advances
//     Param2 steps per bolt (19 steps of 16 directions = 3, a spiral);
//   - where the orb ends (wall or expiry) HitSubMissile1 flies out in every
//     direction (orbRing) and the orb ends.
//
// The sub missiles carry the damage the cast rolled (the orb's own damage is
// zeroed at creation: frozenorb has no Skill column).
const (
	orbDirections = 16
	orbRingCount  = 16
)

func (s *Sim) orbEmit(m *Missile) {
	period := m.Spec.Param[0]
	if period < 1 {
		period = 1
	}

	n := m.Total - m.Life
	if n < 0 || n%period != 0 {
		return
	}

	sub := s.lookup(m.Spec.SubMissile[0])
	if sub == nil {
		return
	}

	step := m.Spec.Param[1]
	dir := ((n / period) * step) % orbDirections
	s.orbShoot(m, sub, float64(dir)*2*math.Pi/orbDirections)
}

func (s *Sim) orbRing(m *Missile) {
	sub := s.lookup(m.Spec.HitSubMissile[0])
	if sub == nil {
		return
	}

	for i := 0; i < orbRingCount; i++ {
		s.orbShoot(m, sub, float64(i)*2*math.Pi/orbRingCount)
	}
}

func (s *Sim) orbShoot(m *Missile, sub *Spec, angle float64) {
	dmg := m.childDamage
	_, _ = s.Create(CreateParams{Spec: sub, Parent: m, Owner: m.Owner, SkillID: m.SkillID, Level: m.Level, Damage: dmg,
		X: m.X, Y: m.Y, DestX: m.X + 1, DestY: m.Y, Angle: angle})
}
