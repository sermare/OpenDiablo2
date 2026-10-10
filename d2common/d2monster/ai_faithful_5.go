package d2monster

// Batch 5 of faithful ports (notes: d2-re-notes/ai-faithful-batch5.md).
//
// The audit of batch 4 found no entry left on a generic stand-in; what remains
// are ports that were written with documented UNVERIFIED simplifications.
// This batch re-reads those handlers in Game.exe 1.14b and closes the gaps:
//
//	Diablo      0x5e8150 (+ decision 0x5e7710, target search 0x5e7ec0, score 0x5e7420)
//	UberDiablo  0x5e8ea0, UberMephisto 0x5f72f0: bare returns in the exe
//	UberIzual   0x5f7df0
//	Vulture     0x5f2150
//	BloodRaven  0x5e5260
//	Nihlathak   0x5ed6d0 (fix in ai_fb_1.go, the host-facing extras live here)
//	Izual 0x5f7b30, Summoner 0x5f7720, Duriel 0x5f5880 (hooks; flow re-read)
//
// Conventions as in batch 3/4: Scratch[0..2] are AiGeneral +0x14/+0x18/+0x1c,
// aipN at monstats +0x56+6*(N-1) plus 2*difficulty. Everything the engine
// owns is an optional host interface with a documented fallback.

// ---------------------------------------------------------------- host hooks

// LineChecker is COLLISION_CanTraceLineBetweenUnits(unit, target, mask) 0x622c80:
// true when it returns nonzero. The helper MONAI_IsNotImmune 0x5dbfd0 (a
// misleading Ghidra name) is "mask 4 result == 0"; UberIzual asks with mask 6.
// Polarity (nonzero = the line is blocked) is inferred from the uses
// (UNVERIFIED). Absent means never blocked.
type LineChecker interface {
	LineBlocked(b *Brain, t Target, mask int) bool
}

// AuraStateSource answers skills.txt aurastate of a monstats slot (-1 for
// none), for the "keep the buff up" casts of Diablo's Skill8 and UberIzual's
// Skill2. Absent means none, so those casts never happen.
type AuraStateSource interface {
	SlotAuraState(b *Brain, slot int) int
}

// QuestBossHook is the quest callback some boss handlers make:
// QUEST_A4_TheFallenAngel_IzualThinkHook ("izual", first think),
// QUEST_A2_TheSummoner_OnSummonerThink ("summoner", first think) and
// QUEST_A5_BetrayalOfHarrogath_RefreshLogOnNihlathakThink ("nihlathak", every
// think). Absent means no quest bookkeeping.
type QuestBossHook interface {
	BossThink(b *Brain, boss string)
}

// RightSkillChecker is UNIT_GetRightSkill(unit) != 0, which Duriel uses to
// decide whether his Holy Freeze aura has been made the active skill yet.
type RightSkillChecker interface {
	HasRightSkill(b *Brain) bool
}

// MeleeRanger is COMBAT_IsTargetWithinMeleeRange(unit, target, 0) 0x622e40 for
// a target other than the tick's. Absent: the tick's InRange flag for the tick
// target, false for anything else.
type MeleeRanger interface {
	InMeleeRange(b *Brain, t Target) bool
}

// RoomMatcher is "UNIT_GetRoom(a) == UNIT_GetRoom(b)" for the Vulture's
// take-off rule. Absent means the same room.
type RoomMatcher interface {
	SameRoom(b *Brain, t Target) bool
}

// WaypointCellFinder is COLLISION_FindNearestFreeCellDefault(room, point, size,
// 0xffff, 1) in the room of the target. Absent: the point is kept.
type WaypointCellFinder interface {
	NearestFreeCell(b *Brain, p Point) (Point, bool)
}

// DiaStats is what MONAI_Diablo_ScoreTargetResists 0x5e7420 reads about a
// candidate. The stat ids are the exe's (STATS_GetUnitStat); their names are
// the usual stat table ones and UNVERIFIED for the damage group.
type DiaStats struct {
	S24, S25, S27, S29, S2B int // 0x24/0x25 resists, 0x27 fire, 0x29 lightning, 0x2b cold
	S16                     int // 0x16 max damage
	S31, S33, S35, S37, S3A int // 0x31 fire, 0x33 lightning, 0x35 magic, 0x37 cold, 0x3a poison (>>8) max damage
	LifePct                 int
	HasState0xb             bool
	LeftRank, LeftLevel     int // skills.txt attackrank and total level of the left skill
	RightRank, RightLevel   int
	LeftID, RightID         int // skill ids (0 when none)
	IsMonster               bool
	Threat                  int // monstats threat when IsMonster
}

