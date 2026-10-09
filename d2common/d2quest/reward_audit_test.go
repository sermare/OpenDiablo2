package d2quest

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// Reward audit (d2-re-notes quests-2.md section 3, quests.md). The rules pinned here:
//
//	VERIFIED   (binary) Den of Evil claim (Akara msg 76): RG set, RP cleared, +1 skill point and the slot-41 respec.
//	SOURCE     (D2MOO, not re-derived in the binary) Radament +1 skill point when the Book of Skill is read (not
//	           at the claim), Fallen Angel +2 skill points, Lam Esen +5 stat points, Golden Bird +20 life,
//	           Prison of Ice +10 all resistances (the difficulty penalty is applied separately by d2difficulty).
//	UNVERIFIED the exact handler addresses of the A2Q1/A3Q1/A3Q4/A4Q1/A5Q3 reward code.
//
// Exe addresses still to confirm (Game.exe 1.14b; the quest node init functions are reached from the static
// tables at 0x72F000 / 0x72F37C, QUEST_CreateGameQuests): the ev11 (message acked) handler of each of
// A2Q1 (Atma 334 / Book of Skill use), A3Q1 (Alkor 564), A3Q4 (Alkor 538, life), A4Q1 (Tyrael 676, 2 points),
// A5Q3 (Malah/Anya scroll read: resist +10 and when it is applied), A5Q1 (Larzuk sockets), plus the item-use
// code that adds STAT_SKILLPTS / STAT_NEWSKILLS for the Book of Skill, and the load path QUESTREC_LoadFromBuffer
// 0x65e9e0 + QUEST_SyncPlayerOnGameEnter 0x544140 (does a saved RP+bit15 node still accept its claim line?
// the engine assumes yes, because the Den of Evil handler 0x543490/ev11 clearly does).
// UNVERIFIED engine choices: the Prison of Ice resist bonus is only logged (no quest-resist stat slot), and
// it is paid at Malah's claim, whereas the original applies it when the scroll is read.
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
			name: "Golden Bird (Potion of Life)", id: QuestGoldenBird, town: LevelKurastDocktown,
			want: rewardTally{"life-boost": 20},
			drive: func(g *Game) {
				moveTo(g, 1, LevelKurastDocktown)
				pickup(g, ItemJadeFigurine)
				talk(g, NPCCain3)
				talk(g, NPCMeshif2)
				// Alkor takes the bird (534); the potion line (538) is the claim
				g.Hear(NPCAlkor, 534)
			},
			claim: claimBy(NPCAlkor),
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
			want: rewardTally{"resist-bonus": 10},
			drive: func(g *Game) {
				moveTo(g, 1, LevelHarrogath)
				talk(g, NPCMalah)
				moveTo(g, LevelHarrogath, LevelFrozenRiver)
				talk(g, NPCAnyaFrozen)
				pickup(g, ItemMalahScroll)
				g.Dispatch(Event{Kind: EvItemRemoved, Item: ItemMalahScroll})
				moveTo(g, LevelFrozenRiver, LevelHarrogath)
			},
			claim: claimBy(NPCMalah),
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
