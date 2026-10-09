package d2gamescreen

import (
	"os"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// OD2_AUTOQUEST=<quest id|name|act1> runs a scripted quest line through the real
// quest runtime: NPC talks, area changes, kills and object use are injected
// the way the engine reports them, and every stage logs the quest bits before
// and after (QUEST lines), the chosen NPC message with its Sounds.txt row
// (QUEST SPEECH lines) and AUTOQUEST expect lines with the checks. The run ends
// with "AUTOQUEST RESULT PASS|FAIL"; OD2_AUTOEXIT quits with the matching exit
// code (and saves the hero, so the .d2s export carries the quest bits).
//
//	OD2_AUTOQUEST          den (1), burial (2), tools (3), cain (4), tower (5),
//	                       andariel (6), radament (8) or act1 (all Act 1 quests in order);
//	                       act2, act3, act4, act5 or acts2-5 (one quest per act), or a
//	                       single Acts 2-5 stage (see autoquest_acts.go)
//	OD2_AUTOQUEST_REAL=1   den: spawn real monsters and let the hero kill them
//	                       (otherwise kill events are injected)
//	OD2_AUTOQUEST_AREA     levels.txt id the map stands for when the run starts
//	OD2_AUTOQUEST_DIFF     difficulty (0..2) of the quest record
//
// The run resets the quest slots it exercises in the hero's record first (and
// marks the predecessors of the quest as done), so it can start from any save.
// actLastQuestSlot is the last record slot of Act 5 (slots 8..40 belong to Acts 2-5).
const actLastQuestSlot = 40

const (
	autoQuestDelay       = 3.0   // seconds after the hero exists
	autoQuestStepGap     = 0.05  // seconds between steps
	autoQuestTimeout     = 90.0  // seconds one step may wait
	autoQuestRealTimeout = 240.0 // ... when real monsters have to be killed
	autoQuestRealFoes    = 3
)

// autoHost is what the scenario needs from the game: the engine implements it
// with the real quest runtime (engineHost) and the unit tests with a plain
// d2quest.Game, so the scripts themselves are tested without a display.
type autoHost interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	// Q is the quest system.
	Q() *d2quest.Game
	// Move reports an area change; Dispatch an event; Apply the effects of a
	// call made directly on Q; Tick advances the quest timers by frames.
	Move(area int)
	Dispatch(e d2quest.Event)
	Apply(effects []d2quest.Effect)
	Tick(frames int)
	Area() int
	// Talk plays the Talk row of an NPC once and returns the message ids voiced
	// (empty when the NPC had nothing spoken to say).
	Talk(class int) []int
	SkillPoints() int
	HeroLevel() int
	RogueHire() bool
	ImbuePending() bool
	LogText(act, index int) string
	// FoesAlive/Fight/SpawnFoes drive the real-monster Den of Evil.
	FoesAlive() int
	Fight()
	SpawnFoes(n int)
	Save()
	Exit(pass bool)
}

type autoStep struct {
	desc string
	fn   func(h autoHost, a *autoQuest) bool
}

type autoQuest struct {
	spec    string
	stages  []string
	steps   []autoStep
	idx     int
	elapsed float64
	gap     float64
	waited  float64
	checks  int
	failed  int
	done    bool
	real    bool
	started bool
	// scratch values the steps share
	skillBefore int
}

//nolint:gochecknoglobals // static lookup data
var autoQuestNames = map[string]string{
	"1": "den", "den": "den", "denofevil": "den",
	"2": "burial", "burial": "burial", "bloodraven": "burial",
	"3": "tools", "tools": "tools", "toolsofthetrade": "tools", "malus": "tools",
	"4": "cain", "cain": "cain", "searchforcain": "cain",
	"5": "tower", "tower": "tower", "forgottentower": "tower", "countess": "tower",
	"6": "andariel", "andariel": "andariel", "slaughter": "andariel", "sisterstotheslaughter": "andariel",
	"8": "radament", "radament": "radament",
	"act1": "act1", "all": "act1",
}

