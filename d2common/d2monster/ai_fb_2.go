package d2monster

// Faithful ports (batch 2): ZakarumPriest, ZakarumZealot, OblivionKnight,
// Overseer, Regurgitator, VileMother, VileDog, ThornHulk, PinHead,
// PutridDefiler and QuillMother. Skill slot i below is Skill_(i+1) of monstats
// (exe offset 0x170 + 2*i, mode byte 0x180 + i). Scratch[0..2] are AiGeneral
// +0x14/+0x18/+0x1c. Every aip read follows the exe's offset table (0x56 aip1,
// 0x5c aip2, 0x62 aip3, 0x68 aip4, 0x6e aip5, 0x74 aip6, 0x7a aip7, 0x80 aip8).

func init() {
	fbxReg("VileDog", thinkVileDogB)
	fbxReg("PinHead", thinkPinHeadB)
	fbxReg("QuillMother", thinkQuillMotherB)
	fbxReg("ThornHulk", thinkThornHulkB)
	fbxReg("PutridDefiler", thinkPutridDefilerB)
	fbxReg("VileMother", thinkVileMotherB)
	fbxReg("Regurgitator", thinkRegurgitatorB)
	fbxReg("ZakarumZealot", thinkZealotB)
	fbxReg("ZakarumPriest", thinkPriestB)
	fbxReg("OblivionKnight", thinkOblivionKnightB)
	fbxReg("Overseer", thinkOverseerB)
}

func fbxDistSq(ax, ay, bx, by int) int {
	dx, dy := ax-bx, ay-by

	return dx*dx + dy*dy
}

// FBXUnitFinder resolves a unit id to a target and its mode (server unit
// lookup by id and type). Fallback: nothing is found.
type FBXUnitFinder interface {
	FBXUnit(b *Brain, id uint32) (Target, Mode, bool)
}

func fbxUnit(c *Ctx, id uint32) (Target, Mode, bool) {
	if f, ok := c.W.(FBXUnitFinder); ok {
		return f.FBXUnit(c.B, id)
	}

	return Target{}, 0, false
}

// FBXCanCast is MONAI_CanCastSkillAtTarget for a ground point. Fallback true.
type FBXCanCast interface {
	FBXCanCastAt(b *Brain, slot int, p Point) bool
}

// FBXSkillGate stands for MONAI_MonsterHasSkillId (the unit holds the skill id
// in one of its eight slots) evaluated for who. The skill id it tests is hidden
// in the decompilation (UNVERIFIED), so the host answers for the slot that is
// about to be cast. Fallback true.
type FBXSkillGate interface {
	FBXHasSkill(b *Brain, who Target, slot int) bool
}

func fbxHasSkill(c *Ctx, who Target, slot int) bool {
	if g, ok := c.W.(FBXSkillGate); ok {
		return g.FBXHasSkill(c.B, who, slot)
	}

	return true
}

// thinkVileDogB is MONAI_Think_VileDog 0x5f93f0 (VERIFIED). aip1 bite%,
// aip2 stall, aip3 approach%.
func thinkVileDogB(c *Ctx) {
	b := c.B

	if b.Scratch[0] == 0 {
		b.Scratch[0] = 1
		c.Sleep(5)

		return
	}

	t := fbxTargetOrSelf(c)

	if c.InRange {
		if b.Roll(100) < b.AIP(1) {
			c.Attack(ModeAttack1, t)

			return
		}

		c.Sleep(b.AIP(2))

		return
	}

	if b.Roll(100) < b.AIP(3) {
		fbxWalk(c, t, 7)

		return
	}

	c.Sleep(10)
}

// thinkPinHeadB is MONAI_Think_PinHead 0x5f5450 (VERIFIED). aip1 repeat gate,
// aip2 stall, aip3 approach gate, aip4 stall, aip5/aip6 skill1/skill2 %.
func thinkPinHeadB(c *Ctx) {
	b := c.B
	t := fbxTargetOrSelf(c)

	if !c.InRange {
		b.Scratch[0] = 0

		if b.AIP(3) <= b.Roll(100) {
			c.Sleep(b.AIP(4))

			return
		}

		fbxWalk(c, t, 7)

		return
	}

	if b.Scratch[0] != 0 && b.AIP(1) <= b.Roll(100) {
		c.Sleep(b.AIP(2))

		return
	}

	b.Scratch[0] = 1

	if fbxSkill(b, 0) && b.Roll(100) < b.AIP(5) {
		c.Cast(0, t)

		return
	}

	if fbxSkill(b, 1) && b.Roll(100) < b.AIP(6) {
		c.Cast(1, t)

		return
	}

	c.Attack(ModeAttack1, t)
}

