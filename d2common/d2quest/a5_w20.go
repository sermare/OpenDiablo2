package d2quest

// Wave 20 rewrite of the Act 5 quests whose rewards the earlier table-driven version got wrong: Rescue on Mount
// Arreat (A5Q2), Prison of Ice (A5Q3), Betrayal of Harrogath (A5Q4) and Rite of Passage (A5Q5). Siege on Harrogath
// (A5Q1) stays in a345.go with the `hold` option.
//
// Evidence. VERIFIED means read in Game.exe 1.14b (read-only Ghidra):
//
//	Siege (0x584d60, 0x5856a0): the thanks line (20090) clears the progress bits and sets bit 5 but leaves "reward
//	  pending"; Larzuk's item action then sets bit 0 and clears bit 1 of slot 35 (Game.ClaimSocket).
//	Prison of Ice message handler (0x587460, verify-quest-rewards.md): Malah 20127 gives the Malah potion "ice",
//	  20132 gives the Scroll of Resistance "tr2" and sets bit 8, Anya (Drehya) 20136 gives an item and sets bits
//	  9/10; the quest is rewarded (bit 0 set, bit 1 cleared) once both halves happened, in either order. The scroll
//	  itself gives the resistances when read (Game.ReadScrollOfResistance).
//	Rite of Passage (0x58a7b0, 0x58a3d0): the three Ancients (classes 540-542) must die with the hero in the
//	  summit and the hero level at least 20 * (difficulty + 1); the last death sets bits 0 and 0xd (no reward
//	  pending) and pays experience: 1,400,000 / 20,000,000 / 40,000,000, capped to the experience span of the
//	  current level.
//
// VERIFIED in w25 (Game.exe 1.14b): Qual-Kehk's 20110 (0x5859d0) needs RP, gives r07/r08/r09 (table 0x731780) by
// slot-36 bit 6 (two runes), bit 7 (one rune), else three, and only then sets RG / clears RP; Anya's item tables
// (0x733404 Nightmare above clvl 45, 0x73343c Hell above clvl 65, 0x733474 otherwise; per class, rare, ilvl = hero
// level) match anyaRewards; Malah hands the potion out once per game (0x587460); the Betrayal kill (lambda 0x589300)
// sets RP + goal only for heroes near the corpse whose Prison of Ice is rewarded or pending, others get bit 14.
// w27: the "game type == 3" Anya test is explained in w27_quests2.go (AnyaTier); the Cain rescue, the Ancients and the
// Baal gold are there too. w37 VERIFIED for Rite of Passage: OnAreaChange 0x58a9a0 (entering Harrogath, level 0x6d, with
// state 2 and bit 0 clear moves it to state 3; entering the summit 0x78 raises it to state 3), OnNpcMessageHeard 0x58a0c0
// (message 0x4eb9 from NPC 0x203 moves state 1 to 2; message 0x4e22 starts the Ancients). UNVERIFIED: the availability
// triggers of the other Act 5 quests.

// ObjectFrozenAnya is the frozen Anya block (objects.txt row 558, OperateFn 67): operating it with Malah's
// potion thaws her.
const ObjectFrozenAnya = 558

// Levels of the Act 5 triggers (levels.txt).
const (
	LevelCrystallinePassage = 113
)

// ItemDefrostPotion is Malah's potion ("ice").
const ItemDefrostPotion = ItemMalahPotion

// RiteExperience is the experience Rite of Passage pays by difficulty (VERIFIED 0x58a3d0). The engine grants it
// capped to the span of the hero's current level.
func RiteExperience(difficulty int) int {
	switch difficulty {
	case Nightmare:
		return 20000000
	case Hell:
		return 40000000
	}

	return 1400000
}

// RiteLevelGate is the hero level Rite of Passage needs (VERIFIED 0x58a7b0: 0x14 * (difficulty + 1)).
func RiteLevelGate(difficulty int) int { return 20 * (difficulty + 1) }

// ---- A5Q2 Rescue on Mount Arreat ----

type rescueData struct {
	doors, freed, killed int
}

