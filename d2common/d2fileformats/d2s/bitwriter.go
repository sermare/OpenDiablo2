package d2s

// bitSink is the inverse of bitReader: it appends LSB-first bit fields.
type bitSink struct {
	buf []byte
	pos int // position in bits
}

// write appends the low n (0..64) bits of v.
func (w *bitSink) write(v uint64, n int) {
	for i := 0; i < n; i++ {
		if w.pos/8 >= len(w.buf) {
			w.buf = append(w.buf, 0)
		}

		if v>>uint(i)&1 == 1 {
			w.buf[w.pos/8] |= 1 << uint(w.pos%8)
		}

		w.pos++
	}
}

// align pads with zero bits to the next byte boundary.
func (w *bitSink) align() {
	w.pos = (w.pos + 7) &^ 7

	for len(w.buf) < w.pos/8 {
		w.buf = append(w.buf, 0)
	}
}
