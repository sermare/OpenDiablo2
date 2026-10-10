package d2monster

// Batch 3 of faithful ports (rules in d2-re-notes/ai-faithful-batch3.md): the
// think functions DesertTurret 0x5df7c0, CatapultSpotter 0x5ed160, Imp
// 0x5e1ef0 (pre 0x5e1ee0), DeathSentry 0x5e99f0 (pre 0x5e9340, helper
// 0x5e9360), DruidBear 0x5ec860 and DruidWolf 0x5ec840 (selectors 0x5ebff0 /
// 0x5ec3c0), plus DarkWanderer 0x5e91e0 and BladeCreeper 0x5e95e0 (pre
// 0x5e95c0). The file name sorts before ai_rest.go, so these register first
// and the stand-ins there skip them. Whatever needs the engine (skill levels,
// corpses, quest state, free cells) is an optional host interface; without it
// the documented fallback applies.
//
// Scratch[0..2] are AiGeneral +0x14/+0x18/+0x1c.

// DirectionMapper is UNIT_GetDirectionToPoint + DIR_GetMappedDirectionByte
// (0x621f80 / 0x63f810): the 0..7 direction index from the unit to a point.
// UNVERIFIED mapping, so the fallback picks the index whose sign vector in
// turretVec matches the offset.
type DirectionMapper interface {
	MappedDirection(b *Brain, x, y int) int
}

// PointCaster is MONAI_CanCastSkillAtTarget (0x5fc6b0) for a ground point;
// absent means "can cast".
type PointCaster interface {
	CanCastAtPoint(b *Brain, slot int, p Point) bool
}

// FreeCellFinder is COLLISION_FindNearestFreeCellMax50 (CatapultSpotter,
// size 2, mask 0x805, 3 tries): the free cell nearest to p, ok false when
// none. Absent means p itself is free.
type FreeCellFinder interface {
	FreeCellNear(b *Brain, p Point) (Point, bool)
}

// DeadPlayerProbe is MONAI_CatapultSpotterComputeAimOffset 0x5ed080: three
// probes around the spotter's room for a player unit in the dead mode
// (ROOM_HasDeadPlayerUnitAtPoint). Absent means none.
type DeadPlayerProbe interface {
	DeadPlayerNear(b *Brain) bool
}

// SkillCalc gives the skill level of a monstats slot on the unit
// (SKILL_GetTotalLevel, 0 when the unit lacks the skill) and the skills.txt
// calc4 value (SKILL_EvalCalcBytecode) at that level. Fallbacks: level 1 when
// the slot is used, calc4 0 (so a sentry would expire at once, hence
// sentryDefaultLife).
type SkillCalc interface {
	SkillLevel(b *Brain, slot int) int
	SkillCalc4(b *Brain, slot, level int) int
}

// ReviveHost serves DeathSentry: SKILL_FindReviveCorpseNearUnit and the
// skill's linear radius (SKILLS_CalcLinearValueByLevel).
type ReviveHost interface {
	FindReviveCorpse(b *Brain, slot, level int) (Target, bool)
	ReviveRadius(b *Brain, slot, level int) int
}

// ClassAIPSource gives aipN of another monstats class at the game
// difficulty (Imp reads the rows 0x1ec..0x1ef, not its own).
type ClassAIPSource interface {
	ClassAIP(b *Brain, class, n int) (int, bool)
}

// ImpHost serves the Imp's remembered unit and its offset cast.
type ImpHost interface {
	// ImpLinkedUnit is the monster with unit id id when it is alive, not
	// owned, and has alignment 0; ok false resets the link.
	ImpLinkedUnit(b *Brain, id uint32) (Target, bool)
	// RandomOffsetCast is MONAI_QueueRandomOffsetSkillCast(unit, range,
	// skill, mode) 0x5de650: a random point around the unit.
	RandomOffsetCast(b *Brain, slot, rng int) bool
}

// UnitFinder is SERVER_FindUnitByIdAndType for players (type 0).
type UnitFinder interface {
	PlayerByID(b *Brain, id uint32) (Target, bool)
}

// ChaseProbe is MONAI_PickChaseOffsetByEdgeDistance 0x5db3a0: line probes to
// the target; the exe's returned flag is not recoverable from the (void)
// decompilation, so true ("reachable") is the fallback. UNVERIFIED.
type ChaseProbe interface {
	ChaseOK(b *Brain, t Target) bool
}

