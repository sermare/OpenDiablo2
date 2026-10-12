package d2hero

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// TestCorpseItemsRoundTrip puts the worn items of the real sample save on a corpse, exports, and reads the corpse
// back: the corpse section holds exactly those items and the body section none of them.
func TestCorpseItemsRoundTrip(t *testing.T) {
	data, tables, state := realSave(t)

	orig, err := d2s.Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	known := func(string) bool { return true }
	corpse := &Corpse{X: 5120, Y: 5040}
	worn := 0

	for i := range orig.Items {
		it := orig.Items[i]
		if it.Location != d2s.LocationEquipped || it.Ear {
			continue
		}

		s, skip := storedFromD2S(&it, PageEquipped, known)
		if skip != "" {
			t.Fatalf("%s: %s", it.Code, skip)
		}

		s.X, s.D2S = int(it.Equipped), &it
		corpse.Items = append(corpse.Items, s)
		worn++
	}

	if worn == 0 {
		t.Skip("sample has no worn items")
	}

	state.Death = &DeathState{Deaths: 1, Corpse: corpse}

	out, _, err := ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}

	got, err := d2s.Parse(out, tables)
	if err != nil {
		t.Fatal(err)
	}

	if !got.HasCorpse || len(got.Corpse) != worn {
		t.Fatalf("corpse section: has=%v items=%d want %d", got.HasCorpse, len(got.Corpse), worn)
	}

	for i := range got.Items {
		if got.Items[i].Location == d2s.LocationEquipped {
			t.Fatalf("worn item %s still on the body", got.Items[i].Code)
		}
	}

	back := &HeroState{}
	importCorpse(back, got, known)

	if back.Death == nil || back.Death.Corpse == nil || len(back.Death.Corpse.Items) != worn ||
		back.Death.Corpse.X != 5120 || back.Death.Corpse.Y != 5040 {
		t.Fatalf("imported corpse %+v", back.Death)
	}

	for _, s := range back.Death.Corpse.Items {
		if s.Page != PageEquipped || s.X < 1 || s.X > 12 {
			t.Fatalf("imported item %+v", s)
		}
	}
}
