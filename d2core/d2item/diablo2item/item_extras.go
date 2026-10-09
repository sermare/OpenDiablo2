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
	f.rollExtrasFlags(t, item, difficulty, false)
}

// rollExtrasFlags is rollExtras with the request flag 0x2 of the original
// ("skip the ethereal roll", VERIFIED in 0x554d90): vendor stock is built
// with it set (0x574110 calls ITEMGEN_BuildItemRequest with that argument 1
// and the socket-block argument 0), so shop items can be socketed but are
// never ethereal. Without the roll no random number is drawn for it.
func (f *ItemFactory) rollExtrasFlags(t *dropTables, item *Item, difficulty int, skipEthereal bool) {
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

	if !skipEthereal && d2drop.RollEthereal(rng, eth) {
		item.attributes.ethereal = true
		item.attributes.applyEtherialBonus()
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

// ItemFromCodeForVendor is ItemFromCode followed by the socket roll of the
// generator, as shop stock gets it (see rollExtrasFlags: sockets yes,
// ethereal never). Everything but the socket count is identical to
// ItemFromCode for the same arguments, because the roll uses its own
// generator seeded from the item seed.
func (f *ItemFactory) ItemFromCodeForVendor(code string, q d2drop.Quality, ilvl int, seed uint32,
	difficulty int) (*Item, error) {
	item, err := f.ItemFromCode(code, q, ilvl, seed)
	if err != nil {
		return nil, err
	}

	f.rollExtrasFlags(f.dropTables(), item, difficulty, true)

	return item, nil
}

// etherealMul and etherealDiv are the ethereal bonus: FUN_00660a40 (called by
// ITEMGEN_RollEthereal 0x554d90 after setting flag 0x400000) replaces the
// base stats with (v * 3) / 2 in C integer division: min/max damage and the
// secondary and throw damage (stats 0x15..0x18, 0x9f, 0xa0) of a weapon,
// base defense (stat 0x1f) of anything else. VERIFIED in Game.exe.
const (
	etherealMul = 3
	etherealDiv = 2
)

func etherealBoost(v int) int { return v * etherealMul / etherealDiv }

// applyEtherialBonus applies the +50% damage / +50% defense of an ethereal
// item to the rolled base values. It is called exactly once per item, when
// the ethereal flag is set (generator roll or ItemFromSpec).
func (a *itemAttributes) applyEtherialBonus() {
	for _, d := range []*minMaxEnhanceable{&a.damageOneHand, &a.damageTwoHand, &a.damageMissile} {
		d.min, d.max = etherealBoost(d.min), etherealBoost(d.max)
	}

	a.defense = etherealBoost(a.defense)
}
