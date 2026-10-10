package d2missile

import (
	"math"
	"sort"
)

// This file holds the movement and hit functions that were the last gaps of
// the skills audit (docs/skills-coverage.md): SrvDoFunc 13, 16, 20, 30, 35 and
// hit functions 17, 18, 21, 22, 37, 53. All were read from Game.exe 1.14b (the
// addresses are given); a comment says UNVERIFIED where a detail was not
// traced. Numbers only: no decompiled text.

// SkillInfo is what the hit functions read from the casting skill's record,
// evaluated by the caller at cast time for the skill level of the missile.
type SkillInfo struct {
	// Param is skills.txt Param1..Param6 (index 1..6; record +0x148..+0x15c).
	Param [7]int
	// AuraTargetState is auratargetstate (the state the missile applies); empty
	// when the column is empty (the exe tests the state id for < 0).
	AuraTargetState string
	// AuraLen is auralencalc in frames (the time of the applied state).
	AuraLen int
	// Filtered is aurafilter != 0: hit function 21 then tests the target with
	// the enemy filter 0xa783.
	Filtered bool
	// ElemLen is the elemental length of the skill at its level (ELen + tiers,
	// SKILL_GetElementalLength 0x6462f0, as d2skill.Skill.ElemLen): the longest
	// time Rabies' contagion may hand on.
	ElemLen int
}

// Stateful is optionally implemented by a Target: whether it already carries a
// state of skills.txt / States.txt by name (hit function 17 skips those).
type Stateful interface{ HasStateNamed(name string) bool }

// Ownable is optionally implemented by a Target that can own missiles (a
// monster or player): hit function 53 puts Rabies' plague on a newly infected
// target as the plague's owner, following it around.
type Ownable interface{ AsOwner() Owner }

// packShorts packs two signed 16 bit values the way SrvDoFunc 35 keeps them in
// data field 0x2c: y in the high half, x in the low half.
func packShorts(x, y int) uint32 { return uint32(int32(y))<<16 | uint32(uint16(int16(x))) }

// unpackShorts is the inverse (both halves sign extended, 0x5ae210).
func unpackShorts(v uint32) (x, y int) { return int(int16(v)), int(int16(v >> 16)) }

func (m *Missile) ownerGone() bool { return m.Owner.Gone != nil && m.Owner.Gone() }

func (s *Sim) effect(m *Missile, e Event) {
	s.emit(e)

	if m.OnEffect != nil {
		m.OnEffect(e)
	}
}

// ---- movement functions ----

// boneWallStep is SrvDoFunc 13 (Bone Wall's maker, 0x5ac920, verified): with
// no owner, or no walls left to summon (data field 0x2c is 0) the missile is
// destroyed (return 2). Otherwise, in a frame in which the last step entered a
// new subtile, one wall monster of the casting skill is summoned at the
// missile, linked to the leader unit the missile marks (Mark; data field
// 0x28), and the counter drops by one; without that unit nothing is summoned
// and the counter stays. The missile then moves on. (When the skill has no
// summon the exe returns 0 without moving: not modelled.)
func (s *Sim) boneWallStep(m *Missile) bool {
	if m.Owner.ID == "" || m.ownerGone() || m.Data2C == 0 {
		s.vanish(m)
		return false
	}

	if !m.entered || m.Mark == nil {
		return true
	}

	s.effect(m, Event{Kind: EventSummon, Missile: m, Target: m.Mark})
	m.Data2C--

	return true
}

// half is the exe's (v + (v >> 31)) >> 1: division by 2 truncating to zero.
func half(v int) int { return v / 2 }

