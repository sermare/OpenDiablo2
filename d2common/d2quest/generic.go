package d2quest

// A table-driven quest node for the quests that follow the common skeleton of
// quests-2.md section 3 (available -> quest-giver line acked -> left town /
// entered area -> goal reached with reward pending -> reward line acked).
// The Act 2 quests after Radament and the quests of Acts 3-5 are described
// as specs; the dialogue (which NPC says which message in which state) is
// read from the speech tables of data/messages.csv, which were dumped from
// the 1.14b binary. The state machines themselves are only verified for the
// Act 2 quests (D2MOO source + binary notes); the Act 3-5 specs follow the
// pointers of quests.md section 5 and are UNVERIFIED in their details (levels,
// monster classes, rewards); each spec says so.

// townLevels are the first level of every act (levels.txt).
//
//nolint:gochecknoglobals // static lookup data
var townLevels = map[int]bool{
	LevelRogueEncampment: true, LevelLutGholein: true, LevelKurastDocktown: true,
	LevelPandemonium: true, LevelHarrogath: true,
}

// step is a dialogue driven transition: the hero heard message msg of npc
// while the quest was in state from.
type step struct {
	from     int
	npc, msg int
	to       int  // new state (0: unchanged)
	goal     bool // reaching the goal: primary goal done + reward pending
	claim    bool // the quest is completed (reward granted) at once
	nomark   bool // do not mirror the new state into the record bits
	fx       func(g *Game, q *Quest) []Effect
}

// trig is a world driven transition.
type trig struct {
	ev       EventKind
	level    int // EvAreaChanged: the new level; EvMonsterKilled: the level of the kill (0: any)
	old      int // EvAreaChanged: the level left (0: any)
	monster  int
	monsters []int                        // any of these classes (instead of monster)
	names    []string                     // or this monster name (Event.Name), as the boss kills are also matched
	cond     func(g *Game, q *Quest) bool // extra condition
	super    string
	object   int
	item     string
	req      []string // items the hero must carry
	min, max int      // the quest state must be in [min, max]; min 0 means 1 (-1: 0), max 0 means goal-1
	to       int
	goal     bool
	direct   bool // goal: grant the reward at once (no pending reward)
	count    int  // fires on the count-th match (0 and 1: the first)
	page     int  // quest log page to store (0: derived)
	bit      int  // record bit to set (-1: none; 0: mirror the state)
	fx       func(g *Game, q *Quest) []Effect
	exe      *exeKill // verified kill bits of the exe, used when Game.ExeBossBits is set (boss_exe.go)
}

// spec describes a table driven quest.
type spec struct {
	id, slot, act, logIndex int
	name, label             string
	start                   int // initial state
	goal                    int // state of the goal (reward pending)
	tbl                     map[int]int
	rp, done                int // speech tables of reward pending / just rewarded
	rpExp, doneExp          int // tables used instead in the expansion (0: none)
	steps                   []step
	trigs                   []trig
	claimMsgs               []int // messages that claim the reward (none: every spoken line of the rp table)
	claimFx                 func(g *Game, q *Quest) []Effect
	noLeaveRule             bool // do not turn "left town in state 2" into state 3
	tableFn                 func(g *Game, q *Quest, def int) int
	extra                   func(g *Game, q *Quest, npc int) []Speech
	setup                   func(q *Quest, s *spec)
	logPage                 func(g *Game, q *Quest) int
}

type genData struct {
	hits     []int
	rewarded bool
}

func (s *spec) rpTable(g *Game) int {
	if g.Expansion && s.rpExp != 0 {
		return s.rpExp
	}

	return s.rp
}

func (s *spec) doneTable(g *Game) int {
	if g.Expansion && s.doneExp != 0 {
		return s.doneExp
	}

	return s.done
}

