package d2quest

// Quests of Acts 3, 4 and 5. The speech tables (who says which message in
// which state) are dumped from the binary; the record slots and quest ids are
// from the quest table (quests.md section 2). The triggers (levels, monsters,
// items, objects) follow the pointers of quests.md section 5 and the usual
// shape of the original quests and are UNVERIFIED: no handler of these quests
// was read from the binary. Every spec says what it assumes.

// Objects of Acts 3 and 4 (objects.txt rows, found by name).
const (
	ObjectLamEsenTome   = 193
	ObjectCompellingOrb = 404
	ObjectHellforge     = 376
	ObjectGidbinnAltar  = 251 // "gidbinn altar" (objects.txt): the blade of the Old Religion lies on it
	ObjectGidbinn       = 252 // "gidbinn" (the decoy: the blade on the altar; the Flayer Jungle has one)
)

// ---- prologues ----

// newPrologue builds the welcome node of an act: the NPC says its first line
// once and the slot gets RewardGranted when it was heard.
func newPrologue(id, slot, act int, label string, npc int, altClass int) *Quest {
	q := &Quest{ID: id, Slot: slot, Act: act, Name: label + " prologue", Label: label, Active: true,
		NoSetState: true, SeqID: -1, tables: speechTables(label)}

	q.activate = func(g *Game, q *Quest, n int) []Speech {
		if n != npc || g.get(q, FlagRewardGranted) {
			return nil
		}

		if altClass >= 0 && g.Hero.Class == altClass && len(q.tables) > 1 {
			return q.pick(n, 1)
		}

		return q.pick(n, 0)
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != npc {
			return
		}

		for _, t := range q.tables {
			for _, s := range t {
				if s.NPC == npc && s.Msg == e.Msg {
					g.set(q, FlagRewardGranted, "act welcome heard")
					g.globalDone(q)

					return
				}
			}
		}
	}

	q.active = func(g *Game, q *Quest, n int) bool { return n == npc && !g.get(q, FlagRewardGranted) }

	q.on[EvGameStarted] = func(g *Game, q *Quest, _ *Event) {
		if g.get(q, FlagRewardGranted) {
			g.globalDone(q)
		}
	}

	return q
}

// A3Q0: Hratli's welcome (the second line is the Sorceress variant).
func newA3Prologue() *Quest {
	return newPrologue(QuestA3Prologue, 16, 2, "A3Q0", NPCHratli, ClassSorceress)
}

// A4Q0: Tyrael's welcome.
func newA4Prologue() *Quest {
	return newPrologue(QuestA4Prologue, 24, 3, "A4Q0", NPCTyrael2, -1)
}

// ---- Act 3 ----

// A3Q1 Lam Esen's Tome (Alkor). UNVERIFIED: the tome sits in the Ruined Temple
// (level 94) and is picked up as item bbb; the reward is 5 stat points.
func newLamEsen() *Quest {
	return newSpecQuest(&spec{
		id: QuestLamEsen, slot: 17, act: 2, logIndex: 1, name: "Lam Esen's Tome", label: "A3Q1",
		start: 1, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 2}, rp: 3, done: 4,
		steps: []step{{from: 1, npc: NPCAlkor, msg: 549, to: 2}},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelRuinedTemple, max: 3, to: 3},
			// the tome lies on the altar of the Ruined Temple (object 193): operating it gives the item; taking the item
			// is the goal (found by playing: the object was never operated as a quest object, so no tome appeared)
			{ev: EvObjectOperated, object: ObjectLamEsenTome, min: -1, max: 3, bit: -1,
				fx: func(g *Game, q *Quest) []Effect {
					if g.hasItem(ItemLamEsenTome) {
						return nil
					}

					return []Effect{{Kind: EffectSpawn, Quest: q.ID, Code: ItemLamEsenTome, Note: "Lam Esen's Tome on the altar"}}
				}},
			{ev: EvItemPickedUp, item: ItemLamEsenTome, goal: true},
		},
		claimFx: func(g *Game, q *Quest) []Effect {
			g.dropItem(ItemLamEsenTome)

			return []Effect{
				{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemLamEsenTome, Note: "Alkor keeps the tome"},
				reward("stat-points", 5, "Lam Esen's Tome: 5 stat points"),
			}
		},
	})
}