// thinkQuillMotherB is MONAI_Think_QuillMother 0x5fa420 (VERIFIED). An engaged
// (aggressive) mother raises the group alarm every think; otherwise aip1/aip2
// gate the melee/approach rolls and aip3/aip4 are the stalls.
func thinkQuillMotherB(c *Ctx) {
	b := c.B
	t := fbxTargetOrSelf(c)

	if b.Aggressive {
		b.Broadcast(Command{Type: CmdAlert, Count: 1, Target: t.ID})

		if !c.InRange {
			fbxWalk(c, t, 7)

			return
		}

		c.Attack(ModeAttack1, t)

		return
	}

	if !c.InRange {
		if b.AIP(2) <= b.Roll(100) {
			c.Sleep(b.AIP(4))

			return
		}

		fbxWalk(c, t, 7)

		return
	}

	if b.AIP(1) <= b.Roll(100) {
		c.Sleep(b.AIP(3))

		return
	}

	c.Attack(ModeAttack1, t)
}

// FBXEventClock is EVENT_GetSoonestFrameForType(1): the frame of the unit's
// next queued attack-end event (0 when none). Fallback 0.
type FBXEventClock interface {
	FBXSoonestEvent(b *Brain) int
}

// thinkThornHulkB is MONAI_Think_ThornHulk 0x5f38f0 (VERIFIED). aip1 stand-off
// gate, aip2 A1/A2 split, aip3 stall gate, aip4 spike volley %, aip5 volley
// pacing, aip6 volley length. Scratch[0] volley shots left, Scratch[1]
// cool-down counter. The volley is cast with mode 5 (A2) as a literal.
func thinkThornHulkB(c *Ctx) {
	b := c.B
	now := c.W.Frame()

	pace := func(t Target) {
		c.Cast(0, t)

		if e, ok := c.W.(FBXEventClock); ok {
			if soon := e.FBXSoonestEvent(b); now < soon {
				c.Sleep(b.AIP(5) - now + soon)
			}
		}
	}

	if !c.InRange || c.Target == nil {
		b.Scratch[0] = 0
		c.SetSpeed(0)

		if c.Target != nil {
			fbxWalk(c, *c.Target, 7)
		} else {
			c.Sleep(10)
		}

		return
	}

	t := *c.Target

	if b.Scratch[0] < 1 {
		if b.AIP(1) <= b.Roll(100) {
			if b.AIP(3) <= b.Roll(100) {
				c.Sleep(15)

				return
			}

			c.Circle(t, 4)

			return
		}

		if b.Scratch[1] < 1 && b.Roll(100) < b.AIP(4) {
			pace(t)

			b.Scratch[0] = b.AIP(6)

			return
		}

		b.Scratch[1]--

		if b.AIP(2) <= b.Roll(100) {
			c.Attack(ModeAttack1, t)

			return
		}

		c.Attack(ModeAttack2, t)

		return
	}

	pace(t)

	b.Scratch[0]--

	if b.Scratch[0] < 1 {
		b.Scratch[1] = 3
	}
}

// thinkPutridDefilerB is MONAI_Think_PutridDefiler 0x5eebc0 (VERIFIED flow;
// scan filter UNVERIFIED). aip1 flee gate, aip2 flee length. It bites what is
// in range, otherwise goes for the unit its scan finds and casts skill id 300
// (mode S1) once adjacent.
func thinkPutridDefilerB(c *Ctx) {
	b := c.B

	if c.Target != nil && c.InRange {
		c.Attack(ModeAttack1, *c.Target)

		return
	}

	r := fbxScan(c, FBXScanQuery{Kind: FBXScanDefilerCorpse, Radius2: 25})
	if !r.Found {
		if c.Dist < b.AIP(1) && c.Target != nil {
			if !c.WalkAway(*c.Target, b.AIP(2)) {
				c.Sleep(10)
			}

			return
		}

		c.Sleep(25)

		return
	}

	d := EdgeDistance(b.X-r.T.X, b.Y-r.T.Y, b.Size)
	if c.W.InRange(b, r.T, d) {
		if sc, ok := c.W.(SkillCaster); ok && sc.CastSkillAt(b, 300, ModeSkill1, Point{r.T.X, r.T.Y}) {
			c.busy()

			return
		}

		c.Attack(ModeSkill1, r.T)

		return
	}

	fbxWalk(c, r.T, 0)
}