// chainAfter lists the quests that become available when a quest is
// completed (QUESTS_SequenceCycler of the notes; the Act 3-5 links are the
// natural order of the dialogue tables and UNVERIFIED).
//
//nolint:gochecknoglobals // static lookup data
var chainAfter = map[int][]int{
	QuestRadament:        {QuestSevenTombs},
	QuestArcane:          {QuestSevenTombs},
	QuestTaintedSun:      {QuestArcane},
	QuestHoradricStaff:   {QuestArcane},
	QuestBlade:           {QuestBlackenedTemple},
	QuestBlackenedTemple: {QuestGuardian},
	QuestRite:            {QuestEve},
}

// chainFrom makes the quests after q available.
func (g *Game) chainFrom(q *Quest) {
	for _, id := range chainAfter[q.ID] {
		if n := g.byID[id]; n != nil && n.State == 0 && n.NotIntro && !g.get(n, FlagRewardGranted) &&
			!g.get(n, FlagCompletedEarly) {
			g.setState(n, 1)
		}
	}
}

// markState mirrors the quest state into the record bits (state 2 STARTED,
// state 3 LEAVETOWN or ENTERAREA depending on the log page, higher states the
// CUSTOM bits).
func (g *Game) markState(q *Quest) {
	if g.grantedOrPending(q) {
		return
	}

	switch {
	case q.State == 2:
		g.set(q, FlagStarted, "started")
	case q.State == 3 && q.LastState == 1:
		g.set(q, FlagLeaveTown, "left town")
	case q.State == 3:
		g.set(q, FlagEnterArea, "entered the area")
	case q.State >= 4 && q.State <= 7:
		g.set(q, FlagCustom1+q.State-4, "quest stage")
	}
}

// reachGoal marks the goal of the quest: primary goal done and the reward
// pending (or granted at once when direct). The log page 3 is pushed after
// the original's timers.
func (g *Game) reachGoal(q *Quest, goalState int, direct bool) {
	if g.grantedOrPending(q) || !q.NotIntro {
		return
	}

	g.setState(q, goalState)
	g.set(q, FlagPrimaryGoal, "goal reached")

	if direct {
		g.set(q, FlagRewardGranted, "goal reached, no reward step")
	} else {
		g.set(q, FlagRewardPending, "goal reached")
	}

	g.globalDone(q)
	g.after(8, func() { g.cycle(q, 3, true) })

	if direct {
		g.chainFrom(q)
	}
}

// claimReward completes a quest: reward granted, reward pending cleared.
func (g *Game) claimReward(q *Quest, s *spec) {
	if g.get(q, FlagRewardGranted) {
		return
	}

	if d, ok := q.data.(*genData); ok {
		d.rewarded = true
	}

	if s != nil {
		g.setState(q, s.goal+1)
	}

	g.set(q, FlagRewardGranted, "reward granted")
	g.clear(q, FlagRewardPending, "reward taken")
	g.globalDone(q)
	q.LastState = 13
	g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 13})
	g.chainFrom(q)
}

// hasGiverLine reports whether npc has a spoken line (mode 0 or 1) in table idx.
func (q *Quest) hasGiverLine(npc, idx int) bool {
	for _, s := range q.pick(npc, idx) {
		if s.Mode == ModeGiver || s.Mode == ModeSpoken {
			return true
		}
	}

	return false
}

func (s *spec) isClaim(q *Quest, g *Game, npc, msg int) bool {
	if len(s.claimMsgs) > 0 {
		for _, m := range s.claimMsgs {
			if m == msg {
				return true
			}
		}

		return false
	}

	for _, l := range q.pick(npc, s.rpTable(g)) {
		if l.Msg == msg && (l.Mode == ModeGiver || l.Mode == ModeSpoken) {
			return true
		}
	}

	return false
}