// Rune codes of the Rescue reward: Tal, Ral, Ort.
//
//nolint:gochecknoglobals // static lookup data
var rescueRunes = []string{"r07", "r08", "r09"}

// A5Q2 Rescue on Mount Arreat (Qual-Kehk): free the barbarian groups by killing the prison doors (class 434, three
// groups of five). Twelve or more freed barbarians finish the goal; Qual-Kehk's thanks (20110) pays one rune per
// ... 15 freed: three runes, 14: two, 12-13: one (D2MOO ACT5Q2_Callback11_ScrollMessage). Event.Freed carries the
// number of barbarians the door freed (0: the five of a full group).
func newRescue() *Quest {
	d := &rescueData{}

	return newSpecQuest(&spec{
		id: QuestRescue, slot: 36, act: 4, logIndex: 2, name: "Rescue on Mount Arreat", label: "A5Q2",
		start: 0, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 2}, rp: 3, done: 4,
		steps: []step{{from: 1, npc: NPCQualKehk, msg: 20096, to: 2}},
		trigs: []trig{
			// exe 0x5860f0: levels 111 and 112 both count, page 1 when none yet
			{ev: EvAreaChanged, level: LevelFrigidHighlands, max: 3, to: 3, page: -1},
			{ev: EvAreaChanged, level: LevelArreatPlateau, max: 3, to: 3, page: -1},
			{ev: EvMonsterKilled, monster: NPCPrisonDoor, count: 3, goal: true,
				cond: func(g *Game, q *Quest) bool { return d.doors < 3 || (d.freed >= 12 && d.killed < 5) },
				fx: func(g *Game, q *Quest) []Effect {
					switch {
					case d.freed >= 15:
						g.set(q, FlagCustom1, "all fifteen barbarians freed")
					case d.freed == 14:
						g.set(q, FlagCustom2, "fourteen barbarians freed")
					default:
						g.set(q, FlagCustom3, "twelve or thirteen barbarians freed")
					}

					return nil
				}},
		},
		setup: func(q *Quest, _ *spec) {
			inner := q.on[EvMonsterKilled]
			q.on[EvMonsterKilled] = func(g *Game, q *Quest, e *Event) {
				// a freed barbarian that died (VERIFIED 0x585f30: five dead ones fail the quest)
				if e.Monster == NPCWussie && q.NotIntro && !g.grantedOrPending(q) {
					d.killed++
				}

				if e.Monster == NPCPrisonDoor && q.NotIntro && !g.grantedOrPending(q) {
					n := e.Freed
					switch {
					case n < 0: // the door freed nobody
						n = 0
					case n == 0: // the scripted walks do not say: a full group
						n = 5
					}

					d.doors++
					d.freed += n
				}

				inner(g, q, e)
			}
		},
		claimFx: func(g *Game, q *Quest) []Effect {
			n := 3

			switch {
			case g.get(q, FlagCustom2):
				n = 2
			case g.get(q, FlagCustom3):
				n = 1
			}

			g.resetProgress(q, "Qual-Kehk paid the runes")

			out := make([]Effect, 0, n)
			for _, code := range rescueRunes[:n] {
				out = append(out, Effect{Kind: EffectGiveItem, Quest: q.ID, Code: code,
					Note: "Qual-Kehk's rune reward (UNVERIFIED in the exe, D2MOO ACT5Q2)"})
			}

			return out
		},
	})
}

// ---- A5Q3 Prison of Ice ----

