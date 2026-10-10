package d2dcc

import (
	"encoding/binary"
	"testing"
)

// bitW writes bits least significant first, like the DCC bit reader reads them.
type bitW struct {
	buf  []byte
	bits int
}

func (w *bitW) put(v uint64, n int) {
	for i := 0; i < n; i++ {
		if w.bits%8 == 0 {
			w.buf = append(w.buf, 0)
		}

		if v>>uint(i)&1 == 1 {
			w.buf[w.bits/8] |= 1 << uint(w.bits%8)
		}

		w.bits++
	}
}

// synthDCC builds a one direction, one frame, 4x4 pixel DCC in memory (no game data): a single
// cell coded with pixel displacements and 2 bit palette picks.
func synthDCC() []byte {
	var d bitW

	d.put(0, 32) // OutSizeCoded
	d.put(0, 2)  // compression flags: no equal cells, no raw/encoding streams

	d.put(0, 4) // Variable0 bits: crazyBitTable[0] = 0
	d.put(4, 4) // width bits: crazyBitTable[4] = 6
	d.put(4, 4) // height bits
	d.put(4, 4) // x offset bits
	d.put(4, 4) // y offset bits
	d.put(0, 4) // optional bits: 0
	d.put(0, 4) // coded bytes bits: 0

	// frame header
	d.put(4, 6) // width
	d.put(4, 6) // height
	d.put(0, 6) // x offset
	d.put(3, 6) // y offset (bottom row of the frame)
	d.put(0, 1) // not bottom up

	d.put(0, 20) // pixel mask bitstream size (the cell uses the default mask)

	for i := 0; i < 256; i++ { // palette: the first 16 indices are used
		if i < 16 {
			d.put(1, 1)
		} else {
			d.put(0, 1)
		}
	}

	// pixel codes: four displacements (1, 2, 3, 4), then 16 pixels of 2 bits
	for _, v := range []uint64{1, 2, 3, 4} {
		d.put(v, 4)
	}

	for i := 0; i < 16; i++ {
		d.put(uint64(i%4), 2)
	}

	dir := d.buf

	out := []byte{0x74, 6, 1}
	out = binary.LittleEndian.AppendUint32(out, 1) // frames per direction
	out = binary.LittleEndian.AppendUint32(out, 1) // the constant 1
	out = binary.LittleEndian.AppendUint32(out, uint32(len(dir)))
	out = binary.LittleEndian.AppendUint32(out, uint32(len(out)+4)) // direction offset (bytes)

	return append(out, dir...)
}

func TestSynthDCC(t *testing.T) {
	d, err := Load(synthDCC())
	if err != nil {
		t.Fatal(err)
	}

	if len(d.Directions) != 1 || len(d.Directions[0].Frames) != 1 {
		t.Fatalf("unexpected structure: %+v", d)
	}
}

// FuzzLoad feeds arbitrary bytes to the DCC loader; it must never panic.
func FuzzLoad(f *testing.F) {
	f.Add([]byte{})
	f.Add(synthDCC())
	f.Add([]byte{0x74, 6, 1, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 12, 0, 0, 0})
	f.Add(append([]byte{0x74, 6, 1, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 12, 0, 0, 0}, make([]byte, 100)...))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Load(data)
	})
}