// QuestWanderer is the Act 3 Dark Wanderer quest node (id 0x1c): the walk
// target, ok false when the quest is not active, and the one-shot arrival
// timer (QUEST_A3_DarkWanderer_StartTimerOnce).
type QuestWanderer interface {
	WandererTarget(b *Brain) (Point, bool)
	WandererArrived(b *Brain)
}

// BladeHost serves the Blade Creeper's self-cast missile.
type BladeHost interface {
	// BladeReady is false when the unit lacks its Skill1 (the creeper dies).
	// delay is skills.txt calc4 at the skill level.
	BladeReady(b *Brain) (delay int, ok bool)
	// BladeFire creates the skill's server missile (srvmissilea) at the
	// creeper's own position, owned by the leader (or itself), and copies the
	// attack rating (stat 0x13) and mastery stat (0x77) onto it.
	BladeFire(b *Brain) bool
}

const (
	desertTurretSleep   = 10
	desertTurretFar     = 15
	spotterRoll         = 3
	spotterNoCell       = 15
	spotterAttackMode   = ModeAttack1
	sentryDefaultLife   = 1000 // used only when the host gives no calc4 (unverified)
	bearTeleportDist    = 50   // 0x32
	bearRunDist         = 28   // 0x1c
	bearGapDist         = 18   // 0x12
	townWolfSleep       = 33   // 0x21
	impClassA           = 0x1ec
	impClassB           = 0x1ed
	impClassC           = 0x1ee
	impClassD           = 0x1ef
	darkWandererRetries = 3
	wolfClassA          = 0x1a4
	bladeWaitIdle       = 3
)

// turretVec is the 8-entry (dx, dy) table at 0x6e4930; turretNext the 8x8
// transition table at 0x6e4970, indexed [direction][previous].
var turretVec = [8][2]int{{1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1}, {1, 0}}

var turretNext = [8][8]int{
	{0, 0, 1, 2, 5, 6, 7, 0},
	{1, 1, 1, 2, 3, 6, 7, 0},
	{1, 2, 2, 2, 3, 4, 7, 0},
	{1, 2, 3, 3, 3, 4, 5, 6},
	{1, 2, 3, 4, 4, 4, 5, 6},
	{7, 2, 3, 4, 5, 5, 5, 6},
	{7, 0, 3, 4, 5, 6, 6, 6},
	{7, 0, 1, 4, 5, 6, 7, 7},
}

// spotterSkills is the 5-entry skill id table at 0x6e4aa4.
var spotterSkills = [5]int{0x11f, 0x120, 0x12f, 0x130, 0x131}

func init() {
	register("DesertTurret", standInMode("DesertTurret"), thinkDesertTurret)
	register("CatapultSpotter", standInMode("CatapultSpotter"), thinkCatapultSpotter)
	register("Imp", standInMode("Imp"), thinkImp)
	register("DeathSentry", standInMode("DeathSentry"), thinkDeathSentry)
	register("DruidBear", standInMode("DruidBear"), thinkDruidBear)
	register("DruidWolf", standInMode("DruidWolf"), thinkDruidWolf)
	register("DarkWanderer", standInMode("DarkWanderer"), thinkDarkWanderer)
	register("BladeCreeper", standInMode("BladeCreeper"), thinkBladeCreeper)
}

// FaithfulBatch3 lists the names ported in this file.
func FaithfulBatch3() []string {
	return []string{"DesertTurret", "CatapultSpotter", "Imp", "DeathSentry", "DruidBear", "DruidWolf",
		"DarkWanderer", "BladeCreeper"}
}

func (c *Ctx) castAtPoint(slot int, p Point) bool {
	return c.Cast(slot, Target{X: p.X, Y: p.Y})
}

func (c *Ctx) canCastAtPoint(slot int, p Point) bool {
	if g, ok := c.W.(PointCaster); ok {
		return g.CanCastAtPoint(c.B, slot, p)
	}

	return true
}

func (c *Ctx) mappedDirection(x, y int) int {
	if m, ok := c.W.(DirectionMapper); ok {
		return m.MappedDirection(c.B, x, y) & 7
	}

	sx, sy := sign(x-c.B.X), sign(y-c.B.Y)
	for i, v := range turretVec {
		if v[0] == sx && v[1] == sy {
			return i
		}
	}

	return 0
}