// Anya's reward lists by class (.d2s class order) and tier: normal, exceptional, elite (D2MOO; UNVERIFIED in the
// exe). The assassin has seven entries per tier, the others five.
//
//nolint:gochecknoglobals // static lookup data
var anyaRewards = [7][3][]string{
	{{"am1", "am2", "am3", "am4", "am5"}, {"am6", "am7", "am8", "am9", "ama"}, {"amb", "amc", "amd", "ame", "amf"}},
	{{"ob1", "ob2", "ob3", "ob4", "ob5"}, {"ob6", "ob7", "ob8", "ob9", "oba"}, {"obb", "obc", "obd", "obe", "obf"}},
	{{"ne1", "ne2", "ne3", "ne4", "ne5"}, {"ne6", "ne7", "ne8", "ne9", "nea"}, {"neb", "nec", "ned", "nee", "nef"}},
	{{"pa1", "pa2", "pa3", "pa4", "pa5"}, {"pa6", "pa7", "pa8", "pa9", "paa"}, {"pab", "pac", "pad", "pae", "paf"}},
	{{"ba1", "ba2", "ba3", "ba4", "ba5"}, {"ba6", "ba7", "ba8", "ba9", "baa"}, {"bab", "bac", "bad", "bae", "baf"}},
	{{"dr1", "dr2", "dr3", "dr4", "dr5"}, {"dr6", "dr7", "dr8", "dr9", "dra"}, {"drb", "drc", "drd", "dre", "drf"}},
	{
		{"ktr", "wrb", "axf", "ces", "clw", "btl", "skr"},
		{"9ar", "9wb", "9xf", "9cs", "9lw", "9tw", "9qr"},
		{"7ar", "7wb", "7xf", "7cs", "7lw", "7tw", "7qr"},
	},
}

// AnyaRewardCodes lists the items Anya may give a hero of a class on a difficulty (D2MOO ACT5Q3_Callback11): the
// exceptional tier from Nightmare with a hero above level 45, the elite tier in Hell above level 65.
func AnyaRewardCodes(class, difficulty, heroLevel int) []string {
	return AnyaRewardCodesFor(class, difficulty, heroLevel, GameTypeHost)
}

type prisonData struct {
	scrollRegiven bool
	potionGiven   bool // Malah hands the potion out once per game (VERIFIED 0x587460: counter at node data +0xd8)
	malahIntro    bool
	rewarded      bool // heard the closing lines in this session (the player GUID list)
}

// Bits of the Prison of Ice record: 7 scroll read, 8 scroll given, 9 Anya's item given, 10 Anya's lock.
const (
	prisonScrollRead  = FlagCustom3
	prisonScrollGiven = FlagCustom4
	prisonAnyaGave    = FlagCustom5
	prisonAnyaLock    = FlagCustom6
)