// DiaCandidate is one unit Diablo may pick as his target.
type DiaCandidate struct {
	Target Target
	Stats  DiaStats
	// Dead is dwMode 0 (death) or 0x11 (dead): the candidate scores 0 but is
	// still counted.
	Dead bool
}

// DiabloTargetSource lists the candidates in the order MONAI_Diablo_FindBestTarget
// walks them (players first), already filtered by MONAI_Diablo_IsTargetNear
// (same area as his anchor, octagonal distance to it below 0x3fc). Absent: the
// tick's target is the only candidate, scored from ResistFinder.
type DiabloTargetSource interface {
	DiabloCandidates(b *Brain) []DiaCandidate
}

// DiabloPrisonTester is MONAI_CanCastSkillAtTarget(game, unit, 199, t): the raw
// result for Diablo's prison skill (id 199), which spawns the group in test
// mode. Absent: zero (the prison stays available).
type DiabloPrisonTester interface {
	PrisonTest(b *Brain, t Target) int
}

// DiabloPortalSource is PLAYER_GetLastPortalUnitId + SERVER_FindUnitByIdAndType:
// the target's last town-portal object, its level id and its position.
type DiabloPortalSource interface {
	LastPortal(b *Brain, t Target) (id uint32, p Point, levelID int, ok bool)
}

// PlayerScaler is the second output of MONSTER_CalcPlayerCountScaling 0x571760
// (read by Diablo's decision as "local_8 < 2"; the meaning, probably the player
// count, is UNVERIFIED). Absent: 1.
type PlayerScaler interface {
	PlayerScaling(b *Brain) int
}

// FootprintChecker is COLLISION_TestFootprintMask(room, x, y, 2, 0x40) at the
// monster's own tile; true when blocked. Absent: free.
type FootprintChecker interface {
	FootprintBlocked(b *Brain) bool
}

func init() {
	register("Diablo", TargetStandard, thinkDiabloF5)
	register("UberDiablo", TargetStandard, thinkBareReturn)
	register("UberMephisto", TargetStandard, thinkBareReturn)
	register("UberIzual", TargetStandard, thinkUberIzual)
	register("Vulture", TargetStandard, thinkVultureF5)
	register("BloodRaven", TargetStandard, thinkBloodRavenF5)
}

// FaithfulBatch5 lists the names this batch counts as newly ported: handlers
// whose earlier Go version had documented gaps against the exe.
func FaithfulBatch5() []string {
	return []string{"Diablo", "UberDiablo", "UberMephisto", "UberIzual", "Vulture", "BloodRaven", "Nihlathak"}
}

// ReverifiedBatch5 lists handlers that were re-read against the exe in this
// batch and only needed engine hooks (quest callbacks, the aura test): they
// were already faithful in flow and are not counted again.
func ReverifiedBatch5() []string {
	return []string{"Izual", "Summoner", "Duriel", "Andariel", "Smith"}
}

// thinkBareReturn is the think function of UberDiablo (0x5e8ea0) and
// UberMephisto (0x5f72f0): a 3-byte RET. Nothing is queued, so the monster is
// never scheduled again by its own AI (like UberBaal in ai_fb_3.go).
func thinkBareReturn(c *Ctx) { fbxStop(c) }

func (c *Ctx) lineBlocked(t Target, mask int) bool {
	if l, ok := c.W.(LineChecker); ok {
		return l.LineBlocked(c.B, t, mask)
	}

	return false
}

func (c *Ctx) auraState(slot int) int {
	if a, ok := c.W.(AuraStateSource); ok {
		return a.SlotAuraState(c.B, slot)
	}

	return -1
}

func (c *Ctx) questHook(boss string) {
	if q, ok := c.W.(QuestBossHook); ok {
		q.BossThink(c.B, boss)
	}
}

