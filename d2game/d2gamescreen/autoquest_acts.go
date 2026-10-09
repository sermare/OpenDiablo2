package d2gamescreen

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
)

// The Acts 2-5 stages of OD2_AUTOQUEST. They use the same host as the Act 1
// stages: NPC talks play the Talk row, kills/objects/items/area changes are
// injected, and every stage checks the record bits, the quest state and the
// effects. OD2_AUTOQUEST=act2|act3|act4|act5|acts2-5 or a single stage
// (staff, sun, arcane, summoner, tombs, lamesen, khalim, blade, bird, temple,
// guardian, fallen, terror, hellforge, siege, rescue, prison, betrayal, rite,
// eve). Each stage marks the quests it depends on as done in the record, so it
// can run alone. The Acts 3-5 triggers are UNVERIFIED (see d2quest/generic.go).

//nolint:gochecknoglobals // static lookup data
var actStageOrder = map[string][]string{
	"act2":    {"radament", "staff", "sun", "arcane", "summoner", "tombs"},
	"act3":    {"lamesen", "khalim", "blade", "bird", "temple", "guardian"},
	"act4":    {"fallen", "terror", "hellforge"},
	"act5":    {"siege", "rescue", "prison", "betrayal", "rite", "eve"},
	"acts2-5": {"sun", "arcane", "tombs", "lamesen", "blade", "fallen", "terror", "siege", "rite"},
}

// actStageSlots lists the record slots a stage resets before it runs and the
// quests (by slot) that must be done for it to start.
//
//nolint:gochecknoglobals // static lookup data
var actStageSlots = map[string][2][]int{
	"staff":     {{10}, nil},
	"sun":       {{11}, nil},
	"arcane":    {{12}, {11}},
	"summoner":  {{13, 12}, nil},
	"tombs":     {{14}, {9}},
	"lamesen":   {{17}, nil},
	"khalim":    {{18}, nil},
	"blade":     {{19}, nil},
	"bird":      {{20}, nil},
	"temple":    {{21}, {19}},
	"guardian":  {{22}, {19, 21}},
	"fallen":    {{25}, nil},
	"terror":    {{26}, nil},
	"hellforge": {{27}, nil},
	"siege":     {{35}, nil},
	"rescue":    {{36}, nil},
	"prison":    {{37}, nil},
	"betrayal":  {{38}, nil},
	"rite":      {{39}, nil},
	"eve":       {{40}, {39}},
}

func init() {
	for name := range actStageSlots {
		autoQuestNames[name] = name
	}

	for name := range actStageOrder {
		autoQuestNames[name] = name
	}

	autoQuestNames["acts25"] = "acts2-5" // the spec is normalised without punctuation
}

// says talks to an NPC until it has nothing spoken left and checks the
// messages it voiced.
func (a *autoQuest) says(h autoHost, why string, npc int, want ...int) []int {
	msgs := a.talk(h, npc)
	ok := true

	for _, m := range want {
		ok = ok && contains(msgs, m)
	}

	a.expect(h, fmt.Sprintf("%s (npc %d says %v)", why, npc, want), ok)

	return msgs
}

// offer plays the quest giver's first line, unless an earlier talk of the
// script already voiced it (the Talk row plays every line an NPC has, so the
// next quest of a chain is often offered in the same talk that closed the last).
func (a *autoQuest) offer(h autoHost, id, npc, msg int) {
	q := a.quest(h, id)
	a.expect(h, q.Label+" is available", q.State >= 1)

	if q.State == 1 {
		a.says(h, q.Label+" offer", npc, msg)
	}

	a.expect(h, q.Label+" STARTED bit set, state 2 or later", a.bit(h, id, d2quest.FlagStarted) && q.State >= 2)
}

func (a *autoQuest) hasEffect(h autoHost, eff []d2quest.Effect, kind d2quest.EffectKind, code string) bool {
	for _, e := range eff {
		if e.Kind == kind && (code == "" || e.Code == code) {
			return true
		}
	}

	return false
}

// bitsOf reports whether all bits of a quest's record slot are set.
func (a *autoQuest) bitsOf(h autoHost, id int, bits ...int) bool {
	for _, b := range bits {
		if !a.bit(h, id, b) {
			return false
		}
	}

	return true
}

func (a *autoQuest) toTown(h autoHost, town int) {
	h.Move(town)
}

