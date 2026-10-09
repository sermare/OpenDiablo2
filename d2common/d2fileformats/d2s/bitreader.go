package d2s

import "errors"

// ErrUnexpectedEOF is returned when a bit read runs past the end of the data.
var ErrUnexpectedEOF = errors.New("d2s: unexpected end of data")

// bitReader reads little-endian, LSB-first bit fields from a byte slice, the
// packing Diablo II uses for items and (in newer versions) for the stats
// section: the first bit read is bit 0 of byte 0.
type bitReader struct {
	data []byte
	pos  int // position in bits
}

func newBitReader(data []byte) *bitReader {
	return &bitReader{data: data}
}

// read returns the next n (0..64) bits as an unsigned integer.
func (r *bitReader) read(n int) (uint64, error) {
	if n < 0 || n > 64 {
		return 0, errors.New("d2s: bad bit count")
	}

	if r.pos+n > len(r.data)*8 {
		return 0, ErrUnexpectedEOF
	}

	var v uint64

	for i := 0; i < n; {
		p := r.pos + i
		take := 8 - p%8

		if rest := n - i; take > rest {
			take = rest
		}

		chunk := (uint64(r.data[p/8]) >> uint(p%8)) & (1<<uint(take) - 1)
		v |= chunk << uint(i)
		i += take
	}

	r.pos += n

	return v, nil
}

// bit reads a single bit as a bool.
func (r *bitReader) bit() (bool, error) {
	v, err := r.read(1)
	return v == 1, err
}

// align skips to the next byte boundary.
func (r *bitReader) align() {
	r.pos = (r.pos + 7) &^ 7
}

// bytesUsed is the number of bytes touched so far (rounded up).
func (r *bitReader) bytesUsed() int {
	return (r.pos + 7) / 8
}