// sidestep is MONAI_RandomizedSidestepCommand 0x5de480 (VERIFIED, decompile):
// the Wander offset scheme (a step whose low bit picks which axis gets the full
// n, a bounded roll for the other, then one step for each axis' sign) applied
// around the target's position, queued as a run (mode 0xf) or, with state
// 0x3c, as a walk.
func (c *Ctx) sidestep(t Target, n int) bool {
	b := c.B
	s1 := b.Seed.Step()
	r := int(b.Seed.Roll(int32(n)))
	sx := b.Seed.Step()
	sy := b.Seed.Step()

	dx, dy := r, n
	if s1&1 != 0 {
		dx, dy = n, r
	}

	if sx&1 != 0 {
		dx = -dx
	}

	if sy&1 != 0 {
		dy = -dy
	}

	return c.move(Point{t.X + dx, t.Y + dy}, nil, 0, !c.W.HasState(b, 0x3c))
}

// runToPoint is MONAI_RunToPoint 0x5ddbc0 (run, or walk with state 0x3c).
func (c *Ctx) runToPoint(p Point) bool {
	return c.move(p, nil, 0, !c.W.HasState(c.B, 0x3c))
}

// walkToPointOrClear is MONAI_WalkToPointOrClearEvent 0x5ddc30: a walk to the
// point; the cancellation of pending events on failure has no Go counterpart.
func (c *Ctx) walkToPointOrClear(p Point) bool { return c.move(p, nil, 0, false) }

// ---------------------------------------------------------------- Diablo

// Action indexes of the decision table (index 0 and 11 both wait).
// DiaSituation is everything MONAI_DiabloChooseAction reads.
type DiaSituation struct {
	HasTarget   bool
	Melee       bool // COMBAT_IsTargetWithinMeleeRange(monster, target)
	LineClear   bool // MONAI_IsNotImmune: no blocked line to the target
	Stats       DiaStats
	EdgeDist    int  // monster to target edge distance
	TargetAway  bool // target edge distance to the anchor > 0x55
	TargetFar   bool // ... > 0x69
	GroundUser  bool // the target's left/right skill is Meteor, Blizzard, Fire Wall > 3 or Immolation Arrow > 7
	PortalNear  bool // the target's portal lies in level 0x6c within 0x55 of the anchor
	Count       int  // candidates counted by the target search
	Score       int  // score of the chosen target
	Scaling     int  // PlayerScaler
	PrisonTest  int  // CanCast(199, target) raw result
	PortalTest  int  // CanCast(199, portal) raw result (only read when PortalNear)
	HasPortalOb bool
}

// groundSkillUser is the exe's test on the target's left and right skills
// (VERIFIED immediates 0x3b, 0x38, 0x33 with total level > 3, 0x1b with total
// level > 7). Which skills those are (Blizzard, Meteor, Fire Wall,
// Immolation Arrow) is from the usual id table, UNVERIFIED.
func groundSkillUser(id, level int) bool {
	switch id {
	case 0x3b, 0x38:
		return true
	case 0x33:
		return level > 3
	case 0x1b:
		return level > 7
	}

	return false
}

