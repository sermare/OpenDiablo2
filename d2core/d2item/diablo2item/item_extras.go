package diablo2item

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// rollExtras applies the ethereal and socket rolls of the generator (see
// d2drop.RollEthereal and d2drop.RollSockets) to a freshly generated item.
// The game rolls ethereal first (qualities other than low and set), then
// sockets (normal and superior only). The rolls use a generator seeded from
// the item seed, so the drop stream and every other item are unchanged.
func (f *ItemFactory) rollExtras(t *dropTables, item *Item, difficulty int) {
	icr := f.asset.Records.Item.All[item.CommonCode]
	info := t.items[item.CommonCode]

	if icr == nil || info == nil || item.attributes == nil || item.genQuality == d2drop.QualityNone {
		return
	}

	rng := d2rand.New(uint32(item.Seed))
	hasDur := item.attributes.durable && icr.Durability > 0

	eth := d2drop.EtherealInput{
		WeaponOrArmor: info.HasType("weap") || info.HasType("armo"),
		Durability:    hasDur,
		Quest:         info.Quest,
		Quality:       item.genQuality,
	}

	if d2drop.RollEthereal(rng, eth) {
		item.attributes.ethereal = true
		item.attributes.durability.max = d2drop.EtherealMaxDurability(icr.Durability)
		item.attributes.currentDurability = item.attributes.durability.max
	}

	if item.genQuality > d2drop.QualitySuperior {
		return
	}

	maxSock := 0

	if ty := f.asset.Records.Item.Types[icr.Type]; ty != nil {
		maxSock = icr.GemSockets

		if lim := (&d2equip.Type{MaxSock1: ty.MaxSock1, MaxSock25: ty.MaxSock25, MaxSock40: ty.MaxSock40}).
			MaxSocketsByLevel(item.itemLevel); lim < maxSock {
			maxSock = lim
		}
	}

	item.attributes.numSockets = d2drop.RollSockets(rng, d2drop.SocketInput{
		Quality: item.genQuality, HasInventory: icr.HasInventory, Stackable: icr.Stackable,
		MaxSockets: maxSock, Difficulty: difficulty, InitSeed: uint32(item.Seed),
		Cells: icr.InventoryWidth * icr.InventoryHeight,
	})
}

// NumSockets is the number of sockets the generator rolled (0 when it did not
// roll or rolled none).
func (i *Item) NumSockets() int {
	if i.attributes == nil {
		return 0
	}

	return i.attributes.numSockets
}

// IsEthereal reports the ethereal attribute.
func (i *Item) IsEthereal() bool {
	return i.attributes != nil && i.attributes.ethereal
}