// A3Q2 Khalim's Will (Cain only). The reports follow the table order (eye,
// heart, brain, flail; the table indexes 1-4 are the "early" lines of each
// part); the quest is done when the Will smashes the Compelling Orb in
// Travincal. UNVERIFIED: object 404 and level 83; the part-to-bit mapping.
func newKhalim() *Quest {
	q := &Quest{ID: QuestKhalim, Slot: 18, Act: 2, Name: "Khalim's Will", Label: "A3Q2", LogIndex: 2,
		Active: true, NotIntro: true, NoSetState: true, SeqID: -1, tables: speechTables("A3Q2")}

	type part struct {
		items []string
		bit   int
		msg   int
		tbl   int
		topc  int
	}

	parts := []part{
		{[]string{ItemKhalimEye}, FlagCustom1, 545, 1, 7},
		{[]string{ItemKhalimHeart}, FlagCustom1 + 1, 544, 2, 8},
		{[]string{ItemKhalimBrain}, FlagCustom1 + 2, 546, 3, 9},
		{[]string{ItemKhalimFlail, ItemKhalimWill}, FlagCustom1 + 3, 547, 4, 10},
	}

	carries := func(g *Game, p part) bool {
		for _, c := range p.items {
			if g.hasItem(c) {
				return true
			}
		}

		return false
	}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		if npc != NPCCain3 {
			return nil
		}

		switch {
		case g.get(q, FlagRewardGranted):
			return q.pick(npc, 11)
		case g.get(q, FlagRewardPending):
			return q.pick(npc, 5)
		case !g.get(q, FlagStarted):
			return q.pick(npc, 0)
		}

		var out []Speech

		for _, p := range parts {
			if carries(g, p) && !g.get(q, p.bit) {
				out = append(out, q.pick(npc, p.tbl)...)

				break
			}
		}

		out = append(out, q.pick(npc, 6)...)

		for _, p := range parts {
			if g.get(q, p.bit) {
				out = append(out, q.pick(npc, p.topc)...)
			}
		}

		return out
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		if npc != NPCCain3 || g.get(q, FlagRewardGranted) {
			return false
		}

		if g.get(q, FlagRewardPending) || !g.get(q, FlagStarted) {
			return true
		}

		for _, p := range parts {
			if carries(g, p) && !g.get(q, p.bit) {
				return true
			}
		}

		return false
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCCain3 {
			return
		}

		switch {
		case e.Msg == 543 && !g.get(q, FlagStarted):
			g.set(q, FlagStarted, "Cain explained Khalim's Will")
			g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: g.LogPage(q)})
		case e.Msg == 548 && g.get(q, FlagRewardPending):
			g.set(q, FlagRewardGranted, "Cain: the orb is broken")
			g.clear(q, FlagRewardPending, "reward taken")
			g.globalDone(q)
			g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 13})
		default:
			for _, p := range parts {
				if e.Msg == p.msg {
					g.set(q, p.bit, "Cain reported a part of Khalim's Will")
					g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: g.LogPage(q)})
				}
			}
		}
	}

	q.on[EvObjectOperated] = func(g *Game, q *Quest, e *Event) {
		if e.Object != ObjectCompellingOrb || g.grantedOrPending(q) || !g.hasItem(ItemKhalimWill) {
			return
		}

		g.set(q, FlagPrimaryGoal, "the Compelling Orb is smashed")
		g.set(q, FlagRewardPending, "the Compelling Orb is smashed")
		g.globalDone(q)
		g.after(8, func() { g.cycle(q, 3, true) })
	}

	q.status = func(g *Game, q *Quest) int {
		if g.get(q, FlagRewardGranted) || !g.get(q, FlagStarted) {
			return 0
		}

		n := 1

		for _, p := range parts {
			if g.get(q, p.bit) {
				n++
			}
		}

		return n // page numbers are UNVERIFIED
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, _ *Event) {
		if g.get(q, FlagRewardGranted) {
			g.globalDone(q)
		}
	}

	return q
}

