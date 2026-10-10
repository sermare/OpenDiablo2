package d2dt1

import "testing"

// FuzzLoadDT1 feeds arbitrary bytes to the DT1 loader; it must never panic.
func FuzzLoadDT1(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{7, 0, 0, 0, 6, 0, 0, 0})
	f.Add(append([]byte{7, 0, 0, 0, 6, 0, 0, 0}, make([]byte, 300)...))
	f.Fuzz(func(t *testing.T, data []byte) {
		d, err := LoadDT1(data)
		if err != nil {
			return
		}

		for i := range d.Tiles {
			if i > 32 {
				break
			}

			tile := &d.Tiles[i]
			pixels := make([]byte, 4096)
			DecodeTileGfxData(tile.Blocks, &pixels, 16, 64)
		}

		_ = d.Marshal()
	})
}