// act1Order is the order the Act 1 quests chain in.
//
//nolint:gochecknoglobals // static lookup data
var act1Order = []string{"den", "burial", "cain", "tools", "tower", "andariel"}

// stagePrereq lists the quests that must be done before a stage can run alone.
//
//nolint:gochecknoglobals // static lookup data
var stagePrereq = map[string][]int{
	"burial":   {d2quest.QuestDenOfEvil},
	"cain":     {d2quest.QuestDenOfEvil, d2quest.QuestBurial},
	"tools":    {d2quest.QuestDenOfEvil, d2quest.QuestBurial, d2quest.QuestCain},
	"andariel": {d2quest.QuestDenOfEvil, d2quest.QuestBurial, d2quest.QuestCain, d2quest.QuestTools},
}

func newAutoQuest(spec string) *autoQuest {
	a := &autoQuest{spec: spec, real: os.Getenv("OD2_AUTOQUEST_REAL") == "1"}

	key := strings.ToLower(strings.NewReplacer(" ", "", "'", "", "-", "", "_", "").Replace(spec))

	name, ok := autoQuestNames[key]
	if !ok {
		a.stages = nil
		return a
	}

	switch {
	case name == "act1":
		a.stages = act1Order
	case actStageOrder[name] != nil:
		a.stages = actStageOrder[name]
	default:
		a.stages = []string{name}
	}

	for _, s := range a.stages {
		a.addStage(s)
	}

	a.add("finish", func(h autoHost, a *autoQuest) bool { a.finish(h); return true })

	return a
}

// resetRecord clears the quest slots the run uses and marks the prerequisites
// of the first stage as done, so the run does not depend on the save.
func (a *autoQuest) resetRecord(p *d2hero.HeroProgress, diff int) {
	rec := p.QuestRecord(diff)

	for _, slot := range []int{0, 1, 2, 3, 4, 5, 6, 7, 9, d2s.QuestSlotAkaraRespec} {
		rec.SetSlot(slot, 0)
	}

	for bit := 0; bit < 64; bit++ {
		p.NPC.SetIntroBit(diff, bit, false)
		p.NPC.SetReturnBit(diff, bit, false)
	}

	// the Acts 2-5 stages start from a clean Acts 2-5 record (a real save has
	// most of these quests done); the first stage needs its predecessors done
	for _, st := range a.stages {
		if _, ok := actStageSlots[st]; ok {
			for slot := d2s.QuestSlotAct1Finished + 1; slot <= actLastQuestSlot; slot++ {
				rec.SetSlot(slot, 0)
			}

			break
		}
	}

	if len(a.stages) > 0 {
		for _, id := range stagePrereq[a.stages[0]] {
			rec.SetSlot(id, 1<<d2quest.FlagRewardGranted) // slot == quest id for Act 1
		}

		for _, slot := range actStageSlots[a.stages[0]][1] {
			rec.SetSlot(slot, 1<<d2quest.FlagRewardGranted)
		}
	}
}

func (a *autoQuest) add(desc string, fn func(h autoHost, a *autoQuest) bool) {
	a.steps = append(a.steps, autoStep{desc, fn})
}

// once adds a step that runs a function and is done.
func (a *autoQuest) once(desc string, fn func(h autoHost, a *autoQuest)) {
	a.add(desc, func(h autoHost, a *autoQuest) bool { fn(h, a); return true })
}

func (a *autoQuest) expect(h autoHost, desc string, ok bool) {
	a.checks++

	if !ok {
		a.failed++
		h.Infof("AUTOQUEST expect %s: FAIL", desc)

		return
	}

	h.Infof("AUTOQUEST expect %s: ok", desc)
}