// A3Q3 The Blade of the Old Religion (Hratli): fetch Gidbinn (g33) from the
// Flayer Dungeon, hand it to Ormus, then Asheara; Ormus's last line pays out.
// UNVERIFIED: the levels (88, 89, 91) and the reward (Iron Wolf mercenaries).
func newBlade() *Quest {
	gidbinn := func(g *Game, q *Quest) []Effect {
		if g.hasItem(ItemGidbinn) {
			return nil
		}

		return []Effect{{Kind: EffectSpawn, Quest: q.ID, Code: ItemGidbinn, Note: "Gidbinn on the altar"}}
	}

	return newSpecQuest(&spec{
		id: QuestBlade, slot: 19, act: 2, logIndex: 3, name: "The Blade of the Old Religion", label: "A3Q3",
		start: 1, goal: 6, tbl: map[int]int{1: 0, 2: 1, 3: 2, 4: 3, 5: 5}, rp: 6, done: 7,
		steps: []step{
			{from: 1, npc: NPCHratli, msg: 571, to: 2},
			{from: 4, npc: NPCOrmus, msg: 587, to: 5, fx: func(g *Game, q *Quest) []Effect {
				g.dropItem(ItemGidbinn)

				return []Effect{{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemGidbinn, Note: "Ormus takes the blade"}}
			}},
			{from: 5, npc: NPCAsheara, msg: 589, to: 6, goal: true},
		},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelFlayerDungeon1, max: 3, to: 3},
			{ev: EvAreaChanged, level: LevelFlayerDungeon2, max: 3, to: 3},
			{ev: EvAreaChanged, level: LevelFlayerDungeon3, max: 3, to: 3},
			// operating the altar of the Flayer Dungeon gives Gidbinn (found by playing: nothing did, so the blade could
			// not be had in the real world)
			{ev: EvObjectOperated, object: ObjectGidbinnAltar, min: -1, max: 3, bit: -1, fx: gidbinn},
			{ev: EvObjectOperated, object: ObjectGidbinn, min: -1, max: 3, bit: -1, fx: gidbinn},
			{ev: EvItemPickedUp, item: ItemGidbinn, max: 3, to: 4},
		},
		// the log pages (qstsa3q31..35): 1 look for Gidbinn in the Flayer Jungle, 2 pick it up, 3 return it to Ormus,
		// 4 talk to Asheara, 5 talk to Ormus (the reward). The pages followed the first area change only, so the log
		// kept showing page 1 until the end (found by playing). UNVERIFIED order of the original's status bytes.
		logPage: func(g *Game, q *Quest) int {
			if g.get(q, FlagRewardGranted) {
				return 0
			}

			switch q.State {
			case 2:
				return 1
			case 3:
				return 2
			case 4:
				return 3
			case 5:
				return 4
			case 6:
				return 5
			}

			return 0
		},
		claimFx: fxs(reward("hire-ironwolves", 0, "Asheara's Iron Wolf mercenaries become hirable")),
	})
}

// A3Q4 The Golden Bird (Cain, Meshif, Alkor): the Jade Figurine (j34) starts
// the quest, Meshif trades it for the Golden Bird (g34), Alkor takes the bird
// and pays +20 life. UNVERIFIED: the hand-over order and the reward.
func newGoldenBird() *Quest {
	return newSpecQuest(&spec{
		id: QuestGoldenBird, slot: 20, act: 2, logIndex: 4, name: "The Golden Bird", label: "A3Q4",
		start: 0, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 2}, rp: 5, done: 6,
		steps: []step{
			{from: 1, npc: NPCCain3, msg: 527, to: 2},
			{from: 2, npc: NPCMeshif2, msg: 529, to: 3, fx: func(g *Game, q *Quest) []Effect {
				g.dropItem(ItemJadeFigurine)

				return []Effect{
					{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemJadeFigurine, Note: "Meshif takes the figurine"},
					{Kind: EffectSpawn, Quest: q.ID, Code: ItemGoldenBird, Note: "Meshif gives the Golden Bird"},
				}
			}},
			{from: 3, npc: NPCAlkor, msg: 534, to: 4, goal: true, fx: func(g *Game, q *Quest) []Effect {
				g.dropItem(ItemGoldenBird)

				return []Effect{{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemGoldenBird, Note: "Alkor takes the bird"}}
			}},
		},
		trigs: []trig{
			{ev: EvItemPickedUp, item: ItemJadeFigurine, min: -1, max: -1, to: 1, bit: -1},
		},
		noLeaveRule: true,
		// VERIFIED (Game.exe 0x5b7f40, msg 538 = 0x21a): the claim hands out the Potion of Life ("xyz") and
		// sets CUSTOM1 (bit 5); the +20 life is paid when the potion is drunk (DrinkPotionOfLife).
		claimFx: func(g *Game, q *Quest) []Effect {
			g.set(q, FlagCustom1, "Potion of Life earned")

			return []Effect{{Kind: EffectSpawn, Quest: q.ID, Code: ItemPotionOfLife, Note: "Alkor gives the Potion of Life"}}
		},
	})
}

