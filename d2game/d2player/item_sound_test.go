package d2player

import "testing"

type soundItem struct {
	InventoryItem
	handle string
}

func (s soundItem) DropSoundHandle() string { return s.handle }

func TestItemSound(t *testing.T) {
	var got []string

	SetItemSoundHook(func(handle, kind string) { got = append(got, kind+":"+handle) })
	t.Cleanup(func() { SetItemSoundHook(nil) })

	for _, tc := range []struct {
		name string
		it   InventoryItem
		want int
	}{
		{"item with a drop sound", soundItem{handle: "item_sword"}, 1},
		{"item without a drop sound", soundItem{}, 0},
		{"item type without the method", nil, 0},
	} {
		got = nil

		itemSound(tc.it, "item-drop")

		if len(got) != tc.want {
			t.Errorf("%s: got %v, want %d sounds", tc.name, got, tc.want)
		}
	}
}
