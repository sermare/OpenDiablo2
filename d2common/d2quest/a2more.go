package d2quest

// Objects of Act 2 (objects.txt rows; the chests and the altar are from
// quests-2.md section 3, the others were read from objects.txt by name and are
// UNVERIFIED as quest objects).
const (
	ObjectTaintedSunAltar = 149
	ObjectOrifice         = 152 // Tal Rasha's staff orifice
	ObjectCubeChest       = 354
	ObjectScrollChest     = 355
	ObjectStaffChest      = 356
	ObjectHorazonJournal  = 357 // "Tome" of the Arcane Sanctuary
)

// flagAssembled is CUSTOM7 (bit 11) of the Horadric Staff slot: the cube made
// the staff.
const flagAssembled = FlagCustom1 + 6

func (g *Game) dropItem(code string) {
	if g.Items[code] > 0 {
		g.Items[code]--
	}
}

// ---- A2Q2 The Horadric Staff - slot 10 (Cain only) [SRC, quests-2.md] ----

type staffData struct{ scroll bool }

func newHoradricStaff() *Quest {
	d := &staffData{}
	q := &Quest{ID: QuestHoradricStaff, Slot: 10, Act: 1, Name: "The Horadric Staff", Label: "A2Q2", LogIndex: 2,
		Active: true, NotIntro: true, NoSetState: true, SeqID: -1, tables: speechTables("A2Q2"), data: d}

	// the reports Cain takes in order: scroll, amulet, staff, cube. Each sets
	// LEAVETOWN as well, so the scroll report is tracked on the side.
	type report struct {
		item string
		bit  int
		msg  int
		tbl  int // line when the hero carries the item and Cain has not heard of it
		topc int // topic once reported
	}

	reports := []report{
		{ItemHoradricScroll, FlagLeaveTown, 335, 0, 6},
		{ItemViperAmulet, FlagEnterArea, 336, 1, 7},
		{ItemStaffOfKingsShaft, FlagCustom1, 337, 2, 8},
		{ItemHoradricCube, FlagCustom2, 338, 3, 9},
	}

	reported := func(g *Game, q *Quest, i int) bool {
		if i == 0 {
			return d.scroll
		}

		return g.get(q, reports[i].bit)
	}

	assembled := func(g *Game, q *Quest) bool {
		return g.get(q, flagAssembled) || g.hasItem(ItemHoradricStaff)
	}

	// pending lists the reports Cain still wants to hear
	pending := func(g *Game, q *Quest) []int {
		var out []int

		for i, r := range reports {
			if g.hasItem(r.item) && !reported(g, q, i) {
				out = append(out, i)
			}
		}

		return out
	}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		if npc != NPCCain2 {
			return nil
		}

		switch {
		case g.get(q, FlagRewardGranted):
			return q.pick(npc, 5)
		case g.get(q, FlagRewardPending) || assembled(g, q):
			return q.pick(npc, 4)
		}

		var out []Speech

		if p := pending(g, q); len(p) > 0 {
			out = append(out, q.pick(npc, reports[p[0]].tbl)...)
		}

		for i, r := range reports {
			if reported(g, q, i) {
				out = append(out, q.pick(npc, r.topc)...)
			}
		}

		return out
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		return npc == NPCCain2 && !g.get(q, FlagRewardGranted) &&
			(g.get(q, FlagRewardPending) || assembled(g, q) || len(pending(g, q)) > 0)
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCCain2 {
			return
		}

		for i, r := range reports {
			if e.Msg != r.msg {
				continue
			}

			if i == 0 {
				d.scroll = true

				g.dropItem(ItemHoradricScroll)
				g.emit(Effect{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemHoradricScroll, Note: "the scroll is deciphered"})
			}

			g.set(q, FlagLeaveTown, "Cain reported "+r.item)
			g.set(q, r.bit, "Cain reported "+r.item)
			g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: g.LogPage(q)})

			return
		}

		if e.Msg == 339 {
			g.clear(q, FlagRewardPending, "Cain: the staff is assembled")

			for _, b := range []int{FlagLeaveTown, FlagCustom6, FlagEnterArea, FlagCustom2, FlagCustom1} {
				g.set(q, b, "Cain: the staff is assembled")
			}

			g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: g.LogPage(q)})
		}
	}

	q.on[EvObjectOperated] = func(g *Game, q *Quest, e *Event) {
		if g.get(q, FlagRewardGranted) {
			return
		}

		code := ""

		switch e.Object {
		case ObjectCubeChest:
			code = ItemHoradricCube
		case ObjectScrollChest:
			code = ItemHoradricScroll
		case ObjectStaffChest:
			code = ItemStaffOfKingsShaft
		}

		if code != "" && !g.hasItem(code) && !g.hasItem(ItemHoradricStaff) {
			g.emit(Effect{Kind: EffectSpawn, Quest: q.ID, Code: code, Note: "quest chest of the Horadric Staff"})
		}

		// Tal Rasha's orifice takes the assembled staff: the quest is done
		if e.Object == ObjectOrifice && g.hasItem(ItemHoradricStaff) && e.Level >= LevelTalRashaFirst &&
			e.Level <= LevelTalRashaLast {
			g.completeStaff(q)
			g.emit(Effect{Kind: EffectPortal, Quest: q.ID, Note: "the portal to Duriel's Lair opens (delay from missile 338, unverified)"})
		}
	}

	q.on[EvItemPickedUp] = func(g *Game, q *Quest, e *Event) {
		// the cube recipe made the staff: the palace opens (Arcane Sanctuary quest)
		if e.Item == ItemHoradricStaff && !g.get(q, flagAssembled) {
			g.set(q, flagAssembled, "Horadric Staff assembled")
			g.chainFrom(q)
		}
	}

	q.status = func(g *Game, q *Quest) int {
		if g.get(q, FlagRewardGranted) {
			return 0
		}

		n := 0

		for i := range reports {
			if reported(g, q, i) {
				n++
			}
		}

		if n == 0 && !g.get(q, FlagStarted) {
			return 0
		}

		return n + 1 // page numbers are UNVERIFIED
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, _ *Event) {
		d.scroll = g.get(q, FlagLeaveTown)

		if g.get(q, FlagRewardGranted) {
			g.globalDone(q)
		}
	}

	return q
}