func newPrison() *Quest {
	d := &prisonData{}
	q := &Quest{ID: QuestPrison, Slot: 37, Act: 4, Name: "Prison of Ice", Label: "A5Q3", LogIndex: 3,
		Active: true, NotIntro: true, State: 0, InitNo: 5, SeqID: QuestBetrayal, tables: speechTables("A5Q3")}

	q.seq = chainSeq(true, 5)

	complete := func(g *Game, q *Quest, why string) {
		d.rewarded = true

		g.clear(q, FlagRewardPending, why)
		g.set(q, FlagRewardGranted, why)
		g.globalDone(q)
		q.LastState = 13
		g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 13})
		g.chainFrom(q)
	}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		if npc == NPCAnyaFrozen {
			// the original sends the thawing hint as a scroll message when the frozen block is operated without the
			// potion (OperateFn 67); the dialog gives the same line
			if q.NotIntro && q.State >= 3 && q.State < 5 && !g.hasItem(ItemDefrostPotion) {
				return q.pick(npc, 3)
			}

			return nil
		}

		if npc == NPCMalah {
			if q.State == 4 {
				if !g.hasItem(ItemDefrostPotion) {
					return q.pick(npc, 3)
				}

				return nil
			}

			if (g.get(q, FlagRewardGranted) || g.get(q, FlagCompletedEarly)) && !d.scrollRegiven &&
				g.get(q, prisonScrollGiven) && !g.get(q, prisonScrollRead) && !g.hasItem(ItemMalahScroll) {
				return q.pick(npc, 5)
			}
		}

		if g.get(q, FlagRewardPending) {
			if (npc == NPCDrehya && g.get(q, prisonAnyaGave)) || (npc == NPCMalah && g.get(q, prisonScrollGiven)) {
				return nil
			}

			return q.pick(npc, 5)
		}

		switch {
		case d.rewarded:
			return q.pick(npc, 6)
		case g.get(q, FlagRewardGranted), (q.State >= 5 && !g.get(q, FlagPrimaryGoal)), !q.NotIntro:
			return nil
		case q.State >= 1 && q.State <= 6:
			return q.pick(npc, q.State-1)
		}

		return nil
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		switch npc {
		case NPCMalah:
			if g.get(q, FlagRewardPending) && !g.get(q, prisonScrollGiven) {
				return true
			}

			if g.get(q, FlagRewardGranted) || g.get(q, FlagRewardPending) {
				return false
			}

			return q.State == 1 || (q.State == 4 && !g.hasItem(ItemDefrostPotion))
		case NPCAnyaFrozen:
			return q.NotIntro && q.State < 5 && !g.hasItem(ItemDefrostPotion)
		case NPCDrehya:
			return g.get(q, FlagRewardPending) && !g.get(q, prisonAnyaGave)
		}

		return false
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		if e.NewLevel == LevelArreatPlateau && q.NotIntro && q.State == 0 {
			g.setState(q, 1)
		}

		// VERIFIED (0x587c70): entering level 0x71/0x72 moves the state to 3 when it was below, and an unset log page
		// becomes 1 (LastState 1 "Look for Anya", sent to the client)
		if (e.NewLevel == LevelCrystallinePassage || e.NewLevel == LevelFrozenRiver) && q.NotIntro {
			if q.State <= 2 {
				g.setState(q, 3)
				g.set(q, FlagLeaveTown, "entered the ice caves")
			}

			prisonFirstPage(g, q)
		}

		if e.OldLevel == LevelHarrogath && q.State == 2 && !g.grantedOrPending(q) {
			g.setState(q, 3)
			g.set(q, FlagLeaveTown, "left town")
			prisonFirstPage(g, q)
		}
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		switch {
		case e.NPC == NPCMalah && e.Msg == 20116:
			g.setState(q, 2)
			g.set(q, FlagStarted, "Malah asked for Anya")
		case e.NPC == NPCMalah && e.Msg == 20127:
			// the potion is given once (the original refuses when the hero or a party member carries one)
			if !g.hasItem(ItemDefrostPotion) && !d.potionGiven {
				d.potionGiven = true

				g.emit(Effect{Kind: EffectGiveItem, Quest: q.ID, Code: ItemDefrostPotion, Note: "Malah's potion thaws Anya"})
			}
		case e.Msg == 20131 && q.NotIntro:
			if q.State <= 3 {
				g.setState(q, 4)
			}

			// VERIFIED (0x587460, message 0x4ea3): below page 3 the log goes to page 3 (the potion is next)
			if q.LastState < 3 {
				g.cycle(q, 3, true)
			}
		case e.NPC == NPCMalah && e.Msg == 20132:
			g.malahLine(q, d, complete)
		case e.NPC == NPCDrehya && e.Msg == 20136:
			g.anyaLine(q, complete)
		}
	}

	// VERIFIED shape (OperateFn 67): the frozen Anya block melts when the hero carries Malah's potion.
	q.on[EvObjectOperated] = func(g *Game, q *Quest, e *Event) {
		if e.Object != ObjectFrozenAnya || g.grantedOrPending(q) || !g.hasItem(ItemDefrostPotion) {
			return
		}

		g.dropItem(ItemDefrostPotion)
		g.emit(Effect{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemDefrostPotion, Note: "the potion thaws Anya"})
		g.setState(q, 5)
		g.set(q, FlagPrimaryGoal, "Anya is thawed")
		g.set(q, FlagRewardPending, "Anya is thawed")
		g.globalDone(q)
		q.LastState = 5
		g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: g.LogPage(q)})
		g.chainFrom(q)
	}

	q.status = func(g *Game, q *Quest) int {
		switch {
		case g.get(q, FlagRewardGranted):
			return 0
		case g.get(q, FlagRewardPending) || g.get(q, FlagPrimaryGoal):
			switch {
			case g.get(q, prisonScrollGiven) && !g.get(q, prisonAnyaGave):
				return 6
			case !g.get(q, prisonScrollGiven):
				return 5
			}

			return 13
		case !q.NotIntro:
			return 0
		case g.hasItem(ItemDefrostPotion):
			return 4
		case g.get(q, FlagCompletedNow):
			return 12
		case q.State < 5:
			return q.LastState
		}

		return 0
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, _ *Event) {
		switch {
		case g.get(q, FlagRewardGranted):
			g.globalDone(q)

			return
		case !q.NotIntro:
			return
		case g.get(q, FlagLeaveTown):
			q.State, q.LastState = 3, 1
		case g.get(q, FlagStarted):
			q.State, q.LastState = 2, 1
		}

		if g.hasItem(ItemDefrostPotion) {
			q.State, q.LastState = 3, 4
		}
	}

	return q
}

