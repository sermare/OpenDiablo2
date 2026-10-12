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

// A3Q4 The Golden Bird (Cain, Meshif, Alkor): the Jade Figurine (j34) starts
// the quest, Meshif trades it for the Golden Bird (g34), Alkor takes the bird
// and pays +20 life. UNVERIFIED: the hand-over order and the reward.
func newGoldenBird() *Quest {
	return newSpecQuest(&spec{
		id: QuestGoldenBird, slot: 20, act: 2, logIndex: 4, name: "The Golden Bird", label: "A3Q4", logPage: goldenBirdLogPage,
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
			// VERIFIED 0x5b7d90: with the bird in the pack the log shows page 3 ("ask Cain about the Golden Bird") until bit 4
			// is set, then page 4 ("give the bird to Alkor"); the page sounds of the client table are 529 (Meshif) and 531 (Cain).
			{from: 3, npc: NPCCain3, msg: 531, nomark: true, fx: func(g *Game, q *Quest) []Effect {
				g.set(q, FlagEnterArea, "Cain explained the Golden Bird")

				return nil
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

// A3Q6 The Guardian (Ormus): through the Durance of Hate and Mephisto's death.
// Levels 100-102 and class 242 (Mephisto) checked w28 against Levels.txt / MonStats.txt row numbers (table check, not read in the exe).
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
// points. VERIFIED w25: the 2 points (0x5b15c0) and the kill (0x5b1c50: RP + goal for the killer, whoever sits in
// Izual's room or the next ones). Both Izual rows (256, ghost 406) are accepted; MonStats.txt confirms both ids (w28, table check).
func newFallenAngel() *Quest {
	return newSpecQuest(&spec{
		id: QuestFallenAngel, slot: 25, act: 3, logIndex: 1, name: "The Fallen Angel", label: "A4Q1",
		start: 1, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 2}, rp: 3, done: 4,
		steps: []step{{from: 1, npc: NPCTyrael2, msg: 670, to: 2}},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelPlainsDespair, max: 3, to: 3, page: -1}, // exe 0x5b1d60: page 1 when none yet
			{ev: EvMonsterKilled, monsters: []int{NPCIzual, NPCIzualGhost}, goal: true},
		},
		claimMsgs: []int{676},
		claimFx:   fxs(Effect{Kind: EffectSkillPoint, Value: 2, Note: "Fallen Angel: 2 skill points"}),
	})
}

// A4Q2 Terror's End (Tyrael): Diablo in the Chaos Sanctuary. The expansion
// uses the SUCCESSEXP lines (tables 4 and 5). VERIFIED w25: Diablo is class 243 (0x5b2f10); Level 108 is Chaos Sanctum in Levels.txt (w28, table check).
func newTerrorsEnd() *Quest {
	return newSpecQuest(&spec{
		id: QuestTerrorsEnd, slot: 26, act: 3, logIndex: 2, name: "Terror's End", label: "A4Q2",
		start: 1, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 1}, rp: 2, done: 3, rpExp: 4, doneExp: 5, kill: terrorSpeech(),
		steps: []step{{from: 1, npc: NPCTyrael2, msg: 681, to: 2}},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelChaosSanctum, max: 3, to: 3, page: -1}, // exe 0x5b2ad0: page 1 when none yet
			{ev: EvMonsterKilled, monster: NPCDiablo, names: []string{"diablo"}, min: -1, goal: true, exe: &exeKill{classicBits: []int{6, 7}}},
		},
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
// Foothills; Larzuk sockets an item. Level 110 is Bloody Foothills in Levels.txt (w28, table check); the super is the superuniques.txt row
// "Siege Boss", displayed "Shenk the Overseer" (string.tbl).
func newSiege() *Quest {
	return newSpecQuest(&spec{
		id: QuestSiege, slot: 35, act: 4, logIndex: 1, name: "Siege on Harrogath", label: "A5Q1", logPage: siegeLogPage,
		start: 1, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 2}, rp: 3, done: 4,
		steps: []step{{from: 1, npc: NPCLarzuk, msg: 20077, to: 2}},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelBloodyFoothills, max: 3, to: 3, page: -1}, // exe 0x5852c0: page 1 when none yet
			{ev: EvMonsterKilled, super: "Shenk the Overseer", goal: true},
		},
		// VERIFIED (Game.exe 0x584d60, msg 20090 = 0x4e7a): the ack clears the progress bits and sets bit 5; it
		// does not set RG or clear RP itself. VERIFIED (0x5856a0, called by Larzuk's item action): the socketing
		// sets bit 0 and clears bit 1 of slot 35 (Game.ClaimSocket); the action needs only RP (SocketOwed).
		hold:    true,
		claimFx: fxs(reward("socket-quest", 1, "Larzuk adds sockets to one item")),
	})
}

// A5Q6 Eve of Destruction (everyone): the Worldstone Keep, the Throne and
// Baal. Available once Rite of Passage is done. Levels 128/131 are the Worldstone Keep 1 and the Throne in Levels.txt (w28 table check); Baal's class is 544 (baalcrab, MonStats.txt; 543 is the throne Baal).
func newEve() *Quest {
	return newSpecQuest(&spec{
		id: QuestEve, slot: 40, act: 4, logIndex: 6, name: "Eve of Destruction", label: "A5Q6",
		start: 0, goal: 3, tbl: map[int]int{1: 0, 2: 0}, rp: 1, done: 3, noLeaveRule: true, kill: eveSpeech(),
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelWorldstone1, max: 1, to: 2},
			{ev: EvAreaChanged, level: LevelThrone, max: 2, to: 2},
			{ev: EvMonsterKilled, monster: NPCBaalCrab, names: []string{"baal"}, min: -1, goal: true, exe: &exeKill{level: LevelWorldstoneChamber, gold: true}},
		},
		claimFx: fxs(reward("unlock-difficulty", 0, "Baal is dead: the next difficulty opens"),
			reward("game-complete", 0, "end of the game")),
	})
}