func (a *autoQuest) finish(h autoHost) {
	r := h.Q()

	h.Save()

	for _, q := range r.Quests {
		if q.LogIndex > 0 {
			h.Infof("AUTOQUEST final %s", r.Describe(q))
		}
	}

	for _, l := range r.Log() {
		h.Infof("AUTOQUEST log act=%d quest=%d status=%d page=%d text=%q", l.Act, l.Index, l.Status, l.Page,
			shorten(h.LogText(l.Act, l.Index), 60))
	}

	pass := a.failed == 0 && a.checks > 0

	result := "PASS"
	if !pass {
		result = "FAIL"
	}

	h.Infof("AUTOQUEST RESULT %s stages=%v checks=%d failed=%d", result, a.stages, a.checks, a.failed)
	a.done = true
	h.Exit(pass)
}

func (a *autoQuest) advance(h autoHost, elapsed float64) {
	if a.done {
		return
	}

	if len(a.stages) == 0 {
		h.Errorf("AUTOQUEST: unknown quest %q", a.spec)
		h.Infof("AUTOQUEST RESULT FAIL stages=[] checks=0 failed=1")

		a.done = true
		h.Exit(false)

		return
	}

	a.elapsed += elapsed
	if a.elapsed < autoQuestDelay {
		return
	}

	a.gap += elapsed
	if a.gap < autoQuestStepGap || a.idx >= len(a.steps) {
		return
	}

	st := a.steps[a.idx]

	if !a.started {
		a.started = true
		h.Infof("AUTOQUEST start spec=%q stages=%v real_kills=%v", a.spec, a.stages, a.real)
	}

	if st.fn(h, a) {
		a.idx++
		a.waited = 0
		a.gap = 0

		return
	}

	a.waited += elapsed

	limit := autoQuestTimeout
	if a.real {
		limit = autoQuestRealTimeout
	}

	if a.waited > limit {
		h.Infof("AUTOQUEST step %q timed out", st.desc)

		a.failed++
		a.checks++
		a.idx = len(a.steps) - 1 // jump to finish
		a.waited = 0
	}
}

// ---- helpers used by the stages ----

func (a *autoQuest) quest(h autoHost, id int) *d2quest.Quest { return h.Q().Quest(id) }

func (a *autoQuest) bit(h autoHost, id, bit int) bool {
	q := a.quest(h, id)

	return h.Q().Rec.Get(q.Slot, bit)
}

// talk plays the Talk row for an NPC class until nothing spoken is left and
// returns the voiced message ids.
func (a *autoQuest) talk(h autoHost, class int) []int {
	var all []int

	for i := 0; i < 12; i++ {
		msgs := h.Talk(class)
		if len(msgs) == 0 {
			break
		}

		all = append(all, msgs...)
	}

	return all
}

// topics returns the message ids of the topics an NPC offers.
func (a *autoQuest) topics(h autoHost, class int) []int {
	var out []int

	for _, t := range h.Q().Activate(class).Topics() {
		out = append(out, t.Msg)
	}

	return out
}

func contains(list []int, n int) bool {
	for _, x := range list {
		if x == n {
			return true
		}
	}

	return false
}

func (a *autoQuest) state(h autoHost, id int, why string) {
	h.Infof("AUTOQUEST state [%s] %s", why, h.Q().Describe(h.Q().Quest(id)))
}

// move reports an area change to the quest system.
func (a *autoQuest) move(h autoHost, to int) { h.Move(to) }

func (a *autoQuest) tick(h autoHost, frames int) {
	h.Tick(frames)
}

func (a *autoQuest) kill(h autoHost, e d2quest.Event) {
	e.Kind = d2quest.EvMonsterKilled
	e.Level = h.Area()
	h.Dispatch(e)
}

func (a *autoQuest) object(h autoHost, id int) {
	h.Dispatch(d2quest.Event{Kind: d2quest.EvObjectOperated, Object: id, Level: h.Area()})
}

func (a *autoQuest) pickup(h autoHost, code string) {
	h.Dispatch(d2quest.Event{Kind: d2quest.EvItemPickedUp, Item: code})
}

// ---- stages ----

