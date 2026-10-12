package diablo2item

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// TestSaveRoundTripManySeeds rolls many items of every quality and checks that
// none loses a stat in the save or changes on reload.
func TestSaveRoundTripManySeeds(t *testing.T) {
	f := realCreatorFactory(t)
	tables := saveTables(t)

	n := 0

	for _, q := range []d2drop.Quality{
		d2drop.QualityNormal, d2drop.QualityMagic, d2drop.QualityRare, d2drop.QualityCrafted,
	} {
		for _, code := range []string{"cap", "lsd", "uap", "lrg"} {
			for seed := uint32(100); seed < 140; seed++ {
				it, err := f.Create(CreateParams{Code: code, ILvl: 30 + int(seed%65), Quality: q, Seed: seed, FreshGame: true})
				if err != nil {
					continue
				}

				it.Identify()

				_, after := asSaved(t, it, tables)

				spec, err := f.SpecFromD2S(&after)
				if err != nil {
					t.Fatalf("%s q%d seed %d: %v", code, q, seed, err)
				}

				again, err := f.ItemFromSpec(spec)
				if err != nil {
					t.Fatal(err)
				}

				sameItem(t, code, it, again, tables)

				n++
			}
		}
	}

	if n < 300 {
		t.Errorf("only %d items rolled", n)
	}
}