// DrinkPotionOfLife is the hero drinking Alkor's Potion of Life. VERIFIED (ITEMACT_ServerUseItem 0x55bfd0):
// only while A3Q4 bit 5 is set; the bit is cleared and the base max-life stat (7) rises by 20 (0x1400 in
// 8.8 fixed point).
func (g *Game) DrinkPotionOfLife() []Effect {
	q := g.byID[QuestGoldenBird]
	if !g.get(q, FlagCustom1) {
		return nil
	}

	g.clear(q, FlagCustom1, "Potion of Life drunk")
	g.emit(reward("life-boost", 20, "Potion of Life: +20 max life"))

	return g.TakeEffects()
}

// ReadScrollOfResistance is the hero reading Malah's Scroll of Resistance ("tr2"). VERIFIED (0x55bfd0,
// FUN_00587f90 0x587f90, FUN_00587ee0 0x587ee0): allowed while A5Q3 bit 8 (scroll given) is set and bit 7
// (scroll read) is clear; it sets bit 7 and adds a stat list with base stats 39/41/43/45 (fire, lightning,
// cold, poison resist). The value is 10 for every difficulty record whose bit 7 is set (the exe sums the
// three records, so the bonus accumulates across difficulties), and the exe re-applies it when a player
// joins a game (SERVER_ClientAddPlayerToGame 0x537455). The Value of the effect is the whole bonus of
// this record only; callers combine the records.
func (g *Game) ReadScrollOfResistance() []Effect {
	q := g.byID[QuestPrison]
	if !g.get(q, FlagCustom4) || g.get(q, FlagCustom3) {
		return nil
	}

	g.set(q, FlagCustom3, "Scroll of Resistance read")
	g.emit(reward("resist-bonus", 10, "Scroll of Resistance: +10 fire/lightning/cold/poison resist"))

	return g.TakeEffects()
}

// ResistBonus is the permanent resistance bonus this record has earned (10 once the scroll was read).
func (g *Game) ResistBonus() int {
	if g.get(g.byID[QuestPrison], FlagCustom3) {
		return 10
	}

	return 0
}

// A3Q5 The Blackened Temple (Ormus): kill the three Council members in
// Travincal. UNVERIFIED: the classes 345-347 and the level 83.
func newBlackenedTemple() *Quest {
	return newSpecQuest(&spec{
		id: QuestBlackenedTemple, slot: 21, act: 2, logIndex: 5, name: "The Blackened Temple", label: "A3Q5",
		start: 0, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 2}, rp: 5, done: 6,
		steps: []step{{from: 1, npc: NPCOrmus, msg: 594, to: 2}},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelTravincal, max: 3, to: 3},
			{ev: EvMonsterKilled, monsters: []int{NPCCouncilA, NPCCouncilB, NPCCouncilC}, level: LevelTravincal,
				count: 3, goal: true},
		},
	})
}

