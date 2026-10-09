package d2records

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
)

func TestTreasureClassFileOrder(t *testing.T) {
	txt := "Treasure Class\tgroup\tlevel\tPicks\tItem1\tProb1\n" +
		"Zeta\t1\t1\t1\tgld\t5\n" +
		"\t\t\t\t\t\n" + // rows without a name do not take a position
		"Alpha\t1\t9\t1\tgld\t5\n" +
		"Mid\t0\t0\t1\tgld\t5\n"

	recs, err := treasureClassCommonLoader(d2txt.LoadDataDictionary([]byte(txt)))
	if err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]int{"Zeta": 0, "Alpha": 1, "Mid": 2} {
		if got := recs[name].Index; got != want {
			t.Errorf("%s index = %d, want %d", name, got, want)
		}
	}
}
