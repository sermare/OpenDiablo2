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

// NumSockets returns the number of (empty) sockets a quest reward gave the
// item; sockets of socketed gems are not modelled by this engine yet.
func (i *Item) NumSockets() int {
	if i.attributes == nil {
		return 0
	}

	return i.attributes.numSockets
}

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

// PersonalizeInfo describes the item for the personalisation rule.
func (i *Item) PersonalizeInfo() d2reward.PersonalizeItem {
	p := d2reward.PersonalizeItem{MagicOrBetter: i.quality >= qualityMagic || i.UniqueCode != "" || i.SetItemCode != "" || len(i.PrefixCodes)+len(i.SuffixCodes) > 0, Personalized: i.PersonalName() != ""}

	if rec := i.CommonRecord(); rec != nil {
		p.Nameable = rec.Nameable
	}

	return p
}
