package d2s

import (
	"bytes"
	"errors"
	"testing"
)

// walkItems calls f for every item of the character, socketed children included.
func walkItems(c *Character, f func(it *Item)) {
	var rec func(list []Item)

	rec = func(list []Item) {
		for i := range list {
			f(&list[i])
			rec(list[i].Children)
		}
	}

	rec(c.Items)
	rec(c.Corpse)
	rec(c.MercItems)
}

// Every item of the real sample: parse -> EncodeItem -> parse again gives the
// same bits, and NewItem, which re-derives the flags and the property order,
// reproduces the original bits (so the encoder's rules are the game's).
func TestEncodeItemRealItems(t *testing.T) {
	data, tables := realSample(t)

	c, err := Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	n, extended, withProps, sockets, ethereal := 0, 0, 0, 0, 0
	quality := map[uint8]int{}

	walkItems(c, func(it *Item) {
		if it.Ear {
			return
		}

		n++

		orig, err := EncodeItem(it, tables)
		if err != nil {
			t.Fatalf("item %q: %v", it.Code, err)
		}

		if it.Simple != tables.IsCompact(it.Code) {
			t.Errorf("item %q: simple flag %v but compactsave %v", it.Code, it.Simple, tables.IsCompact(it.Code))
		}

		clone := *it
		clone.Children = nil
		clone.SocketCount = 0

		if len(it.Children) > 0 {
			clone.Children = it.Children
			clone.SocketCount = it.SocketCount
		}

		built, err := NewItem(clone, tables)
		if err != nil {
			t.Fatalf("item %q: NewItem: %v", it.Code, err)
		}

		got, err := EncodeItem(&built, tables)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(got, orig) {
			t.Errorf("item %q (quality %d): NewItem bits differ at byte %d", it.Code, it.Quality, firstDiff(got, orig))
		}

		if !it.Simple {
			extended++
			quality[it.Quality]++
		}

		if len(it.Properties) > 0 {
			withProps++
		}

		if it.Socketed {
			sockets++
		}

		if it.Ethereal {
			ethereal++
		}
	})

	t.Logf("%d items: %d extended, %d with properties, %d socketed, %d ethereal, qualities %v",
		n, extended, withProps, sockets, ethereal, quality)

	if n == 0 {
		t.Fatal("the sample has no items")
	}
}

func TestNewItemSynthetic(t *testing.T) {
	tb := miniTables(t)

	// a plain item gets the version and the normal quality
	k, err := NewItem(Item{Code: "key", ID: 7, Level: 5}, tb)
	if err != nil {
		t.Fatal(err)
	}

	if k.Quality != QualityNormal || k.Version != DefaultItemVersion {
		t.Fatalf("plain item: %+v", k)
	}

	if _, err := NewItem(Item{Code: "key", Quality: 9}, tb); !errors.Is(err, ErrItemQuality) {
		t.Fatalf("quality 9: %v", err)
	}

	if _, err := NewItem(Item{Code: "zzz"}, tb); !errors.Is(err, ErrUnknownItem) {
		t.Fatalf("unknown code: %v", err)
	}

	if _, err := NewItem(Item{Code: "key", Ear: true}, tb); err == nil {
		t.Fatal("ear accepted")
	}
}

func TestSortProperties(t *testing.T) {
	in := []Property{{ID: 48, Value: 1}, {ID: 49, Value: 2}, {ID: 7, Value: 3}, {ID: 0, Value: 4}}

	out, err := SortProperties(in)
	if err != nil {
		t.Fatal(err)
	}

	want := []int{0, 7, 48, 49}
	for i, p := range out {
		if p.ID != want[i] {
			t.Fatalf("order %v", out)
		}
	}

	for name, bad := range map[string][]Property{
		"missing follower": {{ID: 48}},
		"orphan follower":  {{ID: 49}},
	} {
		if _, err := SortProperties(bad); !errors.Is(err, ErrPropertyOrder) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
