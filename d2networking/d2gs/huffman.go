package d2gs

import (
	"errors"
	"fmt"
	"sort"
)

// huffLengths are the 256 code lengths of the packet Huffman code, read from
// the table at 0x00707778 of Game.exe 1.14b with a read-only Ghidra session
// (the same bytes FUN_00406600 copies there while building the code). Index =
// byte value, entry = code length in bits (1..11).
var huffLengths = [256]uint8{
	1, 4, 6, 7, 7, 6, 7, 7, 7, 8, 7, 8, 8, 7, 9, 8,
	8, 8, 7, 6, 6, 7, 8, 8, 8, 9, 9, 10, 10, 8, 7, 8,
	8, 10, 10, 10, 10, 9, 10, 10, 9, 10, 10, 10, 10, 10, 10, 11,
	9, 9, 10, 10, 10, 10, 10, 10, 10, 10, 10, 9, 10, 10, 10, 10,
	9, 10, 10, 9, 10, 9, 10, 10, 10, 10, 10, 10, 9, 9, 10, 11,
	9, 8, 9, 10, 11, 9, 9, 10, 9, 10, 10, 10, 11, 10, 10, 10,
	10, 10, 9, 9, 9, 10, 10, 7, 7, 7, 10, 9, 8, 7, 10, 10,
	10, 11, 10, 10, 10, 10, 10, 10, 10, 10, 10, 11, 10, 10, 10, 10,
	7, 10, 10, 10, 11, 11, 11, 11, 11, 10, 10, 11, 11, 11, 11, 10,
	9, 11, 11, 10, 11, 9, 9, 9, 10, 11, 11, 11, 11, 11, 11, 11,
	10, 11, 10, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11,
	10, 11, 11, 11, 11, 11, 11, 11, 10, 11, 11, 11, 11, 11, 11, 11,
	10, 11, 10, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 10, 11,
	11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11,
	10, 11, 10, 11, 11, 11, 11, 11, 10, 11, 11, 11, 11, 11, 11, 11,
	10, 11, 11, 11, 11, 11, 11, 11, 10, 11, 11, 11, 11, 11, 11, 6,
}

// Huffman is the packet compressor (FUN_004069c0 compresses, FUN_00406a70
// decompresses). The code is canonical and assigned exactly like the table
// builder FUN_00406600: symbols are sorted by code length descending (ties by
// byte value ascending); the first symbol gets code 0 and every next symbol
// gets (previous code + 1) >> (previous length - length). Bits are packed
// most-significant first and the last byte is zero padded.
//
// Verified: the length table, the sort and code assignment (from the
// decompilation) and the packing loop. Unverified: the original also stores
// each code in ONE byte (so codes of 9..11 bits would be truncated); we
// check in tests that no code exceeds its byte (it does not), so the byte
// storage never matters. Not compared against real captured traffic.
type Huffman struct {
	code [256]uint16
	// decode maps (length<<16 | code) to the byte value.
	decode map[uint32]byte
	minLen int
	maxLen int
}

// NewHuffman builds the code from the Game.exe length table.
func NewHuffman() *Huffman { return newHuffman(huffLengths) }

func newHuffman(lens [256]uint8) *Huffman {
	order := make([]int, 256)
	for i := range order {
		order[i] = i
	}

	sort.SliceStable(order, func(a, b int) bool { return lens[order[a]] > lens[order[b]] })

	h := &Huffman{decode: make(map[uint32]byte, 256), minLen: 99}

	var code uint32

	for k, sym := range order {
		if k > 0 {
			prev := order[k-1]
			code = (code + 1) >> (lens[prev] - lens[sym])
		}

		h.code[sym] = uint16(code)
		h.decode[uint32(lens[sym])<<16|code] = byte(sym)

		if int(lens[sym]) < h.minLen {
			h.minLen = int(lens[sym])
		}

		if int(lens[sym]) > h.maxLen {
			h.maxLen = int(lens[sym])
		}
	}

	return h
}

// ErrBadHuffman is returned for a stream that does not decode.
var ErrBadHuffman = errors.New("d2gs: invalid huffman data")

// Compress appends the compressed form of src to dst.
func (h *Huffman) Compress(dst, src []byte) []byte {
	var acc uint32

	nbits := 0

	for _, b := range src {
		l := int(huffLengths[b])
		acc = acc<<uint(l) | uint32(h.code[b])
		nbits += l

		for nbits >= 8 {
			nbits -= 8
			dst = append(dst, byte(acc>>uint(nbits)))
		}

		acc &= 1<<uint(nbits) - 1
	}

	if nbits > 0 {
		dst = append(dst, byte(acc<<uint(8-nbits)))
	}

	return dst
}

// Decompress appends the decoded bytes of src to dst. The final byte may carry
// up to 7 padding bits; decoding stops when the remaining bits cannot form a
// code (as the original does when it runs out of input).
func (h *Huffman) Decompress(dst, src []byte) ([]byte, error) {
	var (
		code uint32
		l    int
	)

	for _, by := range src {
		for bit := 7; bit >= 0; bit-- {
			code = code<<1 | uint32(by>>uint(bit)&1)
			l++

			if l >= h.minLen {
				if sym, ok := h.decode[uint32(l)<<16|code]; ok {
					dst = append(dst, sym)
					code, l = 0, 0

					continue
				}
			}

			if l > h.maxLen {
				return dst, fmt.Errorf("%w: no code of %d bits", ErrBadHuffman, l)
			}
		}
	}

	// whatever is left must be padding (fewer than 8 bits, all zero)
	if l >= 8 || code != 0 {
		return dst, fmt.Errorf("%w: %d trailing bits", ErrBadHuffman, l)
	}

	return dst, nil
}