// A3Q6 The Guardian (Ormus): through the Durance of Hate and Mephisto's death.
// UNVERIFIED: levels 100-102 and class 242 (Mephisto).
func newGuardian() *Quest {
	return newSpecQuest(&spec{
		id: QuestGuardian, slot: 22, act: 2, logIndex: 6, name: "The Guardian", label: "A3Q6",
		start: 0, goal: 5, tbl: map[int]int{1: 0, 2: 1, 3: 2, 4: 3}, rp: 5, done: 6, kill: guardianSpeech(),
		steps: []step{{from: 1, npc: NPCOrmus, msg: 628, to: 2}},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelDurance1, max: 3, to: 3},
			{ev: EvAreaChanged, level: LevelDurance3, max: 3, to: 4},
			{ev: EvMonsterKilled, monster: NPCMephisto, names: []string{"mephisto"}, min: -1, goal: true, exe: &exeKill{bits: []int{exeBitMephisto}, dropCode: ItemMephistoSoulstone}},
		},
	})
}

// ---- Act 4 ----

// A4Q1 The Fallen Angel (Tyrael): Izual in the Plains of Despair, +2 skill
// points. UNVERIFIED: Izual's class (256 or the ghost 406) and the reward.
func newFallenAngel() *Quest {
	return newSpecQuest(&spec{
		id: QuestFallenAngel, slot: 25, act: 3, logIndex: 1, name: "The Fallen Angel", label: "A4Q1",
		start: 1, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 2}, rp: 3, done: 4,
		steps: []step{{from: 1, npc: NPCTyrael2, msg: 670, to: 2}},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelPlainsDespair, max: 3, to: 3},
			{ev: EvMonsterKilled, monsters: []int{NPCIzual, NPCIzualGhost}, goal: true},
		},
		claimMsgs: []int{676},
		claimFx:   fxs(Effect{Kind: EffectSkillPoint, Value: 2, Note: "Fallen Angel: 2 skill points"}),
	})
}

// A4Q2 Terror's End (Tyrael): Diablo in the Chaos Sanctuary. The expansion
// uses the SUCCESSEXP lines (tables 4 and 5). UNVERIFIED: level 108, class 243.
func newTerrorsEnd() *Quest {
	return newSpecQuest(&spec{
		id: QuestTerrorsEnd, slot: 26, act: 3, logIndex: 2, name: "Terror's End", label: "A4Q2",
		start: 1, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 1}, rp: 2, done: 3, rpExp: 4, doneExp: 5, kill: terrorSpeech(),
		steps: []step{{from: 1, npc: NPCTyrael2, msg: 681, to: 2}},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelChaosSanctum, max: 3, to: 3},
			{ev: EvMonsterKilled, monster: NPCDiablo, names: []string{"diablo"}, min: -1, goal: true, exe: &exeKill{classicBits: []int{6, 7}}},
		},
	})
}

// A4Q3 The Hellforge (Cain): smash the Mephisto Soulstone on the Hellforge
// with the Hellforge Hammer. Cain's first line depends on the soulstone.
// VERIFIED: the object (376, OperateFn 49 = 0x5b3820 asks for the hammer "hfh "), the hammer drop by Hephasto (see below).
func newHellforge() *Quest {
	return newSpecQuest(&spec{
		id: QuestHellforge, slot: 27, act: 3, logIndex: 3, name: "The Hellforge", label: "A4Q3",
		start: 1, goal: 3, tbl: map[int]int{1: 0}, rp: 2, done: 3,
		tableFn: func(g *Game, q *Quest, def int) int {
			if q.State == 1 && !g.hasItem(ItemMephistoSoulstone) {
				return 1
			}

			return def
		},
		steps: []step{
			{from: 1, npc: NPCCain4, msg: 678, to: 2},
			{from: 1, npc: NPCCain4, msg: 679, to: 2},
		},
		trigs: []trig{
			{ev: EvObjectOperated, object: ObjectHellforge, req: []string{ItemMephistoSoulstone, ItemHellforgeHammer},
				goal: true, fx: func(g *Game, q *Quest) []Effect {
					g.dropItem(ItemMephistoSoulstone)

					return []Effect{{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemMephistoSoulstone,
						Note: "the soulstone is smashed on the Hellforge"}}
				}},
			// Hephasto's kill drops the hammer (unless Game.LegacyBossBits). VERIFIED (QUEST_A4_TheHellforge_OnMonsterKilled 0x5b4190, attached to
			// class 0x199 by 0x5af8c0): while the node is active the dying unit's item code (+0xb8) is stamped "hfh " and dropped through
			// 0x557980; no check of the quest state beyond the node being active, no bit is set.
			{ev: EvMonsterKilled, monster: NPCHephasto, names: []string{"hephasto"}, min: 1, bit: -1,
				cond: func(g *Game, q *Quest) bool { return !g.LegacyBossBits },
				fx: func(g *Game, q *Quest) []Effect {
					return []Effect{{Kind: EffectGiveItem, Quest: q.ID, Code: ItemHellforgeHammer,
						Note: "Hephasto drops the hammer (exe: unit code override \"hfh \", 0x5b4190)"}}
				}},
		},
		noLeaveRule: true,
	})
}

