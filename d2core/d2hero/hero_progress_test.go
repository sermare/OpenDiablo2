package d2hero

import (
	"encoding/json"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func TestHeroProgressJSONRoundTrip(t *testing.T) {
	p := &HeroProgress{}
	p.Quests[1].SetCompleted(2, 3)
	p.Waypoints.Set(1, d2s.WPLutGholein, true)
	p.NPC.SetReturnBit(0, 4, true)

	data, err := json.Marshal(&HeroState{HeroName: "x", Progress: p})
	if err != nil {
		t.Fatal(err)
	}

	var got HeroState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	if !got.Progress.QuestRecord(1).Completed(2, 3) || got.Progress.QuestRecord(0).Completed(2, 3) ||
		!got.Progress.Waypoints.Has(1, d2s.WPLutGholein) || !got.Progress.NPC.ReturnBit(0, 4) {
		t.Fatalf("progress lost in round trip: %s", data)
	}
}

func TestHeroStateWithoutProgressIsBackwardsCompatible(t *testing.T) {
	var got HeroState

	if err := json.Unmarshal([]byte(`{"heroName":"old","act":1}`), &got); err != nil {
		t.Fatal(err)
	}

	if got.Progress != nil || got.Progress.QuestRecord(0) != nil {
		t.Fatal("old hero files must load with nil progress")
	}

	data, _ := json.Marshal(&got)
	if string(data) == "" || json.Valid(data) == false {
		t.Fatal("marshal")
	}

	var m map[string]json.RawMessage

	_ = json.Unmarshal(data, &m)

	if _, ok := m["progress"]; ok {
		t.Fatal("nil progress should be omitted")
	}
}