func (a *autoQuest) addStage(name string) {
	switch name {
	case "den":
		a.stageDen()
	case "burial":
		a.stageBurial()
	case "tools":
		a.stageTools()
	case "cain":
		a.stageCain()
	case "tower":
		a.stageTower()
	case "andariel":
		a.stageAndariel()
	case "radament":
		a.stageRadament()
	default:
		a.addActStage(name)
	}
}

func (a *autoQuest) stageDen() {
	const id = d2quest.QuestDenOfEvil

	a.once("den: fresh quest", func(h autoHost, a *autoQuest) {
		a.state(h, id, "fresh")
		a.expect(h, "den starts available (state 1, no bits)", a.quest(h, id).State == 1 && h.Q().Rec.Slot(1) == 0)
		a.skillBefore = h.SkillPoints()
	})
	a.once("den: Akara hands out the quest", func(h autoHost, a *autoQuest) {
		msgs := a.talk(h, d2quest.NPCAkara)
		a.state(h, id, "after Akara")
		a.expect(h, "Akara says her first-meeting line and message 64", contains(msgs, 64) && a.quest(h, id).State == 2)
		a.expect(h, "STARTED bit set", a.bit(h, id, d2quest.FlagStarted))
		a.expect(h, "quest intro flag recorded in the NPC block", h.Q().QuestIntroDone(d2quest.NPCAkara))
	})
	a.once("den: everyone comments", func(h autoHost, a *autoQuest) {
		ok := true
		for npc, msg := range map[int]int{d2quest.NPCKashya: 66, d2quest.NPCCharsi: 67, d2quest.NPCGheed: 69, d2quest.NPCWarriv1: 70} {
			ok = ok && contains(a.topics(h, npc), msg)
		}

		a.expect(h, "Kashya, Charsi, Gheed and Warriv have a topic (66,67,69,70)", ok)
	})
	a.once("den: leave town", func(h autoHost, a *autoQuest) {
		a.move(h, 2)
		a.state(h, id, "left town")
		a.expect(h, "LEAVETOWN bit set", a.bit(h, id, d2quest.FlagLeaveTown) && a.quest(h, id).State == 3)
	})
	a.once("den: enter the den", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelDenOfEvil)
		a.state(h, id, "entered the den")
		a.expect(h, "ENTERAREA bit set, log page 2", a.bit(h, id, d2quest.FlagEnterArea) && a.quest(h, id).LastState == 2)

		if a.real {
			a.spawnFoes(h)
		} else {
			h.Q().SetDenMonsters(autoQuestRealFoes + 3)
		}
	})
	a.add("den: kill every monster of the den", func(h autoHost, a *autoQuest) bool {
		if !a.real {
			for i := 0; i < autoQuestRealFoes+3; i++ {
				a.kill(h, d2quest.Event{Monster: 3})
			}

			return true
		}

		if h.FoesAlive() > 0 {
			h.Fight()
			return false
		}

		return true
	})
	a.once("den: cleared", func(h autoHost, a *autoQuest) {
		a.state(h, id, "den cleared")
		a.expect(h, "reward pending + primary goal, state 4", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal) && a.quest(h, id).State == 4)
		a.expect(h, "Sisters' Burial Grounds is not available yet", a.quest(h, d2quest.QuestBurial).State == 0)
		a.tick(h, 8)
		a.expect(h, "log page 5 after the timer", a.quest(h, id).LastState == 5)
		a.move(h, 2)
		a.move(h, d2quest.LevelRogueEncampment)
	})
	a.once("den: Akara's reward", func(h autoHost, a *autoQuest) {
		a.expect(h, "Akara shows the quest marker", h.Q().Marker(d2quest.NPCAkara))

		msgs := a.talk(h, d2quest.NPCAkara)
		a.state(h, id, "reward claimed")
		a.expect(h, "Akara says message 76", contains(msgs, 76))
		a.expect(h, "reward granted, pending cleared, progress bits cleared", a.bit(h, id, d2quest.FlagRewardGranted) &&
			!a.bit(h, id, d2quest.FlagRewardPending) && h.Q().Rec.Slot(1)&0x0FFC == 0)
		a.expect(h, "one skill point granted", h.SkillPoints() == a.skillBefore+1)
		a.expect(h, "Akara's respec flag (slot 41 bits 1 and 13)", h.Q().Rec.AkaraRespecAvailable())
		a.expect(h, "Sisters' Burial Grounds is now available", a.quest(h, d2quest.QuestBurial).State == 1)
		a.expect(h, "Charsi, Gheed and Warriv have the success topic (78,79,80)", contains(a.topics(h, d2quest.NPCCharsi), 78) &&
			contains(a.topics(h, d2quest.NPCGheed), 79) && contains(a.topics(h, d2quest.NPCWarriv1), 80))
	})
}