// prisonFirstPage is the log page 1 the exe sets when the hero goes to the Crystalline Passage / Frozen River, or comes
// back to Harrogath with the quest at state 2, while no page is pushed yet (0x587c70).
func prisonFirstPage(g *Game, q *Quest) {
	if q.LastState == 0 && !g.grantedOrPending(q) {
		g.cycle(q, 1, true)
	}
}

// malahLine is Malah's 20132: the Scroll of Resistance. The quest is rewarded when Anya's item was given too.
func (g *Game) malahLine(q *Quest, d *prisonData, complete func(*Game, *Quest, string)) {
	if !g.get(q, FlagRewardPending) || g.get(q, FlagRewardGranted) {
		// a hero who lost the scroll gets it again once per game (VERIFIED shape, D2MOO)
		if (g.get(q, FlagRewardGranted) || g.get(q, FlagCompletedEarly)) && !d.scrollRegiven && g.get(q, prisonScrollGiven) &&
			!g.get(q, prisonScrollRead) && !g.hasItem(ItemMalahScroll) {
			d.scrollRegiven = true

			g.emit(Effect{Kind: EffectGiveItem, Quest: q.ID, Code: ItemMalahScroll, Note: "Malah gives the Scroll of Resistance again"})
		}

		return
	}

	g.emit(Effect{Kind: EffectGiveItem, Quest: q.ID, Code: ItemMalahScroll, Note: "Malah gives the Scroll of Resistance"})
	g.set(q, prisonScrollGiven, "Malah gave the scroll")

	if g.get(q, prisonAnyaGave) {
		complete(g, q, "the scroll and Anya's item were given")

		return
	}

	if q.NotIntro && q.LastState < 6 {
		q.LastState = 6
		g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: g.LogPage(q)})
	}
}

// anyaLine is Anya's 20136: a rare item of her choice for the hero's class. The quest is rewarded when Malah's
// scroll was given too.
func (g *Game) anyaLine(q *Quest, complete func(*Game, *Quest, string)) {
	if !g.get(q, FlagRewardPending) || g.get(q, FlagRewardGranted) || g.get(q, prisonAnyaLock) {
		return
	}

	codes := AnyaRewardCodes(g.Hero.Class, g.Difficulty, g.Hero.Level)
	if len(codes) == 0 {
		return
	}

	code := codes[g.Rand.Intn(len(codes))]
	g.emit(Effect{Kind: EffectGiveItem, Quest: q.ID, Code: code, Quality: 2, Value: g.Hero.Level,
		Note: "Anya's rare class item (UNVERIFIED lists, D2MOO)"})
	g.set(q, prisonAnyaGave, "Anya gave her item")

	if g.get(q, prisonScrollGiven) {
		complete(g, q, "the scroll and Anya's item were given")
	}

	g.set(q, prisonAnyaLock, "Anya's item is given once")
}

// ---- A5Q4 Betrayal of Harrogath ----