// A4Q4 Malachai's gossip (slot 33): the angel comments on the Mephisto
// Soulstone, and on its destruction.
func newMalachai() *Quest {
	q := &Quest{ID: QuestMalachai, Slot: 33, Act: 3, Name: "Malachai", Label: "A4Q4", Active: true,
		NoSetState: true, SeqID: -1, tables: speechTables("A4Q4")}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		hell := g.byID[QuestHellforge]

		switch {
		case npc != NPCMalachai:
			return nil
		case g.get(hell, FlagPrimaryGoal) || g.get(hell, FlagRewardGranted):
			return q.pick(npc, 1)
		case g.hasItem(ItemMephistoSoulstone):
			return q.pick(npc, 0)
		}

		return nil
	}

	return q
}

// ---- Act 5 ----

// A5Q1 Siege on Harrogath (Larzuk): Shenk the Overseer in the Bloody
// Foothills; Larzuk sockets an item. UNVERIFIED: level 110 and the super name.
func newSiege() *Quest {
	return newSpecQuest(&spec{
		id: QuestSiege, slot: 35, act: 4, logIndex: 1, name: "Siege on Harrogath", label: "A5Q1",
		start: 1, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 2}, rp: 3, done: 4,
		steps: []step{{from: 1, npc: NPCLarzuk, msg: 20077, to: 2}},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelBloodyFoothills, max: 3, to: 3},
			{ev: EvMonsterKilled, super: "Shenk the Overseer", goal: true},
		},
		// VERIFIED (Game.exe 0x584d60, msg 20090 = 0x4e7a): the ack clears the progress bits and sets bit 5; it
		// does not set RG or clear RP itself (UNRESOLVED: where the socketing action sets RG; kept at the claim).
		claimFx: fxs(reward("socket-quest", 1, "Larzuk adds sockets to one item")),
	})
}

// A5Q2 Rescue on Mount Arreat (Qual-Kehk): free the barbarian groups by
// killing the demons at their prison doors (class 434, three groups).
// UNVERIFIED: the count and the Frigid Highlands level.
func newRescue() *Quest {
	return newSpecQuest(&spec{
		id: QuestRescue, slot: 36, act: 4, logIndex: 2, name: "Rescue on Mount Arreat", label: "A5Q2",
		start: 1, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 2}, rp: 3, done: 4,
		steps: []step{{from: 1, npc: NPCQualKehk, msg: 20096, to: 2}},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelFrigidHighlands, max: 3, to: 3},
			{ev: EvMonsterKilled, monster: NPCPrisonDoor, count: 3, goal: true},
		},
		claimFx: fxs(reward("hire-barbarians", 0, "Qual-Kehk's barbarian mercenaries become hirable")),
	})
}

