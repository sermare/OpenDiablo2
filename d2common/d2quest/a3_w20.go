package d2quest

// Wave 20 rewrite of three Act 3 quests that the earlier table-driven version got wrong: Khalim's Will (A3Q2),
// The Blade of the Old Religion (A3Q3) and The Blackened Temple (A3Q5).
//
// Evidence. VERIFIED means read in Game.exe 1.14b (read-only Ghidra):
//
//	Compelling Orb (objects.txt OperateFn 53 -> 0x5b9550): needs the weapon in the hero's hand to be "qf2 " (Khalim's
//	  Will); the node of quest id 0x13 keeps a hit counter (+0x38), the first operation does nothing, the second sets
//	  bits 0 (done) and 0xd (primary goal) of slot 0x12 at once (no reward pending, no claim talk), deletes the Will,
//	  and, when slot 0x15 (Blackened Temple) has bit 4 set and bit 0 clear, sets bit 0 of slot 0x15. The orb is the
//	  Blackened Temple node's private flag (+0xc) that SERVER_EnterWarpTile reads for the Durance of Hate stairs.
//	Blade of the Old Religion message handler (0x5b6ea0): Ormus (0xff) message 0x24b (587) with a "g33 " in the
//	  inventory and bit 5 set / bit 8 clear removes the item and sets bit 6; Ormus message 0x251 (593) with bit 6 set
//	  and bit 8 clear sets bit 8 and creates a RARE ring ("rin ") of item level 21 / 35 / 75 (Normal / Nightmare /
//	  Hell); Asheara (0xfc) message 0x24d (589) with bit 7 clear sets bit 7 and hires her Iron Wolves
//	  (HIRE_ServerSpawnQuestRewardMercenary(0xfc)); once bits 7 and 8 are both set (either order) bits 0xd and 0 are set.
//	Blackened Temple kill handler (0x5b97d0): the first Council death drops Khalim's Flail ("qf1 ") for every hero
//	  without one, the second drops the Horadric Cube ("box ") for every hero without one; the last death sets state 6
//	  (or 7 with the orb smashed).
//
// UNVERIFIED in the exe (taken from the clean-room D2MOO sources, which agree with the binary everywhere compared):
// the record bits of the Khalim reports (eye = LEAVETOWN, heart = CUSTOM2, brain = ENTERAREA, flail = CUSTOM1,
// the cubed Will = CUSTOM3), the talk tables chosen per state, the availability triggers (Great Marsh, Flayer
// Jungle, Kurast Causeway), the state numbers of the Blackened Temple, and the log pages.

// Levels of the Act 3 availability triggers (levels.txt: Jungle 2, Jungle 3, Kurast 4).
const (
	LevelGreatMarsh     = 77
	LevelFlayerJungle   = 78
	LevelKurastCauseway = 82
)

// Bits of the Khalim's Will record (D2MOO ACT3Q2_Callback11_ScrollMessage).
const (
	khalimEyeBit   = FlagLeaveTown // message 545 reported the eye
	khalimHeartBit = FlagCustom2   // 544
	khalimBrainBit = FlagEnterArea // 546
	khalimFlailBit = FlagCustom1   // 547
	khalimWillBit  = FlagCustom3   // 548
)

// OrbSmashed reports whether the Compelling Orb of Travincal is broken in this game (Khalim's Will record bit 0).
func (g *Game) OrbSmashed() bool { return g.orbSmashed }

type khalimData struct {
	hits int // the orb's operate counter (node +0x38): the second hit breaks it
}

// khalimItems are the five quest items the temple quest deletes once the orb is broken.
//
//nolint:gochecknoglobals // static lookup data
var khalimItems = []string{ItemKhalimEye, ItemKhalimHeart, ItemKhalimBrain, ItemKhalimFlail, ItemKhalimWill}

// deleteKhalimItems removes the parts and the Will (QUESTS_DeleteItem for qey, qhr, qbr, qf1, qf2).
func (g *Game) deleteKhalimItems(q *Quest) {
	for _, c := range khalimItems {
		if g.hasItem(c) {
			g.dropItem(c)
			g.emit(Effect{Kind: EffectDeleteItem, Quest: q.ID, Code: c, Note: "Khalim's Will quest item"})
		}
	}
}

