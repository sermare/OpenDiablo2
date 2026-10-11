package d2monster

// Shadow Master / Shadow Warrior skill scoring, the tail of
// MONAI_Think_ShadowMaster 0x5ea9f0 (VERIFIED from the decompile, helpers
// 0x5ea730 classify, 0x5ea850 other-in-state-group, 0x5ea6b0 valid target,
// 0x5ea920 cast chosen skill).
//
// monstats words (the exe reads the three per-difficulty words of aip1 and aip2
// at fixed offsets as separate parameters, monai.txt labels in brackets):
//
//	+0x56 aip1 normal  [approach distance]      range, squared against dist^2
//	+0x58 aip1 nightmare [melee bonus]          added to melee-kind scores
//	+0x5a aip1 hell    [progressive bonus]      kind 4/0xc/0xd bonus and a flag
//	+0x5c aip2 normal  [random pick]            bound of the score roll
//	+0x5e aip2 nightmare [ignore range]         target dropped beyond it (dist)
//	+0x60 aip2 hell    [boss leash]             owner pull-back, squared
//	+0x62+2*diff aip3  [attack chance]          base percent of the plain attack

// Shadow parameter slots, see the table above.
const (
	shadowApproach = iota
	shadowMeleeBonus
	shadowProgressive
	shadowRandomPick  = shadowApproach
	shadowIgnoreRange = shadowMeleeBonus
	shadowLeash       = shadowProgressive
)

// shadowWord returns word slot (0..2) of aip<n>. Profiles that only carry the
// per-difficulty AIP (tests, older callers) answer with AIP[n] for every slot.
func shadowWord(b *Brain, n, slot int) int {
	raw := b.Profile.AIPRaw[n]
	if raw == [3]int{} {
		return b.AIP(n)
	}

	return raw[slot]
}

// ShadowSkill is the skills.txt view of one skill the shadow owns (all columns
// as the exe reads them).
type ShadowSkill struct {
	ID              int
	AIType          int // skills.txt "aitype" (bAitype)
	AIBonus         int // "aibonus"
	ReqLevel        int // "reqlevel"
	EType           int // "etype": picks the resist the score is lowered by
	AuraState       int // "aurastate" (-1 none)
	AuraTargetState int // "auratargetstate"
	AuraStat1       int // "aurastat1" (stat id read from the state's stat list)
	SrvMissile      int // "srvmissile" id (-1 none)
	SrvMissileA     int // "srvmissilea" id (-1 none)
	MissileRange    int // range of srvmissilea, -1 when it has no record
	Bits4           int // skills.txt flag word 4 (bit 4 = progressive skill)
	DoFunc          int // "srvdofunc"
	Level           int // SKILL_GetTotalLevel
}

// ShadowScan is the result of MONAI_ShadowMaster_ClassifyTarget 0x5ea730 over
// the units around the shadow (hostile ones only; the exe walks the unit-mode
// list with callback 0x5ea730).
type ShadowScan struct {
	Nearest        *Target // [1] nearest hostile to the shadow (within dist^2 0x400)
	NearestDist2   int     // [2] its dist^2 (0x7fffffff when none)
	CloseCount     int     // [3] hostile units within dist^2 < 0x65 of the shadow
	NearOwner      *Target // [4] hostile nearest to the owner
	NearOwnerDist2 int     // [5]
	OwnerCount     int     // [6] hostile units within dist^2 < 0x65 of the owner
	WithinCount    int     // [7] hostile units within dist^2 <= 0x400 of the shadow
	SpecialCount   int     // [8] friendly units of the classes 0x19a-0x19d, 0x19f, 0x1a0
	ValidTarget    *Target // [9] last hostile within 0x400 that IsValidTargetUnit accepts
}

