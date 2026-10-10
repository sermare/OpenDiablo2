package d2mapentity

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

var _ d2interface.MapEntity = &Monster{}

// MonsterEventKind says what finished or happened in a Monster's animation.
type MonsterEventKind int

// Monster events, drained by the owner with TakeEvents.
const (
	// MonsterEventHitFrame fires once when an attack/skill animation passes
	// its halfway point: the moment damage is applied.
	MonsterEventHitFrame MonsterEventKind = iota
	// MonsterEventModeDone fires when a one-shot mode (attack, skill, hit
	// recovery) has played once, or when a walk/run ends at its destination.
	MonsterEventModeDone
	// MonsterEventDied fires when the death animation has played once.
	MonsterEventDied
	// MonsterEventBlocked fires every rendering tick in which a step into
	// another unit's subtile was refused by the Blocker.
	MonsterEventBlocked
)

// MonsterEvent is one animation event.
type MonsterEvent struct {
	Kind MonsterEventKind
	Mode d2monster.Mode
}

// MonsterVitals are the combat numbers of a spawned monster, filled in by the
// owner from monstats/monlvl (see d2monsters).
type MonsterVitals struct {
	Level           int
	HP, MaxHP       int
	Defense         int
	Experience      int
	TreasureClass   string
	A1, A2          MonsterAttack
	S1              MonsterAttack // S1MinD/S1MaxD/S1TH: the damage of modes SC and S1 (VERIFIED 0x5a2960)
	Difficulty      d2monster.Difficulty
	DeathSeedSource uint32
}

// MonsterAttack is the damage profile of one attack mode.
type MonsterAttack struct {
	ToHit, Min, Max int
	// Elem is an added elemental part (skills.txt EType/EMin/EMax or a
	// missile's own columns), rolled min..max and reduced by the defender's
	// resistance; ElemType is "fire", "cold", "ltng", "mag" or "pois".
	ElemType         string
	ElemMin, ElemMax int
}

// Monster is a hostile unit: an animated composite driven by a
// d2monster.Brain. It only knows how to move and animate; the AI, combat and
// loot rules live in the owner (d2monsters).
type Monster struct {
	mapEntity
	composite *d2asset.Composite
	Stat      *d2records.MonStatRecord
	StatEx    *d2records.MonStat2Record
	Brain     *d2monster.Brain
	Vitals    MonsterVitals
	// TypeFlags is the per-spawn monster type byte (Game.exe monster data +0x16,
	// tested by 0x59dd60 with masks 0xa and 2). Not a monstats column.
	TypeFlags uint16
	// SuperUnique is the SuperUniques.txt key (empty for other monsters) and
	// SuperUniqueIdx its row's hcIdx (0 unless TypeFlags has MonTypeSuperUnique): the exe keeps the index in
	// the monster data next to the type word (unit data + 0x26).
	SuperUnique    string
	SuperUniqueIdx int
	// Modifiers are the monumod.txt ids the monster carries (super unique
	// Mod1..3, rolled champion / unique modifiers, or copied from the leader).
	Modifiers []int
	// LeaderID is the Brain id of the leader of a minion, 0 when none.
	LeaderID uint32
	name     string

	mode       d2monster.Mode
	hitFired   bool
	events     []MonsterEvent
	dead       bool
	deadTime   float64
	walkSpeed  float64
	runSpeed   float64
	slowPct    int // movement speed change in percent (negative slows), skills states
	selectable bool

	// Blocker, if set, is asked before the monster enters a new subtile; true
	// refuses the step and the monster stays where it was (unit-vs-unit
	// collision, see d2monsters).
	Blocker func(x, y int) bool
}

// subtile speed of a monstats velocity: V/16 subtile per 25 Hz frame
// (missiles-pathing.md section (c), VERIFIED), i.e. V*25/16 per second.
const velocityToSubtilesPerSecond = retailFps / 16.0

// ID returns the monster's uuid.
func (m *Monster) ID() string { return m.uuid }

