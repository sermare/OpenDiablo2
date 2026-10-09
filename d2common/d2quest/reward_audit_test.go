package d2quest

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// Reward audit (d2-re-notes quests-2.md section 3, quests.md, verify-quest-rewards.md). The rules pinned here:
//
//	VERIFIED (Game.exe 1.14b) Den of Evil claim (Akara msg 76, 0x58dba0): RG set, RP cleared, +1 skill point and
//	           the slot-41 respec. Radament (Atma msg 334, 0x596710): RG set, RP cleared, NO reward; the Book of
//	           Skill ("ass", ITEMACT_ServerUseItem 0x55bfd0) gives +1 skill point when read while A2Q1 bit 5 is set.
//	           Lam Esen (Alkor msg 564, 0x5b53e0 + queued 0x5b5220): +5 stat points (stat 4). Golden Bird (Alkor
//	           msg 538, 0x5b7f40): the claim hands out the Potion of Life ("xyz"); drinking it (0x55bfd0, A3Q4 bit
//	           5) adds +20 base max life. Fallen Angel (Tyrael msg 676, 0x5b15c0): +2 skill points (stat 5).
//	           Prison of Ice: Malah's line (msg 20132) gives the Scroll of Resistance ("tr2"); reading it (0x55bfd0,
//	           A5Q3 bit 8 set / bit 7 clear) applies +10 to stats 39/41/43/45 through 0x587f90, summed over the
//	           three difficulty records and re-applied on game join. Siege (Larzuk msg 20090, 0x584d60): no points.
//	           Betrayal, Rite of Passage and Eve of Destruction message handlers (0x589040, 0x58a0c0, 0x58b720) pay
//	           nothing. All rewards are per difficulty record; the amounts do not scale with difficulty.
//	UNRESOLVED where Larzuk's RG is set (the exe's msg handler does not), the exact Anya/Malah RG condition of
//	           Prison of Ice, and the completion/reward code of Rite of Passage and Eve of Destruction (not in ev11).
//
// Load path (VERIFIED, QUESTREC_LoadFromBuffer 0x65e9e0 + QUEST_SyncPlayerOnGameEnter 0x544140): a save load clears
// bits 13/14 and turns RP (bit 1) into COMPLETEDBEFORE (bit 15). The join code then makes every non-prologue
// node with RG or bit 15 inert (active flags +9/+A/+B cleared), so it ignores the unforced events (kills, area
// changes, item events). The NPC click (ev0) and message-acked (ev11) events are dispatched FORCED
// (0x5415c0 with param_1 = 1), and the claim handlers test only the record bits (RP), so a saved pending-reward
// node stays claimable: the engine's choice (quests-2.md) is right, quests.md "inert" is true only for the
// other events.
//
// Invariants for every reward: it is emitted exactly once per difficulty record, only at the claim (the
// reward-pending flag is set by the kill and survives death and save/load), and a reload never re-grants it.

type rewardTally map[string]int

func tally(effects []Effect) rewardTally {
	t := rewardTally{}

	for _, e := range effects {
		switch e.Kind {
		case EffectSkillPoint:
			t["skill-points"] += e.Value
		case EffectReward:
			t[e.Code] += e.Value
		}
	}

	return t
}

func (r rewardTally) equal(want rewardTally) bool {
	for k, v := range want {
		if r[k] != v {
			return false
		}
	}

	for k, v := range r {
		if want[k] != v {
			return false
		}
	}

	return true
}

type rewardCase struct {
	name  string
	id    int
	town  int
	want  rewardTally
	drive func(g *Game) // everything up to, and not including, the claim
	claim func(g *Game) []Effect
	// afterClaim is a grant that happens outside the claim (the Radament Book of Skill).
	afterClaim func(g *Game) []Effect
	afterWant  rewardTally
}

func claimBy(npc int) func(g *Game) []Effect {
	return func(g *Game) []Effect {
		_, eff := talk(g, npc)

		return eff
	}
}