// A3Q2 Khalim's Will (Cain only). Cain explains the quest (543); every part the hero brings is reported once
// (eye 545, heart 544, brain 546, flail 547); the cubed Will (548) is smashed against the Compelling Orb.
func newKhalim() *Quest {
	d := &khalimData{}
	q := &Quest{ID: QuestKhalim, Slot: 18, Act: 2, Name: "Khalim's Will", Label: "A3Q2", LogIndex: 2,
		Active: true, NotIntro: true, NoSetState: true, SeqID: -1, tables: speechTables("A3Q2"), data: d}

	type part struct {
		item string
		bit  int
		msg  int
		tbl  int // the spoken line while the hero carries the part and Cain has not heard of it
		topc int // the topic
	}

	eye := part{ItemKhalimEye, khalimEyeBit, 545, 1, 7}
	heart := part{ItemKhalimHeart, khalimHeartBit, 544, 2, 8}
	brain := part{ItemKhalimBrain, khalimBrainBit, 546, 3, 9}
	flail := part{ItemKhalimFlail, khalimFlailBit, 547, 4, 10}
	parts := []part{flail, eye, heart, brain} // the order of ACT3Q2_Callback00_NpcActivate

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		if npc != NPCCain3 || g.get(q, FlagRewardGranted) {
			return nil
		}

		if !g.get(q, FlagStarted) {
			if q.State >= 1 {
				return q.pick(npc, 0)
			}

			return nil
		}

		cubed := g.hasItem(ItemKhalimWill)
		if cubed && !g.get(q, khalimWillBit) && !g.orbSmashed {
			return q.pick(npc, 5)
		}

		n := 0

		for _, p := range parts {
			if !g.hasItem(p.item) {
				continue
			}

			n++

			if !g.get(q, p.bit) {
				return q.pick(npc, p.tbl)
			}
		}

		if n == 0 {
			if cubed {
				return q.pick(npc, 11)
			}

			return q.pick(npc, 6)
		}

		for _, p := range []part{flail, heart, eye, brain} {
			if g.hasItem(p.item) {
				return q.pick(npc, p.topc)
			}
		}

		return q.pick(npc, 6)
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		if npc != NPCCain3 || g.get(q, FlagRewardGranted) {
			return false
		}

		if !g.get(q, FlagStarted) {
			return q.State >= 1
		}

		for _, p := range parts {
			if g.hasItem(p.item) && !g.get(q, p.bit) {
				return true
			}
		}

		return g.hasItem(ItemKhalimWill) && !g.get(q, khalimWillBit) && !g.orbSmashed
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCCain3 || g.get(q, FlagRewardGranted) {
			return
		}

		switch e.Msg {
		case 543:
			g.setState(q, 2)
			g.set(q, FlagStarted, "Cain explained Khalim's Will")
			g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: g.LogPage(q)})
		case 545, 544, 546, 547, 548:
			for _, p := range append(parts, part{ItemKhalimWill, khalimWillBit, 548, 5, 11}) {
				if p.msg == e.Msg {
					g.set(q, p.bit, "Cain reported a part of Khalim's Will")
				}
			}

			g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: g.LogPage(q)})
		}
	}

	// VERIFIED 0x5b9550 (OperateFn 53): wielding the Will, the second operation breaks the orb.
	q.on[EvObjectOperated] = func(g *Game, q *Quest, e *Event) {
		if e.Object != ObjectCompellingOrb || g.orbSmashed || e.Wielded != ItemKhalimWill {
			return
		}

		d.hits++
		if d.hits < 2 {
			return
		}

		g.orbSmashed = true
		g.set(q, FlagRewardGranted, "the Compelling Orb is smashed")
		g.set(q, FlagPrimaryGoal, "the Compelling Orb is smashed")
		g.globalDone(q)

		if g.hasItem(ItemKhalimWill) {
			g.dropItem(ItemKhalimWill)
			g.emit(Effect{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemKhalimWill, Note: "the Will breaks the orb"})
		}

		if t := g.byID[QuestBlackenedTemple]; t != nil && g.get(t, FlagEnterArea) && !g.get(t, FlagRewardGranted) {
			g.templeDone(t, "the orb is smashed after the Council died")
		}

		g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 13})
	}

	// ACT3Q2_Callback03_ChangedLevel: the Great Marsh makes the quest available.
	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		if e.NewLevel == LevelGreatMarsh && q.NotIntro && q.State == 0 {
			g.setState(q, 1)
		}
	}

	q.status = func(g *Game, q *Quest) int {
		if g.hasItem(ItemKhalimWill) {
			if g.orbSmashed {
				return 12
			}

			return 6
		}

		have := 0
		for _, p := range parts {
			if g.hasItem(p.item) {
				have++
			}
		}

		switch {
		case have == 4:
			if g.get(q, khalimFlailBit) {
				return 5
			}

			return 7
		case have == 0:
			// VERIFIED 0x5bb390: started, or past state 1 while the quest runs, shows page 1
			if g.get(q, FlagStarted) || (q.NotIntro && q.State > 1) {
				return 1
			}

			return 0
		case !g.hasItem(ItemKhalimEye):
			return 1
		case !g.hasItem(ItemKhalimBrain):
			return 2
		case !g.hasItem(ItemKhalimHeart):
			return 4
		case g.hasItem(ItemKhalimFlail):
			return 7
		}

		return 3
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, _ *Event) {
		if g.get(q, FlagRewardGranted) {
			g.orbSmashed = true
			g.globalDone(q)
		}
	}

	return q
}

