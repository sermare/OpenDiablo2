package diablo2item

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// cubeQualityTokens are the quality words of the "output" columns of
// CubeMain.txt.
var cubeQualityTokens = map[string]d2drop.Quality{
	"low": d2drop.QualityLow, "nor": d2drop.QualityNormal, "sup": d2drop.QualitySuperior,
	"mag": d2drop.QualityMagic, "set": d2drop.QualitySet, "rar": d2drop.QualityRare,
	"uni": d2drop.QualityUnique, "crf": d2drop.QualityCrafted,
}

// CubeResult makes the item a Horadric Cube recipe produces. tokens are the
// modifier words of the recipe's output ("rar", "uni", "eth", "noe", "sock",
// "nos" ...; unknown ones such as "pre=" are ignored), ilvl the item level the
// recipe gives (the level of the input or of the player). A recipe without a
// quality word lets the game roll the quality like a vendor does.
func (f *ItemFactory) CubeResult(code string, tokens []string, ilvl int, seed uint32) (*Item, error) {
	p := CreateParams{Code: code, ILvl: ilvl, Seed: seed, Difficulty: f.Difficulty, Classic: f.Classic}

	for _, tok := range tokens {
		tok = strings.ToLower(strings.TrimSpace(tok))

		switch {
		case cubeQualityTokens[tok] != 0:
			p.Quality = cubeQualityTokens[tok]
		case tok == "eth":
			p.Flags |= d2drop.FlagForceEthereal
		case tok == "noe":
			p.Flags |= d2drop.FlagNoEthereal
		case tok == "sock":
			p.Flags |= d2drop.FlagForceSockets
		case tok == "nos":
			p.Flags |= d2drop.FlagNoSockets
		}
	}

	item, err := f.Create(p)
	if err != nil {
		return nil, fmt.Errorf("cube result %q: %w", code, err)
	}

	return item, nil
}