func (a *autoQuest) addActStage(name string) {
	switch name {
	case "staff":
		a.stageStaff()
	case "sun":
		a.stageSun()
	case "arcane":
		a.stageArcane()
	case "summoner":
		a.stageSummoner()
	case "tombs":
		a.stageTombs()
	case "lamesen":
		a.stageLamEsen()
	case "khalim":
		a.stageKhalim()
	case "blade":
		a.stageBlade()
	case "bird":
		a.stageBird()
	case "temple":
		a.stageTemple()
	case "guardian":
		a.stageGuardian()
	case "fallen":
		a.stageFallen()
	case "terror":
		a.stageTerror()
	case "hellforge":
		a.stageHellforge()
	case "siege":
		a.stageSimple(d2quest.QuestSiege, d2quest.LevelHarrogath, d2quest.NPCLarzuk, 20077, d2quest.LevelBloodyFoothills,
			[]d2quest.Event{{Monster: 999, Super: "Shenk the Overseer"}}, d2quest.NPCLarzuk, "socket-quest")
	case "rescue":
		a.stageSimple(d2quest.QuestRescue, d2quest.LevelHarrogath, d2quest.NPCQualKehk, 20096, d2quest.LevelFrigidHighlands,
			[]d2quest.Event{{Monster: d2quest.NPCPrisonDoor}, {Monster: d2quest.NPCPrisonDoor}, {Monster: d2quest.NPCPrisonDoor}},
			d2quest.NPCQualKehk, "hire-barbarians")
	case "prison":
		a.stagePrison()
	case "betrayal":
		a.stageSimple(d2quest.QuestBetrayal, d2quest.LevelHarrogath, d2quest.NPCDrehya, 20137, d2quest.LevelNihlathakTemple,
			[]d2quest.Event{{Monster: d2quest.NPCNihlathakBoss}}, d2quest.NPCDrehya, "personalize")
	case "rite":
		a.stageSimple(d2quest.QuestRite, d2quest.LevelHarrogath, d2quest.NPCQualKehk, 20153, d2quest.LevelArreatSummit,
			[]d2quest.Event{{Monster: d2quest.NPCAncient1}, {Monster: d2quest.NPCAncient2}, {Monster: d2quest.NPCAncient3}},
			d2quest.NPCMalah, "")
	case "eve":
		a.stageEve()
	}
}

// stageSimple scripts a giver / area / kills / claim quest of the common skeleton.
func (a *autoQuest) stageSimple(id, town, giver, introMsg, area int, kills []d2quest.Event, claimer int, rewardName string) {
	label := ""

	a.once("simple: the giver offers the quest", func(h autoHost, a *autoQuest) {
		a.move(h, town)
		label = a.quest(h, id).Label
		a.offer(h, id, giver, introMsg)
		a.state(h, id, "after the giver")
	})
	a.once("simple: go to the area", func(h autoHost, a *autoQuest) {
		a.move(h, area)
		a.state(h, id, "entered the area")
		a.expect(h, label+" ENTERAREA bit set, state 3", a.bit(h, id, d2quest.FlagEnterArea) && a.quest(h, id).State == 3)
	})
	a.once("simple: the goal", func(h autoHost, a *autoQuest) {
		for _, k := range kills {
			a.kill(h, k)
		}

		a.state(h, id, "goal reached")
		a.expect(h, label+" reward pending + primary goal", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal))
		a.move(h, town)
	})
	a.once("simple: the reward", func(h autoHost, a *autoQuest) {
		a.expect(h, label+" shows the quest marker", h.Q().Marker(claimer))
		a.says(h, label+" reward line", claimer)
		a.state(h, id, "reward claimed")
		a.expect(h, label+" reward granted, pending cleared", a.bit(h, id, d2quest.FlagRewardGranted) &&
			!a.bit(h, id, d2quest.FlagRewardPending))
	})

	_ = rewardName
}

func (a *autoQuest) stageStaff() {
	const id = d2quest.QuestHoradricStaff

	a.once("staff: scroll, amulet, staff, cube reported to Cain", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelLutGholein)

		for _, c := range []struct {
			obj  int
			item string
			msg  int
		}{
			{d2quest.ObjectScrollChest, d2quest.ItemHoradricScroll, 335},
			{0, d2quest.ItemViperAmulet, 336},
			{d2quest.ObjectStaffChest, d2quest.ItemStaffOfKingsShaft, 337},
			{d2quest.ObjectCubeChest, d2quest.ItemHoradricCube, 338},
		} {
			if c.obj != 0 {
				a.object(h, c.obj)
			}

			a.pickup(h, c.item)
			a.expect(h, "Cain marks "+c.item, h.Q().Marker(d2quest.NPCCain2))
			a.says(h, "Cain reports "+c.item, d2quest.NPCCain2, c.msg)
		}

		a.state(h, id, "all four reported")
		a.expect(h, "the staff quest has LEAVETOWN, ENTERAREA, CUSTOM1, CUSTOM2",
			a.bitsOf(h, id, d2quest.FlagLeaveTown, d2quest.FlagEnterArea, d2quest.FlagCustom1, d2quest.FlagCustom2))
	})
	a.once("staff: cube it and place it", func(h autoHost, a *autoQuest) {
		a.pickup(h, d2quest.ItemHoradricStaff)
		a.says(h, "Cain's final staff line", d2quest.NPCCain2, 339)
		a.expect(h, "CUSTOM6 set by Cain's final line", a.bit(h, id, d2quest.FlagCustom6))
		a.move(h, 66)
		a.object(h, d2quest.ObjectOrifice)
		a.state(h, id, "staff placed")
		a.expect(h, "staff quest done: RG + PGD", a.bitsOf(h, id, d2quest.FlagRewardGranted, d2quest.FlagPrimaryGoal) &&
			h.Q().Items[d2quest.ItemHoradricStaff] == 0)
	})
}