// novaTurn is SrvDoFunc 16 (Frozen Orb's nova, 0x5accf0, verified bytes): while
// the elapsed life is below Param1 and a multiple of Param2 (min 1) the
// missile's heading pair (data fields 0x28, 0x2c) = (a, b) becomes
// ((a-b)/2, (a+b)/2), truncating, and its path is re-aimed at its own integer
// position plus the new pair: a turn of 45 degrees per
// application at 1/sqrt(2) of the length, so the nova curls in its first
// frames and then flies straight.
func (s *Sim) novaTurn(m *Missile) {
	sp := m.Spec

	period := sp.Param[1]
	if period < 1 {
		period = 1
	}

	el := m.Elapsed()
	if el >= sp.Param[0] || el%period != 0 {
		return
	}

	a, b := int(int32(m.Data28)), int(int32(m.Data2C))
	na, nb := half(a-b), half(a+b)

	m.Data28, m.Data2C = uint32(int32(na)), uint32(int32(nb))
	aim(m, math.Floor(m.X)+float64(na)+0.5, math.Floor(m.Y)+float64(nb)+0.5)
}

// followOwner is SrvDoFunc 20 (Blade Sentinel's blade creeper, 0x5ad090,
// verified): without a living owner the missile ends through the hit function
// with no target (return 2). Otherwise it is put on the owner's position
// (0x5a7450) and its remaining life reset to 10 before the standard move, so
// it lasts as long as its owner.
func (s *Sim) followOwner(m *Missile) bool {
	if m.Owner.Pos == nil || m.ownerGone() {
		s.finish(m, EventExpire)
		m.dead = true

		return false
	}

	x, y := m.Owner.Pos()
	m.X, m.Y = math.Floor(x)+0.5, math.Floor(y)+0.5
	m.Life = 10

	return true
}

// contagionStep is SrvDoFunc 30 (Rabies' plague, 0x5adbf0, verified): the
// missile follows its owner, the infected monster, and every Param1 (min 1)
// frames of elapsed life creates SubMissile1 (rabiescontagion) for the caster
// it marks (MarkOwner) at a random offset of up to Param2 subtiles in each axis
// (two rolls of 2*Param2+1; the exe rolls the caster's seed, the sim the
// missile's, UNVERIFIED), and hands the contagion the frame the infection
// expires on (data field 0x28). It ends through the hit function with no
// target when the missile has no sub missile, no caster, no living owner or no
// life left.
func (s *Sim) contagionStep(m *Missile) bool {
	sp := m.Spec

	if sp.SubMissile[0] == "" {
		s.vanish(m)
		return false
	}

	if m.MarkOwner == nil || m.Owner.Pos == nil || m.ownerGone() || m.Life < 0 {
		s.finish(m, EventExpire)
		m.dead = true

		return false
	}

	x, y := m.Owner.Pos()
	m.X, m.Y = math.Floor(x)+0.5, math.Floor(y)+0.5

	period := sp.Param[0]
	if period < 1 {
		period = 1
	}

	if m.Elapsed()%period != 0 {
		return true
	}

	sub := s.lookup(sp.SubMissile[0])
	if sub == nil {
		return true
	}

	r := sp.Param[1]
	dx := int(m.seed.Roll(int32(2*r+1))) - r
	dy := int(m.seed.Roll(int32(2*r+1))) - r

	child, err := s.Create(CreateParams{Spec: sub, Parent: m, Owner: *m.MarkOwner, SkillID: m.SkillID, Level: m.Level,
		Damage: m.childDamage, Skill: m.Skill, OnEffect: m.OnEffect,
		X: m.X, Y: m.Y, DestX: m.X + float64(dx), DestY: m.Y + float64(dy)})
	if err == nil && child != nil && m.Skill.AuraTargetState != "" && m.Owner.StateExpire != nil {
		if frame, ok := m.Owner.StateExpire(m.Skill.AuraTargetState); ok {
			child.Data28 = uint32(frame)
		}
	}

	return true
}