// completeStaff is the staff placed in the orifice or boarding Meshif: the
// quest is completed and the staff items are deleted.
func (g *Game) completeStaff(q *Quest) {
	if g.get(q, FlagRewardGranted) {
		return
	}

	g.set(q, FlagRewardGranted, "staff placed")
	g.set(q, FlagPrimaryGoal, "staff placed")
	g.globalDone(q)

	for _, c := range []string{ItemViperAmulet, ItemStaffOfKingsShaft, ItemHoradricStaff} {
		if g.hasItem(c) {
			g.dropItem(c)
			g.emit(Effect{Kind: EffectDeleteItem, Quest: q.ID, Code: c, Note: "staff items are used up"})
		}
	}

	g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 13})
}

// ---- A2Q3 The Tainted Sun - slot 11 (Drognan) ----

func newTaintedSun() *Quest {
	viper := func(g *Game, q *Quest) []Effect {
		staff := g.byID[QuestHoradricStaff]
		if g.hasItem(ItemViperAmulet) || g.hasItem(ItemHoradricStaff) || g.get(staff, FlagRewardGranted) {
			return nil
		}

		return []Effect{{Kind: EffectSpawn, Quest: q.ID, Code: ItemViperAmulet, Note: "the altar drops the Amulet of the Viper"}}
	}

	return newSpecQuest(&spec{
		id: QuestTaintedSun, slot: 11, act: 1, logIndex: 3, name: "The Tainted Sun", label: "A2Q3",
		start: 0, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 2}, rp: 3, done: 4,
		steps: []step{{from: 1, npc: NPCDrognan, msg: 348, to: 2}},
		trigs: []trig{
			// the sun darkens when the hero enters the Lost City or the Valley of Snakes
			{ev: EvAreaChanged, level: LevelLostCity, min: -1, max: -1, to: 1, bit: -1},
			{ev: EvAreaChanged, level: LevelValleySnakes, min: -1, max: -1, to: 1, bit: -1},
			{ev: EvObjectOperated, object: ObjectTaintedSunAltar, goal: true, fx: func(g *Game, q *Quest) []Effect {
				return append(viper(g, q), Effect{Kind: EffectSound, Quest: q.ID, Value: 52, Note: "sound id unverified"})
			}},
		},
	})
}

// ---- A2Q4 The Arcane Sanctuary - slot 12 ----

func newArcaneSanctuary() *Quest {
	return newSpecQuest(&spec{
		id: QuestArcane, slot: 12, act: 1, logIndex: 4, name: "The Arcane Sanctuary", label: "A2Q4",
		start: 0, goal: 5, tbl: map[int]int{1: 0, 2: 1, 3: 2, 4: 3}, rp: 4, done: 5,
		steps: []step{
			{from: 1, npc: NPCDrognan, msg: 373, to: 2},
			{from: 2, npc: NPCJerhyn, msg: 377, to: 3},
		},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelArcane, to: 4, max: 4},
			{ev: EvObjectOperated, object: ObjectHorazonJournal, level: LevelArcane, goal: true},
		},
		noLeaveRule: true,
		// the palace guard (class 331): open once Drognan sent the hero in
		extra: func(g *Game, q *Quest, npc int) []Speech {
			if npc != NPCGuard2 {
				return nil
			}

			if q.State >= 2 {
				return q.pick(npc, 8+g.Rand.Intn(3))
			}

			return q.pick(npc, 7)
		},
	})
}