// spawnFoes puts a few real monsters next to the hero (OD2_AUTOQUEST_REAL=1).
func (a *autoQuest) spawnFoes(h autoHost) { h.SpawnFoes(autoQuestRealFoes) }

func (a *autoQuest) stageBurial() {
	const id = d2quest.QuestBurial

	a.once("burial: Kashya asks for Blood Raven's head", func(h autoHost, a *autoQuest) {
		a.expect(h, "Sisters' Burial Grounds is available", a.quest(h, id).State == 1)
		a.expect(h, "Kashya shows the quest marker", h.Q().Marker(d2quest.NPCKashya))

		msgs := a.talk(h, d2quest.NPCKashya)
		a.state(h, id, "after Kashya")
		a.expect(h, "Kashya says message 81", contains(msgs, 81) && a.quest(h, id).State == 2)
		a.expect(h, "STARTED bit set", a.bit(h, id, d2quest.FlagStarted))
	})
	a.once("burial: travel", func(h autoHost, a *autoQuest) {
		a.move(h, 2)
		a.expect(h, "LEAVETOWN bit set", a.bit(h, id, d2quest.FlagLeaveTown))
		a.move(h, d2quest.LevelBurialGrounds)
		a.state(h, id, "entered the burial grounds")
		a.expect(h, "ENTERAREA bit set, state 3", a.bit(h, id, d2quest.FlagEnterArea) && a.quest(h, id).State == 3)
	})
	a.once("burial: Blood Raven dies", func(h autoHost, a *autoQuest) {
		a.kill(h, d2quest.Event{Monster: d2quest.NPCBloodRaven})
		a.state(h, id, "Blood Raven dead")
		a.expect(h, "reward pending + primary goal, state 4", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal) && a.quest(h, id).State == 4)
		a.tick(h, 15)
		a.expect(h, "log page 3 after the timer", a.quest(h, id).LastState == 3)
		a.move(h, 2)
		a.move(h, d2quest.LevelRogueEncampment)
	})
	a.once("burial: Kashya's reward", func(h autoHost, a *autoQuest) {
		msgs := a.talk(h, d2quest.NPCKashya)
		a.state(h, id, "reward claimed")
		a.expect(h, "Kashya says message 92", contains(msgs, 92))
		a.expect(h, "reward granted, state 5", a.bit(h, id, d2quest.FlagRewardGranted) && !a.bit(h, id, d2quest.FlagRewardPending) &&
			a.quest(h, id).State == 5)
		a.expect(h, "Kashya's rogues are hirable", h.RogueHire())
		a.expect(h, "The Search for Cain is now available", a.quest(h, d2quest.QuestCain).State == 1)
	})
}

