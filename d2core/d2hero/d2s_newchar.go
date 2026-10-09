package d2hero

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// ErrD2SExists is returned when a new character's .d2s would replace a file.
var ErrD2SExists = errors.New("d2s: a file with that name already exists, not overwritten")

// D2SClassOf returns the .d2s class of a hero.
func D2SClassOf(hero d2enum.Hero) (d2s.Class, bool) {
	for class, h := range d2sClassToHero {
		if h == hero {
			return class, true
		}
	}

	return 0, false
}

// HeroOfD2SClass returns the hero of a .d2s class.
func HeroOfD2SClass(class d2s.Class) (d2enum.Hero, bool) {
	hero, ok := d2sClassToHero[class]

	return hero, ok
}

// NewCharacterD2S builds the bytes of the .d2s a new hero starts with.
func NewCharacterD2S(state *HeroState, created time.Time) ([]byte, error) {
	class, ok := D2SClassOf(state.HeroType)
	if !ok {
		return nil, fmt.Errorf("d2s: hero type %v has no .d2s class", state.HeroType)
	}

	return d2s.NewCharacter(state.HeroName, class,
		d2s.NewCharacterFlags{Expansion: state.Expansion, Hardcore: state.Hardcore, Ladder: state.Ladder, Created: created},
		d2s.DefaultAppearance(class))
}

// CreateNewHero is the character creation path: it creates the hero state
// with the class defaults, saves it as an .od2 and writes the real .d2s of a
// new character (see D2SPath for where). An existing .d2s is never replaced;
// then the hero is returned without a link to a .d2s and the error is
// ErrD2SExists (wrapped) in the export result.
func (f *HeroStateFactory) CreateNewHero(name string, hero d2enum.Hero, expansion, hardcore, ladder bool,
) (*HeroState, *D2SExport, error) {
	if err := d2s.ValidateName(name); err != nil {
		return nil, nil, err
	}

	classStats := f.asset.Records.Character.Stats[hero]
	stats := f.CreateHeroStatsState(hero, classStats)

	state, err := f.CreateHeroState(name, hero, stats)
	if err != nil {
		return nil, nil, err
	}

	state.Expansion, state.Hardcore, state.Ladder = expansion, hardcore, ladder

	res := &D2SExport{Path: D2SPath(state)}

	data, err := NewCharacterD2S(state, time.Now())
	if err != nil {
		return state, res, err
	}

	if err = writeNewFile(res.Path, data); err != nil {
		res.Warnings = append(res.Warnings, err.Error())
	} else {
		state.D2SBase = data
		res.Summary = SummarizeD2S(data, nil)
	}

	if err := f.Save(state); err != nil {
		return state, res, err
	}

	return state, res, nil
}

// writeNewFile creates path with data and fails if it exists.
func writeNewFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), mkdirPermission); err != nil {
		return err
	}

	file, err := os.OpenFile(filepath.Clean(path), os.O_WRONLY|os.O_CREATE|os.O_EXCL, writefilePermission)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%w: %s", ErrD2SExists, path)
		}

		return err
	}

	if _, err = file.Write(data); err != nil {
		_ = file.Close()
		return err
	}

	return file.Close()
}
