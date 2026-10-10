package d2reward

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
)

// The reward NPCs through the quest system: a finished quest whose reward waits
// (RP and PGD set in its slot, like after the kill or a reload), the quest
// giver's Talk speaks the reward line, and the claim yields the effect the
// engine turns into a menu row (Larzuk, Anya, Charsi, Akara), a hire row
// (Kashya, Qual-Kehk) or points.
func TestRewardNPCsClaimThroughTalk(t *testing.T) {
	tests := []struct {
		name  string
		act   int // d2s act (1..5)
		quest int // quest number inside the act (1..)
		npc   int
		items map[string]int
		want  func(t *testing.T, effects []d2quest.Effect, s *State)
	}{
		{
			name: "Akara: Den of Evil skill point and the reset", act: 1, quest: 1, npc: d2quest.NPCAkara,
			want: func(t *testing.T, effects []d2quest.Effect, s *State) {
				if !has(effects, d2quest.EffectSkillPoint, "") || !has(effects, d2quest.EffectRespec, "") {
					t.Errorf("effects %+v", effects)
				}
			},
		},
		{
			name: "Kashya: the rogues become hirable", act: 1, quest: 2, npc: d2quest.NPCKashya,
			want: func(t *testing.T, effects []d2quest.Effect, s *State) {
				if !has(effects, d2quest.EffectHireRogues, "") {
					t.Errorf("effects %+v", effects)
				}
			},
		},
		{
			name: "Charsi: the imbue", act: 1, quest: 3, npc: d2quest.NPCCharsi, items: map[string]int{d2quest.ItemHoradricMalus: 1},
			want: func(t *testing.T, effects []d2quest.Effect, s *State) {
				if !has(effects, d2quest.EffectImbue, "") {
					t.Errorf("effects %+v", effects)
				}
			},
		},
		{
			name: "Larzuk: sockets owed", act: 5, quest: 1, npc: d2quest.NPCLarzuk,
			want: func(t *testing.T, effects []d2quest.Effect, s *State) {
				if s.SocketPending != 1 || !OwedFrom(s, false).Of(KindSocket) {
					t.Errorf("socket pending %d effects %+v", s.SocketPending, effects)
				}
			},
		},
		{
			name: "Qual-Kehk: the barbarians become hirable", act: 5, quest: 2, npc: d2quest.NPCQualKehk,
			want: func(t *testing.T, effects []d2quest.Effect, s *State) {
				if !s.Hired["barbarians"] {
					t.Errorf("hired %v effects %+v", s.Hired, effects)
				}
			},
		},
		{
			name: "Anya: personalisation owed", act: 5, quest: 4, npc: d2quest.NPCDrehya,
			want: func(t *testing.T, effects []d2quest.Effect, s *State) {
				if s.PersonalizePending != 1 {
					t.Errorf("personalize pending %d effects %+v", s.PersonalizePending, effects)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rec d2s.QuestRecord

			g := d2quest.New(&rec, &d2s.NPCBlock{}, 0)
			g.Expansion = true
			g.Hero.Level = 30

			for code, n := range tt.items {
				g.Items[code] = n
			}

			slot, ok := d2s.QuestSlot(tt.act, tt.quest)
			if !ok {
				t.Fatalf("no slot for %d/%d", tt.act, tt.quest)
			}

			g.Start()

			if tt.act == 5 { // arriving in Harrogath opens the Act 5 quests
				g.Dispatch(d2quest.Event{Kind: d2quest.EvAreaChanged, OldLevel: 1, NewLevel: d2quest.LevelHarrogath})
			}

			rec.Set(slot, d2quest.FlagRewardPending)
			rec.Set(slot, d2quest.FlagPrimaryGoal)

			var (
				effects []d2quest.Effect
				state   State
			)

			for i := 0; i < 12; i++ {
				s, more := g.Activate(tt.npc).NextSpoken(g)
				if !more {
					break
				}

				effects = append(effects, g.Hear(tt.npc, s.Msg)...)
			}

			effects = append(effects, g.Close(tt.npc)...)

			for _, e := range effects {
				if e.Kind == d2quest.EffectReward {
					state.Apply(e)
				}
			}

			tt.want(t, effects, &state)
		})
	}
}

func has(effects []d2quest.Effect, kind d2quest.EffectKind, code string) bool {
	for _, e := range effects {
		if e.Kind == kind && (code == "" || e.Code == code) {
			return true
		}
	}

	return false
}