// templeDone completes the Blackened Temple (bit 0 of slot 21) and makes the Guardian available.
func (g *Game) templeDone(t *Quest, why string) {
	if g.get(t, FlagRewardGranted) {
		return
	}

	g.set(t, FlagRewardGranted, why)
	g.clear(t, FlagEnterArea, why)
	g.globalDone(t)
	t.LastState = 13
	g.emit(Effect{Kind: EffectLogUpdate, Quest: t.ID, Value: 13})
	g.chainFrom(t)
}

// ---- A3Q3 The Blade of the Old Religion ----

// Bits of the Blade record (D2MOO / 0x5b6ea0).
const (
	bladeCarry   = FlagCustom1 // the hero carries Gidbinn
	bladeBrought = FlagCustom2 // Ormus has the blade (message 587)
	bladeWolves  = FlagCustom3 // Asheara's Iron Wolves were granted (589)
	bladeRing    = FlagCustom4 // Ormus gave the ring (593)
	bladeSeen    = FlagCustom5 // the pick-up sound/mark of the blade
)

type bladeData struct {
	dropped  bool // Gidbinn left the boss (ACT3Q3 bGidbinnDropped)
	boss     bool // the fetish boss was woken (ACT3Q3 bBossHasSpawned)
	hratli   bool
	rewarded bool
}

// ringLevel is the item level of Ormus's ring by difficulty (VERIFIED 0x5b6ea0: 0x15, 0x23, 0x4b).
func ringLevel(difficulty int) int {
	switch difficulty {
	case Nightmare:
		return 35
	case Hell:
		return 75
	}

	return 21
}