// Monster type flag bits of TypeFlags (the word at +0x16 of the monster data, unit+0x14). All five masks and their
// names are VERIFIED from the exe's own predicates, which pass the mask to MONSTER_TestTypeFlags (0x4a8f50):
// MONSTER_IsSuperUnique 0x4aab10 (0x2), MONSTER_IsChampion 0x4aaa90 (0x4), MONSTER_IsUnique 0x4aaad0 (0x8),
// MONSTER_IsMinion 0x4aaaf0 (0x10) and MONSTER_IsGhostly 0x4aaab0 (0x40); the spawn packet writer
// SCMD_SendOpAC (0x53c110) tests them in the order 4, 8, 2, 0x10, 0x40. Setters seen in the exe:
// MONSTER_SetFlagsAndAddUniqueMod 0x5a2390 ORs in 0x5 (champion: 0x4 plus the bit 0x1), MONSTER_InitAsUniqueWithRolledMods
// 0x5a2410 ORs in 0x1 and rolls the unique modifiers, MONSTER_SpawnMinionGroup 0x59e7a0 ORs in 0x10 after
// MONAI_AddMinionToLeader. Two bits have no predicate: 0x1 (the unique modifiers were rolled) and 0x20 (ORed in by
// MONSTER_MakeChampion 0x59edb0 together with the minion level and experience penalty; its meaning is still unknown
// and that function's name does not match the 0x4 champion bit). structs-server.md line 36 had these wrong.
const (
	MonTypeModsRolled  uint16 = 0x1
	MonTypeSuperUnique uint16 = 0x2
	MonTypeChampion    uint16 = 0x4
	MonTypeUnique      uint16 = 0x8
	MonTypeMinion      uint16 = 0x10
	MonTypeGhostly     uint16 = 0x40
)

// Label is the monster's display name.
func (m *Monster) Label() string { return m.name }

// Selectable is true while the monster is alive.
func (m *Monster) Selectable() bool { return m.selectable && !m.dead }

// Alive reports whether the monster has not started dying.
func (m *Monster) Alive() bool { return !m.dead }

// Mode is the monster's current mode.
func (m *Monster) Mode() d2monster.Mode { return m.mode }

// GetPosition returns the monster's position.
func (m *Monster) GetPosition() d2vector.Position { return m.mapEntity.Position }

// GetVelocity returns the monster's velocity vector.
func (m *Monster) GetVelocity() d2vector.Vector { return m.mapEntity.velocity }

// GetSize returns the current frame size.
func (m *Monster) GetSize() (width, height int) { return m.composite.GetSize() }

// MonstatID returns the monstats class id.
func (m *Monster) MonstatID() int { return m.Stat.ID }

// AnimationFrames is the frame count of the current animation.
func (m *Monster) AnimationFrames() int { return m.composite.GetFrameCount() }

// SubtilePos is the integer subtile the monster stands on.
func (m *Monster) SubtilePos() (x, y int) {
	return int(m.Position.X()), int(m.Position.Y())
}

// Moving reports whether the monster still has somewhere to walk.
func (m *Monster) Moving() bool { return !m.atTarget() || m.hasPath() }

// TakeEvents returns and clears the pending animation events.
func (m *Monster) TakeEvents() []MonsterEvent {
	ev := m.events
	m.events = nil

	return ev
}

// Render draws the animated composite.
func (m *Monster) Render(target d2interface.Surface) {
	renderOffset := m.Position.RenderOffset()
	target.PushTranslation(
		int((renderOffset.X()-renderOffset.Y())*magicOffsetScalarY),
		int(((renderOffset.X()+renderOffset.Y())*magicOffsetScalarX)-magicOffsetX),
	)

	defer target.Pop()

	_ = m.composite.Render(target)
}

