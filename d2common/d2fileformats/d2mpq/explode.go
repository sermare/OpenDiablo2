package d2mpq

import (
	"errors"
	"sync"
)

// This file decodes the PKWare Data Compression Library ("implode") format of MPQ sectors. It is a
// rewrite for speed of the github.com/JoshVarga/blast reader (itself a port of Mark Adler's blast.c
// from zlib, zlib/libpng licence: (C) 2003, 2012, 2013 Mark Adler, (C) 2018 Josh Varga). Altered
// source version, plainly marked: the tables below are the ones of blast.c, but the Huffman codes are
// decoded with a lookup table over a 64 bit bit buffer instead of one bit at a time, the input
// is the whole sector and the output is one pre-sized slice. Decoding DT1/DS1/DCC sectors was the
// largest cost of loading a level; the old reader also allocated 16 KB per call.
//
// Format (see blast.c): byte 0 is 1 when literals are Huffman coded, byte 1 (4, 5 or 6) the extra
// distance bits; then, LSB first, a flag bit per item: 0 literal, 1 length/distance pair; length
// 519 ends the stream. Codes are stored bit-reversed and inverted relative to the canonical order.

var (
	errExplodeHeader   = errors.New("explode: invalid header")
	errExplodeDict     = errors.New("explode: invalid dictionary")
	errExplodeDistance = errors.New("explode: distance is too far back")
	errExplodeEOF      = errors.New("explode: unexpected end of input")
	errExplodeCode     = errors.New("explode: invalid code")
)

const (
	explodeMaxBits = 13
	explodeEndCode = 519
)

// code lengths in the compact form of blast.c: high four bits + 1 = repeat count, low four bits = length
var (
	explodeLiteralLengths = []byte{ //nolint:gochecknoglobals // constant tables
		11, 124, 8, 7, 28, 7, 188, 13, 76, 4, 10, 8, 12, 10, 12, 10, 8, 23, 8,
		9, 7, 6, 7, 8, 7, 6, 55, 8, 23, 24, 12, 11, 7, 9, 11, 12, 6, 7, 22, 5,
		7, 24, 6, 11, 9, 6, 7, 22, 7, 11, 38, 7, 9, 8, 25, 11, 8, 11, 9, 12,
		8, 12, 5, 38, 5, 38, 5, 11, 7, 5, 6, 21, 6, 10, 53, 8, 7, 24, 10, 27,
		44, 253, 253, 253, 252, 252, 252, 13, 12, 45, 12, 45, 12, 61, 12, 45,
		44, 173}
	explodeLengthLengths   = []byte{2, 35, 36, 53, 38, 23}         //nolint:gochecknoglobals // constant tables
	explodeDistanceLengths = []byte{2, 20, 53, 230, 247, 151, 248} //nolint:gochecknoglobals // constant tables
	explodeLenBase         = [16]int{3, 2, 4, 5, 6, 7, 8, 9, 10, 12, 16, 24, 40, 72, 136, 264}
	explodeLenExtra        = [16]uint{0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8}
)

// explodeTable maps the next explodeMaxBits stream bits to symbol<<4 | code length (0: invalid code).
type explodeTable [1 << explodeMaxBits]uint16

//nolint:gochecknoglobals // built once
var (
	explodeOnce                               sync.Once
	explodeLit, explodeLenTab, explodeDistTab *explodeTable
)