// DiabloActionWeights returns the 17-entry weight table of
// MONAI_DiabloChooseAction 0x5e7710 (VERIFIED against the decompile and the
// disassembly, including the order of the adjustments).
func DiabloActionWeights(s DiaSituation) [diaWeights]int {
	var w [diaWeights]int

	st := s.Stats
	fire, light := st.S27, st.S29

	switch {
	case s.Melee:
		w[diaA1], w[diaLight], w[diaCold] = 40, 40, 40
		w[diaA2], w[diaFire], w[diaWall] = 70, 24, 15

		if st.LifePct < 20 {
			w[diaA1] = 50
		}

		if st.HasState0xb {
			w[diaCold], w[diaWall] = 0, 0
		}

		if light < fire {
			w[diaFire] -= 10
		}

		if fire < light {
			w[diaFire] += 10
		}

		if !s.LineClear {
			w[diaLight], w[diaFire] = 0, 0
		}

		if s.PortalNear {
			w[diaPrisonAny] = 10
		}
	case s.LineClear:
		w[diaLight], w[diaFire], w[diaPrison], w[diaCircle], w[diaWall], w[diaRun] = 25, 25, 20, 20, 15, 10

		if s.EdgeDist > 25 {
			w[diaWall] -= 5
			w[diaRun] = 20
			w[diaLight] = 0
		}

		if light < fire {
			w[diaFire] -= 10
			w[diaWall] -= 10
		}

		if fire < light {
			w[diaFire] += 10
			w[diaWall] += 10
		}

		if s.Count < 2 {
			w[diaFire] -= 10
		}

		if s.Count > 3 {
			w[diaFire] += 5
		}

		if s.Score > 0x3c {
			w[diaPrison] += 10
		}

		if s.Count < 2 && s.Scaling < 2 {
			w[diaPrison] = 0
		}

		away := func() {
			w[diaWalkHome], w[diaCircle], w[diaRun], w[diaFirewall] = 0, 10, 0, 15
			if w[diaPrison] == 0 {
				w[diaPrison] = 10
			}
		}

		switch {
		case s.GroundUser && !s.TargetAway:
			w[diaFire] += 10
			w[diaRun] = 30
			w[diaFirewall] = 15
		case s.GroundUser || s.TargetAway:
			away()
		}

		if s.TargetFar {
			w[diaPrison], w[diaRunHome] = 20, 60
		}

		if s.PortalNear {
			w[diaPrisonAny] = 15
		}
	default:
		w[diaAttack11], w[diaFire], w[diaWall], w[diaPrison], w[diaCircle] = 5, 25, 25, 40, 25

		if s.Count < 2 {
			w[diaWall] -= 5
			w[diaFire] = 0
			w[diaWalkHome] = 25
			w[diaPrison] = 0
		}

		away := func() {
			w[diaWalkHome], w[diaCircle], w[diaFirewall] = 0, 15, 25
			if w[diaPrison] == 0 {
				w[diaPrison] = 20
			}

			if s.Count < 2 {
				w[diaPrison] -= 5
			}
		}

		switch {
		case s.GroundUser && !s.TargetAway:
			w[diaFirewall] = 15
		case s.GroundUser || s.TargetAway:
			away()
		}

		if s.TargetFar {
			w[diaRunHome] = 60
		}

		if s.PortalNear {
			w[diaPrisonAny] = 20
		}
	}

	if s.PrisonTest != 0 {
		w[diaPrison] = 0
	}

	if w[diaPrisonAny] != 0 && s.HasPortalOb && s.PortalTest == 0 {
		w[diaPrisonAny] = 0
	}

	return w
}

// DiabloScore is MONAI_Diablo_ScoreTargetResists 0x5e7420 (VERIFIED formula).
// melee: target within melee reach; lineClear: no blocked line; leashBonus: the
// monster is farther than 0x55 from the anchor while the target is nearer.
func DiabloScore(s DiaStats, melee, lineClear, leashBonus bool) int {
	if s.IsMonster && s.Threat < 2 {
		return 0
	}

	zone := 0
	if leashBonus {
		zone = 100
	}

	cold, reach := 0, 0

	switch {
	case melee:
		cold, reach = s.S2B, 100
	case lineClear:
		reach = 0x4b
	}

	rank := s.RightRank*s.RightLevel + s.LeftRank*s.LeftLevel
	lowLife := 0

	if s.LifePct < 20 {
		lowLife = 100
	}

	state := 0
	if s.HasState0xb {
		state = 100
	}

	dmg := ((s.S3A >> 8) + s.S37 + s.S35 + s.S33 + s.S31 + s.S16) / 2
	v := lowLife*3 + (cold+(s.S24+s.S25*2)*4+s.S29+s.S27)/0xf + (zone+reach)*5 +
		(rank/4+state*2+dmg)*2

	v /= 0x16
	if v == 0 {
		v = 1
	}

	return v
}

func (c *Ctx) diabloMelee(t Target) bool {
	if m, ok := c.W.(MeleeRanger); ok {
		return m.InMeleeRange(c.B, t)
	}

	return c.Target != nil && t.ID == c.Target.ID && c.InRange
}

// diabloCandidates returns the candidates, scored.
func (c *Ctx) diabloPick(anchor *Command) (best *Target, score, count int, bestStats DiaStats) {
	b := c.B

	var cands []DiaCandidate

	if src, ok := c.W.(DiabloTargetSource); ok {
		cands = src.DiabloCandidates(b)
	} else if c.Target != nil {
		st := DiaStats{LifePct: 100, IsMonster: !c.Target.IsPlayer}
		if rf, ok := c.W.(ResistFinder); ok {
			st.S27, st.S2B, st.S29 = rf.Resists(*c.Target)
		}

		cands = []DiaCandidate{{Target: *c.Target, Stats: st}}
	}

	selfAway := anchor != nil && b.DistanceTo(anchor.X, anchor.Y) > diabloAway

	for i := range cands {
		cd := cands[i]
		count++

		s := 0

		if !cd.Dead {
			leash := false
			if anchor != nil && selfAway {
				leash = EdgeDistance(cd.Target.X-anchor.X, cd.Target.Y-anchor.Y, cd.Target.Size) < diabloAway
			}

			s = DiabloScore(cd.Stats, c.diabloMelee(cd.Target), !c.lineBlocked(cd.Target, 4), leash)
		}

		if s > score {
			score = s
			t := cd.Target
			best = &t
			bestStats = cd.Stats
		}
	}

	return best, score, count, bestStats
}