func (a *autoQuest) stageTools() {
	const id = d2quest.QuestTools

	a.once("tools: Charsi explains the Malus", func(h autoHost, a *autoQuest) {
		a.expect(h, "Tools of the Trade is available", a.quest(h, id).State == 1)

		msgs := a.talk(h, d2quest.NPCCharsi)
		a.state(h, id, "after Charsi")
		a.expect(h, "Charsi says message 146", contains(msgs, 146) && a.quest(h, id).State == 2)
		a.move(h, 3)
		a.expect(h, "LEAVETOWN bit set, state 3", a.bit(h, id, d2quest.FlagLeaveTown) && a.quest(h, id).State == 3)
	})
	a.once("tools: the chest", func(h autoHost, a *autoQuest) {
		a.expect(h, "hero is level 8 or more (needed to open the chest)", h.HeroLevel() >= 8)
		a.object(h, d2quest.ObjectHoradricMalus)
		a.state(h, id, "chest opened")
		a.expect(h, "state 4 after the chest opened", a.quest(h, id).State == 4)
		a.pickup(h, d2quest.ItemHoradricMalus)
		a.expect(h, "CUSTOM2 (first pickup) set, log page 2", a.bit(h, id, d2quest.FlagCustom2) && a.quest(h, id).LastState == 2)
		a.move(h, d2quest.LevelRogueEncampment)
	})
	a.once("tools: hand it in", func(h autoHost, a *autoQuest) {
		a.expect(h, "Charsi shows the quest marker", h.Q().Marker(d2quest.NPCCharsi))

		msgs := a.talk(h, d2quest.NPCCharsi)
		a.state(h, id, "malus handed in")
		a.expect(h, "Charsi says message 163", contains(msgs, 163))
		a.expect(h, "reward pending + primary goal, state 5", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal) && a.quest(h, id).State == 5)
		a.expect(h, "imbue is available", h.ImbuePending())
		a.tick(h, 20)
		a.expect(h, "Sisters to the Slaughter becomes available", a.quest(h, d2quest.QuestAndariel).State == 1 ||
			a.bit(h, d2quest.QuestAndariel, d2quest.FlagRewardGranted))

		h.Apply(h.Q().ClaimImbue())
		a.state(h, id, "imbued")
		a.expect(h, "reward granted after the imbue, quest inert", a.bit(h, id, d2quest.FlagRewardGranted) &&
			!a.bit(h, id, d2quest.FlagRewardPending) && !a.quest(h, id).Active)
	})
}

func (a *autoQuest) stageCain() {
	const id = d2quest.QuestCain

	a.once("cain: Akara hands out the scroll quest", func(h autoHost, a *autoQuest) {
		a.expect(h, "The Search for Cain is available", a.quest(h, id).State == 1)

		msgs := a.talk(h, d2quest.NPCAkara)
		a.state(h, id, "after Akara")
		a.expect(h, "Akara says message 97", contains(msgs, 97) && a.quest(h, id).State == 2)
		a.expect(h, "STARTED bit set", a.bit(h, id, d2quest.FlagStarted))
		a.move(h, 5)
	})
	a.once("cain: the Inifuss tree", func(h autoHost, a *autoQuest) {
		a.object(h, d2quest.ObjectInifussTree)
		a.pickup(h, d2quest.ItemScrollOfInifuss)
		a.state(h, id, "scroll found")
		a.expect(h, "state 4 and LEAVETOWN set after the scroll dropped", a.quest(h, id).State == 4 &&
			a.bit(h, id, d2quest.FlagLeaveTown))
		a.move(h, d2quest.LevelRogueEncampment)
	})
	a.once("cain: Akara deciphers it", func(h autoHost, a *autoQuest) {
		msgs := a.talk(h, d2quest.NPCAkara)
		a.expect(h, "Akara says message 112 (instructions)", contains(msgs, 112))
		a.pickup(h, d2quest.ItemDecipheredScroll)
		a.state(h, id, "deciphered")
		a.expect(h, "state 5 with the deciphered scroll", a.quest(h, id).State == 5 &&
			h.Q().Items[d2quest.ItemScrollOfInifuss] == 0 && h.Q().Items[d2quest.ItemDecipheredScroll] == 1)
		a.move(h, 5)
	})
	a.once("cain: the cairn stones", func(h autoHost, a *autoQuest) {
		order := h.Q().StoneOrder()
		h.Infof("AUTOQUEST stone order %v", order)

		wrong := order[1]
		a.object(h, wrong)
		a.expect(h, "a stone out of order is ignored", !a.bit(h, id, d2quest.FlagEnterArea))

		for _, s := range order {
			a.object(h, s)
		}

		a.state(h, id, "stones solved")
		a.expect(h, "ENTERAREA bit set, deciphered scroll consumed", a.bit(h, id, d2quest.FlagEnterArea) &&
			h.Q().Items[d2quest.ItemDecipheredScroll] == 0)
		a.move(h, d2quest.LevelTristram)
	})
	a.once("cain: free Cain", func(h autoHost, a *autoQuest) {
		a.object(h, d2quest.ObjectCainGibbet)
		a.state(h, id, "Cain freed")
		a.expect(h, "reward pending + primary goal", a.bit(h, id, d2quest.FlagRewardPending) && a.bit(h, id, d2quest.FlagPrimaryGoal))
		a.move(h, d2quest.LevelRogueEncampment)
	})
	a.once("cain: Akara's ring", func(h autoHost, a *autoQuest) {
		msgs := a.talk(h, d2quest.NPCAkara)
		a.state(h, id, "reward claimed")
		a.expect(h, "Akara says message 118", contains(msgs, 118))
		a.expect(h, "reward granted, state 6", a.bit(h, id, d2quest.FlagRewardGranted) && a.quest(h, id).State == 6)
		a.expect(h, "Tools of the Trade is now available", a.quest(h, d2quest.QuestTools).State == 1 ||
			a.bit(h, d2quest.QuestTools, d2quest.FlagRewardGranted))
	})
}

