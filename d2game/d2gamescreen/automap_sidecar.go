package d2gamescreen

import (
	"path/filepath"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2automap"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// automapSidecar names the automap sidecar files of the local hero: next to the character's .d2s (so
// OD2_D2S_WRITEBACK moves them too), game id = the hero's map seed (UNVERIFIED that this is the exe's
// *DAT_007986c0; it is stable across sessions of one character, which is what the lifecycle needs).
// A remote hero has no local save folder: the result is invalid and the automap stays in memory.
func (v *Game) automapSidecar() d2automap.Sidecar {
	st := v.gameClient.LocalHeroState()
	if st == nil || st.HeroName == "" {
		return d2automap.Sidecar{}
	}

	id := st.MapSeed
	if id == 0 {
		id = uint32(v.gameClient.Seed)
	}

	return d2automap.Sidecar{Dir: filepath.Dir(d2hero.D2SPath(st)), Name: st.HeroName, GameID: id}
}