// diabloChoose is MONAI_DiabloChooseAction.
func (c *Ctx) diabloChoose(t *Target, anchor *Command, score, count int, st DiaStats) int {
	b := c.B

	if b.Scratch[0] != 0 {
		return b.Scratch[0]
	}

	if t == nil {
		if f, ok := c.W.(FootprintChecker); ok && f.FootprintBlocked(b) {
			return diaWander
		}

		if b.Roll(1000) > 0 {
			return diaWait
		}

		return diaAttack11
	}

	s := DiaSituation{HasTarget: true, Stats: st, Count: count, Score: score, Scaling: 1}
	s.Melee = c.diabloMelee(*t)
	s.LineClear = !c.lineBlocked(*t, 4)
	s.EdgeDist = EdgeDistance(b.X-t.X, b.Y-t.Y, b.Size)
	s.GroundUser = groundSkillUser(st.LeftID, st.LeftLevel) || groundSkillUser(st.RightID, st.RightLevel)

	if anchor != nil {
		d := EdgeDistance(t.X-anchor.X, t.Y-anchor.Y, t.Size)
		s.TargetAway, s.TargetFar = d > diabloAway, d > diabloVeryFar
	}

	var portal *Point

	if t.IsPlayer {
		if ps, ok := c.W.(DiabloPortalSource); ok {
			if _, p, lvl, found := ps.LastPortal(b, *t); found {
				s.HasPortalOb = true
				portal = &p

				if lvl == 0x6c && anchor != nil &&
					EdgeDistance(p.X-anchor.X, p.Y-anchor.Y, 0) < diabloAway {
					s.PortalNear = true
				}
			}
		}
	}

	if ps, ok := c.W.(PlayerScaler); ok {
		s.Scaling = ps.PlayerScaling(b)
	}

	if pt, ok := c.W.(DiabloPrisonTester); ok {
		s.PrisonTest = pt.PrisonTest(b, *t)

		if s.PortalNear && portal != nil {
			s.PortalTest = pt.PrisonTest(b, Target{X: portal.X, Y: portal.Y})
		}
	} else {
		s.PortalTest = 1
	}

	w := DiabloActionWeights(s)

	return pickWeighted(b, w[:])
}