func (c *Ctx) skillLevel(slot int) int {
	if s, ok := c.W.(SkillCalc); ok {
		return s.SkillLevel(c.B, slot)
	}

	if c.skill(slot) {
		return 1
	}

	return 0
}

// thinkDesertTurret is MONAI_Think_DesertTurret (VERIFIED from the
// disassembly). S0 next-shot frame, S1 shots in the burst, S2 the rotating
// direction index. aip1 gap between shots, aip2 burst size, aip3 reload,
// aip4 range, aip5 scatter distance.
func thinkDesertTurret(c *Ctx) {
	b := c.B
	s := &b.Scratch
	frame := c.frame()

	// Opening: with Skill1 and no timer yet, one cast at (0,0) and arm S0.
	if c.skill(slotA) && s[0] == 0 {
		c.setMode(b.Profile.Skills[slotA].Mode)
		s[2] = 0
		s[0] = frame

		return
	}

	if frame < s[0] {
		c.Sleep(desertTurretSleep)

		return
	}

	if c.Dist > b.AIP(4) || c.Target == nil {
		if s[1] > 0 {
			s[1]--
		}

		c.Sleep(desertTurretFar)

		return
	}

	t := *c.Target
	dir := c.mappedDirection(t.X, t.Y)
	s[2] = turretNext[dir][s[2]&7]
	v := turretVec[s[2]]
	p := Point{b.X + v[0]*b.AIP(5), b.Y + v[1]*b.AIP(5)}

	if c.skill(slotA) && c.canCastAtPoint(slotA, p) && c.canCastAtPoint(slotA, Point{t.X, t.Y}) {
		c.castAtPoint(slotA, p)

		s[1]++
		if s[1] > b.AIP(2) {
			s[0] = b.AIP(3) + frame
			s[1] = 0
		} else {
			s[0] = b.AIP(1) + frame
		}

		return
	}

	if s[1] > 0 {
		s[1]--
	}

	c.Sleep(desertTurretSleep)
}

// thinkCatapultSpotter is MONAI_Think_CatapultSpotter. S0 the chosen skill
// index, S1 shots left in the volley, S2 the frame of the last volley shot
// (1 after the first think). aip1 fire percent, aip2 gap, aip4 scatter, aip5
// volley size.
func thinkCatapultSpotter(c *Ctx) {
	b := c.B
	s := &b.Scratch
	frame := c.frame()

	probe := s[2] == 0 || b.Roll(100) < spotterRoll
	if probe {
		if p, ok := c.W.(DeadPlayerProbe); ok && p.DeadPlayerNear(b) {
			c.setMode(ModeDying) // UNVERIFIED: the exe queues mode 0 on the target slot
			return
		}
	}

	if s[2] == 0 {
		s[2] = 1
	}

	wait := b.AIP(2)

	if wait-frame+s[2] < 1 {
		if c.Target != nil && b.Roll(100) < b.AIP(1) {
			skill := s[0]

			if s[1] < 1 {
				s[0] = b.Roll(5)
				skill = s[0]
				s[1] = b.AIP(5)
			} else {
				s[1]--
			}

			sp := b.AIP(4)
			x := b.Roll(sp*2) - sp + c.Target.X
			y := b.Roll(sp*2) - sp + c.Target.Y
			pt := Point{x, y}

			if f, ok := c.W.(FreeCellFinder); ok {
				q, found := f.FreeCellNear(b, pt)
				if !found {
					c.Sleep(spotterNoCell)

					return
				}

				pt = q
			}

			c.castSkillIDAt(spotterSkills[skill%len(spotterSkills)], pt)
			s[2] = frame

			return
		}
	}

	c.Sleep(wait)
}

// castSkillIDAt casts a skill by id (A1 mode) at a ground point.
func (c *Ctx) castSkillIDAt(id int, p Point) {
	if r, ok := c.W.(RawCaster); ok {
		if r.CastSkillID(c.B, id, spotterAttackMode, nil, &p) {
			c.busy()

			return
		}
	} else if c.skill(slotA) {
		c.castAtPoint(slotA, p)

		return
	}

	c.Sleep(10)
}

// impAIP is aipN of one of the exe's fixed Imp rows (falls back to the unit's
// own aip when the host cannot give the class).
func (c *Ctx) impAIP(class, n int) int {
	if s, ok := c.W.(ClassAIPSource); ok {
		if v, ok := s.ClassAIP(c.B, class, n); ok {
			return v
		}
	}

	return c.B.AIP(n)
}