// A5Q3 Prison of Ice (Malah): find the frozen Anya, get Malah's scroll, read
// it. Reading the scroll (it leaves the inventory) completes the goal.
// UNVERIFIED: level 114 and the scroll flow.
func newPrison() *Quest {
	give := func(g *Game, q *Quest) []Effect {
		return []Effect{{Kind: EffectSpawn, Quest: q.ID, Code: ItemMalahScroll, Note: "Malah's Scroll of Resistance"}}
	}

	return newSpecQuest(&spec{
		id: QuestPrison, slot: 37, act: 4, logIndex: 3, name: "Prison of Ice", label: "A5Q3",
		start: 1, goal: 6, tbl: map[int]int{1: 0, 2: 1, 3: 2, 4: 3, 5: 4}, rp: 5, done: 6,
		steps: []step{
			{from: 1, npc: NPCMalah, msg: 20116, to: 2},
			{from: 4, npc: NPCMalah, msg: 20127, to: 5, fx: give},
			{from: 4, npc: NPCAnyaFrozen, msg: 20131, to: 5, fx: give},
		},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelFrozenRiver, max: 3, to: 4},
			{ev: EvItemRemoved, item: ItemMalahScroll, min: 5, max: 5, goal: true},
		},
		// VERIFIED (Game.exe 0x587460): the claim pays no resistance; Malah's line gives the Scroll of Resistance
		// (tr2) and sets bit 8; the +10 comes from reading it (ReadScrollOfResistance). UNRESOLVED: the original
		// needs both Malah's scroll and Anya's item (msg 20136) before RG is set, and Malah's "ice" potion
		// (msg 20127) thaws Anya; this flow still follows the earlier approximation.
		claimFx: func(g *Game, q *Quest) []Effect {
			g.set(q, FlagCustom4, "Malah's scroll given")

			return []Effect{{Kind: EffectSpawn, Quest: q.ID, Code: ItemMalahScroll, Note: "Malah gives the Scroll of Resistance"}}
		},
	})
}

// A5Q4 Betrayal of Harrogath (Anya): Nihlathak in his temple. UNVERIFIED:
// level 121 and class 526; the reward is Anya personalising an item.
func newBetrayal() *Quest {
	return newSpecQuest(&spec{
		id: QuestBetrayal, slot: 38, act: 4, logIndex: 4, name: "Betrayal of Harrogath", label: "A5Q4",
		start: 1, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 2}, rp: 3, done: 4,
		steps: []step{{from: 1, npc: NPCDrehya, msg: 20137, to: 2}},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelNihlathakTemple, max: 3, to: 3},
			{ev: EvMonsterKilled, monster: NPCNihlathakBoss, goal: true},
		},
		claimFx: fxs(reward("personalize", 1, "Anya personalises one item")),
	})
}

// A5Q5 Rite of Passage (Qual-Kehk): the three Ancients on Mount Arreat; the
// statues speak when the quest reached the summit. UNVERIFIED: classes 540-542.
func newRite() *Quest {
	return newSpecQuest(&spec{
		id: QuestRite, slot: 39, act: 4, logIndex: 5, name: "Rite of Passage", label: "A5Q5",
		start: 1, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 2}, rp: 5, done: 3,
		steps: []step{{from: 1, npc: NPCQualKehk, msg: 20153, to: 2}},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelArreatSummit, max: 3, to: 3},
			{ev: EvMonsterKilled, monsters: []int{NPCAncient1, NPCAncient2, NPCAncient3}, count: 3, goal: true},
		},
		extra: func(g *Game, q *Quest, npc int) []Speech {
			if q.State == 3 && npc >= NPCAncientStatue1 && npc <= NPCAncientStatue3 {
				return q.pick(npc, 4)
			}

			return nil
		},
	})
}

// A5Q6 Eve of Destruction (everyone): the Worldstone Keep, the Throne and
// Baal. Available once Rite of Passage is done. UNVERIFIED: levels 128/131
// and Baal's class (543).
func newEve() *Quest {
	return newSpecQuest(&spec{
		id: QuestEve, slot: 40, act: 4, logIndex: 6, name: "Eve of Destruction", label: "A5Q6",
		start: 0, goal: 3, tbl: map[int]int{1: 0, 2: 0}, rp: 1, done: 3, noLeaveRule: true, kill: eveSpeech(),
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelWorldstone1, max: 1, to: 2},
			{ev: EvAreaChanged, level: LevelThrone, max: 2, to: 2},
			{ev: EvMonsterKilled, monster: NPCBaalCrab, names: []string{"baal"}, min: -1, goal: true, exe: &exeKill{level: LevelWorldstoneChamber}},
		},
		claimFx: fxs(reward("unlock-difficulty", 0, "Baal is dead: the next difficulty opens"),
			reward("game-complete", 0, "end of the game")),
	})
}