func newBlade() *Quest {
	d := &bladeData{}
	q := &Quest{ID: QuestBlade, Slot: 19, Act: 2, Name: "The Blade of the Old Religion", Label: "A3Q3", LogIndex: 3,
		Active: true, NotIntro: true, InitNo: 4, SeqID: QuestBlackenedTemple, tables: speechTables("A3Q3"), data: d}

	q.seq = chainSeq(true, 5)

	// ACT3Q3_UnitIterate_UpdateQuestStateFlags
	flags := func(g *Game, q *Quest) {
		if g.get(q, FlagRewardGranted) || g.get(q, bladeBrought) {
			return
		}

		switch {
		case g.hasItem(ItemGidbinn):
			g.set(q, bladeCarry, "the hero carries Gidbinn")
		case d.dropped:
			g.set(q, FlagEnterArea, "Gidbinn was found")

			if !g.get(q, FlagLeaveTown) {
				g.set(q, FlagStarted, "started")
			}
		case q.State == 2:
			g.set(q, FlagLeaveTown, "left town")
		}
	}

	done := func(g *Game, q *Quest, why string) {
		g.set(q, FlagPrimaryGoal, why)
		g.set(q, FlagRewardGranted, why)
		g.globalDone(q)
		d.rewarded = true
		q.LastState = 13
		g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 13})
		g.chainFrom(q)
	}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		switch {
		case g.get(q, FlagRewardGranted):
			if d.rewarded {
				return q.pick(npc, 4)
			}

			return nil
		case g.hasItem(ItemGidbinn) && g.get(q, bladeCarry) && !g.get(q, bladeRing):
			return q.pick(npc, 3)
		case g.get(q, bladeBrought):
			if npc == NPCAsheara {
				if !g.get(q, bladeWolves) {
					return q.pick(npc, 5)
				}
			} else {
				if npc == NPCOrmus && !g.get(q, bladeRing) {
					return q.pick(npc, 6)
				}

				if !g.get(q, bladeWolves) {
					return q.pick(npc, 5)
				}
			}

			if !g.get(q, bladeRing) {
				return q.pick(npc, 6)
			}

			return nil
		case g.get(q, bladeCarry):
			return q.pick(npc, 4)
		case q.State >= 1 && q.State < 4 && !g.hasItem(ItemGidbinn):
			return q.pick(npc, q.State-1)
		}

		return nil
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		if g.get(q, FlagRewardGranted) {
			return false
		}

		switch npc {
		case NPCHratli:
			return !g.hasItem(ItemGidbinn) && q.State == 1
		case NPCOrmus:
			if g.hasItem(ItemGidbinn) {
				return true
			}

			return g.get(q, bladeBrought) && !g.get(q, bladeRing)
		case NPCAsheara:
			return g.get(q, bladeBrought) && !g.get(q, bladeWolves)
		}

		return false
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		if e.NewLevel == LevelFlayerJungle && q.NotIntro && q.State == 0 {
			g.setState(q, 1)
		}

		if e.OldLevel != LevelKurastDocktown {
			return
		}

		switch {
		case q.State == 2 && !g.get(q, FlagRewardGranted) && !g.get(q, FlagCompletedEarly):
			g.setState(q, 3)
			flags(g, q)
		case q.State == 5:
			q.Active = false
		}
	}

	q.on[EvItemPickedUp] = func(g *Game, q *Quest, e *Event) {
		if e.Item == ItemGidbinn && q.NotIntro {
			if !g.get(q, bladeSeen) {
				g.set(q, bladeSeen, "Gidbinn picked up")
			}

			d.dropped = true
			g.setState(q, 4)
		}

		flags(g, q)
	}

	// The Gidbinn object (OperateFn 31): operating it wakes the fetish boss (class 407, D2MOO ACT3Q3_SpawnGidbinnBoss,
	// a timer of seven ticks when a hero stands near); the boss's death drops the blade at the boss (D2MOO
	// ACT3Q3_Callback08_MonsterKilled sets dwDropItemCode "g33 "). The altar (251) is scenery (OperateFn 0).
	q.on[EvObjectOperated] = func(g *Game, q *Quest, e *Event) {
		if e.Object != ObjectGidbinn {
			return
		}

		if g.get(q, FlagRewardGranted) || g.get(q, bladeWolves) || g.get(q, bladeRing) || !q.NotIntro {
			return
		}

		if q.State == 0 {
			g.setState(q, 1)
		}

		if g.hasItem(ItemGidbinn) || d.dropped || d.boss {
			return
		}

		d.boss = true

		g.emit(Effect{Kind: EffectSpawn, Quest: q.ID, Code: GidbinnBossCode, Note: "the fetish boss wakes (D2MOO)"})
		flags(g, q)
	}

	q.on[EvMonsterKilled] = func(g *Game, q *Quest, e *Event) {
		if e.Monster != NPCGidbinnBoss || !d.boss || d.dropped || !q.NotIntro {
			return
		}

		d.boss = false
		d.dropped = true

		g.emit(Effect{Kind: EffectSpawn, Quest: q.ID, Code: GidbinnDropCode, Note: "the boss drops Gidbinn where it fell"})
		flags(g, q)
	}

	q.on[EvNpcDeactivate] = func(g *Game, q *Quest, e *Event) {
		if e.NPC == NPCHratli && d.hratli {
			d.hratli = false
			q.LastState = 2

			flags(g, q)
		}
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if g.get(q, FlagRewardGranted) {
			return
		}

		switch {
		case e.NPC == NPCOrmus && e.Msg == 587:
			if g.hasItem(ItemGidbinn) && g.get(q, bladeCarry) && !g.get(q, bladeRing) {
				g.dropItem(ItemGidbinn)
				g.emit(Effect{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemGidbinn, Note: "Ormus takes the blade"})
				g.set(q, bladeBrought, "Ormus has Gidbinn")
			}

			if q.NotIntro {
				g.globalDone(q)
			}

			if q.NotIntro && q.State != 5 {
				g.setState(q, 5)
				g.cycle(q, 13, true)
				q.seq(g, q)
			}
		case e.NPC == NPCOrmus && e.Msg == 593:
			if !g.get(q, bladeBrought) || g.get(q, bladeRing) {
				return
			}

			g.set(q, bladeRing, "Ormus gave the ring")
			g.emit(Effect{Kind: EffectGiveItem, Quest: q.ID, Code: ItemReward, Quality: 2, Value: ringLevel(g.Difficulty),
				Note: "Ormus: a rare ring (VERIFIED 0x5b6ea0)"})

			if g.get(q, bladeWolves) {
				done(g, q, "ring and Iron Wolves")
			}
		case e.NPC == NPCAsheara && e.Msg == 589:
			if g.get(q, bladeWolves) {
				return
			}

			g.set(q, bladeWolves, "Asheara's Iron Wolves")
			g.emit(Effect{Kind: EffectReward, Quest: q.ID, Code: "hire-ironwolves", Note: "Asheara's Iron Wolf mercenaries become hirable"})

			if g.get(q, bladeRing) {
				done(g, q, "ring and Iron Wolves")
			}
		case e.NPC == NPCHratli && e.Msg == 571:
			g.setState(q, 2)
			d.hratli = true
		}
	}

	// ACT3Q3_StatusFilterCallback (without the Act 2 finished test)
	q.status = func(g *Game, q *Quest) int {
		switch {
		case g.get(q, FlagRewardGranted):
			if g.get(q, FlagPrimaryGoal) {
				return 13
			}

			return 11
		case g.hasItem(ItemGidbinn):
			return 4
		}

		page := 0

		switch {
		case g.get(q, bladeBrought):
			if !g.get(q, bladeRing) {
				page = 6
			}

			if !g.get(q, bladeWolves) {
				page = 5
			}
		case g.get(q, FlagEnterArea):
			page = 2
			if d.dropped {
				page = 3
			}
		case g.get(q, FlagLeaveTown):
			page = 2
		case g.get(q, FlagStarted):
			page = 1
		}

		return page
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, _ *Event) {
		switch {
		case g.get(q, FlagRewardGranted):
			q.State = 0
			g.globalDone(q)
		case g.hasItem(ItemGidbinn):
			q.State, q.LastState = 4, 4
		case g.get(q, bladeBrought):
			q.State = 5
			d.dropped = true

			if g.get(q, bladeWolves) {
				q.LastState = 6
			} else {
				q.LastState = 5
			}
		default:
			if g.get(q, bladeCarry) {
				q.State, q.LastState = 3, 2
			}

			if g.get(q, FlagEnterArea) || g.get(q, FlagLeaveTown) {
				q.State, q.LastState = 2, 2
			} else if g.get(q, FlagStarted) {
				q.State, q.LastState = 3, 1
			}
		}
	}

	return q
}