// FBXBrood is the VileMother's spawn chain: the class id of the young linked
// to her (MONSTER_GetNextLinkedClassAtDepth) and a free-cell test for a point.
// Fallbacks: class -1 and free cells.
type FBXBrood interface {
	FBXBroodClass(b *Brain) int
	FBXCellFreeAt(b *Brain, p Point) bool
}

// fbxRing is the eight compass steps the VileMother tries, in the order of
// rotating direction (UNVERIFIED mapping of the exe's direction table).
var fbxRing = [8]Point{{1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1}}

// thinkVileMotherB is MONAI_Think_VileMother 0x5f9190 (VERIFIED flow). aip1
// brood limit, aip2 young alive limit, aip3 spawn %, aip4 melee %, aip5 hover
// gate, aip6 hover %, aip8 stall. Scratch[0] young spawned.
func thinkVileMotherB(c *Ctx) {
	b := c.B
	t := fbxTargetOrSelf(c)

	if b.Scratch[0] < b.AIP(1) && b.Roll(100) < b.AIP(3) {
		class := -1
		if f, ok := c.W.(FBXBrood); ok {
			class = f.FBXBroodClass(b)
		}

		alive := fbxScan(c, FBXScanQuery{Kind: FBXScanLinkedClass, Class: class, Radius2: 25 * 25}).Count
		if alive < b.AIP(2) && fbxVileSpawn(c, t) {
			b.Scratch[0]++

			return
		}
	}

	if c.InRange {
		if b.Roll(100) < b.AIP(4) {
			c.Attack(ModeAttack1, t)

			return
		}

		c.Sleep(b.AIP(8))

		return
	}

	if b.Scratch[0] < b.AIP(1) && b.AIP(5) <= b.Roll(100) {
		if b.Roll(100) < b.AIP(6) {
			c.Circle(t, 4)

			return
		}

		c.Sleep(15)

		return
	}

	fbxWalk(c, t, 7)
}

// fbxVileSpawn is MONAI_VileMotherSpawnSkillInDirection: starting from the
// direction of the target it tries the eight directions, three subtiles out,
// and casts Skill1 on the first free cell.
func fbxVileSpawn(c *Ctx, t Target) bool {
	b := c.B
	if !fbxSkill(b, 0) {
		return false
	}

	start := 0
	best := -1 << 30

	for i, d := range fbxRing {
		if s := d.X*sign(t.X-b.X) + d.Y*sign(t.Y-b.Y); s > best {
			best, start = s, i
		}
	}

	for k := 0; k < 8; k++ {
		d := fbxRing[(start+k)%8]
		p := Point{b.X + d.X*3, b.Y + d.Y*3}

		free := true
		if f, ok := c.W.(FBXBrood); ok {
			free = f.FBXCellFreeAt(b, p)
		}

		if free {
			return c.Cast(0, Target{X: p.X, Y: p.Y})
		}
	}

	return false
}