func (a *autoQuest) stageTower() {
	const id = d2quest.QuestTower

	a.once("tower: walk in", func(h autoHost, a *autoQuest) {
		a.move(h, 6)
		a.move(h, d2quest.LevelForgottenTower)
		a.state(h, id, "entered the tower")
		a.expect(h, "STARTED bit set, state 2", a.bit(h, id, d2quest.FlagStarted) && a.quest(h, id).State == 2)
		a.move(h, d2quest.LevelTowerCellar5)
		a.state(h, id, "entered the cellar")
		a.expect(h, "ENTERAREA bit set, state 3", a.bit(h, id, d2quest.FlagEnterArea) && a.quest(h, id).State == 3)
	})
	a.once("tower: the Countess dies", func(h autoHost, a *autoQuest) {
		a.kill(h, d2quest.Event{Monster: 45, Super: "The Countess"})
		a.state(h, id, "Countess dead")
		a.expect(h, "reward granted + primary goal at once, state 5", a.bit(h, id, d2quest.FlagRewardGranted) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal) && !a.bit(h, id, d2quest.FlagRewardPending) && a.quest(h, id).State == 5)
		a.move(h, d2quest.LevelRogueEncampment)
	})
	a.once("tower: the town congratulates", func(h autoHost, a *autoQuest) {
		a.expect(h, "Akara shows the quest marker, Gheed does not", h.Q().Marker(d2quest.NPCAkara) &&
			!h.Q().Marker(d2quest.NPCGheed))

		msgs := a.talk(h, d2quest.NPCKashya)
		a.expect(h, "Kashya says message 140", contains(msgs, 140))
		a.expect(h, "Tools of the Trade is available after the congratulations", a.quest(h, d2quest.QuestTools).State >= 1 ||
			a.bit(h, d2quest.QuestTools, d2quest.FlagRewardGranted))
	})
}