func (c *Ctx) impOffsetCast(rng int) bool {
	if h, ok := c.W.(ImpHost); ok && h.RandomOffsetCast(c.B, slotA, rng) {
		c.busy()

		return true
	}

	return false
}

// thinkImp is MONAI_Think_Imp (pre: S0 = -1). S0 holds a remembered unit id.
// Tunables come from the monstats rows 0x1ec..0x1ef: A = 0x1ec, B = 0x1ed,
// C = 0x1ee, D = 0x1ef (VERIFIED indices, the roles of the rows UNVERIFIED).
func thinkImp(c *Ctx) {
	b := c.B
	s := &b.Scratch

	c.pre(func() { s[0] = -1 })

	var t Target
	if c.Target != nil {
		t = *c.Target
	}

	if s[0] != -1 {
		if h, ok := c.W.(ImpHost); ok {
			if u, found := h.ImpLinkedUnit(b, uint32(s[0])); found {
				r := c.impAIP(impClassB, 1)
				dx, dy := u.X-b.X, u.Y-b.Y

				if r*r < dx*dx+dy*dy {
					c.WalkTo(u, 0)

					return
				}

				if c.skill(slotA) {
					c.Cast(slotA, u)

					return
				}
			} else {
				s[0] = -1
			}
		} else {
			s[0] = -1
		}
	}

	if c.InRange {
		if c.skill(slotA) && b.HPPercent < c.impAIP(impClassA, 1) {
			if c.impOffsetCast(c.impAIP(impClassA, 2)) {
				return
			}
		}

		if b.Roll(100) < c.impAIP(impClassC, 2) && c.WalkAway(t, 5) {
			return
		}
	}

	if c.skill(slotA) && b.Roll(100) < c.impAIP(impClassA, 3) {
		if c.impOffsetCast(c.impAIP(impClassA, 2)) {
			return
		}
	}

	if c.Dist < c.impAIP(impClassC, 1) {
		if b.Roll(100) < c.impAIP(impClassC, 2) {
			c.WalkAway(t, 5)

			return
		}
	}

	if c.skill(3) {
		if at, d, ok := c.W.AttackTarget(b); ok {
			if d < c.impAIP(impClassC, 3) {
				if b.Roll(100) < c.impAIP(impClassC, 4) {
					c.Cast(3, at)

					return
				}
			} else if d < c.impAIP(impClassD, 3) && b.Roll(100) < c.impAIP(impClassD, 4) {
				c.Cast(3, at)

				return
			}
		}
	}

	if b.Roll(100) < 0x21 {
		c.WalkTo(t, 4)

		return
	}

	if b.Roll(100) > 0x13 {
		c.Sleep(10)

		return
	}

	c.Wander(8)
}

// sentryFollow is MONAI_Sentry_FollowLeaderOrIdle 0x5e9360: true when it
// ended the tick (death queued). S1 is the remaining life counter (-1 until
// first use, then skill calc4).
func sentryFollow(c *Ctx, dec bool) bool {
	b := c.B
	s := &b.Scratch

	if _, ok := fbnLeader(c); !ok || c.townRoom() {
		c.setMode(ModeDying)

		return true
	}

	if s[1] < 0 {
		if !c.skill(slotA) {
			c.setMode(ModeDying)

			return true
		}

		lv := c.skillLevel(slotA)
		if lv == 0 {
			c.setMode(ModeDying)

			return true
		}

		if sc, ok := c.W.(SkillCalc); ok {
			s[1] = sc.SkillCalc4(b, slotA, lv)
		} else {
			s[1] = sentryDefaultLife
		}
	}

	if s[1] > 0 {
		if dec {
			s[1]--
		}

		return false
	}

	c.setMode(ModeDying)

	return true
}