// buildExplodeTable builds the lookup table of one canonical code given as blast.c's compact lengths.
func buildExplodeTable(rep []byte) *explodeTable {
	var lengths [256]int

	n := 0

	for _, r := range rep {
		for k := int(r>>4) + 1; k > 0; k-- {
			lengths[n] = int(r & 15)
			n++
		}
	}

	var count [explodeMaxBits + 1]int
	for i := 0; i < n; i++ {
		count[lengths[i]]++
	}

	// symbols sorted by length, then by value (blast.c's symbol[])
	var offs [explodeMaxBits + 2]int
	for l := 1; l <= explodeMaxBits; l++ {
		offs[l+1] = offs[l] + count[l]
	}

	symbol := make([]int, n)

	for i := 0; i < n; i++ {
		if lengths[i] != 0 {
			symbol[offs[lengths[i]]] = i
			offs[lengths[i]]++
		}
	}

	t := new(explodeTable)
	first, index := 0, 0

	for l := 1; l <= explodeMaxBits; l++ {
		for c := 0; c < count[l]; c++ {
			code := first + c // value of the inverted bits, first stream bit as the most significant one
			sym := symbol[index+c]

			// the stream delivers bit k (LSB first) as NOT(code bit l-1-k)
			rev := 0
			for k := 0; k < l; k++ {
				rev |= (((code >> (l - 1 - k)) & 1) ^ 1) << k
			}

			for hi := 0; hi < 1<<(explodeMaxBits-l); hi++ {
				t[rev|hi<<l] = uint16(sym<<4 | l)
			}
		}

		index += count[l]
		first = (first + count[l]) << 1
	}

	return t
}

func initExplode() {
	explodeLit = buildExplodeTable(explodeLiteralLengths)
	explodeLenTab = buildExplodeTable(explodeLengthLengths)
	explodeDistTab = buildExplodeTable(explodeDistanceLengths)
}

// explode decompresses a PKWare DCL stream. sizeHint is the expected output size (0 if unknown).
//
//nolint:funlen,gocyclo // one tight loop
func explode(in []byte, sizeHint int) ([]byte, error) {
	explodeOnce.Do(initExplode)

	if len(in) < 2 {
		return nil, errExplodeEOF
	}

	if in[0] > 1 {
		return nil, errExplodeHeader
	}

	coded := in[0] == 1
	dict := uint(in[1])

	if dict < 4 || dict > 6 {
		return nil, errExplodeDict
	}

	out := make([]byte, 0, sizeHint)
	pos := 2

	var (
		acc uint64
		cnt uint
	)

	// need makes sure at least n (<= 32) bits are buffered; past the end of the input the buffer
	// is padded with zero bits and the loop below checks overrun at the end of every item.
	pad := 0
	refill := func() {
		for cnt <= 56 {
			if pos < len(in) {
				acc |= uint64(in[pos]) << cnt
				pos++
			} else {
				pad++
			}

			cnt += 8
		}
	}

	for {
		refill()

		if pad > 8 { // consumed more than the input holds
			return nil, errExplodeEOF
		}

		if acc&1 == 0 { // literal
			acc >>= 1
			cnt--

			var b byte

			if coded {
				e := explodeLit[acc&(1<<explodeMaxBits-1)]
				if e == 0 {
					return nil, errExplodeCode
				}

				l := uint(e & 15)
				b = byte(e >> 4)
				acc >>= l
				cnt -= l
			} else {
				b = byte(acc)
				acc >>= 8
				cnt -= 8
			}

			out = append(out, b)

			continue
		}

		acc >>= 1
		cnt--

		e := explodeLenTab[acc&(1<<explodeMaxBits-1)]
		if e == 0 {
			return nil, errExplodeCode
		}

		l := uint(e & 15)
		sym := int(e >> 4)
		acc >>= l
		cnt -= l

		x := explodeLenExtra[sym]
		length := explodeLenBase[sym] + int(acc&(1<<x-1))
		acc >>= x
		cnt -= x

		if length == explodeEndCode {
			break
		}

		e = explodeDistTab[acc&(1<<explodeMaxBits-1)]
		if e == 0 {
			return nil, errExplodeCode
		}

		l = uint(e & 15)
		dsym := uint(e >> 4)
		acc >>= l
		cnt -= l

		shift := dict
		if length == 2 {
			shift = 2
		}

		dist := int(dsym<<shift) + int(acc&(1<<shift-1)) + 1
		acc >>= shift
		cnt -= shift

		if dist > len(out) {
			return nil, errExplodeDistance
		}

		// forward copy: overlapping runs repeat the last dist bytes
		start := len(out) - dist
		if dist >= length {
			out = append(out, out[start:start+length]...)
		} else {
			for i := 0; i < length; i++ {
				out = append(out, out[start+i])
			}
		}
	}

	if pad > 0 && cnt < uint(pad)*8 { // the end code used bits that were not in the input
		return nil, errExplodeEOF
	}

	return out, nil
}
