package d2s

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
)

// DefaultItemVersion is the item version field of items made by the 1.14
// game (the sample saves hold 101 for every item).
const DefaultItemVersion = 101

// Errors returned by NewItem.
var (
	ErrItemQuality  = errors.New("d2s: unsupported item quality")
	ErrItemMismatch = errors.New("d2s: item does not survive an encode/parse cycle")
)

// EncodeItem returns the bits of one item followed by its socketed children,
// exactly as the item sits in a 'JM' list (without the list header). It is the
// inverse of ParseItemList for one entry.
func EncodeItem(it *Item, t *ItemTables) ([]byte, error) {
	if t == nil {
		return nil, ErrNoTables
	}

	if !it.Simple && int(it.SocketCount) != len(it.Children) {
		return nil, ErrSocketCount
	}

	out, err := marshalItem(it, t)
	if err != nil {
		return nil, err
	}

	for i := range it.Children {
		cb, err := marshalItem(&it.Children[i], t)
		if err != nil {
			return nil, fmt.Errorf("socketed item %d: %w", i, err)
		}

		out = append(out, cb...)
	}

	return out, nil
}

// SortProperties puts a property list in the order the game writes it:
// ascending stat id (a stat that occurs several times, with different
// parameters, keeps its relative order), with the followers of a group stat
// (min/max damage pairs) directly behind their leader. A group stat whose
// followers are missing is an error.
func SortProperties(props []Property) ([]Property, error) {
	follower := map[int]bool{}

	for _, fs := range groupFollowers {
		for _, f := range fs {
			follower[f] = true
		}
	}

	var groups [][]Property

	for i := 0; i < len(props); i++ {
		p := props[i]
		if follower[p.ID] {
			return nil, fmt.Errorf("%w: follower stat %d without its leader", ErrPropertyOrder, p.ID)
		}

		g := []Property{p}

		for _, f := range groupFollowers[p.ID] {
			i++

			if i >= len(props) || props[i].ID != f {
				return nil, fmt.Errorf("%w: stat %d needs follower %d", ErrPropertyOrder, p.ID, f)
			}

			g = append(g, props[i])
		}

		groups = append(groups, g)
	}

	sort.SliceStable(groups, func(a, b int) bool { return groups[a][0].ID < groups[b][0].ID })

	out := make([]Property, 0, len(props))
	for _, g := range groups {
		out = append(out, g...)
	}

	return out, nil
}

// NewItem completes a hand-built item (position, code, quality and the
// quality's identity fields, base stats, properties) into an item that can be
// written: it derives the flag bits (simple, socketed, runeword) and the item
// version from the data, sorts the property lists into the order the game
// writes them and proves the result by encoding it, parsing the bits again and
// comparing. It never invents a value: ids, rolls and base stats are the
// caller's.
func NewItem(it Item, t *ItemTables) (Item, error) {
	if t == nil {
		return it, ErrNoTables
	}

	if it.Ear {
		return it, errors.New("d2s: NewItem does not build ears")
	}

	if it.Version == 0 {
		it.Version = DefaultItemVersion
	}

	it.Simple = t.IsCompact(it.Code)

	if it.Simple {
		// the save format has no room for anything but the code
		it.Quality, it.Level, it.ID = 0, 0, 0
		it.Socketed, it.Runeword, it.Personalized = false, false, false
	} else {
		if it.Quality == 0 {
			it.Quality = QualityNormal
		}

		if it.Quality > QualityCrafted {
			return it, fmt.Errorf("%w: %d", ErrItemQuality, it.Quality)
		}

		it.Socketed = it.TotalSockets > 0
		it.Runeword = it.RunewordID != 0

		var err error

		if it.Properties, err = SortProperties(it.Properties); err != nil {
			return it, err
		}

		if it.Runeword {
			if it.RunewordProperties, err = SortProperties(it.RunewordProperties); err != nil {
				return it, err
			}
		}

		for i := range it.SetProperties {
			if it.SetProperties[i], err = SortProperties(it.SetProperties[i]); err != nil {
				return it, err
			}
		}
	}

	b, err := EncodeItem(&it, t)
	if err != nil {
		return it, err
	}

	list := append([]byte{'J', 'M', 1, 0}, b...)

	back, n, err := ParseItemList(list, t)
	if err != nil {
		return it, fmt.Errorf("%w: %v", ErrItemMismatch, err)
	}

	if n != len(list) || len(back) != 1 {
		return it, ErrItemMismatch
	}

	again, err := EncodeItem(&back[0], t)
	if err != nil || !bytes.Equal(again, b) {
		return it, ErrItemMismatch
	}

	return it, nil
}