func (t *trig) monsterMatches(e *Event) bool {
	class := e.Monster

	for _, n := range t.names {
		if nameIs(e, n) {
			return true
		}
	}

	if len(t.names) > 0 && len(t.monsters) == 0 && t.monster == 0 {
		return false
	}

	if len(t.monsters) == 0 {
		return t.monster == 0 || t.monster == class
	}

	for _, m := range t.monsters {
		if m == class {
			return true
		}
	}

	return false
}

func (t *trig) matches(g *Game, q *Quest, goal int, e *Event) bool {
	if t.ev != e.Kind {
		return false
	}

	max := t.max
	if max == 0 {
		max = goal - 1
	} else if max < 0 {
		max = 0
	}

	if t.cond != nil && !t.cond(g, q) {
		return false
	}

	for _, code := range t.req {
		if !g.hasItem(code) {
			return false
		}
	}

	min := t.min
	if min == 0 {
		min = 1
	} else if min < 0 {
		min = 0 // -1: the trigger also works while the quest is dormant
	}

	if q.State < min || q.State > max {
		return false
	}

	switch e.Kind {
	case EvAreaChanged:
		return (t.level == 0 || t.level == e.NewLevel) && (t.old == 0 || t.old == e.OldLevel)
	case EvMonsterKilled:
		return t.monsterMatches(e) && (t.super == "" || t.super == e.Super) &&
			(t.level == 0 || t.level == e.Level || e.Level == 0)
	case EvObjectOperated:
		return t.object == e.Object
	case EvItemPickedUp, EvItemRemoved, EvItemDropped:
		return t.item == e.Item
	}

	return false
}