// ShadowEnv is the host side of the scoring: the skill records, the scan and
// the state / stat queries. All random numbers are drawn from the brain.
type ShadowEnv interface {
	ShadowSkills(b *Brain) []ShadowSkill
	ShadowScan(b *Brain, owner *Target) ShadowScan
	// TargetStat is STATS_GetUnitStat(target, stat) for the resist stats the
	// etype column selects (0x24, 0x27, 0x29, 0x25, 0x2b, 0x2d).
	TargetStat(t Target, stat int) int
	SelfHasState(b *Brain, state int) bool
	TargetHasState(t Target, state int) bool
	// SelfInCastMask is STATE_UnitHasStateInMaskDC(self) != 0 (mask not decoded).
	SelfInCastMask(b *Brain) bool
	// OtherInStateGroup is MONAI_ShadowMaster_ScanOtherEntries: the shadow has
	// another state of the same states.txt group as state.
	OtherInStateGroup(b *Brain, state int) bool
	// SelfStateStat is the value of stat in the shadow's own stat list of state
	// (STATS_FindStateStatList + STATS_GetListBaseStatAlt); ok false without a list.
	SelfStateStat(b *Brain, state, stat int) (v int, ok bool)
	// TargetMonsterThreat reports whether the target is a monster and the byte at
	// monstats +0xa0+difficulty of its record (meaning not decoded).
	TargetMonsterThreat(t Target, diff Difficulty) (isMonster bool, v int)
	// SelectActiveSkill is SKILL_SelectActiveSkillForUnit (aitype 6).
	SelectActiveSkill(b *Brain, skillID int)
	// LeftSkillSet is UNIT_GetLeftSkill(self) != nil.
	LeftSkillSet(b *Brain) bool
}

// ShadowCandidate is one entry of the exe's candidate stack.
type ShadowCandidate struct {
	Skill  int
	Score  int
	Target *Target // nil: no target (kind 7)
}

// shadowResistStat maps skills.txt etype to the resist stat that lowers the
// score (VERIFIED switch; etype 0 physical, 1 fire, 2 lightning, 3 magic, 4 and
// 0xc cold, 5 poison; others none).
func shadowResistStat(etype int) (stat int, ok bool) {
	switch etype {
	case 0:
		return 0x24, true
	case 1:
		return 0x27, true
	case 2:
		return 0x29, true
	case 3:
		return 0x25, true
	case 4, 0xc:
		return 0x2b, true
	case 5:
		return 0x2d, true
	}

	return 0, false
}

// shadowResistTerm is the exe's signed division of the resist by -10 written as
// a multiply-high: trunc(resist / -10).
func shadowResistTerm(resist int) int {
	x := int32((int64(resist) * -0x66666667) >> 32)

	return int(x>>2) - int(x>>31)
}

// ShadowScoreInput is everything the per-skill scoring needs besides the env.
type ShadowScoreInput struct {
	B        *Brain
	Env      ShadowEnv
	Target   Target
	Owner    *Target
	Scan     ShadowScan
	Dist2    int  // squared distance shadow-target
	InRange  bool // target within melee range
	Life     int  // shadow life percent
	Blocked  bool // no clear line (mask 4) between shadow and target
	CastMask bool // SelfInCastMask
	Progress *int // the running "progressive" accumulator (local_424)
}

