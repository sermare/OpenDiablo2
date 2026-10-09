package d2missile

import "math"

// meteorOffsets are the subtile offsets (x, y) at which hit function 14
// (Meteor, 0x5a8680) spawns HitSubMissile1: two parallel 18 entry int tables
// at 0x6e3a00 (x) and 0x6e3a48 (y), walked with a stride of sHitPar2 entries
// (verified bytes).
var meteorOffsets = [18][2]int{
	{2, -2}, {-2, -2}, {0, 2}, {0, 5}, {-3, 3}, {0, 3}, {3, 3}, {-1, 2}, {1, 1},
	{-1, -1}, {2, -1}, {-4, -2}, {-3, -2}, {-1, -3}, {0, -4}, {1, -3}, {3, -3}, {4, -2},
}

// hitFunc runs the missile's pSrvHitFunc (0x739a68 table) for target t (nil on
// expiry, wall or the end of the path) and returns the value the exe returns,
// which replaces the result bits A of ProcessHitOrExpire. ok is false for hit
// functions that are not modelled (and for id 0, which is not called).
//
// Verified returns: 1 -> 1 (area damage only, no direct damage), 2 and 4 -> 3,
// 14 -> a skill record pointer in the exe (the decompiled code leaves the
// pointer in the return register; it has bits 0-2 clear in practice, else
// Meteor would never end), taken as 0. 10 is the Guided Arrow logic.
func (s *Sim) hitFunc(m *Missile, t Target) (ret int, ok bool) {
	switch m.Spec.SrvHitFunc {
	case 1:
		s.areaDamage(m)
		return resKill, true
	case 2:
		s.spawnHitSub(m, 1)
		return resKill | resDamage, true
	case 4:
		s.spawnHitSub(m, 4)
		return resKill | resDamage, true
	case 7:
		return s.holyBoltHit(m, t), true
	case 10:
		return s.guidedHit(m, t), true
	case 14:
		s.areaDamage(m)
		s.meteorFire(m)

		return 0, true
	}

	return 0, false
}

// areaDamage is the damage half of hit functions 1 (0x5a7500) and 14: one
// damage roll applied to every enemy whose subtile position is within the
// radius of the missile (squared distance <= r*r, 0x569510, verified units:
// subtiles, the units' path positions are subtile integers). The radius is
// sHitPar1 when positive, otherwise a skill calc the sim does not evaluate
// (Missile.AreaRadius, minimum 1 as in the exe).
func (s *Sim) areaDamage(m *Missile) {
	r := m.Spec.SHitPar[0]
	if r < 1 {
		r = m.AreaRadius
	}

	if r < 1 {
		return
	}

	s.emit(Event{Kind: EventArea, Missile: m, Damage: m.Damage.Roll(m.Owner.Roller), Radius: r})
}

// spawnHitSub runs hit functions 2 (HitSubMissile1 only) and 4 (all four).
func (s *Sim) spawnHitSub(m *Missile, n int) {
	for _, name := range m.Spec.HitSubMissile[:n] {
		sub := s.lookup(name)
		if sub == nil {
			continue
		}

		_, _ = s.Create(CreateParams{Spec: sub, Owner: m.Owner, SkillID: m.SkillID, Level: m.Level,
			Damage: m.Damage, X: m.X, Y: m.Y, DestX: m.X, DestY: m.Y})
	}
}

// meteorFire is the spawn half of hit function 14: HitSubMissile1 (meteorfire)
// at up to 18 offsets around the impact, every sHitPar2-th entry of the table,
// with the lifetime override HitSubRange when positive (flag 0x8000).
func (s *Sim) meteorFire(m *Missile) {
	sub := s.lookup(m.Spec.HitSubMissile[0])
	if sub == nil {
		return
	}

	step := m.Spec.SHitPar[1]
	if step < 1 {
		step = 1
	}

	for i := 0; i < len(meteorOffsets); i += step {
		x, y := m.X+float64(meteorOffsets[i][0]), m.Y+float64(meteorOffsets[i][1])

		_, _ = s.Create(CreateParams{Spec: sub, Owner: m.Owner, SkillID: m.SkillID, Level: m.Level,
			Damage: m.Damage, X: x, Y: y, DestX: x, DestY: y, Range: m.HitSubRange})
	}
}