// thinkDeathSentry is MONAI_Think_DeathSentry. The pre-hook (0x5e9340) sets
// S0 = frame and S1 = -1; S0 later holds the id of the corpse last raised.
// aip2 idle sleep, aip3 percent for the Skill2 attack, aip4 range.
func thinkDeathSentry(c *Ctx) {
	b := c.B
	s := &b.Scratch

	c.pre(func() { s[0], s[1] = c.frame(), -1 })

	if sentryFollow(c, false) {
		return
	}

	lv := c.skillLevel(slotA)
	if !c.skill(slotA) || lv <= 0 {
		return
	}

	t, d, ok := c.W.AttackTarget(b)
	if !ok {
		c.Sleep(b.AIP(2))

		return
	}

	var corpse Target

	found := false
	radius := 0

	if r, ok := c.W.(ReviveHost); ok {
		corpse, found = r.FindReviveCorpse(b, slotA, lv)
		radius = r.ReviveRadius(b, slotA, lv)
	}

	if !found || int(corpse.ID) == s[0] || radius/2 <= Distance(t.X-corpse.X, t.Y-corpse.Y) {
		if d < b.AIP(4) && b.Roll(100) < b.AIP(3) {
			if sentryFollow(c, true) {
				return
			}

			c.Cast(slotB, t)

			return
		}

		c.Sleep(b.AIP(2))

		return
	}

	if !sentryFollow(c, true) {
		s[0] = int(corpse.ID)
		c.Cast(slotA, corpse)
	}
}

func (c *Ctx) chaseOK(t Target) bool {
	if p, ok := c.W.(ChaseProbe); ok {
		return p.ChaseOK(c.B, t)
	}

	return true
}

// petSpeedBoost is the exe's (Run*100/Walk)-100 for the pets: 100 when the
// class has no walk velocity or the excess exceeds 99.
func (p *Profile) petSpeedBoost() int {
	if p.Walk < 1 {
		return 100
	}

	if v := p.Run*100/p.Walk - 100; v <= 99 {
		return v
	}

	return 100
}

// petTargets runs the shared two-stage enemy search of the Druid pets: the
// nearest enemy within radius (with its in-range flag) if it passes the
// chase probe, else the nearest enemy at all when closer than radius and
// passing the probe (then the in-range flag stays 0, as in the exe).
func petTargets(c *Ctx, radius int) (sel *Target, inRange bool) {
	if t, d, ok := fbnEnemy(c, radius); ok && c.chaseOK(t) {
		return &t, c.W.InRange(c.B, t, d)
	}

	if t, d, ok := fbnEnemy(c, 0); ok && d < radius && c.chaseOK(t) {
		return &t, false
	}

	return nil, false
}

// thinkDruidBear is MONAI_Think_DruidBear (VERIFIED flow). aip1 attack wait,
// aip2 percent to boost speed when closing, aip3 percent below which the
// Skill1 cast replaces the swing.
func thinkDruidBear(c *Ctx) {
	b := c.B
	s := &b.Scratch

	l, ok := fbnLeader(c)
	if !ok {
		fbnPetGone(c)

		return
	}

	s[2] = int(l.ID)
	d := EdgeDistance(b.X-l.X, b.Y-l.Y, b.Size)
	boost := b.Profile.petSpeedBoost()

	if d > bearTeleportDist {
		fbnFollow(c, l, 3, 0, 0)

		return
	}

	if d > bearRunDist {
		fbnFollow(c, l, 0, 100, 0)

		return
	}

	if d > bearGapDist {
		if (l.Mode == ModeWalk || l.Mode == ModeGetHit) && fbnFollow(c, l, 0, 0, 0) {
			return
		}

		if l.Mode == 3 && fbnFollow(c, l, 0, 100, 0) {
			return
		}
	}

	sel, inRange := petTargets(c, bearRunDist)

	if !inRange {
		if sel != nil {
			if b.Roll(100) < b.AIP(2) {
				c.SetSpeed(boost)
			}

			c.walkNoTarget(*sel)

			return
		}
	} else if sel != nil {
		if b.Roll(100) >= b.AIP(3) {
			if c.Attack(ModeAttack1, *sel) {
				c.Sleep(b.AIP(1))
			}

			return
		}

		c.Cast(slotA, *sel)

		return
	}

	if d <= bearGapDist-2 {
		c.Sleep(15)

		return
	}

	fbnFollow(c, l, 0, 0, 0)
}

// SkillTargetSearcher is MONAI_SearchNearbyUnitsForSkillTarget(leader, 10):
// a unit near the pack leader the wolf's Skill1 may target.
type SkillTargetSearcher interface {
	NearbySkillTarget(b *Brain, radius int) (Target, bool)
}