// thinkRegurgitatorB is MONAI_Think_Regurgitator 0x5f98a0 (VERIFIED flow).
// States in Scratch[0]: 0/1 hunting, 2 walking to the corpse, 3 at the corpse,
// 4 spitting, 5 backing off; Scratch[1] the corpse id, Scratch[2] step count.
// aip1 melee %, aip2 near-corpse %, aip3 approach %, aip4 far-corpse %,
// aip5 any-corpse %, aip6 corpse radius.
func thinkRegurgitatorB(c *Ctx) {
	b := c.B
	t := fbxTargetOrSelf(c)
	reset := func() { b.Scratch = [3]int{} }

	switch b.Scratch[0] {
	case 5:
		if !c.WalkAway(t, 0x10) {
			c.Sleep(10)
		}

		b.Scratch[0] = 0

		return
	case 2:
		u, m, ok := fbxUnit(c, uint32(b.Scratch[1]))
		if !ok || m != ModeDead {
			reset()
			c.Wander(8)

			return
		}

		if fbxDistSq(b.X, b.Y, u.X, u.Y) > 4 {
			if b.Scratch[2] > 5 {
				reset()
				c.Wander(8)

				return
			}

			fbxWalk(c, u, 1)

			b.Scratch[2]++
		}

		b.Scratch[0] = 3
		c.Sleep(8)

		return
	case 3:
		u, m, ok := fbxUnit(c, uint32(b.Scratch[1]))
		if fbxSkill(b, 0) && ok && m == ModeDead {
			c.Cast(0, u)

			b.Scratch[0] = 4

			return
		}

		reset()
		c.Wander(8)

		return
	case 4:
		if !c.InRange {
			c.Attack(ModeAttack2, t)

			b.Scratch[0] = 5

			return
		}

		if !c.WalkAway(t, 8) {
			c.Sleep(10)
		}

		return
	}

	scan := fbxScan(c, FBXScanQuery{Kind: FBXScanRegurgitate, Radius2: b.AIP(6) * b.AIP(6)})
	toCorpse := func() {
		fbxWalk(c, scan.T, 1)

		b.Scratch[0], b.Scratch[1], b.Scratch[2] = 2, int(scan.T.ID), 0
	}

	if b.Scratch[0] == 1 {
		if scan.Found {
			b.Scratch[1] = int(scan.T.ID)

			if fbxDistSq(b.X, b.Y, scan.T.X, scan.T.Y) > 2 {
				fbxWalk(c, scan.T, 1)

				b.Scratch[0], b.Scratch[2] = 2, 0

				return
			}

			c.Sleep(8)

			b.Scratch[0], b.Scratch[2] = 3, 0

			return
		}

		if b.Roll(100) < 20 {
			reset()

			return
		}

		c.Wander(8)

		return
	}

	if scan.Found {
		if fbxDistSq(b.X, b.Y, scan.T.X, scan.T.Y) < 9 && b.Roll(100) < b.AIP(2) {
			toCorpse()

			return
		}

		if b.Roll(100) < b.AIP(5) {
			toCorpse()

			return
		}
	}

	r := b.Roll(100)

	if c.InRange {
		if r < b.AIP(1) {
			c.Attack(ModeAttack1, t)

			return
		}

		c.Sleep(15)

		return
	}

	if r < b.AIP(3) {
		fbxWalk(c, t, 0)

		return
	}

	if scan.Found && b.Roll(100) < b.AIP(4) {
		toCorpse()

		return
	}

	c.Sleep(8)
}

// FBXAfraid is the Zakarum Zealot's Act 3 fear: in act 3 (level act 2) it runs
// from a player whose quest record has flag 0x15 for the current difficulty
// (the target, or the owner of a minion target). Fallback false.
type FBXAfraid interface {
	FBXAfraidOf(b *Brain, t Target) bool
}

// thinkZealotB is MONAI_Think_ZakarumZealot 0x5f5ef0 (VERIFIED flow). aip1
// repeat gate, aip2 A1/A2 split, aip3 flee life %, aip4 walk-vs-run gate.
// Scratch[0] attacked last think, Scratch[1] flee cool-down.
func thinkZealotB(c *Ctx) {
	b := c.B
	boost := fbxRunSpeed(b.Profile, true)
	t := fbxTargetOrSelf(c)

	if c.Target != nil {
		if a, ok := c.W.(FBXAfraid); ok && a.FBXAfraidOf(b, *c.Target) {
			c.SetSpeed(boost)

			if c.RunAway(t, 8) {
				return
			}

			if !c.Wander(6) {
				c.Sleep(10)
			}

			return
		}
	}

	if b.Aggressive {
		if b.Scratch[1] == 0 && b.HPPercent < b.AIP(3) {
			b.Scratch[0] = 0
			c.SetSpeed(boost)

			if c.RunAway(t, 8) {
				return
			}

			b.Scratch[1] = 5
		}

		if tf, ok := c.W.(TileFlagger); ok && tf.OwnTileFlagged(b) {
			if !c.InRange {
				fbxWalk(c, t, 7)
			} else {
				c.Circle(t, 4)
			}

			return
		}
	}

	if b.Scratch[1] != 0 {
		b.Scratch[1]--
	}

	if !c.InRange {
		b.Scratch[0] = 0

		if b.AIP(4) <= b.Roll(100) {
			fbxWalk(c, t, 7)

			return
		}

		c.SetSpeed(boost)

		if !c.RunTo(t, 0) {
			c.Sleep(10)
		}

		return
	}

	if b.Scratch[0] != 0 && b.AIP(1) <= b.Roll(100) {
		b.Scratch[0] = 0

		if b.Roll(100) > 0x4f {
			c.Circle(t, 4)

			return
		}

		c.Sleep(10)

		return
	}

	b.Scratch[0] = 1

	if b.AIP(2) <= b.Roll(100) {
		c.Attack(ModeAttack1, t)

		return
	}

	c.Attack(ModeAttack2, t)
}