// guidedTurn is SrvDoFunc 7 (0x5ac2c0) before the standard move; false if the
// missile vanished. It destroys the missile when the owner is gone. A homing
// missile (HomeMode bit 1) re-aims at its target only every Param1 frames
// (5 when Param1 < 1) at which its remaining life is a multiple of Param1,
// and only while the distance to the target is 4..24 (verified, 0x5ac2c0 with
// UNIT_GetDistanceToUnit 0x642b10: subtiles, metric max+min/2 with the unit
// sizes subtracted, see Distance); otherwise it flies straight.
// The exe also destroys it inside a town (not modelled).
func (s *Sim) guidedTurn(m *Missile) bool {
	if m.Owner.Gone != nil && m.Owner.Gone() {
		s.vanish(m)
		return false
	}

	t := m.Home
	if m.HomeMode&1 == 0 || t == nil || !t.Alive() || !s.World.IsEnemy(m.Owner, t) {
		return true
	}

	period := m.Spec.Param[0]
	if period < 1 {
		period = 5
	}

	pt, ok := t.(Positioned)
	if !ok || m.Life%period != 0 {
		return true
	}

	tx, ty := pt.SubPos()

	if d := Distance(m.X, m.Y, 0, tx, ty, targetSize(t)); d >= 4 && d <= 24 {
		aim(m, tx, ty)
	}

	return true
}

// guidedHit is hit function 10 (0x5a8100, Guided Arrow) with the mode bits of
// Missile.HomeMode (verified):
//   - no target and the missile was already re-targeted (bit 4): 1 (ends);
//   - homing (bit 1): ends (3) on the intended target and at the end of its
//     path, passes through every other unit (4);
//   - ground aimed (bit 2): passes through units (4) while it has life left,
//     unless re-targeted (3); when its life ran out it looks for a new target
//     (retarget): 1 if it may not, else 4 (it flies on);
//   - any other mode: 3.
func (s *Sim) guidedHit(m *Missile, t Target) int {
	f := m.HomeMode

	if t == nil && f&4 != 0 {
		return resKill
	}

	if f&1 != 0 {
		if t == nil || (m.Home != nil && t.ID() == m.Home.ID()) {
			return resKill | resDamage
		}

		return resKeep
	}

	if f&2 == 0 {
		return resKill | resDamage
	}

	if m.Life <= 0 {
		if s.retarget(m) {
			return resKill
		}

		return resKeep
	}

	if f&4 != 0 {
		return resKill | resDamage
	}

	return resKeep
}

// retarget is 0x5a8060 + 0x5a7f10: a ground aimed guided arrow whose life ran
// out searches for an enemy (verified): the scan 0x569510 visits every unit of
// the rooms around the missile whose subtile position is within Param2
// subtiles (euclidean, squared compare) of the MISSILE, the owner excluded,
// that is a living player or monster, not in a town, flagged targetable, an
// enemy of the owner and in line of sight of the owner (0x569100, flags
// 0xa783). Of those the callback 0x569a40 keeps the one with the LOWEST UNIT
// ID (it compares unit+0xc; the callback's first branch is dead), not the
// nearest: Finder.EnemiesWithin lists the candidates and a Target that
// implements Serial is ordered by it (the first listed wins otherwise). The
// arrow then refills its life to Range + (level-1)*LevRange and either homes
// on the unit (mode 5) or flies another ground leg of the original length
// (mode 6). It reports true when the missile may not re-target (already did).
func (s *Sim) retarget(m *Missile) bool {
	if m.HomeMode&4 != 0 {
		return true
	}

	var t Target

	if f, ok := s.World.(Finder); ok {
		t = lowestSerial(f.EnemiesWithin(m.Owner, m.X, m.Y, m.Spec.Param[1]))
	}

	m.Life = (m.Level-1)*m.Spec.LevRange + m.Spec.Range
	m.CollideFrom = m.Life - m.Spec.Activate
	m.parked = false

	if t != nil {
		m.Home, m.HomeMode = t, 5

		if pt, ok := t.(Positioned); ok {
			tx, ty := pt.SubPos()
			if d := Distance(m.X, m.Y, 0, tx, ty, targetSize(t)); d < 25 {
				aim(m, tx, ty)
			}
		}

		return false
	}

	m.Home, m.HomeMode = nil, 6

	if leg := math.Hypot(m.legX, m.legY); leg > 0 {
		aim(m, m.X+m.legX, m.Y+m.legY)
		m.ClampDist, m.Travel = leg, 0
	}

	return false
}