// ---- A3Q5 The Blackened Temple ----

type templeData struct {
	kills       int
	flail, cube bool // the Council deaths that already dropped the Flail / the Cube
	rewarded    bool
}

func newBlackenedTemple() *Quest {
	d := &templeData{}
	q := &Quest{ID: QuestBlackenedTemple, Slot: 21, Act: 2, Name: "The Blackened Temple", Label: "A3Q5", LogIndex: 5,
		Active: true, NotIntro: true, InitNo: 6, SeqID: QuestGuardian, tables: speechTables("A3Q5"), data: d}

	q.seq = chainSeq(false, 7)

	lamDone := func(g *Game) bool {
		lam := g.byID[QuestLamEsen]

		return lam != nil && g.global.Get(lam.Slot, FlagPrimaryGoal)
	}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		switch {
		case g.get(q, FlagEnterArea):
			if !g.orbSmashed {
				return q.pick(npc, 5)
			}
		case !g.get(q, FlagRewardGranted) || d.rewarded:
			switch {
			case q.State < 6:
				if q.NotIntro && q.State >= 1 {
					return q.pick(npc, q.State-1)
				}
			case g.get(q, FlagPrimaryGoal):
				return q.pick(npc, 6)
			}
		}

		return nil
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		switch {
		case g.get(q, FlagRewardGranted):
			return false
		case npc == NPCOrmus:
			return q.State == 1
		case npc == NPCCain3:
			return g.get(q, FlagEnterArea) && !g.orbSmashed
		}

		return false
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		if e.NewLevel == LevelKurastCauseway && q.NotIntro && q.State == 0 {
			g.setState(q, 1)
		}

		if e.NewLevel == LevelTravincal && q.NotIntro && !g.grantedOrPending(q) {
			g.set(q, FlagLeaveTown, "the Council is in Travincal")
		}

		if e.OldLevel != LevelKurastDocktown || q.State <= 1 || q.State > 3 {
			return
		}

		if g.get(q, FlagRewardGranted) || g.get(q, FlagEnterArea) {
			return
		}

		if lamDone(g) {
			g.setState(q, 4)
		} else {
			g.setState(q, 5)
		}

		q.LastState = 2
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if g.get(q, FlagRewardGranted) {
			return
		}

		switch {
		case e.NPC == NPCOrmus && e.Msg == 594:
			if lamDone(g) {
				g.setState(q, 2)
			} else {
				g.setState(q, 3)
			}

			g.set(q, FlagStarted, "Ormus sent the hero to Travincal")
		case e.NPC == NPCCain3 && e.Msg == 626 && g.get(q, FlagEnterArea):
			if k := g.byID[QuestKhalim]; k != nil && g.get(k, FlagRewardGranted) {
				g.deleteKhalimItems(q)
			}

			if g.get(q, FlagPrimaryGoal) && q.State != 7 {
				g.setState(q, 7)
				g.globalDone(q)
			}

			d.rewarded = true
			g.templeDone(q, "Cain's last word")
		}
	}

	// ACT3Q5_Callback08_MonsterKilled: the Council. The kills are counted by the quest; the first death also drops
	// the Cube for a hero without one (the Flail drop belongs to the engine, see d2act3.FlailDrop).
	q.on[EvMonsterKilled] = func(g *Game, q *Quest, e *Event) {
		if !q.NotIntro || g.get(q, FlagRewardGranted) {
			return
		}

		if e.Monster != NPCCouncilA && e.Monster != NPCCouncilB && e.Monster != NPCCouncilC {
			return
		}

		d.kills++

		// VERIFIED 0x5b97d0: while no Flail was dropped each death tries to drop one (the engine does it, d2act3.FlailDrop;
		// only a hero without Flail, Will or finished Khalim quest is owed one); the death after that one drops the Cube
		// for a hero without one.
		khalim := g.byID[QuestKhalim]

		switch {
		case !d.flail:
			d.flail = !g.hasItem(ItemKhalimFlail) && !g.hasItem(ItemKhalimWill) && !g.get(khalim, FlagRewardGranted)
		case !d.cube:
			d.cube = true

			if !g.hasItem(ItemHoradricCube) {
				g.emit(Effect{Kind: EffectGiveItem, Quest: q.ID, Code: ItemHoradricCube,
					Note: "a Council death leaves a Horadric Cube for a hero without one (VERIFIED 0x5b97d0)"})
			}
		}

		if d.kills < 3 || g.get(q, FlagEnterArea) {
			return
		}

		if g.orbSmashed {
			g.setState(q, 7)
		} else {
			g.setState(q, 6)
		}

		g.globalDone(q)
		g.set(q, FlagPrimaryGoal, "the Council is dead")

		if g.orbSmashed {
			g.templeDone(q, "the Council died after the orb was smashed")
			d.rewarded = true

			return
		}

		g.set(q, FlagEnterArea, "the Council is dead, the orb is whole")
		q.LastState = 4
		g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: g.LogPage(q)})
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, _ *Event) {
		if g.get(q, FlagRewardGranted) || g.get(q, FlagEnterArea) {
			g.globalDone(q)
			q.NotIntro = false

			return
		}

		pick := func(done, not int) int {
			if lamDone(g) {
				return done
			}

			return not
		}

		switch {
		case g.get(q, FlagLeaveTown):
			q.LastState = 3
			q.State = pick(4, 5)
		case g.get(q, FlagStarted):
			q.LastState = 2
			q.State = pick(2, 3)
		}
	}

	return q
}

