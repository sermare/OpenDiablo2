package d2missile

// Frozen Orb (missiles.txt frozenorb: SrvDoFunc 15, SrvHitFunc 29, Param1 1,
// Param2 19, sHitPar1 4; SubMissile1 frozenorbbolt, HitSubMissile1
// frozenorbnova), read from the exe and VERIFIED:
//   - SrvDoFunc 15 (0x5acba0): every Param1 (min 1) frames of elapsed life the
//     orb creates SubMissile1 at its position with a heading read from the
//     64 entry table orbSin / orbCos at the index kept in its data field 0x28
//     (create kind 2, taken as an offset from the orb: UNVERIFIED role of the
//     create struct's two fields), then advances the index by Param2 modulo 64.
//   - hit function 29 (0x5a9640): while the orb has life left it returns 2 (a
//     hit: damage, the orb flies on; the orb carries none itself); when its
//     life is 0 it creates HitSubMissile1 for every sHitPar1-th table entry
//     (min 1) of the 64, giving each the table pair both as heading and as its
//     data fields 0x28 / 0x2c (see novaTurn), and ends (3).
//
// The sub missiles carry the damage the cast rolled (the orb's own damage is
// zeroed at creation: frozenorb has no Skill column).

// orbSinHalf is the table at 0x6e3ae8 (also 0x6e3f28): 30*sin(2*pi*i/64) for
// i = 0..32, rounded; the second half of the 64 entries is the negative of the
// first (bytes checked in TestOrbTableMatchesExe).
var orbSinHalf = [33]int{0, 2, 5, 8, 11, 14, 16, 19, 21, 23, 24, 26, 27, 28, 29, 29, 30, 29, 29, 28, 27, 26, 24, 23, 21, 19, 16, 14, 11, 8, 5, 2, 0}

// orbSin is entry i of the table 0x6e3ae8 (the y component of heading i).
func orbSin(i int) int {
	i &= 63
	if i > 32 {
		return -orbSinHalf[i-32]
	}

	if i == 32 {
		return 0
	}

	return orbSinHalf[i]
}

// orbCos is entry i of the table 0x6e3be8 / 0x6e4028 (the x component): the
// sine table a quarter turn on.
func orbCos(i int) int { return orbSin(i + 16) }

func (s *Sim) orbEmit(m *Missile) {
	period := m.Spec.Param[0]
	if period < 1 {
		period = 1
	}

	if m.Elapsed()%period != 0 {
		return
	}

	sub := s.lookup(m.Spec.SubMissile[0])
	if sub == nil {
		return
	}

	idx := int(m.Data28 & 63)
	m.Data28 = uint32((idx + m.Spec.Param[1]) & 63)

	s.orbShoot(m, sub, orbCos(idx), orbSin(idx), false)
}

// orbHit is hit function 29, see the comment at the top of the file.
func (s *Sim) orbHit(m *Missile) int {
	if m.Spec.HitSubMissile[0] == "" {
		return resKill
	}

	if m.Life != 0 {
		return resDamage
	}

	if m.Owner.Gone != nil && m.Owner.Gone() {
		return resKill
	}

	s.orbRing(m)

	return resKill | resDamage
}

func (s *Sim) orbRing(m *Missile) {
	sub := s.lookup(m.Spec.HitSubMissile[0])
	if sub == nil {
		return
	}

	step := m.Spec.SHitPar[0]
	if step < 1 {
		step = 1
	}

	for i := 0; i < 64; i += step {
		s.orbShoot(m, sub, orbCos(i), orbSin(i), true)
	}
}

// orbShoot creates a bolt or nova flying from the orb along (dx, dy); withData
// stores the pair in the sub missile's data fields 0x28 / 0x2c (the nova).
func (s *Sim) orbShoot(m *Missile, sub *Spec, dx, dy int, withData bool) {
	p := CreateParams{Spec: sub, Parent: m, Owner: m.Owner, SkillID: m.SkillID, Level: m.Level, Damage: m.childDamage,
		X: m.X, Y: m.Y, DestX: m.X + float64(dx), DestY: m.Y + float64(dy)}

	if withData {
		p.Data28, p.Data2C = uint32(int32(dx)), uint32(int32(dy))
	}

	_, _ = s.Create(p)
}