func newSpecQuest(s *spec) *Quest {
	d := &genData{hits: make([]int, len(s.trigs))}
	q := &Quest{ID: s.id, Slot: s.slot, Act: s.act, Name: s.name, Label: s.label, LogIndex: s.logIndex,
		Active: true, NotIntro: true, State: s.start, InitNo: s.goal, SeqID: -1,
		tables: speechTables(s.label), data: d}

	q.seq = func(g *Game, q *Quest) bool {
		if q.State == 0 && q.NotIntro {
			g.setState(q, 1)
		}

		return true
	}

	if s.logPage != nil {
		q.status = s.logPage
	}

	apply := func(g *Game, q *Quest, to int, goal, direct bool, page, bit int, nomark bool) {
		if to > 0 && to > q.State {
			g.setState(q, to)
		}

		switch {
		case goal:
			g.reachGoal(q, s.goal, direct)
		case page > 0:
			q.LastState = page
		}

		switch {
		case goal || nomark:
		case bit > 0:
			g.set(q, bit, "quest stage")
		case bit == 0:
			g.markState(q)
		}

		if !goal && q.LastState != 0 {
			g.cycle(q, q.LastState, true)
		}
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if !q.NotIntro && !g.rewardOwed(q) {
			return
		}

		if g.get(q, FlagRewardPending) && !g.get(q, FlagRewardGranted) && s.isClaim(q, g, e.NPC, e.Msg) {
			g.claimReward(q, s)

			if s.claimFx != nil {
				for _, f := range s.claimFx(g, q) {
					g.emit(f)
				}
			}

			return
		}

		for _, st := range s.steps {
			if st.from != q.State || st.npc != e.NPC || st.msg != e.Msg || g.grantedOrPending(q) {
				continue
			}

			if st.from == 1 {
				q.LastState = 1
			}

			switch {
			case st.claim:
				g.setState(q, st.to)
				g.reachGoal(q, st.to, true)
				d.rewarded = true
				q.LastState = 13
				g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 13})
			default:
				apply(g, q, st.to, st.goal, false, 0, 0, st.nomark)
			}

			if st.fx != nil {
				for _, f := range st.fx(g, q) {
					g.emit(f)
				}
			}

			return
		}
	}

	handle := func(g *Game, q *Quest, e *Event) {
		if !q.NotIntro || g.grantedOrPending(q) {
			return
		}

		for i := range s.trigs {
			t := &s.trigs[i]
			if !t.matches(g, q, s.goal, e) {
				continue
			}

			d.hits[i]++
			if t.count > 1 && d.hits[i] < t.count {
				continue
			}

			page := t.page
			if page == 0 && t.ev == EvAreaChanged && t.to == 3 {
				page = 2
			}

			direct := t.direct

			if g.ExeBossBits && t.exe != nil {
				if !t.exe.applies(g, e) {
					continue
				}

				direct = true
			}

			apply(g, q, t.to, t.goal, direct, page, t.bit, false)

			if g.ExeBossBits && t.exe != nil {
				t.exe.finish(g, q, s)
			}

			if t.fx != nil {
				for _, f := range t.fx(g, q) {
					g.emit(f)
				}
			}
		}

		if e.Kind == EvAreaChanged && !s.noLeaveRule && !g.grantedOrPending(q) && q.State == 2 && townLevels[e.OldLevel] &&
			!townLevels[e.NewLevel] {
			q.LastState = 1
			g.setState(q, 3)
			g.markState(q)
			g.cycle(q, 1, true)
		}
	}

	for _, k := range []EventKind{EvAreaChanged, EvMonsterKilled, EvObjectOperated, EvItemPickedUp, EvItemRemoved} {
		q.on[k] = handle
	}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		if !q.NotIntro && !g.rewardOwed(q) {
			return nil
		}

		var out []Speech

		switch {
		case g.get(q, FlagRewardPending) && !g.get(q, FlagRewardGranted):
			out = q.pick(npc, s.rpTable(g))
		case g.get(q, FlagRewardGranted):
			if d.rewarded {
				out = q.pick(npc, s.doneTable(g))
			}
		default:
			idx, ok := s.tbl[q.State]
			if !ok {
				idx = -1
			}

			if s.tableFn != nil {
				idx = s.tableFn(g, q, idx)
			}

			out = q.pick(npc, idx)
		}

		if s.extra != nil {
			out = append(out, s.extra(g, q, npc)...)
		}

		return out
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		switch {
		case (!q.NotIntro && !g.rewardOwed(q)) || g.get(q, FlagRewardGranted):
			return false
		case g.get(q, FlagRewardPending):
			return q.hasGiverLine(npc, s.rpTable(g))
		}

		for _, st := range s.steps {
			if st.from == q.State && st.npc == npc {
				return true
			}
		}

		return false
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, e *Event) {
		switch {
		case g.get(q, FlagRewardGranted):
			q.State = 0
			g.globalDone(q)
			g.chainFrom(q)
		case g.get(q, FlagCompletedEarly):
			q.State = 0
		default:
			restoreStd(g, q, e)

			for bit := FlagCustom1; bit <= FlagCustom1+3; bit++ {
				if g.get(q, bit) && q.State < 4+bit-FlagCustom1 {
					q.State = 4 + bit - FlagCustom1
				}
			}
		}
	}

	if s.setup != nil {
		s.setup(q, s)
	}

	return q
}

// ---- helpers the specs use ----

func fxs(effects ...Effect) func(g *Game, q *Quest) []Effect {
	return func(g *Game, q *Quest) []Effect {
		out := make([]Effect, len(effects))
		for i, e := range effects {
			e.Quest = q.ID
			out[i] = e
		}

		return out
	}
}

// reward is an EffectReward with the reward name (see EffectReward).
func reward(name string, value int, note string) Effect {
	return Effect{Kind: EffectReward, Code: name, Value: value, Note: note}
}

// questSupers are the super unique monsters of the later acts that quests wait
// for, by the label the engine gives them (UNVERIFIED spelling).
//
//nolint:gochecknoglobals // static lookup data
var questSupers = map[string]bool{"Shenk the Overseer": true, "Eldritch the Rectifier": true}

// IsQuestSuper reports whether a monster label is a super unique a quest of
// Acts 2-5 triggers on.
func IsQuestSuper(label string) bool { return questSupers[label] }