// ---- A2Q5 The Summoner - slot 13 ----

func newSummoner() *Quest {
	return newSpecQuest(&spec{
		id: QuestSummoner, slot: 13, act: 1, logIndex: 5, name: "The Summoner", label: "A2Q5",
		start: 0, goal: 2, tbl: map[int]int{1: 0}, rp: 1, done: 2,
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelArcane, min: -1, max: -1, to: 1, bit: FlagStarted},
			{ev: EvMonsterKilled, monster: NPCSummoner, level: LevelArcane, goal: true, fx: func(g *Game, q *Quest) []Effect {
				// the Summoner's death also completes the Arcane Sanctuary quest
				g.reachGoal(g.byID[QuestArcane], 5, false)

				return []Effect{{Kind: EffectSound, Quest: q.ID, Value: 51, Note: "sound id unverified"}}
			}},
		},
		noLeaveRule: true,
	})
}

// ---- A2Q6 The Seven Tombs - slot 14 ----

func newSevenTombs() *Quest {
	// the NPCs that comment on the finished quest: message and the bit that
	// remembers it (CUSTOM2..CUSTOM7)
	comment := map[int][2]int{
		NPCAtma: {445, FlagCustom1 + 1}, NPCWarriv2: {446, FlagCustom1 + 2}, NPCDrognan: {449, FlagCustom1 + 3},
		NPCLysander: {444, FlagCustom1 + 4}, NPCCain2: {452, FlagCustom1 + 5}, NPCFara: {447, FlagCustom1 + 6},
	}

	s := &spec{
		id: QuestSevenTombs, slot: 14, act: 1, logIndex: 6, name: "The Seven Tombs", label: "A2Q6",
		start: 0, goal: 5, tbl: map[int]int{1: 0, 2: 1, 3: 2, 4: 3, 5: 4},
		noLeaveRule: true,
		steps: []step{
			{from: 1, npc: NPCJerhyn, msg: 430, to: 2},
			{from: 3, npc: NPCTyrael1, msg: 302, to: 4, nomark: true, fx: func(g *Game, q *Quest) []Effect {
				g.set(q, FlagPrimaryGoal, "Tyrael's message")
				g.set(q, FlagLeaveTown, "Tyrael's message")
				g.globalDone(q)

				return []Effect{{Kind: EffectPortal, Quest: q.ID, Note: "Tyrael opens the portal to Lut Gholein"}}
			}},
			{from: 4, npc: NPCJerhyn, msg: 442, to: 5, nomark: true, fx: func(g *Game, q *Quest) []Effect {
				g.clear(q, FlagLeaveTown, "Jerhyn's thanks")
				g.set(q, FlagEnterArea, "Jerhyn's thanks")

				return nil
			}},
			{from: 5, npc: NPCMeshif1, msg: 450, to: 6, claim: true, nomark: true, fx: func(g *Game, q *Quest) []Effect {
				g.clear(q, FlagEnterArea, "Meshif's ship")
				g.completeStaff(g.byID[QuestHoradricStaff])

				return []Effect{{Kind: EffectUnlockAct, Quest: q.ID, Value: 3}}
			}},
		},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelCanyon, min: 1, max: 1, to: 2},
			{ev: EvAreaChanged, level: LevelDurielLair, min: 1, max: 1, to: 2},
			{ev: EvMonsterKilled, monster: NPCDuriel, min: 1, max: 2, to: 3, bit: FlagCustom1, page: 3},
		},
		extra: func(g *Game, q *Quest, npc int) []Speech {
			var out []Speech

			if g.get(q, FlagEnterArea) {
				out = append(out, q.pick(npc, 5)...)
			}

			if c, ok := comment[npc]; ok && g.get(q, FlagPrimaryGoal) && !g.get(q, c[1]) {
				out = append(out, q.pick(npc, 6)...)
			}

			return out
		},
	}

	q := newSpecQuest(s)

	old := q.on[EvMessageHeard]
	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if old != nil {
			old(g, q, e)
		}

		if c, ok := comment[e.NPC]; ok && e.Msg == c[0] && g.get(q, FlagPrimaryGoal) && q.NotIntro {
			g.set(q, c[1], "comment heard")
		}
	}

	return q
}