// ShadowScore is the score of one skill and the target it would be cast at
// (kinds 1 and 8 pick the shadow itself, 7 none, 0xd possibly the unit nearest
// the owner). A zero score means "do not consider".
func ShadowScore(in *ShadowScoreInput, sk ShadowSkill) (score int, target *Target) {
	b, env := in.B, in.Env
	tgt := in.Target
	target = &tgt
	self := &Target{ID: b.ID, X: b.X, Y: b.Y, Size: b.Size}
	pick := uint32(shadowWord(b, 2, shadowRandomPick))
	roll := func() int {
		if pick == 0 {
			return 0
		}

		return int(b.Seed.Roll(int32(pick)))
	}

	approach := shadowWord(b, 1, shadowApproach)
	meleeBonus := shadowWord(b, 1, shadowMeleeBonus)
	progWord := shadowWord(b, 1, shadowProgressive)

	resist := 0
	if st, ok := shadowResistStat(sk.EType); ok {
		resist = env.TargetStat(tgt, st)
	}

	base := sk.AIBonus + sk.ReqLevel/4 + sk.Level + shadowResistTerm(resist)
	nearest := in.Scan.NearestDist2
	close3 := in.Scan.CloseCount
	within := in.Scan.WithinCount
	special := in.Scan.SpecialCount
	prog := in.Progress
	near := func(d int) bool { return d < 0x1a }

	missileOK := func() bool {
		// the exe's long condition: a missile with a record must reach the target
		// (range-1 squared), the other combinations always pass
		if sk.SrvMissile >= 0 || sk.SrvMissileA < 0 || sk.MissileRange < 0 {
			return true
		}

		r := sk.MissileRange - 1

		return in.Dist2 < r*r
	}

	rangedPenalties := func(s int) int {
		if near(nearest) {
			s -= 5
		}

		if near(in.Dist2) {
			s -= 5
		}

		if in.CastMask {
			s -= 5
		}

		return s
	}

	// melee-kind bonus shared by kinds 4 and 0xc
	meleeBase := func(s int) int {
		if approach*approach < in.Dist2 {
			s -= 10
		}

		s += meleeBonus
		if in.InRange || near(in.Dist2) {
			s += 10
		}

		return s
	}

	// progressive skills accumulate the stat of their own state; more than 2 of
	// it silences the skill (returns false)
	progressive := func() bool {
		if sk.AuraState > 0 && env.SelfHasState(b, sk.AuraState) {
			if v, ok := env.SelfStateStat(b, sk.AuraState, sk.AuraStat1); ok {
				*prog += v
				if v > 2 {
					return false
				}
			}
		}

		return true
	}

	switch sk.AIType {
	case 1: // buff on the shadow (the state test is read literally: see ai-skills-batch6.md)
		if sk.AuraState > 0 && !env.SelfHasState(b, sk.AuraState) {
			return 0, nil
		}

		if nearest < 0x1a {
			base -= 6
		}

		if env.OtherInStateGroup(b, sk.AuraState) {
			base -= 10
		} else {
			base += 10
		}

		return roll() + base, self
	case 2: // curse
		hasOwn := sk.AuraState > 0 && env.SelfHasState(b, sk.AuraState)
		hasTgt := sk.AuraTargetState > 0 && env.TargetHasState(tgt, sk.AuraTargetState)

		if hasOwn || hasTgt {
			return 0, nil
		}

		if nearest < 0x1a {
			base -= 10
		}

		return roll() + base, target
	case 3:
		if special > 5 {
			base -= special * 2
		}

		if nearest < 0x1a {
			base -= 7
		}

		if within < 3 {
			base -= 10
		}

		return within*3 - 9 + roll() + base, target
	case 4: // melee skill
		base = meleeBase(base)

		if sk.Bits4&4 == 0 {
			if progWord <= 0 || in.CastMask {
				base += 3 + *prog*4
			} else {
				base -= 10
			}
		} else {
			if !progressive() {
				return 0, nil
			}

			base += progWord
		}

		return roll() + base, target
	case 5: // missile
		if !in.Blocked && missileOK() {
			return roll() + rangedPenalties(base), target
		}

		return 0, nil
	case 6: // choose the active left skill (consumes a roll)
		lim := 6
		if !env.LeftSkillSet(b) {
			lim = 0x14
		}

		if b.Roll(100) < lim {
			env.SelectActiveSkill(b, sk.ID)
		}

		return 0, nil
	case 7: // reposition: wounded only; without an owner two position rolls are drawn
		r := roll()

		if in.Life >= 0x43 {
			return 0, nil
		}

		if in.Owner == nil {
			b.Roll(0x28)
			b.Roll(0x28)
		}

		s := r + base + 10
		if in.Life < 0x2d {
			s = r + base + 0x14
		}

		return s, nil
	case 8: // self heal-like
		r := roll()
		if in.Life >= 0x43 {
			return 0, nil
		}

		s := (r + base) * 2
		if in.Life < 0x2d {
			s = (r + base) * 4
		}

		return s, self
	case 0xb:
		if !in.Blocked && missileOK() {
			s := rangedPenalties(base)

			return within*3 + roll() + s, target
		}

		return 0, nil
	case 0xc:
		isMon, threat := env.TargetMonsterThreat(tgt, b.Diff)
		if isMon && threat <= 0x18 {
			return 0, nil
		}

		base = meleeBase(base)

		if sk.Bits4&4 != 0 && !progressive() {
			return 0, nil
		}

		s := roll() + base
		if in.Life < 0x4b {
			s += 8
		}

		if in.Life < 0x32 {
			s += 0xc
		}

		return s, target
	case 0xd:
		ext := *prog
		if progWord > 0 && !in.CastMask {
			ext = -5
		}

		base += meleeBonus + ext

		if (in.Life < 0x32 || close3 > 3) && in.Scan.NearOwner != nil && in.Scan.OwnerCount < 4 &&
			sqDist(b.X, b.Y, in.Scan.NearOwner.X, in.Scan.NearOwner.Y) > 0x19 {
			base += 0x14

			return roll() + base, in.Scan.NearOwner
		}

		if in.Dist2 < 0x19 {
			return 0, nil
		}

		if in.Dist2 > 0x144 {
			base += 10
		}

		return roll() + base, target
	}

	return 0, nil
}