// thinkDiabloF5 is MONAI_Think_Diablo 0x5e8150 with its helpers (VERIFIED,
// decompile and disassembly). Differences from the earlier port: the target is
// chosen by score over the heroes (host list), "engaged" is the clear line of
// sight test and the weights use the target's life, state and resistances;
// the portal-prison and teleporting-hero adjustments, the prison availability
// test, the Skill8 buff, the walk-home action (it walks to the TARGET) and the
// two-step run home are all in. Slots: Skill1 DiabLight (0x170, mode byte
// 0x180), 2 DiabCold, 3 DiabFire, 4 DiabWall, 5 DiabRun, 6 PrimeFirewall,
// 7 DiabPrison, 8 the buff (0x17e, mode byte 0x187).
//
// UNVERIFIED: the exact membership and order of the candidate lists (the exe
// walks player buckets plus two unit lists and prefers a list-2 unit when the
// best has no path; only the player-style scoring is ported), the polarity of
// the prison test (taken literally from the code) and the PrisonAny cast, which
// the exe issues with argument words that look like leftovers (target none,
// x = portal object id, y = 2).
func thinkDiabloF5(c *Ctx) {
	b := c.B
	p := b.Profile

	anchor := b.FindCommand(CmdAnchor)
	if anchor == nil {
		b.AppendCommand(Command{Type: CmdAnchor, X: b.X, Y: b.Y})
		anchor = b.FindCommand(CmdAnchor)
	}

	best, score, count, st := c.diabloPick(anchor)
	act := c.diabloChoose(best, anchor, score, count, st)

	tgtOrSelf := c.self()
	if best != nil {
		tgtOrSelf = *best
	}

	if act != diaLight && p.Skills[slot8].Used() {
		if as := c.auraState(slot8); as >= 0 && !c.W.HasState(b, as) && c.tryCast(slot8, tgtOrSelf) {
			b.Scratch[0] = 0

			return
		}
	}

	clear := func() { b.Scratch[0] = 0 }

	castOne := func(slot int) {
		if !p.Skills[slot].Used() {
			c.Sleep(2)
			clear()

			return
		}

		c.W.Cast(b, slot, tgtOrSelf)
		c.busy()
		clear()
	}

	switch act {
	case diaWalkHome:
		c.SetSpeed(0x14)

		if best != nil {
			c.moveTo(Point{best.X, best.Y})
		}

		clear()
	case diaA1, diaA2, diaAttack11:
		mode := map[int]Mode{diaA1: ModeAttack1, diaA2: ModeAttack2, diaAttack11: Mode(0xb)}[act]
		if best != nil {
			c.W.Attack(b, mode, *best)
			c.busy()
		}

		clear()
	case diaLight:
		if p.Skills[slot1].Used() {
			if !c.W.HasState(b, 0xc) {
				c.W.Cast(b, slot1, tgtOrSelf)
				c.busy()

				b.Scratch[0] = diaLight

				return
			}

			c.setState(0xc, false)
		}

		c.Sleep(2)
		clear()
	case diaFire:
		castOne(slot3)
	case diaCold:
		castOne(slot2)
	case diaWall:
		castOne(slot4)
	case diaPrison:
		castOne(slot7)
	case diaRun:
		castOne(slot5)
	case diaFirewall:
		castOne(slot6)
	case diaPrisonAny:
		c.diabloPrisonAny(best)
		clear()
	case diaCircle:
		if best != nil {
			c.Circle(*best, 4)
		}

		clear()
	case diaRunHome:
		c.diabloRunHome(anchor)
		clear()
	case diaWander:
		c.SetSpeed(0x14)
		c.Wander(5)
		clear()
	default:
		clear()

		switch b.Diff {
		case Nightmare:
			c.Sleep(8)
		case Hell:
			c.Sleep(4)
		default:
			c.Sleep(12)
		}
	}
}

// diabloPrisonAny is action 0xf: only for a player target; the portal object
// of that player must exist; then DiabPrison is cast with no target (UNVERIFIED
// argument words, see thinkDiabloF5). The Go port casts it at the target's
// portal position when the host gives one, and waits 2 otherwise.
func (c *Ctx) diabloPrisonAny(t *Target) {
	b := c.B

	if t == nil || !t.IsPlayer {
		c.Sleep(3)

		return
	}

	ps, ok := c.W.(DiabloPortalSource)
	if !ok {
		c.Sleep(2)

		return
	}

	_, p, _, found := ps.LastPortal(b, *t)
	if !found || !b.Profile.Skills[slot7].Used() {
		c.Sleep(2)

		return
	}

	c.tryCast(slot7, Target{X: p.X, Y: p.Y})
}

// diabloRunHome is action 0xe: run to the anchor (speed override 0x32/100); if
// no path is queued, cancel and walk to the point half the tile distance
// beyond the anchor on the monster's side, then wait 2.
func (c *Ctx) diabloRunHome(a *Command) {
	b := c.B

	c.SetSpeed(0x32)

	if c.walkToPointOrClear(Point{a.X, a.Y}) {
		return
	}

	d := unitTileDist(b, a.X, a.Y)
	sx, sy := sign(b.X-a.X), sign(b.Y-a.Y)

	c.SetSpeed(0x32)

	if c.walkToPointOrClear(Point{a.X + (d>>1)*sx, a.Y + (d>>1)*sy}) {
		return
	}

	c.Sleep(2)
}

// ---------------------------------------------------------------- UberIzual

// thinkUberIzual is MONAI_Think_UberIzual 0x5f7df0 (VERIFIED, decompile): the
// Izual body of ai_boss2.go preceded by two checks and without Izual's quest
// hook. aip1 melee%, aip2 hold%, aip3 nova% when closing in, aip4 nova% when
// in reach, aip5 stall frames after a nova, aip6 swings that follow it.
// Skill2 is a buff cast when its aura state is missing; when the line to the
// target is blocked (mask 6) Skill3 is cast at the target's position.
func thinkUberIzual(c *Ctx) {
	b, t := c.B, c.Target
	p := b.Profile

	if t == nil {
		c.Sleep(10)

		return
	}

	if p.Skills[slot2].Used() {
		if as := c.auraState(slot2); as >= 0 && !c.W.HasState(b, as) {
			c.Cast(slot2, c.self())

			return
		}
	}

	if c.lineBlocked(*t, 6) {
		c.Cast(slot3, Target{X: t.X, Y: t.Y})

		return
	}

	izualBody(c, *t)
}

