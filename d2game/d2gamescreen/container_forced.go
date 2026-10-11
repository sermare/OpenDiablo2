package d2gamescreen

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2object"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// chestSink adapts the game to d2object.LootSink: Drop rolls the chest class with a forced quality, Create
// makes gold and potions at the container (OBJECT_ServerDropItemByTypeCode 0x583870).
type chestSink struct {
	v     *Game
	id    int
	co    diablo2item.ChestDropOptions
	all   *diablo2item.Loot
	class string
	n     uint32
}

func (s *chestSink) Drop(forced int) (made, magical bool) {
	s.v.ground.chestSeq++

	co := s.co
	co.ForcedQuality = d2drop.Quality(forced)

	loot, class, err := s.v.rollChestOnce(co, s.id, s.v.chestSeed()+s.v.ground.chestSeq)
	if err != nil || loot == nil {
		return false, false
	}

	s.class = class
	s.all.Entries = append(s.all.Entries, loot.Entries...)

	if len(loot.Entries) == 0 {
		return false, false
	}

	first := loot.Entries[0]
	if first.Item == nil {
		return true, false // gold is an item that is not magical
	}

	return true, d2object.IsMagicalQuality(int(first.Item.Quality()))
}

func (s *chestSink) Create(code string, n int) {
	for i := 0; i < n; i++ {
		s.n++

		if code == "gld " {
			rng := d2rand.New(s.v.chestSeed() + s.v.ground.chestSeq + 1000 + s.n)
			s.all.Entries = append(s.all.Entries, diablo2item.LootEntry{Gold: d2drop.GoldAmount(rng, s.co.ILvl, 0)})

			continue
		}

		it, err := s.v.itemFactory().ItemFromCode(strings.TrimSpace(code), d2drop.QualityNormal, s.co.ILvl, s.v.chestSeed()+s.n)
		if err == nil && it != nil {
			s.all.Entries = append(s.all.Entries, diablo2item.LootEntry{Item: it})
		}
	}
}

// openChestForced runs the quality forcing container branches and returns what dropped and the last class.
func (v *Game) openChestForced(ob *d2mapentity.Object, id int, co diablo2item.ChestDropOptions,
	plan containerPlan) (*diablo2item.Loot, string) {
	s := &chestSink{v: v, id: id, co: co, all: &diablo2item.Loot{}}
	rng := v.objRoller(ob, "open")

	if plan.sparkly {
		d2object.OpenSparkly(rng, s)
	} else {
		d2object.OpenGeneric(rng, s, plan.locked, plan.variant)
	}

	return s.all, s.class
}