// ---- A3Q1 Lam Esen's Tome ----

type lamData struct {
	alkorStart bool
	rewarded   bool
}

// A3Q1 Lam Esen's Tome (Alkor). VERIFIED (Game.exe 0x5b52c0, ev3): entering Lower Kurast (79) makes the quest
// available (state 1); leaving the Docks in state 2 moves it to state 3 (D2MOO). Alkor's message 564 with the
// tome in the pack deletes the tome and pays 5 stat points (0x5b53e0 + 0x5b5220), setting the done bits at once:
// the reward is never pending across a save (the exe sets bits 0, 1 and 0xd together and clears bit 1 in the
// same call), so a hero cannot claim without the tome.
func newLamEsen() *Quest {
	d := &lamData{}
	q := &Quest{ID: QuestLamEsen, Slot: 17, Act: 2, Name: "Lam Esen's Tome", Label: "A3Q1", LogIndex: 1,
		Active: true, NotIntro: true, InitNo: 4, SeqID: QuestBlackenedTemple, tables: speechTables("A3Q1")}

	q.seq = chainSeq(false, 5)

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		switch {
		case g.get(q, FlagRewardGranted):
			if d.rewarded {
				return q.pick(npc, 4)
			}

			return nil
		case g.hasItem(ItemLamEsenTome):
			return q.pick(npc, 3)
		case q.NotIntro && q.State >= 1 && q.State < 4:
			return q.pick(npc, q.State-1)
		}

		return nil
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		if npc != NPCAlkor {
			return false
		}

		if q.State == 1 && !g.get(q, FlagRewardGranted) && !g.get(q, FlagCompletedEarly) {
			return true
		}

		return g.hasItem(ItemLamEsenTome)
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		if e.NewLevel == LevelLowerKurast && q.NotIntro && q.State == 0 {
			g.setState(q, 1)
		}

		if e.OldLevel == LevelKurastDocktown && q.State == 2 && !g.get(q, FlagRewardGranted) {
			q.LastState = 1
			g.setState(q, 3)
			g.set(q, FlagStarted, "started")
		}
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCAlkor || g.get(q, FlagRewardGranted) {
			return
		}

		switch e.Msg {
		case 549:
			q.LastState = 1
			g.setState(q, 2)
			g.set(q, FlagStarted, "Alkor asked for the tome")
			d.alkorStart = true
		case 564:
			if !g.hasItem(ItemLamEsenTome) {
				return
			}

			g.dropItem(ItemLamEsenTome)
			g.emit(Effect{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemLamEsenTome, Note: "Alkor keeps the tome"})
			g.emit(Effect{Kind: EffectReward, Quest: q.ID, Code: "stat-points", Value: 5, Note: "Lam Esen's Tome: 5 stat points"})
			g.set(q, FlagPrimaryGoal, "Alkor took the tome")
			g.set(q, FlagRewardGranted, "Alkor took the tome")
			g.globalDone(q)
			d.rewarded = true
			g.setState(q, 5)
			q.LastState = 13
			g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 13})
			g.chainFrom(q)
		}
	}

	q.on[EvObjectOperated] = func(g *Game, q *Quest, e *Event) {
		if e.Object != ObjectLamEsenTome || !q.NotIntro || g.get(q, FlagRewardGranted) {
			return
		}

		if !g.hasItem(ItemLamEsenTome) {
			g.emit(Effect{Kind: EffectSpawn, Quest: q.ID, Code: ItemLamEsenTome, Note: "Lam Esen's Tome on the altar"})
		}

		if q.State != 3 {
			g.setState(q, 3)
		}
	}

	q.on[EvItemPickedUp] = func(g *Game, q *Quest, e *Event) {
		if e.Item == ItemLamEsenTome && q.NotIntro && !g.get(q, FlagRewardGranted) {
			q.LastState = 2
			g.setState(q, 4)
			g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: g.LogPage(q)})
		}
	}

	q.on[EvItemDropped] = func(g *Game, q *Quest, e *Event) {
		if e.Item == ItemLamEsenTome && q.NotIntro && !g.get(q, FlagRewardGranted) {
			q.LastState = 1
			g.setState(q, 3)
		}
	}

	q.status = func(g *Game, q *Quest) int {
		switch {
		case g.get(q, FlagRewardGranted):
			if g.get(q, FlagPrimaryGoal) {
				return 13
			}

			return 11
		case g.hasItem(ItemLamEsenTome):
			return 2
		case !q.NotIntro || q.State < 1:
			return 0
		}

		return 1
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, _ *Event) {
		switch {
		case g.get(q, FlagRewardGranted):
			g.globalDone(q)
		case g.hasItem(ItemLamEsenTome):
			q.State, q.LastState = 4, 2
		case g.get(q, FlagLeaveTown):
			q.State, q.LastState = 3, 1
		case g.get(q, FlagStarted):
			q.State, q.LastState = 2, 1
		}
	}

	return q
}