// ---------------------------------------------------------------- Vulture

// thinkVultureF5 is MONAI_Think_Vulture 0x5f2150 (VERIFIED, decompile). Against
// the earlier port it adds the room-mismatch branch, reuses a pending waypoint
// through the free-cell search, measures with the exe's tile distance and
// fixes the minimum waypoint distance, which is clamp(2*(laps+8), 12, 36) (the
// exe doubles the already incremented lap value). aip1 attack%, aip2 stall,
// aip3 prey hp%, aip4 circle%, aip5 move%. Scratch[0] laps (-1 after landing),
// Scratch[1]/[2] the waypoint.
func thinkVultureF5(c *Ctx) {
	b, t := c.B, c.Target
	laps := b.Scratch[0]

	fl, _ := c.W.(Flier)

	land := func() bool { // MONAI_EnterBurrowedState
		if !b.Airborne {
			return false
		}

		b.Airborne = false

		if fl != nil {
			fl.Land(b)
		}

		return true
	}

	lift := func() { // MONAI_ExitBurrowedState
		b.Airborne = true

		if fl != nil {
			fl.TakeOff(b)
		}
	}

	settle := func(to Target) {
		c.RunToRange(to, 9, 2)
		c.Sleep(vultureHoverStep)

		b.Scratch[0] = -1
	}

	if t == nil {
		if laps < 1 {
			c.Sleep(vultureHoverStep)

			return
		}

		if land() {
			settle(c.self())

			return
		}

		c.Sleep(vultureHoverStep)

		return
	}

	sq := sqDist(b.X, b.Y, t.X, t.Y)

	if rm, ok := c.W.(RoomMatcher); ok && !rm.SameRoom(b, *t) && sq > vultureFarSq {
		if laps < 1 {
			c.WalkToRange(*t, 9, 0)

			return
		}

		if land() {
			c.RunToRange(c.self(), 9, 2)
			c.Sleep(vultureHoverStep)

			b.Scratch[0] = -1

			return
		}
	}

	if laps == 0 && b.Leader == nil && sq > vultureFarSq && b.Roll(100) < vultureTakeOff {
		lift()

		b.Scratch[0] = vultureLapsBase + int(b.Seed.Step()&1)
		c.RunToRange(*t, 8, 8)
		c.Sleep(vultureHoverStep)

		return
	}

	prey := false
	if pf, ok := c.W.(PreyFinder); ok {
		_, prey = pf.NearestPrey(b, 11, b.AIP(3))
	}

	switch {
	case laps >= 2:
		if !prey && (c.Dist > 5 || b.Roll(100) >= 15) {
			lift()
			c.vultureFlyStep(*t, laps)

			return
		}

		b.Scratch[0], laps = 1, 1

		fallthrough
	case laps == 1:
		if land() {
			settle(*t)

			return
		}

		b.Scratch[0] = 8

		c.Sleep(vultureHoverStep)
	}

	if c.InRange {
		if b.Roll(100) < b.AIP(1) {
			c.Attack(ModeAttack1, *t)
		} else {
			c.Sleep(b.AIP(2))
		}

		return
	}

	if laps != -1 && b.Roll(100) >= b.AIP(5) {
		c.Sleep(b.AIP(2))

		return
	}

	if b.Roll(100) < b.AIP(4) {
		c.Circle(*t, 6)
	} else {
		c.WalkToRange(*t, 9, 0)
	}

	b.Scratch[0] = 0
	c.Sleep(vultureHoverStep)
}

// vultureFlyStep is the flight leg: pick (or keep) a waypoint around the
// target and move to it, one lap used.
func (c *Ctx) vultureFlyStep(t Target, laps int) {
	b := c.B

	wx, wy := b.Scratch[1], b.Scratch[2]
	if wx == 0 || wy == 0 || unitTileDist(b, wx, wy) < 2 {
		k := laps + 8
		wx = int(b.Seed.Roll(int32(2*k))) + (t.X - k)
		wy = int(b.Seed.Roll(int32(2*k))) + (t.Y - k)

		min := clamp(2*k, 12, 36)

		for guard := 0; guard < 256 && unitTileDist(b, wx, wy) < min; guard++ {
			wx += sign(wx-b.X) * 1
			wy += sign(wy-b.Y) * 1

			if wx == b.X && wy == b.Y {
				wx++
				wy++
			}
		}

		b.Scratch[1], b.Scratch[2] = wx, wy
		c.SetSpeed(0)
		c.moveTo(Point{wx, wy})
		c.Sleep(vultureHoverStep)

		b.Scratch[0]--

		return
	}

	if fc, ok := c.W.(WaypointCellFinder); ok {
		if p, found := fc.NearestFreeCell(b, Point{wx, wy}); found {
			b.Scratch[1], b.Scratch[2] = p.X, p.Y
		}
	}

	b.Scratch[0]--

	c.SetSpeed(0)
	c.moveTo(Point{b.Scratch[1], b.Scratch[2]})
	c.Sleep(vultureHoverStep)
}

