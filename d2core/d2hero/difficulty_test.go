package d2hero

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2difficulty"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func TestChooseDifficultyNeedsUnlock(t *testing.T) {
	h := &HeroState{HeroName: "T", Expansion: true}

	if u := h.UnlockedDifficulties(); u != [3]bool{true, false, false} {
		t.Fatalf("new hero %v", u)
	}

	if err := h.ChooseDifficulty(d2difficulty.Nightmare, false); err == nil {
		t.Error("Nightmare chosen before Normal was finished")
	}

	if h.Difficulty != d2enum.DifficultyNormal {
		t.Error("a refused choice changed the difficulty")
	}

	// finish Act 5 (and the acts before it) in Normal
	p := h.EnsureProgress()
	for _, a := range [][2]int{{1, 6}, {2, 6}, {3, 6}, {4, 2}, {5, 6}} {
		p.Quests[0].SetCompleted(a[0], a[1])
	}

	if err := h.ChooseDifficulty(d2difficulty.Nightmare, false); err != nil {
		t.Fatal(err)
	}

	if h.Difficulty != d2enum.DifficultyNightmare || !p.Waypoints.Has(1, d2s.WPRogueEncampment) {
		t.Error("Nightmare not active or its Rogue Encampment waypoint missing")
	}

	if err := h.ChooseDifficulty(d2difficulty.Hell, false); err == nil {
		t.Error("Hell chosen before Nightmare was finished")
	}

	if err := h.ChooseDifficulty(d2difficulty.Hell, true); err != nil || h.Difficulty != d2enum.DifficultyHell {
		t.Errorf("forced Hell: %v", err)
	}
}

func TestDeathPenaltyFromTable(t *testing.T) {
	for d, want := range []int{0, 5, 10} {
		if DeathExpPenaltyPercent(d) != want {
			t.Errorf("penalty %d", d)
		}
	}

	if DeathExpPenaltyPercent(7) != 0 || DeathExpPenaltyPercent(-1) != 0 {
		t.Error("out of range")
	}
}

// With the real level-94 save (progression 13: Normal and Nightmare finished) all
// difficulties are unlocked; the chosen one is written to the header and the other
// difficulties' quests and waypoints stay as they were.
func TestRealSaveDifficultyChoice(t *testing.T) {
	data, tables, state := realSave(t)

	if u := state.UnlockedDifficulties(); u != [3]bool{true, true, true} {
		t.Fatalf("unlocked %v, want all (progression %d)", u, state.StoredProgression())
	}

	before, _ := d2s.Parse(data, tables)

	if err := state.ChooseDifficulty(d2difficulty.Nightmare, false); err != nil {
		t.Fatal(err)
	}

	state.Progress.Waypoints.Set(1, d2s.WPRogueEncampment, true)
	state.Progress.Quests[1].Set(d2s.QuestSlotAkaraRespec, 5)

	out, _, err := ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}

	got, err := d2s.Parse(out, tables)
	if err != nil {
		t.Fatal(err)
	}

	if d, _, _ := got.Header.ActiveDifficulty(); d != 1 {
		t.Errorf("active difficulty %d, want 1", d)
	}

	if d2difficulty.Progression(got.Header.Status) != state.StoredProgression() {
		t.Error("the progression changed")
	}

	if !got.Body.QuestRecord(1).Get(d2s.QuestSlotAkaraRespec, 5) {
		t.Error("Nightmare quest edit lost")
	}

	for _, d := range []int{0, 2} {
		if *got.Body.QuestRecord(d) != *before.Body.QuestRecord(d) {
			t.Errorf("quests of difficulty %d changed", d)
		}

		if got.Body.Waypoints[d] != before.Body.Waypoints[d] {
			t.Errorf("waypoints of difficulty %d changed", d)
		}
	}
}

func TestExportProgressionOnlyGrows(t *testing.T) {
	h := &d2s.Header{Status: d2s.StatusExpansion}
	state := &HeroState{Progress: &HeroProgress{}}

	var orig [3]d2s.QuestRecord

	exportProgression(h, orig, state)

	if d2difficulty.Progression(h.Status) != 0 {
		t.Error("unchanged hero changed the progression")
	}

	for _, a := range [][2]int{{1, 6}, {2, 6}, {3, 6}, {4, 2}, {5, 6}} {
		state.Progress.Quests[0].SetCompleted(a[0], a[1])
	}

	exportProgression(h, orig, state)

	if d2difficulty.Progression(h.Status) != 5 || h.Status&d2s.StatusExpansion == 0 {
		t.Errorf("status %#x after Baal", h.Status)
	}

	// a stored progression that is higher stays
	h.Status = d2difficulty.WithProgression(h.Status, 9)
	exportProgression(h, orig, state)

	if d2difficulty.Progression(h.Status) != 9 {
		t.Error("progression lowered")
	}
}
