package diablo2item

import (
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

func TestSetInfoAndTooltip(t *testing.T) {
	f := realCreatorFactory(t)

	skull, err := f.Create(CreateParams{
		Code: "bhm", ILvl: 99, Quality: d2drop.QualitySet, Name: "Tancred's Skull", Seed: 5, FreshGame: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	row := skull.SetRow()

	info, ok := f.SetInfo(row)
	if !ok || len(info.Pieces) != 5 || len(info.Full) == 0 || len(info.Partial[0]) == 0 {
		t.Fatalf("set info %+v (ok %v)", info, ok)
	}

	if skull.SetPieceName() != "Tancred's Skull" {
		t.Fatalf("piece name %q", skull.SetPieceName())
	}

	found := false

	for _, p := range info.Pieces {
		found = found || p == skull.SetPieceName()
	}

	if !found {
		t.Fatalf("the piece %q is not among the set's pieces %v", skull.SetPieceName(), info.Pieces)
	}

	count := func(lines []string, token string) int {
		n := 0

		for _, l := range lines {
			if strings.HasPrefix(l, token) {
				n++
			}
		}

		return n
	}

	worn := func(n int) map[string]bool {
		m := map[string]bool{}
		for _, p := range info.Pieces[:n] {
			m[p] = true
		}

		return m
	}

	nPartial := len(info.Partial[0]) + len(info.Partial[1]) + len(info.Partial[2]) + len(info.Partial[3])

	// two pieces switch the first partial tier on (green), the rest stays grey
	lines := SetTooltipLines(info, worn(2))
	if g := count(lines, "[grey]"); g != nPartial-len(info.Partial[0])+len(info.Full) {
		t.Errorf("2 pieces: %d grey lines in %q", g, lines)
	}

	// the whole set switches the full bonus on, nothing is grey
	lines = SetTooltipLines(info, worn(5))
	if count(lines, "[grey]") != 0 || count(lines, "[red]") != 0 {
		t.Errorf("5 pieces: grey or red left in %q", lines)
	}

	// no piece worn: every piece is red
	if r := count(SetTooltipLines(info, worn(0)), "[red]"); r != 5 {
		t.Errorf("0 pieces: %d red lines", r)
	}
}