// ShadowPick runs the candidate stack of the think function: every skill with a
// score above the current best is pushed (the stack starts with the plain
// attack, skill 0, score 0), so the last entry is the best. The skills are
// visited in the order the env lists them.
func ShadowPick(in *ShadowScoreInput, skills []ShadowSkill) []ShadowCandidate {
	stack := []ShadowCandidate{{Skill: 0, Score: 0, Target: &in.Target}}

	for _, sk := range skills {
		score, tgt := ShadowScore(in, sk)
		if score > stack[len(stack)-1].Score {
			stack = append(stack, ShadowCandidate{Skill: sk.ID, Score: score, Target: tgt})
		}
	}

	return stack
}

// ShadowTargeting is the optional target logic around the scoring: the owner's
// own living hostile target, the valid-target test (0x5ea6b0: a monster whose
// monstats flag word has 0x8000 and not 0x100, boss/flag exclusions) and the
// leader of a unit.
type ShadowTargeting interface {
	OwnerTarget(b *Brain, owner Target) (Target, bool)
	ValidTarget(b *Brain, t Target) bool
	LeaderOf(t Target) (Target, bool)
}

// shadowTail is the part of MONAI_Think_ShadowMaster after the plain-attack
// roll: target adoption when out of reach, the crowd retreat, the candidate
// stack and the cast loop. It always ends the tick (true).
func shadowTail(c *Ctx, env ShadowEnv, book FB1ShadowBook, owner, tgt, ownerTgt *Target) bool {
	b := c.B
	t := *tgt
	scan := env.ShadowScan(b, owner)

	if !c.InRange {
		var cand *Target

		switch {
		case ownerTgt != nil:
			cand = ownerTgt
		case scan.NearOwner != nil:
			cand = scan.NearOwner
		case scan.ValidTarget != nil && sqDist(b.X, b.Y, scan.ValidTarget.X, scan.ValidTarget.Y) < 0x400:
			cand = scan.ValidTarget
		}

		if cand != nil {
			t = *cand
		}

		if st, ok := c.W.(ShadowTargeting); ok && !st.ValidTarget(b, t) {
			if l, found := st.LeaderOf(t); found && sqDist(b.X, b.Y, l.X, l.Y) < 0x400 {
				t = l
			}
		}
	}

	// crowded: step back toward the owner, or run away from the target
	if scan.CloseCount > 3 && b.Roll(0x20) < scan.CloseCount*2 {
		if owner != nil && sqDist(b.X, b.Y, owner.X, owner.Y) > 0x24 {
			c.RunTo(*owner, 0)
			c.busy()

			return true
		}

		if c.RunAway(t, 8) {
			return true
		}
	}

	prog := 0
	castMask := shadowWord(b, 1, shadowProgressive) > 0 && env.SelfInCastMask(b)
	skills := env.ShadowSkills(b)
	in := &ShadowScoreInput{
		B: b, Env: env, Target: t, Owner: owner, Scan: scan,
		Dist2: sqDist(b.X, b.Y, t.X, t.Y), InRange: c.InRange, Life: b.HPPercent,
		Blocked: c.lineBlocked(t, 4), CastMask: castMask, Progress: &prog,
	}

	stack := ShadowPick(in, skills)
	doFunc := map[int]int{}

	for _, s := range skills {
		doFunc[s.ID] = s.DoFunc
	}

	for i := len(stack) - 1; i >= 0; i-- {
		if b.Seed.Step()&3 == 0 {
			continue
		}

		e := stack[i]
		et := Target{ID: b.ID, X: b.X, Y: b.Y, Size: b.Size}

		if e.Target != nil {
			et = *e.Target
		}

		if shadowChosenCast(c, book, owner, e.Skill, et, e.Target != nil) {
			if doFunc[e.Skill] == 0x13 {
				b.Scratch[1], b.Scratch[0] = e.Skill, 0x19
			}

			return true
		}
	}

	if shadowChosenCast(c, book, owner, 0, t, true) {
		return true
	}

	c.Sleep(0xf)

	return true
}

// shadowChosenCast is MONAI_ShadowMaster_CastChosenSkill 0x5ea920: a target
// that is the owner or the shadow itself is refused; otherwise the skill is
// queued (the plain attack is skill 0).
func shadowChosenCast(c *Ctx, book FB1ShadowBook, owner *Target, skill int, t Target, hasTarget bool) bool {
	b := c.B

	if hasTarget && (t.ID == b.ID || owner != nil && t.ID == owner.ID) {
		return false
	}

	inRange := c.InRange && c.Target != nil && t.ID == c.Target.ID

	if m, ok := c.W.(MeleeRanger); ok {
		inRange = m.InMeleeRange(b, t)
	}

	if !book.CastQueued(b, skill, t, inRange) {
		return false
	}

	c.busy()

	return true
}