// animation mode used for each monster mode, with fallbacks for classes that
// lack a mode (monstats2 m* columns): run -> walk, A2/skills -> A1.
var modeFallback = map[d2monster.Mode][]d2enum.MonsterAnimationMode{
	d2monster.ModeDying:       {d2enum.MonsterAnimationModeDeath},
	d2monster.ModeDead:        {d2enum.MonsterAnimationModeDead, d2enum.MonsterAnimationModeDeath},
	d2monster.ModeNeutral:     {d2enum.MonsterAnimationModeNeutral},
	d2monster.ModeWalk:        {d2enum.MonsterAnimationModeWalk},
	d2monster.ModeRun:         {d2enum.MonsterAnimationModeRun, d2enum.MonsterAnimationModeWalk},
	d2monster.ModeGetHit:      {d2enum.MonsterAnimationModeGetHit},
	d2monster.ModeAttack1:     {d2enum.MonsterAnimationModeAttack1},
	d2monster.ModeAttack2:     {d2enum.MonsterAnimationModeAttack2, d2enum.MonsterAnimationModeAttack1},
	d2monster.ModeSkill1:      {d2enum.MonsterAnimationModeSkill1, d2enum.MonsterAnimationModeAttack1},
	d2monster.ModeSkill2:      {d2enum.MonsterAnimationModeSkill2, d2enum.MonsterAnimationModeSkill1, d2enum.MonsterAnimationModeAttack1},
	d2monster.ModeSkill3:      {d2enum.MonsterAnimationModeSkill3, d2enum.MonsterAnimationModeAttack1},
	d2monster.ModeSkill4:      {d2enum.MonsterAnimationModeSkill4, d2enum.MonsterAnimationModeAttack1},
	d2monster.ModeCast:        {d2enum.MonsterAnimationModeCast, d2enum.MonsterAnimationModeAttack1},
	d2monster.ModeSpecialCast: {d2enum.MonsterAnimationModeCast, d2enum.MonsterAnimationModeSkill1, d2enum.MonsterAnimationModeAttack1},
}

// HasMode reports whether the class has an animation for the mode itself (not a
// fallback). Without class data it reports true.
func (m *Monster) HasMode(mode d2monster.Mode) bool {
	fb := modeFallback[mode]
	if m.StatEx == nil || len(fb) == 0 {
		return true
	}

	return m.StatEx.HasAnimationMode[fb[0]]
}

// SetMode switches the unit to a mode and starts its animation. It returns
// false if the class has no animation for the mode or any fallback.
func (m *Monster) SetMode(mode d2monster.Mode) bool {
	for _, am := range modeFallback[mode] {
		if m.StatEx != nil && !m.StatEx.HasAnimationMode[am] {
			continue
		}

		if err := m.composite.SetMode(am, m.StatEx.BaseWeaponClass); err != nil {
			continue
		}

		m.mode = mode
		m.hitFired = false
		m.Brain.Mode = mode

		return true
	}

	return false
}

// MoveAlong starts walking (or running) along path. The monster keeps its
// current mode until it arrives; Advance then emits MonsterEventModeDone.
func (m *Monster) MoveAlong(path []d2vector.Position, run bool) bool {
	if len(path) == 0 {
		return false
	}

	mode, speed := d2monster.ModeWalk, m.walkSpeed
	if run {
		mode, speed = d2monster.ModeRun, m.runSpeed
	}

	speed = m.slowed(speed)

	if !m.SetMode(mode) {
		return false
	}

	m.SetSpeed(speed)
	m.SetPath(path, nil)

	return true
}

func (m *Monster) slowed(speed float64) float64 {
	if m.slowPct == 0 {
		return speed
	}

	f := float64(100+m.slowPct) / 100
	if f < 0.05 {
		f = 0.05
	}

	return speed * f
}

// SetSlow changes the movement speed by pct percent (negative slows; 0
// restores it), also while the monster is walking.
func (m *Monster) SetSlow(pct int) {
	m.slowPct = pct

	switch m.mode {
	case d2monster.ModeWalk:
		m.SetSpeed(m.slowed(m.walkSpeed))
	case d2monster.ModeRun:
		m.SetSpeed(m.slowed(m.runSpeed))
	}
}

// StopMoving stops walking and returns to neutral.
func (m *Monster) StopMoving() {
	m.mapEntity.StopMoving()

	if m.mode == d2monster.ModeWalk || m.mode == d2monster.ModeRun {
		m.SetMode(d2monster.ModeNeutral)
		m.events = append(m.events, MonsterEvent{MonsterEventModeDone, d2monster.ModeWalk})
	}
}

// Face turns the monster toward a position (in subtiles).
func (m *Monster) Face(x, y float64) {
	p := d2vector.NewPosition(x, y)
	m.composite.SetDirection(m.Position.DirectionTo(p.Vector))
}

// Die starts the death animation. It is idempotent.
func (m *Monster) Die() {
	if m.dead {
		return
	}

	m.dead = true
	m.mapEntity.StopMoving()

	if !m.SetMode(d2monster.ModeDying) {
		m.SetMode(d2monster.ModeDead)
	}

	m.Brain.Mode = d2monster.ModeDying
}