// thinkDruidWolf is MONAI_Think_DruidWolf: class 0x1a4 runs selector A
// (calls with Skill1, simple pack logic), every other wolf selector B (calls
// with Skill2, the 0x8a state gate and the Skill1 pounce). The exe schedules
// a follow-up event (frame +2/+4/+6) after the leader-call casts; that
// event's handler is unknown, so only the wait is ported (UNVERIFIED).
func thinkDruidWolf(c *Ctx) {
	if c.B.Class == wolfClassA {
		wolfSelect(c, slotA)
	} else {
		wolfSelect(c, slotB)
	}
}

// wolfSelect is the common frame of selectors A (slot Skill1) and B (slot
// Skill2). aip1 attack wait, aip2 wander percent, aip3 leash for the follow
// modes (A) / pounce percent (B), aip4 pack radius, aip5 follow distance.
func wolfSelect(c *Ctx, slot int) {
	b := c.B
	s := &b.Scratch

	l, ok := fbnLeader(c)
	if !ok {
		if s[2] != 0 && c.skill(slot) {
			if u, found := c.playerByID(uint32(s[2])); found {
				c.castAtPoint(slot, Point{u.X, u.Y})
				c.Sleep(8)

				return
			}
		}

		c.Sleep(10)

		return
	}

	s[2] = int(l.ID)

	if c.townRoom() || fbnInTown(c) {
		if !fbnSelectPet(c, nil, l, false, 6) {
			c.Sleep(townWolfSleep)
		}

		return
	}

	rad := b.AIP(4)
	sel, inRange := petTargets(c, rad)
	boost := b.Profile.petSpeedBoost()
	d := EdgeDistance(b.X-l.X, b.Y-l.Y, b.Size)
	lt := Target{ID: l.ID, X: l.X, Y: l.Y, Size: l.Size}

	if slot == slotB && sel != nil && d > rad &&
		EdgeDistance(sel.X-l.X, sel.Y-l.Y, sel.Size) > rad {
		sel = nil
	}

	if c.skill(slot) && d > bearTeleportDist {
		c.castAtPoint(slot, Point{l.X, l.Y})
		c.Sleep(10)

		return
	}

	if slot == slotB {
		if d > b.AIP(5) {
			fbnFollow(c, l, 0, 100, 0)

			return
		}

		if d > rad {
			if l.Mode == 3 {
				fbnFollow(c, l, 0, 100, 0)

				return
			}

			if l.Mode == ModeWalk || l.Mode == ModeGetHit {
				fbnFollow(c, l, 0, 0, 0)

				return
			}
		}
	} else {
		if d > b.AIP(5) && fbnFollow(c, l, 0, 100, 0) {
			return
		}

		if d > b.AIP(3) {
			if (l.Mode == ModeWalk || l.Mode == ModeGetHit) && fbnFollow(c, l, 0, 0, 0) {
				return
			}

			if l.Mode == 3 && fbnFollow(c, l, 0, 100, 0) {
				return
			}
		}
	}

	if fbnSelectPet(c, sel, l, inRange, 6) {
		return
	}

	if slot == slotB {
		c.wolfB(sel, inRange, boost)

		return
	}

	if sel == nil {
		if b.Roll(100) < b.AIP(2) {
			c.Wander(10)

			return
		}

		c.Sleep(15)

		return
	}

	if inRange {
		if c.Attack(ModeAttack1, *sel) {
			c.Sleep(b.AIP(1))
		}

		return
	}

	if EdgeDistance(sel.X-l.X, sel.Y-l.Y, sel.Size) < rad {
		c.SetSpeed(boost)
		c.RunTo(*sel, 0)

		return
	}

	if d > 10 {
		c.WalkToRange(lt, 8, 6)

		return
	}

	c.Sleep(15)
}

