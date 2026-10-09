package d2mapstamp

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2ds1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
)

const logPrefix = "Map Stamp"

// NewStampFactory creates a MapStamp factory instance
func NewStampFactory(asset *d2asset.AssetManager, l d2util.LogLevel, entity *d2mapentity.MapEntityFactory) *StampFactory {
	result := &StampFactory{
		asset:  asset,
		entity: entity,
		rng:    d2rand.New(0),
	}

	result.Logger = d2util.NewLogger()
	result.Logger.SetLevel(l)
	result.Logger.SetPrefix(logPrefix)

	return result
}

// StampFactory is responsible for loading map stamps. A stamp can be thought of like a
// preset map configuration, like the various configurations of Act 1 town.
type StampFactory struct {
	asset  *d2asset.AssetManager
	entity *d2mapentity.MapEntityFactory
	rng    *d2rand.Seed // picks among a preset's files; reseeded with the map seed

	*d2util.Logger
}

// Reseed restarts the file-picking stream; the map engine calls it with the map seed.
func (f *StampFactory) Reseed(seed int64) {
	f.rng.Init(uint32(seed))
}

// LoadStamp loads the Stamp data from file, using the given level type, level preset index, and
// level file index.
func (f *StampFactory) LoadStamp(levelType d2enum.RegionIdType, levelPreset, fileIndex int) *Stamp {
	preset := f.asset.Records.Level.Presets[levelPreset]

	var levelFilesToPick []string

	for _, fileRecord := range preset.Files {
		if fileRecord != "" && fileRecord != "0" {
			levelFilesToPick = append(levelFilesToPick, fileRecord)
		}
	}

	levelIndex := fileIndex
	if fileIndex < 0 || fileIndex >= len(levelFilesToPick) {
		levelIndex = int(f.rng.Roll(int32(len(levelFilesToPick))))
	}

	if levelFilesToPick == nil {
		panic("no level files to pick from")
	}

	return f.LoadStampPath(levelType, levelPreset, levelFilesToPick[levelIndex])
}

// LoadStampPath is LoadStamp for an explicit DS1 file (relative to
// data/global/tiles). The DRLG outdoor generator uses it because the compiled
// lvlprest.bin lists preset files the txt-based records lack.
func (f *StampFactory) LoadStampPath(levelType d2enum.RegionIdType, levelPreset int, path string) *Stamp {
	stamp := &Stamp{
		factory:     f,
		entity:      f.entity,
		regionID:    levelType,
		levelType:   *f.asset.Records.Level.Types[levelType],
		levelPreset: f.asset.Records.Level.Presets[levelPreset],
	}

	for _, levelTypeDt1 := range &stamp.levelType.Files {
		if levelTypeDt1 == "" || levelTypeDt1 == "0" {
			continue
		}

		dt1, err := f.asset.LoadDT1(levelTypeDt1)
		if err != nil {
			f.Error(err.Error())
			return nil
		}

		stamp.tiles = append(stamp.tiles, dt1.Tiles...)
	}

	stamp.regionPath = path
	fileData, err := f.asset.LoadFile("/data/global/tiles/" + stamp.regionPath)

	if err != nil {
		panic(err)
	}

	stamp.ds1, err = d2ds1.Unmarshal(fileData)
	if err != nil {
		f.Error(err.Error())
		return nil
	}

	return stamp
}
