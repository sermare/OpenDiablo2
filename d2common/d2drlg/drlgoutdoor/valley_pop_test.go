package drlgoutdoor

import "testing"

// The Valley of Snakes (level 45, Levels.txt "Act 2 - Desert 5") has a monster list (NumMon 3, MonDen 880) and still
// spawns no natural monsters in the original: its generator (DRLG_GenerateAct2Outdoors case 0x2d, act2.go) places only
// the 12 desert border presets (Defs 364..375), the LvlSub cliff pieces (376..387) and the centre preset Def 389
// (Desert Tomb 2, Viper1.ds1, 16x16), and every one of them has LvlPrest Populate = 0, which flags the room 0x800000
// (rooms.go) and makes the population pass skip it (MONREGION_ShouldSpawnForRoom). The emulator confirms: 16 of 16 rooms
// carry the flag (scripts/verify.d/9r-pop-045.sh). So a population minimum of 0 is correct, not a missing feature.
func TestValleyOfSnakesRoomsDoNotPopulate(t *testing.T) {
	tb := monasteryTables(t)

	var defs []int

	for d := 364; d <= 387; d++ {
		defs = append(defs, d)
	}

	defs = append(defs, 389)

	for _, d := range defs {
		pr, ok := tb.PrestByDef(d)
		if !ok {
			t.Errorf("Def %d missing", d)

			continue
		}

		if pr.Populate != 0 {
			t.Errorf("Def %d (%s) has Populate %d: level 45 would spawn monsters in it", d, pr.Name, pr.Populate)
		}
	}
}