// A5Q4 Betrayal of Harrogath (Anya): Nihlathak in his temple. VERIFIED (0x589040) Anya's 20148 pays nothing; the
// reward is her personalising an item, which sets the done bits (Game.ClaimPersonalize, D2MOO
// ACT5Q4_SetRewardGranted). D2MOO: Nihlathak's death only counts when Prison of Ice is done or reward pending;
// Anya opens the portal to his temple after her first line.
func newBetrayal() *Quest {
	q := newSpecQuest(&spec{
		id: QuestBetrayal, slot: 38, act: 4, logIndex: 4, name: "Betrayal of Harrogath", label: "A5Q4", logPage: betrayalLogPage,
		start: 0, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 2}, rp: 3, done: 4,
		steps: []step{{from: 1, npc: NPCDrehya, msg: 20137, to: 2, fx: func(g *Game, q *Quest) []Effect {
			return []Effect{{Kind: EffectPortal, Quest: q.ID, Value: LevelNihlathakTemple, Note: "Anya opens the portal to Nihlathak's Temple"}}
		}}},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelNihlathakTemple, max: 3, to: 3, bit: -1}, // VERIFIED 0x589900: only the state changes, bit 4 is Anya's
			{ev: EvMonsterKilled, monster: NPCNihlathakBoss, goal: true, cond: func(g *Game, q *Quest) bool {
				p := g.byID[QuestPrison]

				return BetrayalEligible(p == nil || g.get(p, FlagRewardGranted) || g.get(p, FlagRewardPending),
					g.get(q, betrayalMissedBit))
			}},
		},
		hold: true, holdBit: FlagEnterArea, holdKeep: true,
	})
	q.on[EvMonsterKilled] = betrayalMissed(q.on[EvMonsterKilled])

	return q
}

// betrayalMissed wraps the kill handler: after the grant rules ran, a hero that has neither the done bit, reward
// pending nor the goal bit gets bit 14 (VERIFIED lambda 0x589530).
func betrayalMissed(next func(*Game, *Quest, *Event)) func(*Game, *Quest, *Event) {
	return func(g *Game, q *Quest, e *Event) {
		if next != nil {
			next(g, q, e)
		}

		if e.Monster != NPCNihlathakBoss || g.get(q, FlagRewardGranted) || g.get(q, FlagRewardPending) ||
			g.get(q, FlagPrimaryGoal) {
			return
		}

		g.set(q, betrayalMissedBit, "missed Nihlathak's kill: bit 14 (VERIFIED 0x589530)")
	}
}

// ---- A5Q5 Rite of Passage ----

// A5Q5 Rite of Passage (Qual-Kehk): the three Ancients on the summit. VERIFIED (0x58a7b0, 0x58a3d0): the third
// death with the hero at least level 20 / 40 / 60 completes the quest at once and pays experience. The statues
// speak while the quest runs.
func newRite() *Quest {
	return newSpecQuest(&spec{
		id: QuestRite, slot: 39, act: 4, logIndex: 5, name: "Rite of Passage", label: "A5Q5",
		start: 0, goal: 4, tbl: map[int]int{1: 0, 2: 1, 3: 2}, rp: 5, done: 3,
		steps: []step{{from: 1, npc: NPCQualKehk, msg: 20153, to: 2}},
		trigs: []trig{
			{ev: EvAreaChanged, level: LevelArreatSummit, max: 3, to: 3, page: -1}, // exe 0x58a9a0: page 1 when below 2
			{ev: EvMonsterKilled, monsters: []int{NPCAncient1, NPCAncient2, NPCAncient3}, count: 3, goal: true, direct: true,
				cond: func(g *Game, q *Quest) bool { return g.Hero.Level >= RiteLevelGate(g.Difficulty) },
				fx: func(g *Game, q *Quest) []Effect {
					return []Effect{{Kind: EffectReward, Quest: q.ID, Code: "experience", Value: RiteExperience(g.Difficulty),
						Note: "Rite of Passage: experience, capped to the span of the current level (VERIFIED 0x58a3d0)"}}
				}},
		},
		extra: func(g *Game, q *Quest, npc int) []Speech {
			if q.State == 3 && npc >= NPCAncientStatue1 && npc <= NPCAncientStatue3 {
				return q.pick(npc, 4)
			}

			return nil
		},
	})
}
