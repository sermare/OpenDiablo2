package d2monsters

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// TreasureClassOf returns the monstats treasure class column of a monster for
// the director's difficulty: kind 1 normal (TreasureClass1), 2 champion, 3
// unique, 4 quest (ITEMGEN_DropMonsterTreasure picks the column from the type
// the Barbarian's Find Item rolls, see SRVDO_FindItem in skills-2.md).
func (d *Director) TreasureClassOf(m *d2mapentity.Monster, kind int) string {
	r := m.Stat
	if r == nil {
		return ""
	}

	diff := int(d.opt.Difficulty)
	if diff < 0 || diff > 2 {
		diff = 0
	}

	switch kind {
	case 2:
		return [3]string{r.TreasureClassChampionNormal, r.TreasureClassChampionNightmare, r.TreasureClassChampionHell}[diff]
	case 3:
		return [3]string{r.TreasureClass3UniqueNormal, r.TreasureClass3UniqueNightmare, r.TreasureClass3UniqueHell}[diff]
	case 4:
		return [3]string{r.TreasureClassQuestNormal, r.TreasureClassQuestNightmare, r.TreasureClassQuestHell}[diff]
	}

	return [3]string{r.TreasureClassNormal, r.TreasureClassNightmare, r.TreasureClassHell}[diff]
}

// LootCorpse drops the treasure class of the given kind (see TreasureClassOf)
// of a dead monster on the ground around its corpse (Find Item). It returns
// the treasure class that was rolled.
func (d *Director) LootCorpse(m *d2mapentity.Monster, kind int) string {
	u := d.byEntity[m.ID()]
	if u == nil {
		return ""
	}

	tc := d.TreasureClassOf(m, kind)
	d.dropLootFrom(u, tc)

	return tc
}

// DropItemCode puts one item of a base item code (a potion of Find Potion) on
// the ground at a subtile.
func (d *Director) DropItemCode(code string, subX, subY int) error {
	ent, err := d.engine.NewItem(subX, subY, code)
	if err != nil {
		return fmt.Errorf("item %q: %w", code, err)
	}

	d.Counters.Drops++
	d.engine.AddEntity(ent)

	return nil
}
