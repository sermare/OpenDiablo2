package d2player

// itemSoundHook plays a Sounds.txt handle for an item the hero moves between
// the cursor and a grid. The game screen sets it (it owns the sound engine).
var itemSoundHook func(handle, kind string)

// SetItemSoundHook sets the function that plays item handling sounds.
func SetItemSoundHook(fn func(handle, kind string)) { itemSoundHook = fn }

// itemSound plays the sound of an item being picked up or put down: the item's
// own dropsound column (Sounds.txt handle such as item_sword). Which sound the
// original plays for a pickup from a grid is UNVERIFIED; the dropsound is used
// for both.
func itemSound(it InventoryItem, kind string) {
	if itemSoundHook == nil || it == nil {
		return
	}

	if s, ok := it.(interface{ DropSoundHandle() string }); ok {
		if h := s.DropSoundHandle(); h != "" {
			itemSoundHook(h, kind)
		}
	}
}