func (a *autoQuest) stageAndariel() {
	const id = d2quest.QuestAndariel

	a.once("andariel: Cain sends you to the Catacombs", func(h autoHost, a *autoQuest) {
		a.tick(h, 25)
		a.expect(h, "Sisters to the Slaughter is available", a.quest(h, id).State == 1)

		msgs := a.talk(h, d2quest.NPCCain5)
		a.state(h, id, "after Cain")
		a.expect(h, "Cain says message 166", contains(msgs, 166) && a.quest(h, id).State == 2)
		a.expect(h, "STARTED bit set", a.bit(h, id, d2quest.FlagStarted))
	})
	a.once("andariel: down the Catacombs", func(h autoHost, a *autoQuest) {
		for _, lvl := range []int{d2quest.LevelCatacombs1, 35, 36, d2quest.LevelCatacombs4} {
			a.move(h, lvl)
		}

		a.state(h, id, "entered Catacombs 4")
		a.expect(h, "ENTERAREA bit set, state 3", a.bit(h, id, d2quest.FlagEnterArea) && a.quest(h, id).State == 3)
	})
	a.once("andariel: Andariel dies", func(h autoHost, a *autoQuest) {
		a.kill(h, d2quest.Event{Monster: d2quest.NPCAndariel})
		a.state(h, id, "Andariel dead")
		a.expect(h, "reward pending + primary goal, state 4", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal) && a.quest(h, id).State == 4)
		a.tick(h, 12)
		a.expect(h, "log page 3 after the timers", a.quest(h, id).LastState == 3)
		a.move(h, d2quest.LevelRogueEncampment)
	})
	a.once("andariel: Warriv's thanks and the trip east", func(h autoHost, a *autoQuest) {
		ok := true
		for _, npc := range []int{d2quest.NPCCain5, d2quest.NPCAkara, d2quest.NPCKashya, d2quest.NPCWarriv1} {
			ok = ok && h.Q().Marker(npc)
		}

		a.expect(h, "Cain, Akara, Kashya and Warriv show the quest marker", ok)

		msgs := a.talk(h, d2quest.NPCWarriv1)
		a.state(h, id, "reward claimed")
		a.expect(h, "Warriv says message 183", contains(msgs, 183))
		a.expect(h, "reward granted, state 5", a.bit(h, id, d2quest.FlagRewardGranted) && a.quest(h, id).State == 5)

		h.Apply(h.Q().TravelToAct2())
		a.expect(h, "Act 1 finished word (slot 7) set", h.Q().Rec.ActFinished(1))
		a.expect(h, "Act 1 NPCs get the welcome-back flag", h.Q().ReturnGreetingPending(d2quest.NPCAkara))
	})
}

func (a *autoQuest) stageRadament() {
	const id = d2quest.QuestRadament

	a.once("radament: Atma", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelLutGholein)
		a.expect(h, "Radament's Lair starts available", a.quest(h, id).State == 1)

		a.skillBefore = h.SkillPoints()
		msgs := a.talk(h, d2quest.NPCAtma)
		a.state(h, id, "after Atma")
		a.expect(h, "Atma says message 304", contains(msgs, 304) && a.quest(h, id).State == 2)
		a.move(h, 41)
		a.expect(h, "state 3 after leaving Lut Gholein", a.quest(h, id).State == 3)
	})
	a.once("radament: Radament dies", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelSewers3)
		a.kill(h, d2quest.Event{Monster: d2quest.NPCRadament})
		a.state(h, id, "Radament dead")
		a.expect(h, "reward pending + goal + Book of Skill bit", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal) && a.bit(h, id, d2quest.FlagCustom1) && a.quest(h, id).State == 4)
		a.move(h, d2quest.LevelLutGholein)
	})
	a.once("radament: Atma's thanks and the book", func(h autoHost, a *autoQuest) {
		msgs := a.talk(h, d2quest.NPCAtma)
		a.state(h, id, "reward claimed")
		a.expect(h, "Atma says message 334", contains(msgs, 334))
		a.expect(h, "reward granted", a.bit(h, id, d2quest.FlagRewardGranted))
		h.Apply(h.Q().ReadBookOfSkill())
		a.expect(h, "reading the Book of Skill gives one skill point", h.SkillPoints() == a.skillBefore+1)
	})
}
