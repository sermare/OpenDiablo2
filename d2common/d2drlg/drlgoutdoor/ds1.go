package drlgoutdoor

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Group is one rectangle of a pattern sheet (the original's 0x18 byte record
// {x, y, w, h, 0, nvar}).
type Group struct{ X, Y, W, H, NVar int }

// Pattern is the part of a DS1 file the outdoor generator reads: the wall,
// orientation, floor and "shadow" layers plus the sub-theme group rectangles.
type Pattern struct {
	W, H   int
	Wall   [][]uint32 // [layer][y*W+x]
	Orient [][]uint32
	Floor  [][]uint32
	Shadow []uint32
	Groups []Group
}

// ParsePattern decodes a DS1 file. The buffer is read with 64 zero bytes of
// padding behind it because the game does the same over-read for the version
// 12 Trees.ds1 (13 stored groups, count 14: the 14th group is all zero).
func ParsePattern(data []byte) (*Pattern, error) {
	b := append(append([]byte(nil), data...), make([]byte, 64)...)
	o := 0

	i32 := func() int {
		if o+4 > len(b) {
			o = len(b)
			return 0
		}

		v := int(int32(binary.LittleEndian.Uint32(b[o:])))
		o += 4

		return v
	}

	if len(data) < 16 {
		return nil, errors.New("drlgoutdoor: ds1 too short")
	}

	ver := i32()
	p := &Pattern{}
	p.W = i32() + 1
	p.H = i32() + 1

	if ver >= 8 {
		i32() // act
	}

	subst := 0
	if ver >= 10 {
		subst = i32()
	}

	if ver >= 3 {
		n := i32()
		for k := 0; k < n; k++ {
			for o < len(b) && b[o] != 0 {
				o++
			}

			o++
		}
	}

	if ver >= 9 && ver <= 13 {
		o += 8
	}

	nwall := 1
	if ver >= 4 {
		nwall = i32()
	}

	nfloor := 1
	if ver >= 16 {
		nfloor = i32()
	}

	n := p.W * p.H
	if n <= 0 || nwall < 0 || nfloor < 0 || nwall > 8 || nfloor > 8 || n*4*(2*nwall+nfloor+1) > len(data) {
		return nil, fmt.Errorf("drlgoutdoor: ds1 layer sizes out of range (%dx%d, %d wall, %d floor)", p.W, p.H, nwall, nfloor)
	}

	layer := func() []uint32 {
		l := make([]uint32, n)
		for k := range l {
			l[k] = binary.LittleEndian.Uint32(b[o+4*k:])
		}

		o += 4 * n

		return l
	}

	for k := 0; k < nwall; k++ {
		p.Wall = append(p.Wall, layer())
		p.Orient = append(p.Orient, layer())
	}

	for k := 0; k < nfloor; k++ {
		p.Floor = append(p.Floor, layer())
	}

	p.Shadow = layer()

	if subst == 1 || subst == 2 {
		layer() // substitution layer (unused)
	}

	if ver >= 2 {
		o += 20 * i32() // objects
	}

	if ver >= 12 && (subst == 1 || subst == 2) {
		if ver >= 18 {
			o += 4
		}

		ng := i32()
		if ng < 0 || ng > 4096 {
			return nil, fmt.Errorf("drlgoutdoor: ds1 group count %d", ng)
		}

		for k := 0; k < ng; k++ {
			g := Group{X: i32(), Y: i32(), W: i32(), H: i32()}
			if ver >= 13 {
				g.NVar = i32()
			}

			p.Groups = append(p.Groups, g)
		}
	}

	return p, nil
}

func (p *Pattern) wall(x, y int) uint32 {
	if len(p.Wall) == 0 {
		return 0
	}

	return p.Wall[0][y*p.W+x]
}

func (p *Pattern) floor(x, y int) uint32 {
	if len(p.Floor) == 0 {
		return 0
	}

	return p.Floor[0][y*p.W+x]
}