// chaosIceTurn is SrvDoFunc 35 (Royal Strike's chaos ice, 0x5ae210, verified):
// every Param1 (min 1) frames of elapsed life the missile reseeds itself with
// its data field 0x28 and steps the generator once; the heading (x, y), kept
// as two shorts in data field 0x2c, is turned a little to one side chosen by
// the low bit: bit 0 gives ((4x+y)/4, (4y-x)/4), bit 1 ((4x-y)/4, (4y+x)/4),
// truncating, a zero component becoming 1. The path is re-aimed at the integer
// position plus the new heading, the new seed low word goes back to field
// 0x28 and the heading to 0x2c.
func (s *Sim) chaosIceTurn(m *Missile) {
	period := m.Spec.Param[0]
	if period < 1 {
		period = 1
	}

	if m.Elapsed()%period != 0 {
		return
	}

	m.seed.Init(m.Data28)

	x, y := unpackShorts(m.Data2C)

	var a, b int
	if m.seed.Step()&1 == 0 {
		a, b = y, -x
	} else {
		a, b = -y, x
	}

	nx, ny := (a+x*4)/4, (b+y*4)/4
	if nx == 0 {
		nx = 1
	}

	if ny == 0 {
		ny = 1
	}

	aim(m, math.Floor(m.X)+float64(nx)+0.5, math.Floor(m.Y)+float64(ny)+0.5)
	m.Data28, m.Data2C = m.seed.Lo, packShorts(nx, ny)
}

// ---- hit functions ----

// howlHit is hit function 17 (Howl, 0x5a8aa0, verified): nothing without an
// owner or when the skill has no auratargetstate (1, the missile ends). Only a
// monster that does not carry the state yet and whose level is below
// Param2 + the caster's level + the skill level is frightened: it flees
// Param3 + (lvl-1)*Param4 subtiles for Param5 + (lvl-1)*Param6 frames
// (EventState). Always 0 for the rest: the missile flies on.
func (s *Sim) howlHit(m *Missile, t Target) int {
	if m.Owner.ID == "" {
		return resKill
	}

	if t == nil || t.IsPlayer() {
		return 0
	}

	sk := m.Skill
	if sk.AuraTargetState == "" {
		return resKill
	}

	if st, ok := t.(Stateful); ok && st.HasStateNamed(sk.AuraTargetState) {
		return 0
	}

	if t.Level() < sk.Param[2]+m.Owner.Level+m.Level {
		lv := m.Level - 1
		s.effect(m, Event{Kind: EventState, Missile: m, Target: t, Name: sk.AuraTargetState,
			Distance: sk.Param[3] + lv*sk.Param[4], Frames: sk.Param[5] + lv*sk.Param[6]})
	}

	return 0
}

// shoutHit is hit function 18 (Shout, Battle Orders, Battle Command; 0x5a8ba0,
// verified): a missile of a caster that is gone ends (1); a unit allied to the
// owner (0x552e40; taken as "not an enemy", UNVERIFIED) receives the skill's
// timed state (EventState, the time is auralencalc); anything else, and an
// ally included, returns 0 so the missile flies on.
func (s *Sim) shoutHit(m *Missile, t Target) int {
	if m.Owner.ID == "" {
		return resKill
	}

	if t != nil && !s.World.IsEnemy(m.Owner, t) {
		s.effect(m, Event{Kind: EventState, Missile: m, Target: t, Name: m.Skill.AuraTargetState, Frames: m.Skill.AuraLen})
	}

	return 0
}

// battleCryHit is hit function 21 (Battle Cry, 0x5a9000, verified): with no
// owner, no target, a skill without auratargetstate, or a target the filter
// rejects it returns 1 (the missile ends); otherwise the target gets the
// state for auralencalc frames with the skill's aura stats (EventState) and it
// returns 0. The filter is 0xa783 (enemies) when aurafilter is set, else none.
func (s *Sim) battleCryHit(m *Missile, t Target) int {
	if m.Owner.ID == "" || t == nil || m.Skill.AuraTargetState == "" {
		return resKill
	}

	if m.Skill.Filtered && !s.World.IsEnemy(m.Owner, t) {
		return resKill
	}

	s.effect(m, Event{Kind: EventState, Missile: m, Target: t, Name: m.Skill.AuraTargetState, Frames: m.Skill.AuraLen})

	return 0
}

