package d2itemdesc

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"

// SetItemIndex returns the setitems.txt index name of a set item ("" when it
// is not a known set item).
func (t *Tables) SetItemIndex(it *d2s.Item) string {
	if it.Quality != d2s.QualitySet || int(it.SetID) >= len(t.SetItems) {
		return ""
	}

	return t.SetItems[it.SetID].Index
}