// Healer is optionally implemented by a Target that Holy Bolt can heal
// (amount in 8.8 fixed point, clamped to the maximum life by the target).
type Healer interface{ Heal(amount int) }

// Kinded is optionally implemented by a Target for the class filter of Holy
// Bolt (sHitPar2): monstats lUndead|hUndead and demon flags (byte +0xd bits
// 3,4 and 5 of the monstats record, tests 0x63f9e0 / 0x63f990, verified).
type Kinded interface {
	IsUndead() bool
	IsDemon() bool
}

// holyBoltHit is hit function 7 (0x5a7a40, verified): no target -> 1 (ends).
// With sHitPar1 != 0 (heals allies) and an owner: an ally of the owner
// (tests 0x552320 / 0x552d80, taken as "not an enemy": UNVERIFIED detail) is
// healed by calc1 + rand(calc2 - calc1) of the casting skill (8.8 fixed
// point, added to life and clamped to max life) and the result is 1: the
// missile ends WITHOUT damage. Anything else goes through the class filter
// sHitPar2 (0 all, 1 undead, 2 demons): units that pass return 3 (destroy +
// damage), others 4 (the bolt flies on); a player passes only for 0, a
// non unit (missile) never passes.
func (s *Sim) holyBoltHit(m *Missile, t Target) int {
	if t == nil {
		return resKill
	}

	sp := m.Spec

	if sp.SHitPar[0] != 0 && m.Owner.ID != "" && !s.World.IsEnemy(m.Owner, t) {
		if h, ok := t.(Healer); ok {
			amount := m.HealMin

			if span := m.HealMax - m.HealMin; span > 0 && m.Owner.Roller != nil {
				amount += int(m.Owner.Roller.Roll(int32(span)))
			}

			if amount > 0 {
				h.Heal(amount)
			}

			s.emit(Event{Kind: EventHeal, Missile: m, Target: t, Heal: amount})
		}

		return resKill
	}

	pass := func() int { return resKill | resDamage }

	if sp.SHitPar[1] == 0 {
		return pass()
	}

	if t.IsPlayer() {
		return resKeep
	}

	k, ok := t.(Kinded)
	if !ok {
		return resKeep
	}

	if sp.SHitPar[1] == 2 {
		if k.IsDemon() {
			return pass()
		}

		return resKeep
	}

	if k.IsUndead() {
		return pass()
	}

	return resKeep
}

// tornadoPulse is the part of SrvDoFunc 27 (Tornado, 0x5ad590, verified bytes;
// the function is not defined in the project) that runs before the standard
// move: every `period` frames an area damage like hit function 1 is applied
// around the missile. period is missiles.txt Param1 when > 0, else the casting
// skill's calc4 (record +0x144), minimum 1; the radius is Param2 when > 0, else
// the skill's aurarangecalc (record +0x64), minimum 1, in subtiles; the pulse
// uses the damage the missile carries, the missile record's HitFlags and
// ResultFlags, and the skill's aurafilter (0xa583 for Tornado: enemy filter).
// The pulse fires when the missile's remaining life is a multiple of the
// period (the exe tests the value from 0x64b640 - missile data +0xe minus
// +0x10 - which is taken as the remaining life: UNVERIFIED).
func (s *Sim) tornadoPulse(m *Missile) {
	period := m.Spec.Param[0]
	if period < 1 {
		period = m.PulseEvery
	}

	if period < 1 {
		period = 1
	}

	if m.Life%period != 0 {
		return
	}

	r := m.Spec.Param[1]
	if r < 1 {
		r = m.AreaRadius
	}

	if r < 1 {
		r = 1
	}

	s.emit(Event{Kind: EventArea, Missile: m, Damage: m.Damage.Roll(m.Owner.Roller), Radius: r})
}
