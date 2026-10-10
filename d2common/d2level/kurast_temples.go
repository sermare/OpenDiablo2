package d2level

import "strings"

// warpKurastToTemple is the LvlWarp id "Act 3 Kurast to Temple" (Levels.txt Warp slot of Kurast Bazaar 80, Upper
// Kurast 81 and the Causeway 82 towards their two temples).
const warpKurastToTemple = 61

// TempleEntranceByPreset resolves the temple entrance of the Kurast levels 80, 81 and 82. Each of them has two temple
// links with the same LvlWarp id (80: Ruined Temple 94 and Disused Fane 95, 81: 96 and 97, 82: 98 and 99), so the tile
// style cannot say which one a tile is. What the generated levels show: the temple presets Act3/Kurast/BurbsTemple2.ds1
// and BurbsTemple3.ds1 each carry one entrance tile (style 2 and style 3, the other special tiles of those presets have
// the style 8 of the doors of every Kurast building) and a level places at most one preset of each. OBSERVED: BurbsTemple2
// stands for the first temple link of the level (in Levels.txt slot order), BurbsTemple3 for the second.
// UNVERIFIED against the exe (which temple a preset leads to in the original is not decoded).
func TempleEntranceByPreset(level int, path string, style int) (int, bool) {
	if level < 80 || level > 82 || (style != 2 && style != 3) {
		return 0, false
	}

	idx := -1
	p := strings.ToLower(path)

	switch {
	case strings.Contains(p, "burbstemple2"):
		idx = 0
	case strings.Contains(p, "burbstemple3"):
		idx = 1
	default:
		return 0, false
	}

	n := 0

	for _, l := range allLinks {
		if l.From != level || l.Kind != KindTile || l.Warp != warpKurastToTemple {
			continue
		}

		if n == idx {
			return l.To, true
		}

		n++
	}

	return 0, false
}
