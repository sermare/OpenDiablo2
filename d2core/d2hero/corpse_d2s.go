package d2hero

import (
	"encoding/binary"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// exportCorpseItems writes a corpse that holds its own item list (Corpse.Items, the live game's corpse): the file's
// worn items that are on the corpse move to the corpse section (matched by the save's item id), the rest of the
// corpse (made in the game, or the cursor item) is encoded. Worn items the hero took back stay worn in the file.
func exportCorpseItems(c *d2s.Character, state *HeroState, tables *d2s.ItemTables, ids *AffixIDs, warn func(string, ...interface{})) {
	if state == nil || state.Death == nil || state.Death.Corpse == nil || len(state.Death.Corpse.Items) == 0 ||
		c.Body == nil || c.HasCorpse {
		return
	}

	d := state.Death.Corpse
	onCorpse := map[uint32]*StoredItem{}

	for i := range d.Items {
		if s := &d.Items[i]; s.D2S != nil {
			onCorpse[s.D2S.ID] = s
		}
	}

	kept := c.Items[:0:0]
	done := map[*StoredItem]bool{}

	for i := range c.Items {
		it := c.Items[i]

		if s, ok := onCorpse[it.ID]; ok && !done[s] && it.Location == d2s.LocationEquipped && !it.Ear {
			c.Corpse = append(c.Corpse, it)
			done[s] = true

			continue
		}

		kept = append(kept, it)
	}

	c.Items = kept

	for i := range d.Items {
		s := &d.Items[i]
		if done[s] {
			continue
		}

		var it d2s.Item

		if s.D2S != nil {
			it = *s.D2S
		} else {
			var why string

			if it, _, why = D2SItemFromStored(s, tables, ids); why != "" {
				warn("corpse item %q not written: %s", s.Code, why)
				continue
			}
		}

		if s.Page == PageEquipped {
			it.Location, it.Page, it.X, it.Y, it.Equipped = d2s.LocationEquipped, 0, 0, 0, uint8(s.X)
		} else {
			it.Location, it.Page, it.X, it.Y = d2s.LocationStored, uint8(PageInventory), uint8(s.X), uint8(s.Y)
		}

		c.Corpse = append(c.Corpse, it)
	}

	c.HasCorpse = true
	binary.LittleEndian.PutUint32(c.CorpseHeader[corpseHeaderX:], uint32(d.X))
	binary.LittleEndian.PutUint32(c.CorpseHeader[corpseHeaderY:], uint32(d.Y))
}

// importCorpse reads the corpse section of a save into the hero's death record: the worn items (body location kept in
// X) and the others the corpse holds, with the position of the death from the section header.
func importCorpse(state *HeroState, c *d2s.Character, known func(string) bool) {
	if !c.HasCorpse && len(c.Corpse) == 0 {
		return
	}

	corpse := &Corpse{
		X: int(binary.LittleEndian.Uint32(c.CorpseHeader[corpseHeaderX:])),
		Y: int(binary.LittleEndian.Uint32(c.CorpseHeader[corpseHeaderY:])),
	}

	for i := range c.Corpse {
		it := &c.Corpse[i]

		var (
			s    StoredItem
			skip string
		)

		if it.Location == d2s.LocationEquipped {
			if s, skip = storedFromD2S(it, PageEquipped, known); skip == "" {
				s.X, s.Y = int(it.Equipped), 0
			}
		} else {
			s, skip = StoredFromD2S(it, known)
		}

		if skip == "" {
			corpse.Items = append(corpse.Items, s)
		}
	}

	if state.Death == nil {
		state.Death = &DeathState{}
	}

	state.Death.Corpse = corpse
}
