package diablo2item

import (
	"errors"
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

var (
	errNoFreeSocket  = errors.New("the item has no free socket")
	errNotSocketable = errors.New("the item cannot be put into a socket")
	errNotRolled     = errors.New("the item was not made with the item creator")
)

// socketableTypes are the ItemTypes a socket takes: gems, runes and jewels.
var socketableTypes = map[string]bool{"gem0": true, "gem1": true, "gem2": true, "gem3": true, "gem4": true, "gemz": true, "rune": true, "jewl": true}

// Socketed returns the items in the sockets of the item.
func (i *Item) Socketed() []*Item {
	return i.socketed
}

// RolledRuneword returns the Runes.txt name of the runeword a creator-made item
// has been made into, "" for an item that is not one (cube-made runewords are
// the Runeword field).
func (i *Item) RolledRuneword() string {
	return i.runeword
}

// Socket puts a gem, rune or jewel into the next free socket. When this fills
// the last socket with the runes of a runeword that fits the item, the item
// becomes the runeword: its properties are walked into the runeword stat list
// (d2drop.Creator.ApplyRuneword) and its name becomes the runeword's.
func (i *Item) Socket(child *Item) error {
	if i.rolled == nil {
		return errNotRolled
	}

	if !socketableTypes[child.TypeCode] && !socketableTypes[child.CommonRecord().Type] {
		return fmt.Errorf("%w: %s", errNotSocketable, child.CommonCode)
	}

	if len(i.socketed) >= i.NumSockets() {
		return errNoFreeSocket
	}

	i.socketed = append(i.socketed, child)

	if len(i.socketed) == i.NumSockets() {
		i.makeRuneword()
	}

	return nil
}

// makeRuneword turns a fully socketed item into a runeword, if its runes
// form one.
func (i *Item) makeRuneword() {
	c, err := i.factory.Creator()
	if err != nil || i.runeword != "" {
		return
	}

	codes := make([]string, len(i.socketed))
	for k, s := range i.socketed {
		codes[k] = s.CommonCode
	}

	w := c.FindRunewordFor(i.CommonCode, i.quality, i.NumSockets(), codes)
	if w == nil {
		return
	}

	req := d2drop.Request{
		Code: i.CommonCode, ILvl: i.itemLevel, Quality: i.quality, Version: lodVersion, Expansion: true,
	}

	i.rolled = c.ApplyRuneword(i.rolled, req, w)
	i.runeword = w.Name

	// enhanced defense and the like change the base values the runeword walk wrote
	for _, wr := range i.rolled.Writes {
		if wr.Kind == 'S' && wr.Stat == statIDDefense {
			i.attributes.defense = wr.Value
		}
	}

	i.rolledProperties(c)
	i.name = i.rolledName()
}

// runewordName is the display name of a runeword row ("Runeword1" is the key
// of the string table entry "Ancient's Pledge").
func (f *ItemFactory) runewordName(key string) string {
	return f.translateName(key)
}

// socketedStatItems are the socketed items as the stat list sees them.
func (i *Item) socketedStatItems() []d2statlist.SocketedItem {
	out := make([]d2statlist.SocketedItem, 0, len(i.socketed))

	for _, s := range i.socketed {
		si := d2statlist.SocketedItem{Code: strings.TrimSpace(s.CommonCode)}

		// jewels carry their own properties; gems and runes get theirs from Gems.txt
		if s.rolled != nil && s.quality != d2drop.QualityNormal {
			si.Props = s.StatItem().Props
		}

		out = append(out, si)
	}

	return out
}
