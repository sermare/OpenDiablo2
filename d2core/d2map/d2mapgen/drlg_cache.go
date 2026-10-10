package d2mapgen

import (
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgoutdoor"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
)

// The DRLG tables and the outdoor generator's environment (parsed DS1 patterns and DT1 tile
// headers) only depend on the game archives, which do not change during a run, and the generators
// only read them (drlgoutdoor.Env is documented as shared and read-only). They used to be rebuilt
// for every level load, and the local game builds every level twice (server and client map
// generator), which cost hundreds of milliseconds per outdoor level in MPQ reads and
// decompression. They are now built once per asset manager.
type drlgShared struct {
	tables *d2drlg.Tables
	env    *drlgoutdoor.Env
}

//nolint:gochecknoglobals // per-asset-manager cache
var (
	drlgSharedMu sync.Mutex
	drlgSharedBy = map[*d2asset.AssetManager]*drlgShared{}
)

func loadDRLGShared(a *d2asset.AssetManager) (*drlgShared, error) {
	drlgSharedMu.Lock()
	defer drlgSharedMu.Unlock()

	if s, ok := drlgSharedBy[a]; ok {
		return s, nil
	}

	tb, err := buildDRLGTables(a)
	if err != nil {
		return nil, err
	}

	s := &drlgShared{tables: tb, env: drlgoutdoor.NewEnv(tb, func(file string) ([]byte, error) {
		return a.LoadFileHandoff("/data/global/tiles/" + file)
	})}
	drlgSharedBy[a] = s

	return s, nil
}

// outdoorEnv returns the shared outdoor generator environment of the asset manager.
func outdoorEnv(a *d2asset.AssetManager) (*drlgoutdoor.Env, error) {
	s, err := loadDRLGShared(a)
	if err != nil {
		return nil, err
	}

	return s.env, nil
}
