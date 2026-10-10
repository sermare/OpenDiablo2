package diablo2item

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2reward"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// Item changes that quest rewards make: Larzuk's sockets and Anya's
// personalisation (rules in d2common/d2reward).

// qualityMagic is the first quality id of items Anya may name (magic 4, set 5,
// rare 6, unique 7, crafted 8).
const qualityMagic = d2drop.QualityMagic

// SetNumSockets sets the item's socket count (Larzuk's reward).
func (i *Item) SetNumSockets(n int) {
	if i.attributes != nil && n >= 0 {
		i.attributes.numSockets = n
	}
}

// PersonalName returns the hero name the item was personalised with, or "".
func (i *Item) PersonalName() string {
	if i.attributes == nil {
		return ""
	}

	return i.attributes.personalization
}

// SetPersonalName personalises the item (Anya's reward).
func (i *Item) SetPersonalName(hero string) {
	if i.attributes != nil {
		i.attributes.personalization = hero
	}
}

// IsQuestItem reports whether the item is one the quest system tracks
// (type "ques").
func (i *Item) IsQuestItem() bool { return i.TypeCode == "ques" }

// SocketInfo describes the item for the socket rule of d2reward.
func (i *Item) SocketInfo() d2reward.SocketItem {
	si := d2reward.SocketItem{Code: i.CommonCode, ItemLevel: i.ItemLevel(), Sockets: i.NumSockets(), Quest: i.IsQuestItem()}

	if rec := i.CommonRecord(); rec != nil {
		si.BaseSockets = rec.GemSockets
	}

	if t := i.TypeRecord(); t != nil {
		si.MaxSock1, si.MaxSock25, si.MaxSock40 = t.MaxSock1, t.MaxSock25, t.MaxSock40
	}

	return si
}

// weaponOrArmor reports whether the item type descends from "weap" or "armo"
// (ItemTypes.txt Equiv1/Equiv2).
func (i *Item) weaponOrArmor() bool {
	seen := map[string]bool{}

	var walk func(code string) bool

	walk = func(code string) bool {
		if code == "" || seen[code] {
			return false
		}

		seen[code] = true

		if code == "weap" || code == "armo" {
			return true
		}

		t := i.factory.asset.Records.Item.Types[code]

		return t != nil && (walk(t.Equiv1) || walk(t.Equiv2))
	}

	return walk(i.TypeCode)
}

// ImbueInfo describes the item for Charsi's imbue rule.
func (i *Item) ImbueInfo() d2reward.ImbueItem {
	return d2reward.ImbueItem{
		WeaponOrArmor: i.weaponOrArmor(),
		Quality:       int(i.quality),
		Quest:         i.IsQuestItem(),
		Gems:          len(i.SocketCodes) + len(i.sockets) + len(i.socketed),
	}
}

// Imbue makes the rare item Charsi's reward gives for old: the same base item,
// quality rare, item level ilvl, identified; sockets, ethereal and
// personalisation of the old item are kept (quests-2.md, A1Q3). The old item
// is unchanged; the caller replaces it.
func (f *ItemFactory) Imbue(old *Item, ilvl int, seed uint32) (*Item, error) {
	if err := d2reward.CanImbue(old.ImbueInfo()); err != nil {
		return nil, err
	}

	fresh, err := f.ItemFromCode(old.CommonCode, d2drop.QualityRare, ilvl, seed)
	if err != nil {
		return nil, err
	}

	spec := fresh.Spec()
	spec.Identified = true
	spec.Ethereal = old.IsEthereal()
	spec.Personal = old.PersonalName()

	if n := old.NumSockets(); n > 0 {
		spec.Sockets = n
	}

	if rebuilt, err := f.ItemFromSpec(spec); err == nil {
		rebuilt.Identify()

		return rebuilt, nil
	}

	return fresh.Identify(), nil
}

// PersonalizeInfo describes the item for the personalisation rule.
func (i *Item) PersonalizeInfo() d2reward.PersonalizeItem {
	p := d2reward.PersonalizeItem{MagicOrBetter: i.quality >= qualityMagic || i.UniqueCode != "" || i.SetItemCode != "" || len(i.PrefixCodes)+len(i.SuffixCodes) > 0, Personalized: i.PersonalName() != ""}

	if rec := i.CommonRecord(); rec != nil {
		p.Nameable = rec.Nameable
	}

	return p
}