// wolfB is selector B's tail: a Skill1 pounce on a unit near the leader
// (unless state 0x8a is on the wolf), S0/S1 remember a foe it is running at,
// then the plain chase/attack/wander.
func (c *Ctx) wolfB(sel *Target, inRange bool, boost int) {
	b := c.B
	s := &b.Scratch

	if !c.skill(slotA) || c.W.HasState(b, 0x8a) ||
		(b.Roll(100) >= b.AIP(3) && sel != nil && s[0] == 0) {
		s[0] = 0
	} else if f, ok := c.W.(SkillTargetSearcher); ok {
		if u, found := f.NearbySkillTarget(b, 10); found &&
			EdgeDistance(b.X-u.X, b.Y-u.Y, b.Size) < b.AIP(4)/2 {
			d := EdgeDistance(b.X-u.X, b.Y-u.Y, b.Size)

			switch {
			case c.W.InRange(b, u, d):
				s[0] = 0
				c.Cast(slotA, u)

				return
			case !inRange:
				c.RunTo(u, 0)

				s[0], s[1] = 1, int(u.ID)

				return
			}

			if sel != nil && c.Attack(ModeAttack1, *sel) {
				c.Sleep(b.AIP(1))
			}

			return
		}
	}

	if !inRange {
		if sel != nil {
			c.SetSpeed(boost)
			c.RunTo(*sel, 0)

			return
		}

		if b.Roll(100) >= b.AIP(2) {
			c.Sleep(15)

			return
		}

		c.Wander(10)

		return
	}

	if sel != nil && c.Attack(ModeAttack1, *sel) {
		c.Sleep(b.AIP(1))
	}
}

func (c *Ctx) playerByID(id uint32) (Target, bool) {
	if u, ok := c.W.(UnitFinder); ok {
		return u.PlayerByID(c.B, id)
	}

	return Target{}, false
}

// thinkDarkWanderer is MONAI_Think_DarkWanderer (VERIFIED flow). S0 phase
// (0 init, 1 start walking, 2 walking), S1 retry count. It needs the quest
// host; without it the NPC just waits 10.
func thinkDarkWanderer(c *Ctx) {
	b := c.B
	s := &b.Scratch

	q, ok := c.W.(QuestWanderer)
	if !ok {
		c.Sleep(10)

		return
	}

	p, active := q.WandererTarget(b)
	if !active {
		c.Sleep(10)

		return
	}

	if s[0] == 0 {
		s[0], s[1], s[2] = 1, 0, 0
	}

	if c.Target == nil {
		c.Sleep(10)

		return
	}

	if c.Dist < 20 {
		if s[0] == 1 {
			fbnWalkPoint(c, p)
			s[0] = 2

			return
		}

		if s[0] == 2 {
			if fbnTiles(b, p.X, p.Y) < 2 || s[1] >= darkWandererRetries {
				c.die()
				q.WandererArrived(b)

				return
			}

			s[1]++
			c.SetSpeed(0)
			fbnWalkPoint(c, p)

			return
		}
	}

	c.Sleep(40)
}

// thinkBladeCreeper is MONAI_Think_BladeCreeper (pre 0x5e95c0: S0 = -1, S1
// = 1, S2 = 0). S0 the expiry frame (-1 until armed from calc4), S1 selects
// which of the two queued waypoints is next, S2 the missile-laid flag. Only
// the leader's command queue matters: commands carry two points.
func thinkBladeCreeper(c *Ctx) {
	b := c.B
	s := &b.Scratch

	c.pre(func() { s[0], s[1], s[2] = -1, 1, 0 })

	h, ok := c.W.(BladeHost)
	if !ok {
		c.setMode(ModeDying)

		return
	}

	delay, ready := h.BladeReady(b)
	if !ready {
		c.setMode(ModeDying)

		return
	}

	if s[0] < 0 {
		s[0] = delay + c.frame()
	}

	if c.frame() > s[0] {
		c.setMode(ModeDying)

		return
	}

	if s[2] == 0 {
		h.BladeFire(b)

		s[2] = 1
	}

	cmd := b.PeekCommand()
	if cmd == nil {
		c.Sleep(bladeWaitIdle)

		return
	}

	c.SetSpeed(0)

	// The command node holds point A at +0xc/+0x10 (X, Y) and point B at
	// +0x14/+0x18 (Count, Delay in Command). The creeper tries the point
	// S1 selects, flips S1 and tries the other one when the walk fails.
	pa, pb := Point{cmd.X, cmd.Y}, Point{cmd.Count, cmd.Delay}
	first, second := pa, pb

	if s[1] != 0 {
		first, second = pb, pa
	}

	if c.move(first, nil, 0, false) {
		return
	}

	if s[1] == 0 {
		s[1] = 1
	} else {
		s[1] = 0
	}

	if c.move(second, nil, 0, false) {
		return
	}

	if l, ok := fbnLeader(c); ok {
		if c.WalkNearTarget(&l.Target, 5) {
			return
		}
	}

	c.Sleep(5)
}
