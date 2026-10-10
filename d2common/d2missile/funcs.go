package d2missile

import (
	"math"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// Elapsed is MISSILE_GetElapsedLife (0x64b640, verified): the life the missile
// was created with (data +0xe) minus its remaining life (+0x10).
func (m *Missile) Elapsed() int { return m.Total - m.Life }

// cellMask bits the random sub missile functions refuse (the mask argument of
// COLLISION_GetCellFlagsAt in 0x5a7280): Blizzard passes 5 (walk | wall),
// Eruption 0x45 (plus the missile bit 0x40).
const (
	spawnMaskBlizzard = d2path.FlagWalk | d2path.FlagWall
	spawnMaskEruption = spawnMaskBlizzard | MissileCellBit
)

// doFunc runs the part of a server movement function that precedes the
// standard move for the functions added in funcs.go (the older ones are
// handled in stepOne). It reports false when the missile vanished.
//
// All of these were read from the exe (Game.exe 1.14b) and are VERIFIED unless
// a comment says otherwise:
//   - 3 (0x5abfb0) and 5 (0x5ac050): stamp bit 0x40 into the collision cell
//     under the missile (see CellMarker), then move. 5 additionally runs the
//     frame animation of the ground fires (see fireFrame).
//   - 10 (0x5ac5b0) and 25 (0x5ad3e0): spawnRandomSub with the skill's calc1
//     (scatter radius) and calc2 (frames between spawns).
//   - 14 (0x5acb00): every Param1 frames (1 when empty) dispatch the periodic
//     skill helper Param2 for the missile's skill and level.
//   - 23 and 24 (0x5ad2f0): drop SubMissile1 at the missile's own cell whenever
//     the last step entered a new subtile.
//   - 28 (0x5ad6f0): Volcano's scatter, see volcanoSpread.
func (s *Sim) doFunc(m *Missile) bool {
	switch m.Spec.SrvDoFunc {
	case 3:
		s.markCell(m)
	case 5:
		s.markCell(m)
		s.fireFrame(m)
	case 10:
		s.spawnRandomSub(m, spawnMaskBlizzard)
	case 25:
		s.spawnRandomSub(m, spawnMaskEruption)
	case 14:
		return s.periodicHelper(m)
	case 23, 24:
		return s.spawnAtOwnCell(m)
	case 28:
		s.volcanoSpread(m)
	}

	return true
}

// markCell is the cell stamp of SrvDoFunc 3 and 5. The exe skips the stamp in
// do function 3 when the path's flag at +0x7c (0x649a80) is set and goes
// straight to the standard move; that flag is not modelled (UNVERIFIED role),
// so the stamp is always applied.
func (s *Sim) markCell(m *Missile) {
	if cm, ok := s.World.(CellMarker); ok {
		cm.MarkCell(int(math.Floor(m.X)), int(math.Floor(m.Y)), m.Spec.Size, MissileCellBit)
	}
}

// fireFrame is the animation step of SrvDoFunc 5 (0x5ac050), which only moves
// the sprite frame (missile field 0x44, 8.8) and has no gameplay effect: with
// start = SubStart and stop = SubStop, a frame sitting at start-1 jumps to
// start-1 + rand(stop-start) (the roll uses the missile's own seed); else when
// the remaining life equals start the frame becomes start-3, and when the
// remaining life is below start the frame backs off by 2 (floored at 0).
func (s *Sim) fireFrame(m *Missile) {
	sp := m.Spec
	frame := m.Frame >> 8

	switch {
	case frame == sp.SubStart-1:
		m.Frame = (frame + int(m.seed.Roll(int32(sp.SubStop-sp.SubStart)))) << 8
	case m.Life == sp.SubStart:
		m.Frame = (sp.SubStart - 3) << 8
	case m.Life < sp.SubStart:
		frame -= 2
		if frame < 0 {
			frame = 0
		}

		m.Frame = frame << 8
	}
}

// spawnRandomSub is MISSILE_SpawnRandomSubMissileEveryNFrames (0x5a7280) as
// called by SrvDoFunc 10 and 25: every SpawnEvery frames of elapsed life (no
// spawn when it is 0) the missile reseeds itself with its integer x plus the
// elapsed life, rolls two offsets in [0, 2*(r-1)) and, when the cell at
// (x + dx - (r-1), y + dy - (r-1)) has none of the mask bits, creates its
// SubMissile1 there with the owner, skill and level of the parent. r is
// SpawnRadius; the roll for n < 1 does not advance the seed.
func (s *Sim) spawnRandomSub(m *Missile, mask uint16) {
	sp := m.Spec
	if m.SpawnEvery == 0 || m.Elapsed()%m.SpawnEvery != 0 || (m.Owner.Gone != nil && m.Owner.Gone()) {
		return
	}

	sub := s.lookup(sp.SubMissile[0])
	if sub == nil {
		return
	}

	cx, cy := int(math.Floor(m.X)), int(math.Floor(m.Y))
	m.seed.Init(uint32(cx + m.Elapsed()))

	r := m.SpawnRadius - 1
	x := cx + int(m.seed.Roll(int32(r*2))) - r
	y := cy + int(m.seed.Roll(int32(r*2))) - r

	if s.World.Flags(x, y)&mask != 0 {
		return
	}

	fx, fy := float64(x)+0.5, float64(y)+0.5
	_, _ = s.Create(CreateParams{Spec: sub, Parent: m, Owner: m.Owner, SkillID: m.SkillID, Level: m.Level,
		Damage: m.Damage, AreaRadius: m.AreaRadius, X: fx, Y: fy, DestX: fx, DestY: fy})
}

// periodicHelper is SrvDoFunc 14 (Grim Ward): with period = Param1 (1 when
// not positive), when the elapsed life is a multiple of it and the missile has
// a skill id and level above 0, the periodic helper Param2 is dispatched
// (EventPeriodic); with no skill the missile is destroyed (return 2).
func (s *Sim) periodicHelper(m *Missile) bool {
	period := m.Spec.Param[0]
	if period < 1 {
		period = 1
	}

	if m.Elapsed()%period != 0 {
		return true
	}

	if m.SkillID <= 0 || m.Level <= 0 {
		s.vanish(m)
		return false
	}

	s.emit(Event{Kind: EventPeriodic, Missile: m, Helper: m.Spec.Param[1]})

	return true
}

// spawnAtOwnCell is SrvDoFunc 23 / 24 (Firestorm's emitter): a missile without
// SubMissile1 is destroyed; otherwise, when the last step entered a new
// subtile, SubMissile1 is created at the missile's integer position. A
// positive Param1 is passed as the sub-loop count of create flag 8, which
// lengthens the sub missile's life by (SubStop - SubStart) * Param1 frames.
func (s *Sim) spawnAtOwnCell(m *Missile) bool {
	sub := s.lookup(m.Spec.SubMissile[0])
	if m.Spec.SubMissile[0] == "" {
		s.vanish(m)
		return false
	}

	if sub == nil || !m.entered {
		return true
	}

	loops := 0
	if m.Spec.Param[0] > 0 {
		loops = m.Spec.Param[0]
	}

	x, y := math.Floor(m.X)+0.5, math.Floor(m.Y)+0.5
	_, _ = s.Create(CreateParams{Spec: sub, Parent: m, Owner: m.Owner, SkillID: m.SkillID, Level: m.Level,
		Damage: m.Damage, AreaRadius: m.AreaRadius, X: x, Y: y, DestX: x, DestY: y, SubLoops: loops})

	return true
}

// volcanoSpread is SrvDoFunc 28 (0x5ad6f0): while Param3 < elapsed < Param4
// and the elapsed life is a multiple of the period (Param1, else the skill's
// calc4, minimum 1; PulseEvery) the missile reseeds itself with its data field
// 0x28, rolls two offsets in [0, 2r+1) with r = Param2 (else aurarangecalc,
// minimum 1; AreaRadius), stores the new seed low word back in the field and
// lobs SubMissile1 from its own position to (x + dx - r, y + dy - r) (create
// flags 0x520: explicit destination, lob; the exe also hands Param5 to the
// create struct, role UNVERIFIED). Nothing spawns without an owner.
func (s *Sim) volcanoSpread(m *Missile) {
	sp := m.Spec
	if m.Owner.Gone != nil && m.Owner.Gone() {
		return
	}

	sub := s.lookup(sp.SubMissile[0])
	if sub == nil {
		return
	}

	r := sp.Param[1]
	if r < 1 {
		r = m.AreaRadius
	}

	if r < 1 {
		r = 1
	}

	period := sp.Param[0]
	if period < 1 {
		period = m.PulseEvery
	}

	if period < 1 {
		period = 1
	}

	el := m.Elapsed()
	if el <= sp.Param[2] || el >= sp.Param[3] || el%period != 0 {
		return
	}

	m.seed.Init(m.Data28)
	dx := int(m.seed.Roll(int32(r*2 + 1)))
	dy := int(m.seed.Roll(int32(r*2 + 1)))
	m.Data28 = m.seed.Lo

	cx, cy := int(math.Floor(m.X)), int(math.Floor(m.Y))
	tx, ty := float64(cx+dx-r)+0.5, float64(cy+dy-r)+0.5

	_, _ = s.Create(CreateParams{Spec: sub, Parent: m, Owner: m.Owner, SkillID: m.SkillID, Level: m.Level,
		Damage: m.Damage, AreaRadius: m.AreaRadius, X: m.X, Y: m.Y, DestX: tx, DestY: ty, ClampToDest: true})
}

// ---- hit functions ----

// blazeHit is hit function 8 (0x5a7c20): the fire of Blaze burns everything
// (2: damage, the fire stays) except its caster while the caster is in state
// 13 (blaze), for which it returns 0 (nothing happens).
func (s *Sim) blazeHit(m *Missile, t Target) int {
	if t != nil && t.ID() == m.Owner.ID && m.Owner.HasState != nil && m.Owner.HasState(13) {
		return 0
	}

	return resDamage
}

// wardStartHit is hit function 26 (0x5a93f0, Grim Ward's start missile): at the
// end of its short life it creates HitSubMissile1 (the ward proper) at its own
// cell with the owner, skill and level of the parent and the lifetime
// override sHitPar1 when positive, else the skill's calc1 (HitSubRange, the
// "ward duration"), minimum 5 (create flags 0x8001). The missile ends (1). No
// owner, no ward.
func (s *Sim) wardStartHit(m *Missile) int {
	if m.Owner.ID == "" || (m.Owner.Gone != nil && m.Owner.Gone()) {
		return resKill
	}

	sub := s.lookup(m.Spec.HitSubMissile[0])
	if sub == nil {
		return resKill
	}

	life := m.Spec.SHitPar[0]
	if life < 1 {
		life = m.HitSubRange
		if life < 5 {
			life = 5
		}
	}

	x, y := math.Floor(m.X), math.Floor(m.Y)
	_, _ = s.Create(CreateParams{Spec: sub, Parent: m, Owner: m.Owner, SkillID: m.SkillID, Level: m.Level,
		Damage: m.Damage, X: x + 0.5, Y: y + 0.5, DestX: x + 0.5, DestY: y + 0.5, Range: life})

	return resKill
}

// boulderHit is hit function 47 (0x5aa070, Molten Boulder): a unit that is not
// a monster, or a monster without the class bit 0xb of its monstats2 record
// (0x45f0b0, which bit that is: UNVERIFIED), is only damaged (2: the boulder
// rolls on). Otherwise (that class of monster, or no target at the end of the
// path or on a wall) the boulder explodes: one area damage with radius
// sHitPar1 (else the skill's aurarangecalc, minimum 1), then HitSubMissile1
// (the fire path) at every sHitPar2-th entry of the Meteor offset table, and
// the missile ends (1). BoulderBit is the optional Target interface for that
// class bit.
func (s *Sim) boulderHit(m *Missile, t Target) int {
	if t != nil {
		bb, ok := t.(interface{ BoulderBit() bool })
		if t.IsPlayer() || !ok || !bb.BoulderBit() {
			return resDamage
		}
	}

	s.areaDamage(m)
	s.meteorFire(m)

	return resKill
}

// emergeHit is hit function 48 (0x5aa210, Molten Boulder's emerge missile):
// at its end it creates HitSubMissile1 (the rolling boulder) at its own cell,
// flying to the path destination it was aimed at (create flags 0x21) and the
// emerge missile ends (1).
func (s *Sim) emergeHit(m *Missile) int {
	sub := s.lookup(m.Spec.HitSubMissile[0])
	if sub == nil {
		return resKill
	}

	x, y := math.Floor(m.X)+0.5, math.Floor(m.Y)+0.5
	_, _ = s.Create(CreateParams{Spec: sub, Parent: m, Owner: m.Owner, SkillID: m.SkillID, Level: m.Level,
		Damage: m.Damage, AreaRadius: m.AreaRadius, HitSubRange: m.HitSubRange, X: x, Y: y, DestX: m.destX, DestY: m.destY})

	return resKill
}

// debrisHit is hit function 51 (0x5aa3b0): HitSubMissile1, 2 and 3 (each when
// set) are created at the missile with its owner, skill and level, and the
// missile ends (1).
func (s *Sim) debrisHit(m *Missile) int {
	for _, name := range m.Spec.HitSubMissile[:3] {
		sub := s.lookup(name)
		if sub == nil {
			continue
		}

		_, _ = s.Create(CreateParams{Spec: sub, Parent: m, Owner: m.Owner, SkillID: m.SkillID, Level: m.Level,
			Damage: m.Damage, AreaRadius: m.AreaRadius, X: m.X, Y: m.Y, DestX: m.X, DestY: m.Y})
	}

	return resKill
}

// armageddonHit is hit function 56 (0x5aa7a0, the control missile of
// Armageddon): one area damage (radius sHitPar1, else aurarangecalc, minimum
// 1) at the missile and HitSubMissile1 created at its cell by the skill's
// create-at-target helper (0x56cbf0); the missile ends (1).
func (s *Sim) armageddonHit(m *Missile) int {
	s.areaDamage(m)

	if sub := s.lookup(m.Spec.HitSubMissile[0]); sub != nil {
		x, y := math.Floor(m.X)+0.5, math.Floor(m.Y)+0.5
		_, _ = s.Create(CreateParams{Spec: sub, Parent: m, Owner: m.Owner, SkillID: m.SkillID, Level: m.Level,
			Damage: m.Damage, AreaRadius: m.AreaRadius, X: x, Y: y, DestX: x, DestY: y})
	}

	return resKill
}

// immolationHit is hit function 9 (0x5a7cf0, Immolation Arrow), VERIFIED from
// the disassembly (the function is not defined in the project):
//   - with a HitSubMissile1 it creates a fire disc (0x5a6f60): the sub
//     missile at every cell (dx, dy) with dx*dx + dy*dy <= r*r around the
//     arrow, skipping cells with the wall bit 4, r = sHitPar1 else the skill's
//     calc1 (DiscRadius), minimum 1; the fire lives DiscLife frames (the
//     missile's SHitCalc1) when that is positive (create flag 0x8000);
//   - then one area damage with radius sHitPar2 else the skill's calc2
//     (AreaRadius), minimum 1, from a descriptor whose chill length and
//     freeze stats are zeroed (stats 0x38 and 0x86 are cleared first);
//   - returns 3: the struck target is damaged too and the arrow ends.
func (s *Sim) immolationHit(m *Missile) int {
	sp := m.Spec

	if sub := s.lookup(sp.HitSubMissile[0]); sub != nil {
		r := sp.SHitPar[0]
		if r < 1 {
			r = m.DiscRadius
		}

		if r < 1 {
			r = 1
		}

		cx, cy := int(math.Floor(m.X)), int(math.Floor(m.Y))

		for dx := -r; dx <= r; dx++ {
			for dy := -r; dy <= r; dy++ {
				if dx*dx+dy*dy > r*r || s.World.Flags(cx+dx, cy+dy)&d2path.FlagWall != 0 {
					continue
				}

				x, y := float64(cx+dx)+0.5, float64(cy+dy)+0.5
				_, _ = s.Create(CreateParams{Parent: m, Owner: m.Owner, SkillID: m.SkillID, Level: m.Level, Spec: sub,
					Damage: m.Damage, X: x, Y: y, DestX: x, DestY: y, Range: m.DiscLife})
			}
		}
	}

	r := sp.SHitPar[1]
	if r < 1 {
		r = m.AreaRadius
	}

	if r < 1 {
		r = 1
	}

	d := m.Damage
	d.Cold.Len, d.FreezeLen = 0, 0

	s.emit(Event{Kind: EventArea, Missile: m, Damage: d.Roll(m.Owner.Roller), Radius: r})

	return resKill | resDamage
}

// Dest is the aim point the missile was created with (the path destination
// that hit function 48 reads back).
func (m *Missile) Dest() (x, y float64) { return m.destX, m.destY }
