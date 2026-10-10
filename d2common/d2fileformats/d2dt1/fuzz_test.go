package d2dt1

import "testing"

// synthDT1 builds a one-tile, two-block DT1 (one RLE block, one isometric block) in memory.
func synthDT1() []byte {
	d := New()
	d.numberOfTiles = 1
	d.bodyPosition = 276 // after the file header

	rle := []byte{2, 3, 7, 8, 9, 0, 0, 1, 1, 5} // skip 2, draw 3 pixels; new row; skip 1, draw 1 pixel
	iso := make([]byte, 256)

	for i := range iso {
		iso[i] = byte(i)
	}

	const tileHeader, blockHeader = 96, 20

	t := Tile{
		Direction: 1, Width: 80, Height: -32, Type: 1, Style: 2,
		unknown2:           make([]byte, 4),
		blockHeaderPointer: int32(d.bodyPosition) + tileHeader,
		Blocks: []Block{
			{X: 0, Y: 0, GridX: 0, GridY: 0, format: int16(BlockFormatRLE), Length: int32(len(rle)), FileOffset: 2 * blockHeader, EncodedData: rle},
			{X: 16, Y: 0, GridX: 1, GridY: 0, format: int16(BlockFormatIsometric), Length: int32(len(iso)), FileOffset: 2*blockHeader + int32(len(rle)), EncodedData: iso},
		},
	}
	d.Tiles = []Tile{t}

	return d.Marshal()
}

// TestSynthDT1 checks the seed loads and decodes, so the fuzz target starts from a valid file.
func TestSynthDT1(t *testing.T) {
	d, err := LoadDT1(synthDT1())
	if err != nil {
		t.Fatal(err)
	}

	if len(d.Tiles) != 1 || len(d.Tiles[0].Blocks) != 2 {
		t.Fatalf("tiles %d", len(d.Tiles))
	}

	pixels := make([]byte, 4096)
	DecodeTileGfxData(d.Tiles[0].Blocks, &pixels, 16, 64)
}

// FuzzLoadDT1 feeds arbitrary bytes to the DT1 loader; it must never panic.
func FuzzLoadDT1(f *testing.F) {
	f.Add([]byte{})
	f.Add(synthDT1())
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