// thinkPriestB is MONAI_Think_ZakarumPriest 0x5f6350 (VERIFIED flow; the
// blocked-line test and the scan filter come from FBXLineOfSight and
// FBXScanner). Skills: 1 heal ally, 2 target cast B, 3 teleport away, 4 target
// cast A. aip1 melee/cast gate, aip2 cast %, aip3 cast B %, aip4 blocked-line
// gate, aip5 cooldown, aip6 ally radius. Scratch[0] teleport ready frame,
// Scratch[1] cast A ready frame, Scratch[2] cast B ready frame.
func thinkPriestB(c *Ctx) {
	b, now := c.B, c.W.Frame()
	t := fbxTargetOrSelf(c)

	if c.InRange || b.Aggressive {
		if fbxSkill(b, 2) && b.HPPercent < 0x21 && b.Scratch[0] < now {
			b.Scratch[0] = now + b.AIP(5)*4

			shift := uint(0)
			if c.InRange {
				shift = 2
			}

			dx, dy := b.X-t.X, b.Y-t.Y
			p := Point{b.X + dx<<shift, b.Y + dy<<shift}

			ok := true
			if cc, has := c.W.(FBXCanCast); has {
				ok = cc.FBXCanCastAt(b, 2, p)
			}

			if ok {
				c.Cast(2, Target{X: p.X, Y: p.Y})

				return
			}
		}

		if c.InRange && b.Roll(100) < b.AIP(1) {
			c.Attack(ModeAttack1, t)

			return
		}
	}

	ally := fbxScan(c, FBXScanQuery{Kind: FBXScanPriestAlly, Radius2: b.AIP(6) * b.AIP(6)})
	if fbxSkill(b, 0) && ally.Found && b.Roll(100) < 0x19 {
		c.Cast(0, ally.T)

		return
	}

	castA := func() {
		c.Cast(3, t)

		b.Scratch[1] = b.AIP(5) + now
	}

	if !fbxBlocked(c, t, 4) {
		if b.Roll(100) < b.AIP(1) && fbxSkill(b, 3) && b.Scratch[1] < now && b.Roll(100) < b.AIP(2) {
			castA()

			return
		}
	} else if b.Roll(100) < b.AIP(4) {
		if fbxSkill(b, 3) && b.Scratch[1] < now && b.Roll(100) < b.AIP(2) {
			castA()

			return
		}

		if fbxSkill(b, 1) && b.Scratch[2] < now && b.Roll(100) < b.AIP(3) {
			c.Cast(1, t)

			b.Scratch[2] = now + 0x14

			return
		}
	}

	if b.Roll(100) < 0x1e {
		c.Circle(t, 4)

		return
	}

	c.Sleep(0x14)
}

// FBXKnightHost carries the OblivionKnight's engine questions.
type FBXKnightHost interface {
	// FBXSkillState is skills.txt record +0x82 of the slot (the state the
	// skill puts on a target, 0 = none).
	FBXSkillState(b *Brain, slot int) int
	// FBXTargetHasState reports a state on another unit.
	FBXTargetHasState(b *Brain, t Target, state int) bool
	// FBXSummonsBelow4 is the unit-data byte test before Skill1 (fewer than
	// four of something; UNVERIFIED). Fallback true.
	FBXSummonsBelow4(b *Brain) bool
}

