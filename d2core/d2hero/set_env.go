package d2hero

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// setEnv is the set bonus table of the stat list, built once from Sets.txt
// (nil when the tables cannot be read).
type setEnv struct {
	loaded bool
	env    *d2statlist.Env
	items  *diablo2item.ItemFactory
}

func (f *HeroStateFactory) loadSetEnv() *setEnv {
	if f.sets != nil {
		return f.sets
	}

	f.sets = &setEnv{loaded: true}

	items, err := diablo2item.NewItemFactory(f.asset)
	if err != nil {
		return f.sets
	}

	table, err := items.SetTable()
	if err != nil {
		return f.sets
	}

	f.sets.items = items
	f.sets.env = &d2statlist.Env{Sets: table}

	return f.sets
}

// statEnv is the environment for d2statlist.Compute: the set bonuses that
// become active when enough pieces of a set are worn (partial tiers at 2 to
// 5 pieces, the full bonus with the whole set).
func (f *HeroStateFactory) statEnv() *d2statlist.Env {
	return f.loadSetEnv().env
}

// setResolver maps the set item rows of a save to their sets.
func (f *HeroStateFactory) setResolver() SetResolver {
	s := f.loadSetEnv()
	if s.items == nil {
		return nil
	}

	return s.items.SetOfSetItem
}