func rewardCases() []rewardCase {
	return []rewardCase{
		{
			name: "Den of Evil", id: QuestDenOfEvil, town: LevelRogueEncampment,
			want: rewardTally{"skill-points": 1},
			drive: func(g *Game) {
				talk(g, NPCAkara)
				moveTo(g, LevelRogueEncampment, 2)
				g.SetDenMonsters(3)
				moveTo(g, 2, LevelDenOfEvil)

				for i := 0; i < 3; i++ {
					g.Dispatch(Event{Kind: EvMonsterKilled, Monster: 1, Level: LevelDenOfEvil})
				}

				moveTo(g, LevelDenOfEvil, LevelRogueEncampment)
			},
			claim: func(g *Game) []Effect { return g.Hear(NPCAkara, 76) },
		},
		{
			name: "Radament claim pays nothing, the book pays 1", id: QuestRadament, town: LevelLutGholein,
			want: rewardTally{},
			drive: func(g *Game) {
				moveTo(g, 39, LevelLutGholein)
				talk(g, NPCAtma)
				moveTo(g, LevelLutGholein, 41)
				kill(g, NPCRadament, LevelSewers3)
				moveTo(g, 41, LevelLutGholein)
			},
			claim:      claimBy(NPCAtma),
			afterClaim: func(g *Game) []Effect { return g.ReadBookOfSkill() },
			afterWant:  rewardTally{"skill-points": 1},
		},
		{
			name: "Lam Esen's Tome", id: QuestLamEsen, town: LevelKurastDocktown,
			want: rewardTally{"stat-points": 5},
			drive: func(g *Game) {
				moveTo(g, 1, LevelKurastDocktown)
				talk(g, NPCAlkor)
				moveTo(g, LevelKurastDocktown, LevelRuinedTemple)
				pickup(g, ItemLamEsenTome)
				moveTo(g, LevelRuinedTemple, LevelKurastDocktown)
			},
			claim: claimBy(NPCAlkor),
		},
		{
			name: "Golden Bird (claim gives the potion, drinking pays +20 life)", id: QuestGoldenBird, town: LevelKurastDocktown,
			want: rewardTally{},
			drive: func(g *Game) {
				moveTo(g, 1, LevelKurastDocktown)
				pickup(g, ItemJadeFigurine)
				talk(g, NPCCain3)
				talk(g, NPCMeshif2)
				// Alkor takes the bird (534); the potion line (538) is the claim
				g.Hear(NPCAlkor, 534)
			},
			claim:      claimBy(NPCAlkor),
			afterClaim: func(g *Game) []Effect { return g.DrinkPotionOfLife() },
			afterWant:  rewardTally{"life-boost": 20},
		},
		{
			name: "Fallen Angel", id: QuestFallenAngel, town: LevelPandemonium,
			want: rewardTally{"skill-points": 2},
			drive: func(g *Game) {
				moveTo(g, 1, LevelPandemonium)
				talk(g, NPCTyrael2)
				moveTo(g, LevelPandemonium, LevelPlainsDespair)
				kill(g, NPCIzual, LevelPlainsDespair)
				moveTo(g, LevelPlainsDespair, LevelPandemonium)
			},
			claim: func(g *Game) []Effect { return g.Hear(NPCTyrael2, 676) },
		},
		{
			name: "Prison of Ice", id: QuestPrison, town: LevelHarrogath,
			want: rewardTally{},
			drive: func(g *Game) {
				moveTo(g, 1, LevelHarrogath)
				talk(g, NPCMalah)
				moveTo(g, LevelHarrogath, LevelFrozenRiver)
				talk(g, NPCAnyaFrozen)
				pickup(g, ItemMalahScroll)
				g.Dispatch(Event{Kind: EvItemRemoved, Item: ItemMalahScroll})
				moveTo(g, LevelFrozenRiver, LevelHarrogath)
			},
			claim:      claimBy(NPCMalah),
			afterClaim: func(g *Game) []Effect { return g.ReadScrollOfResistance() },
			afterWant:  rewardTally{"resist-bonus": 10},
		},
		{
			name: "Siege (sockets, no points)", id: QuestSiege, town: LevelHarrogath,
			want: rewardTally{"socket-quest": 1},
			drive: func(g *Game) {
				moveTo(g, 1, LevelHarrogath)
				talk(g, NPCLarzuk)
				moveTo(g, LevelHarrogath, LevelBloodyFoothills)
				g.Dispatch(Event{Kind: EvMonsterKilled, Monster: 999, Super: "Shenk the Overseer", Level: LevelBloodyFoothills})
				moveTo(g, LevelBloodyFoothills, LevelHarrogath)
			},
			claim: claimBy(NPCLarzuk),
		},
	}
}

func newGameOn(body *d2s.Body, diff int) *Game {
	g := New(body.QuestRecord(diff), body.NPCFlags(), diff)
	g.Hero = Hero{Class: ClassSorceress, Level: 10}
	g.Start()

	return g
}