// thinkOblivionKnightB is MONAI_Think_OblivionKnight 0x5fa060 (VERIFIED flow,
// filter of the ally scan UNVERIFIED). aip1 close range, aip2 attack range,
// aip3 cool-down, aip4 skill7 %, aip5 cast gate, aip6 skill3 %, aip7 reposition
// %, aip8 reposition distance. Scratch[0] next skill7/curse frame.
func thinkOblivionKnightB(c *Ctx) {
	b, now := c.B, c.W.Frame()
	t := fbxTargetOrSelf(c)
	d1 := b.AIP(1)
	host, _ := c.W.(FBXKnightHost)

	ally := fbxScan(c, FBXScanQuery{Kind: FBXScanKnightAlly})

	castCurse := func(slot int, tt Target) {
		c.Cast(slot, tt)

		b.Scratch[0] = b.AIP(3) + now
	}

	if c.Dist < d1 {
		if fbxSkill(b, 3) && host != nil {
			if st := host.FBXSkillState(b, 3); st > 0 && !host.FBXTargetHasState(b, t, st) {
				castCurse(3, t)

				return
			}
		}

		if ally.Found && fbxDistSq(b.X, b.Y, ally.T.X, ally.T.Y) > d1*d1 && c.WalkTo(ally.T, 4) {
			return
		}

		c.SetSpeed(50)

		if c.WalkAway(t, 10) {
			return
		}

		if fbxSkill(b, 2) {
			c.Cast(2, t)

			return
		}
	}

	at, d, ok := c.W.AttackTarget(b)
	if ok && d < b.AIP(2) {
		if fbxSkill(b, 6) && b.Scratch[0] < now && b.Roll(100) < b.AIP(4) {
			castCurse(6, t)

			return
		}

		if b.Roll(100) < b.AIP(5) {
			if fbxSkill(b, 2) && b.Roll(100) < b.AIP(6) {
				c.Cast(2, at)

				return
			}

			if fbxSkill(b, 0) && (host == nil || host.FBXSummonsBelow4(b)) {
				c.Cast(0, at)

				return
			}
		}
	}

	if b.AIP(8) < c.Dist && b.Roll(100) < b.AIP(7) {
		c.WalkNearTarget(&t, 6)

		return
	}

	if b.Roll(100) > 0x45 {
		c.Sleep(10)

		return
	}

	c.Circle(t, 3)
}

// FBXOverseerHost carries the Overseer's questions: the linked target (the unit
// it is told to attack, valid when it is a live player or a hostile monster).
type FBXOverseerHost interface {
	FBXLinkedTarget(b *Brain) (Target, bool)
}

// thinkOverseerB is MONAI_Think_Overseer 0x5e1640 (VERIFIED flow; the scan
// callback 0x5e1570 is UNVERIFIED: ally in T, enemy in T2, Count of
// candidates). aip1 skill1 cool-down, aip2 skill2 %, aip3 skill3 %, aip4/aip5
// preferred distance and slack, aip6 melee %, aip7 A1/A2 split. Scratch[0] is
// the skill1 ready frame.
func thinkOverseerB(c *Ctx) {
	b, now := c.B, c.W.Frame()

	// every think, before anything else: QUEST_A5_SiegeOnHarrogath_OnOverseerThink
	// 0x5857e0 (hook only; the quest gate is boss number 0x2a)
	c.questHook("overseer")
	t := fbxTargetOrSelf(c)

	var linked Target

	hasLinked := false
	if h, ok := c.W.(FBXOverseerHost); ok {
		linked, hasLinked = h.FBXLinkedTarget(b)
	}

	if fbxSkill(b, 0) && b.Aggressive && b.Scratch[0] < now && hasLinked && fbxHasSkill(c, linked, 0) {
		c.Cast(0, linked)

		b.Scratch[0] = b.AIP(1) + now

		return
	}

	if c.InRange && b.Roll(100) < b.AIP(6) {
		if b.AIP(7) <= b.Roll(100) {
			c.Attack(ModeAttack1, t)
		} else {
			c.Attack(ModeAttack2, t)
		}

		return
	}

	r := fbxScan(c, FBXScanQuery{Kind: FBXScanOverseer, Radius2: 50})

	if fbxSkill(b, 1) && r.Found && b.Roll(100) < b.AIP(2) && fbxHasSkill(c, r.T, 1) {
		c.Cast(1, r.T)

		return
	}

	if fbxSkill(b, 2) && r.Found2 && b.Roll(100) < b.AIP(3) && fbxHasSkill(c, r.T2, 2) {
		c.Cast(2, r.T2)

		return
	}

	if r.Count == 0 {
		if b.Roll(100) < 0x3c {
			fbxWalk(c, t, 0)

			return
		}
	} else {
		lo := b.AIP(4) - b.AIP(5)
		if c.Dist < lo {
			if !c.WalkAway(t, b.AIP(4)-c.Dist) {
				c.Sleep(10)
			}

			return
		}

		if b.AIP(5)+b.AIP(4) < c.Dist {
			fbxWalk(c, t, 0)

			return
		}

		if b.Roll(100) < 0x32 {
			c.Circle(t, 5)

			return
		}
	}

	c.Sleep(10)
}
