package diablo2item

import (
	"fmt"
	"hash/fnv"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

const earCode = "ear" // misc.txt "Player Ear"

// EarInfo is what an ear remembers of the hero it was cut from: the name, the
// level and the class (the .d2s class number: 0 amazon .. 6 assassin).
type EarInfo struct {
	Name  string
	Level int
	Class int
}

// Ear returns the ear payload, nil for an ordinary item.
func (i *Item) Ear() *EarInfo { return i.ear }

// NewEar makes the ear item of a slain hardcore hero. It is a misc.txt "ear"
// item (a 1x1 item with the player ear sprites) that is identified from the
// start and carries the victim's name, level and class. The seed is a hash of
// the name and level, so equal ears are equal items.
func (f *ItemFactory) NewEar(info EarInfo) (*Item, error) {
	h := fnv.New32a()
	_, _ = h.Write([]byte(info.Name))

	it, err := f.ItemFromCode(earCode, d2drop.QualityNormal, info.Level, h.Sum32()^uint32(info.Level))
	if err != nil {
		return nil, err
	}

	e := info
	it.ear = &e
	it.Identify()
	it.name = earName(info)

	return it, nil
}

// earName is the label of an ear. UNVERIFIED: the original builds it from the
// string table ("%s's Ear" style, then the level and class on the next line).
func earName(e EarInfo) string {
	return fmt.Sprintf("%s's Ear (Level %d)", e.Name, e.Level)
}