// TestRewardOnceAtClaim: nothing before the claim, the table's reward at the claim, nothing on a repeat.
func TestRewardOnceAtClaim(t *testing.T) {
	for _, c := range rewardCases() {
		t.Run(c.name, func(t *testing.T) {
			var body d2s.Body

			g := newGameOn(&body, Normal)
			c.drive(g)

			q := g.Quest(c.id)
			if !g.get(q, FlagRewardPending) || g.get(q, FlagRewardGranted) {
				t.Fatalf("before the claim: %s", g.Describe(q))
			}

			if pre := tally(g.TakeEffects()); len(pre) != 0 {
				t.Fatalf("reward paid before the claim: %v", pre)
			}

			if got := tally(c.claim(g)); !got.equal(c.want) {
				t.Fatalf("claim paid %v, want %v", got, c.want)
			}

			if !g.get(q, FlagRewardGranted) || g.get(q, FlagRewardPending) {
				t.Fatalf("after the claim: %s", g.Describe(q))
			}

			// a second claim attempt (same NPC, same line) must not pay again
			if got := tally(c.claim(g)); len(got) != 0 {
				t.Errorf("second claim paid %v", got)
			}

			if c.afterClaim != nil {
				if got := tally(c.afterClaim(g)); !got.equal(c.afterWant) {
					t.Errorf("after-claim grant %v, want %v", got, c.afterWant)
				}

				if got := tally(c.afterClaim(g)); len(got) != 0 {
					t.Errorf("after-claim grant repeated: %v", got)
				}
			}
		})
	}
}

// TestRewardNotRegrantedOnLoad: reloading a record whose quest is claimed must not pay again, whatever
// the hero does afterwards.
func TestRewardNotRegrantedOnLoad(t *testing.T) {
	for _, c := range rewardCases() {
		t.Run(c.name, func(t *testing.T) {
			var body d2s.Body

			g := newGameOn(&body, Normal)
			c.drive(g)
			c.claim(g)

			if c.afterClaim != nil {
				c.afterClaim(g)
			}

			for i := 0; i < 2; i++ { // load twice
				g = newGameOn(&body, Normal)
				if got := tally(g.TakeEffects()); len(got) != 0 {
					t.Fatalf("load %d paid %v", i, got)
				}

				moveTo(g, 1, c.town)

				if got := tally(c.claim(g)); len(got) != 0 {
					t.Errorf("load %d re-claim paid %v", i, got)
				}

				if c.afterClaim != nil {
					if got := tally(c.afterClaim(g)); len(got) != 0 {
						t.Errorf("load %d re-granted the after-claim reward: %v", i, got)
					}
				}
			}
		})
	}
}

// TestRewardSurvivesDeathBeforeClaim: the hero dies (or quits) between the kill and the claim. The pending
// flag is saved, the reload keeps it claimable, and the reward is flagged given exactly once.
func TestRewardSurvivesDeathBeforeClaim(t *testing.T) {
	for _, c := range rewardCases() {
		t.Run(c.name, func(t *testing.T) {
			var body d2s.Body

			g := newGameOn(&body, Normal)
			c.drive(g)

			// "death + reload": a fresh Game over the same saved record
			g = newGameOn(&body, Normal)

			q := g.Quest(c.id)
			if !g.get(q, FlagRewardPending) {
				t.Fatalf("reward-pending lost on load: %s", g.Describe(q))
			}

			// quest items the hero still carries after the reload
			switch c.id {
			case QuestLamEsen:
				g.Items[ItemLamEsenTome] = 1
			case QuestGoldenBird:
				g.Items[ItemGoldenBird] = 1
			}

			moveTo(g, 1, c.town)

			if got := tally(c.claim(g)); !got.equal(c.want) {
				t.Fatalf("claim after reload paid %v, want %v (%s)", got, c.want, g.Describe(q))
			}

			if !g.get(q, FlagRewardGranted) || g.get(q, FlagRewardPending) {
				t.Fatalf("after the claim: %s", g.Describe(q))
			}

			if got := tally(c.claim(g)); len(got) != 0 {
				t.Errorf("claimed twice: %v", got)
			}
		})
	}
}

// TestRewardPerDifficulty: each difficulty owns a record, so every reward is paid once per difficulty
// (the skill-point quests give their points again in Nightmare and Hell) and the amounts do not scale.
func TestRewardPerDifficulty(t *testing.T) {
	for _, c := range rewardCases() {
		t.Run(c.name, func(t *testing.T) {
			var body d2s.Body

			for _, diff := range []int{Normal, Nightmare, 2} {
				g := newGameOn(&body, diff)
				c.drive(g)

				if got := tally(c.claim(g)); !got.equal(c.want) {
					t.Fatalf("difficulty %d claim paid %v, want %v", diff, got, c.want)
				}

				if c.afterClaim != nil {
					if got := tally(c.afterClaim(g)); !got.equal(c.afterWant) {
						t.Fatalf("difficulty %d after-claim %v, want %v", diff, got, c.afterWant)
					}
				}
			}

			g := newGameOn(&body, Normal)
			if !g.get(g.Quest(c.id), FlagRewardGranted) {
				t.Errorf("normal record lost its flag")
			}
		})
	}
}