// DropHitEvents discards pending attack hit-frame events: a monster that is
// hit while winding up loses its blow (hit recovery).
func (m *Monster) DropHitEvents() {
	kept := m.events[:0]

	for _, ev := range m.events {
		if ev.Kind != MonsterEventHitFrame {
			kept = append(kept, ev)
		}
	}

	m.events = kept
}

// CorpseAge is the time in seconds since the monster finished dying.
func (m *Monster) CorpseAge() float64 { return m.deadTime }

// Advance processes one rendering tick.
func (m *Monster) Advance(tickTime float64) {
	if !m.dead {
		m.stepBlocked(tickTime)
	}

	if err := m.composite.Advance(tickTime); err != nil {
		return
	}

	m.checkEvents(tickTime)
}

// stepBlocked moves along the path unless the next subtile is refused.
func (m *Monster) stepBlocked(tickTime float64) {
	if m.Blocker == nil {
		m.Step(tickTime)

		return
	}

	px, py := m.Position.X(), m.Position.Y()
	ox, oy := m.SubtilePos()

	m.Step(tickTime)

	if nx, ny := m.SubtilePos(); (nx != ox || ny != oy) && m.Blocker(nx, ny) {
		m.Position.Set(px, py)
		m.events = append(m.events, MonsterEvent{MonsterEventBlocked, m.mode})
	}
}

func (m *Monster) checkEvents(tickTime float64) {
	played := m.composite.GetPlayedCount() >= 1
	frameCount := m.composite.GetFrameCount()
	if frameCount < 1 {
		frameCount = 1
	}

	half := float64(m.composite.GetCurrentFrame())/float64(frameCount) >= 0.5

	switch m.mode {
	case d2monster.ModeWalk, d2monster.ModeRun:
		if !m.Moving() {
			m.SetMode(d2monster.ModeNeutral)
			m.events = append(m.events, MonsterEvent{MonsterEventModeDone, d2monster.ModeWalk})
		}
	case d2monster.ModeAttack1, d2monster.ModeAttack2, d2monster.ModeSkill1, d2monster.ModeSkill2,
		d2monster.ModeSkill3, d2monster.ModeSkill4, d2monster.ModeCast, d2monster.ModeSpecialCast:
		if half && !m.hitFired {
			m.hitFired = true
			m.events = append(m.events, MonsterEvent{MonsterEventHitFrame, m.mode})
		}

		if played {
			done := m.mode
			m.SetMode(d2monster.ModeNeutral)
			m.events = append(m.events, MonsterEvent{MonsterEventModeDone, done})
		}
	case d2monster.ModeGetHit:
		if played {
			m.SetMode(d2monster.ModeNeutral)
			m.events = append(m.events, MonsterEvent{MonsterEventModeDone, d2monster.ModeGetHit})
		}
	case d2monster.ModeDying:
		if played {
			m.events = append(m.events, MonsterEvent{MonsterEventDied, d2monster.ModeDying})

			if !m.SetMode(d2monster.ModeDead) {
				m.mode = d2monster.ModeDead
			}

			m.Brain.Mode = d2monster.ModeDead
		}
	case d2monster.ModeDead:
		m.deadTime += tickTime
	}
}

// rotate sets the facing; movement modes are chosen by the owner.
func (m *Monster) rotate(direction int) {
	if m.composite.GetDirection() != direction {
		m.composite.SetDirection(direction)
	}
}

// Revive brings a dead monster back to life in neutral mode (used for
// mercenaries; the original sets mode NU and refills the hit points).
func (m *Monster) Revive() {
	m.dead = false
	m.deadTime = 0
	m.events = nil
	m.mapEntity.StopMoving()
	m.SetMode(d2monster.ModeNeutral)
	m.Brain.Mode = d2monster.ModeNeutral
	m.Brain.HasTarget = false
}

// TeleportTo puts the monster on a subtile without walking there.
func (m *Monster) TeleportTo(x, y int) {
	m.mapEntity.StopMoving()
	m.Position.Set(float64(x), float64(y))
	m.Target = m.Position
}

// SetSelectable chooses whether the mouse can pick the monster (mercenaries
// are not attackable by their owner).
func (m *Monster) SetSelectable(v bool) { m.selectable = v }

// SetLabel overrides the display name (mercenaries show their own name).
func (m *Monster) SetLabel(name string) { m.name = name }
