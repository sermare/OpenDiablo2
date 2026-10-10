package d2drop

import "testing"

// The item rules of ITEM_FindRunewordForSockets (0x62c010, VERIFIED): only low,
// normal and superior items, every socket filled, quest items never.
func TestFindRunewordForItemRules(t *testing.T) {
	c := cLoadCreator(t)
	c.Runes = loadRuneTablesFrom(skipTB{t}, testSource(t), c.Props)

	runes := []string{"r08", "r09", "r07"}

	for _, tc := range []struct {
		name    string
		quality Quality
		sockets int
		want    bool
	}{
		{"normal, filled", QualityNormal, 3, true},
		{"superior", QualitySuperior, 3, true},
		{"low quality", QualityLow, 3, true},
		{"magic", QualityMagic, 3, false},
		{"set", QualitySet, 3, false},
		{"rare", QualityRare, 3, false},
		{"unique", QualityUnique, 3, false},
		{"crafted", QualityCrafted, 3, false},
		{"tempered", qualityTempered, 3, false},
		{"a socket is still empty", QualityNormal, 4, false},
		{"more runes than sockets", QualityNormal, 2, false},
	} {
		got := c.FindRunewordFor("lrg", tc.quality, tc.sockets, runes) != nil
		if got != tc.want {
			t.Errorf("%s: found = %v, want %v", tc.name, got, tc.want)
		}
	}

	// fewer runes than the row has never match (the row would need more)
	if c.FindRunewordFor("lrg", QualityNormal, 2, runes[:2]) != nil {
		t.Error("a two rune prefix of a three rune row made a runeword")
	}
}