// fistHit is hit function 22 (Fist of the Heavens' delay missile, 0x5ab7f0,
// verified): without HitSubMissile1 or an owner it ends (1); when the unit it
// marks is gone it returns 0. Otherwise the marked unit takes the missile's
// damage (the lightning, an EventHit) and every enemy within sHitPar1 (else
// aurarangecalc, min 1) of the missile, at most sHitPar2 (else calc4, min 1)
// of them, gets one HitSubMissile1 bolt aimed at it (the callback 0x5a9140
// creates them with the cast's owner, skill and level). That the bolts start
// at the missile and the order the units are visited in are UNVERIFIED: the
// sim visits them by unit id. Returns 1.
func (s *Sim) fistHit(m *Missile) int {
	sub := s.lookup(m.Spec.HitSubMissile[0])
	if m.Spec.HitSubMissile[0] == "" || m.Owner.ID == "" {
		return resKill
	}

	if m.Mark == nil {
		return 0
	}

	s.emit(Event{Kind: EventHit, Missile: m, Target: m.Mark, Damage: m.Damage.Roll(m.Owner.Roller)})

	if sub == nil {
		return resKill
	}

	f, ok := s.World.(Finder)
	if !ok {
		return resKill
	}

	radius := m.Spec.SHitPar[0]
	if radius < 1 {
		radius = m.AreaRadius
	}

	if radius < 1 {
		radius = 1
	}

	count := m.Spec.SHitPar[1]
	if count < 1 {
		count = m.FuryCount
	}

	if count < 1 {
		count = 1
	}

	cands := f.EnemiesWithin(m.Owner, m.X, m.Y, radius)
	sort.SliceStable(cands, func(i, j int) bool { return targetSerial(cands[i]) < targetSerial(cands[j]) })

	for _, c := range cands {
		if count == 0 {
			break
		}

		pt, ok := c.(Positioned)
		if !ok || !c.Alive() {
			continue
		}

		tx, ty := pt.SubPos()
		count--

		_, _ = s.Create(CreateParams{Spec: sub, Parent: m, Owner: m.Owner, SkillID: m.SkillID, Level: m.Level, Damage: m.Damage,
			X: m.X, Y: m.Y, DestX: tx, DestY: ty})
	}

	return resKill
}

// bladeHit is hit function 37 (Blade Sentinel's blade creeper, 0x5a9b50,
// verified bytes): 2 (damage, the missile stays) with a target, 0 without.
func (s *Sim) bladeHit(t Target) int {
	if t != nil {
		return resDamage
	}

	return 0
}

// contagionHit is hit function 53 (Rabies' contagion, 0x5aa5b0, verified): it
// needs an owner (the caster) and a target. With the frames left on the
// infection (data field 0x28, the frame it expires on, minus the game frame:
// the exe reads game+0xa8, the frame counter, as 0x5c5b00 does when it sets the
// expiry) at least 10 and at most the skill's elemental length, the target is
// infected for the time left (0x5c5dc0) and takes the damage (2); otherwise the
// missile ends (1). Infecting (0x5c5b00) only works on a unit that does not carry
// the state yet; then it also puts the plague missile (rabiesplague) on the new
// carrier (0x5c5ce0), marking the caster. EventState is sent only for a new
// infection.
func (s *Sim) contagionHit(m *Missile, t Target) int {
	if m.Owner.ID == "" || t == nil {
		return resKill
	}

	left := int(m.Data28) - s.World.Frame()
	if left < 10 || left > m.Skill.ElemLen {
		return resKill
	}

	if st, ok := t.(Stateful); ok && st.HasStateNamed(m.Skill.AuraTargetState) {
		return resDamage
	}

	s.effect(m, Event{Kind: EventState, Missile: m, Target: t, Name: m.Skill.AuraTargetState, Frames: left})

	if ow, ok := t.(Ownable); ok && m.plague != nil {
		carrier, caster := ow.AsOwner(), m.Owner

		x, y := 0.0, 0.0
		if carrier.Pos != nil {
			px, py := carrier.Pos()
			x, y = math.Floor(px)+0.5, math.Floor(py)+0.5
		}

		_, _ = s.Create(CreateParams{Spec: m.plague, Owner: carrier, MarkOwner: &caster, SkillID: m.SkillID, Level: m.Level,
			Damage: m.Damage, Skill: m.Skill, OnEffect: m.OnEffect, Stationary: true, X: x, Y: y, DestX: x, DestY: y})
	}

	return resDamage
}