// ---------------------------------------------------------------- BloodRaven

// thinkBloodRavenF5 is MONAI_Think_BloodRaven 0x5e5260 (VERIFIED, decompile
// and disassembly). Against the earlier port: the leash tests use the TARGET's
// distance to the anchor (local_2c, 50 or more) or the monster's (over 50),
// the close-in and 5% moves are MONAI_RandomizedSidestepCommand around the
// target (the step replaces the tick distance for the later tests when taken),
// and the arrow spot is r = 5..19 with the coin/modulo/sign rolls of the exe.
// Scratch[0] shot probability (+3 per tick), [1] arrows fired, [2] heading
// home.
func thinkBloodRavenF5(c *Ctx) {
	b, t := c.B, *c.Target
	p := b.Profile

	anchor := b.FindCommand(CmdAnchor)
	if anchor == nil {
		b.AppendCommand(Command{Type: CmdAnchor, X: b.X, Y: b.Y})
		anchor = b.FindCommand(CmdAnchor)
	}

	home := Point{anchor.X, anchor.Y}
	tHome := EdgeDistance(t.X-anchor.X, t.Y-anchor.Y, t.Size)

	if c.Dist > brIgnoreDist {
		c.Sleep(5)

		return
	}

	if tHome >= 0x32 || b.DistanceTo(anchor.X, anchor.Y) > 0x32 {
		b.Scratch[brFleeHome] = 1
		c.SetSpeed(100)

		if c.runToPoint(home) {
			return
		}
	}

	if b.Scratch[brFleeHome] != 0 && b.DistanceTo(anchor.X, anchor.Y) > brHomeDist {
		c.SetSpeed(100)

		if c.runToPoint(home) {
			return
		}
	}

	b.Scratch[brFleeHome] = 0

	d := c.Dist
	if d > brCloseInDist && tHome < 0x32 {
		d = d >> 1
		if d < 12 {
			d = 12
		}

		c.SetSpeed(100)

		if c.sidestep(t, d) {
			return
		}
	}

	b.Scratch[brCharge] += 3

	if p.Skills[slot1].Used() && !c.InRange && b.Scratch[brShots] < int(b.Diff)*2+8 &&
		b.Roll(100) < b.Scratch[brCharge] {
		r := b.Roll(15) + 5
		coin := b.Seed.Step() & 1

		var ox, oy int
		if coin == 0 {
			ox, oy = int(b.Seed.Roll(int32(r))), r
		} else {
			ox, oy = r, int(b.Seed.Roll(int32(r)))
		}

		if b.Seed.Step()&1 != 0 {
			ox = -ox
		}

		if b.Seed.Step()&1 != 0 {
			oy = -oy
		}

		c.tryCast(slot1, Target{X: t.X + ox, Y: t.Y + oy, Size: 1})

		b.Scratch[brShots]++
		b.Scratch[brCharge] = 0

		return
	}

	melee := t

	if d > 5 {
		if b.Roll(100) < 5 && tHome < 0x32 {
			c.SetSpeed(100)
			c.sidestep(t, 12)

			return
		}

		at, _, ok := c.W.AttackTarget(b)
		if ok && !b.Aggressive && b.Roll(100) < 80 {
			if p.Skills[slot2].Used() && b.Roll(100) < (int(b.Diff)+4)*10 {
				c.Cast(slot2, at)

				return
			}

			c.Attack(ModeAttack1, at)

			return
		}

		c.SetSpeed(50)

		if c.Circle(t, 4) {
			return
		}
	}

	if b.Roll(100) < 30 && d < 12 {
		c.SetSpeed(100)

		if c.RunAway(t, 12-d) {
			return
		}
	}

	c.Attack(ModeAttack1, melee)
}
