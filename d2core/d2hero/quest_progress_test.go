package d2hero

import (
	"encoding/json"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
)

// playDenToTheCave drives the quest system through the first steps of Den of Evil.
func playDenToTheCave(p *HeroProgress) *d2quest.Game {
	g := d2quest.New(p.QuestRecord(0), &p.NPC, 0)
	g.Hero = d2quest.Hero{Class: d2quest.ClassPaladin, Level: 5}
	g.Start()

	g.Activate(d2quest.NPCAkara)
	g.Hear(d2quest.NPCAkara, 11) // her first-meeting line
	g.Hear(d2quest.NPCAkara, 64) // the quest
	g.Close(d2quest.NPCAkara)
	g.Dispatch(d2quest.Event{Kind: d2quest.EvAreaChanged, OldLevel: 1, NewLevel: 2})
	g.Dispatch(d2quest.Event{Kind: d2quest.EvAreaChanged, OldLevel: 2, NewLevel: d2quest.LevelDenOfEvil})

	return g
}

func checkDenRestored(t *testing.T, p *HeroProgress) {
	t.Helper()

	g := d2quest.New(p.QuestRecord(0), &p.NPC, 0)
	g.Start()

	den := g.Quest(d2quest.QuestDenOfEvil)
	if den.State != 3 || den.LastState != 2 {
		t.Fatalf("Den of Evil not restored: state %d last %d: %s", den.State, den.LastState, g.Describe(den))
	}

	if !g.QuestIntroDone(d2quest.NPCAkara) {
		t.Error("Akara's intro flag lost")
	}

	if g.Quest(d2quest.QuestBurial).State != 0 {
		t.Error("the next quest must not be available")
	}
}

func TestQuestProgressSurvivesTheHeroFile(t *testing.T) {
	state := &HeroState{HeroName: "q", Progress: &HeroProgress{}}
	playDenToTheCave(state.Progress)

	data, err := json.Marshal(state) // the .od2 save and the save/add-player packets
	if err != nil {
		t.Fatal(err)
	}

	var back HeroState
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}

	checkDenRestored(t, back.Progress)
}

func TestQuestProgressSurvivesTheD2SExport(t *testing.T) {
	data, tables, state := realSave(t)

	rec := state.Progress.QuestRecord(0)
	for slot := 0; slot < d2s.QuestSlots; slot++ { // start from a clean quest block
		rec.SetSlot(slot, 0)
	}

	state.Progress.NPC = d2s.NPCBlock{}

	playDenToTheCave(state.Progress)

	out, _, err := ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}

	got, err := d2s.Parse(out, tables)
	if err != nil {
		t.Fatal(err)
	}

	checkDenRestored(t, &HeroProgress{Quests: quests(got.Body), Waypoints: got.Body.Waypoints, NPC: *got.Body.NPCFlags()})
}
