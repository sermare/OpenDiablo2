package d2s

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Errors returned by ParseStash.
var (
	ErrNotStash      = errors.New("d2s: not a shared stash file")
	ErrStashFormat   = errors.New("d2s: unsupported stash format")
	ErrStashTruncate = errors.New("d2s: stash file is truncated")
)

// Stash file kinds.
const (
	// StashShared is a PlugY shared stash (.sss softcore, .hss hardcore),
	// magic "SSS\0".
	StashShared = "SSS"
	// StashPersonal is a PlugY personal stash (.d2x), magic "CSTM".
	StashPersonal = "CSTM"
)

// Stash is a read-only view of a PlugY stash file: pages of items in the same
// bit format as a .d2s item list.
//
// UNVERIFIED: the header layout below is reconstructed from memory of PlugY's
// format, and there was no real stash file on the development machine to check
// it against. ParseStash is tolerant (it only trusts tags and the item lists)
// and reports the offset of anything it does not understand.
//
//	"SSS\0" | "CSTM"      4 bytes magic
//	version               2 ASCII bytes, "01" or "02"
//	gold                  u32, shared stash only
//	flags                 u32, personal stash version 02 only
//	pages                 u32 page count
//	per page:             "ST", flags u32 (version 02; bit 0 = a name follows),
//	                      NUL-terminated name if flagged, then a 'JM' item list
type Stash struct {
	Kind    string
	Version string
	// Gold is the gold in a shared stash.
	Gold  uint32
	Flags uint32
	Pages []StashPage
}

// StashPage is one stash page.
type StashPage struct {
	Flags uint32
	Name  string
	Items []Item
}

// Items returns every item of every page.
func (s *Stash) Items() []Item {
	var out []Item
	for i := range s.Pages {
		out = append(out, s.Pages[i].Items...)
	}

	return out
}

const maxStashPages = 4096

// ParseStash decodes a PlugY stash file. A Diablo II: Resurrected .d2i (it
// starts like a save, with 0xAA55AA55) is recognised and refused with
// ErrStashFormat because its item encoding differs from 1.14's.
func ParseStash(data []byte, tables *ItemTables) (s *Stash, err error) {
	err = safely(func() error {
		var perr error
		s, perr = parseStash(data, tables)

		return perr
	})
	if err != nil {
		return nil, err
	}

	return s, nil
}

func parseStash(data []byte, tables *ItemTables) (*Stash, error) {
	if tables == nil {
		return nil, ErrNoTables
	}

	if len(data) >= 4 && binary.LittleEndian.Uint32(data) == Magic {
		return nil, fmt.Errorf("%w: a Resurrected .d2i uses a different item encoding", ErrStashFormat)
	}

	s := &Stash{}

	switch {
	case bytes.HasPrefix(data, []byte("SSS\x00")):
		s.Kind = StashShared
	case bytes.HasPrefix(data, []byte("CSTM")):
		s.Kind = StashPersonal
	default:
		return nil, ErrNotStash
	}

	pos := 4
	rd := func(n int) ([]byte, error) {
		if pos+n > len(data) {
			return nil, fmt.Errorf("%w: need %d bytes at 0x%X", ErrStashTruncate, n, pos)
		}

		b := data[pos : pos+n]
		pos += n

		return b, nil
	}

	v, err := rd(2)
	if err != nil {
		return nil, err
	}

	s.Version = string(v)
	if s.Version != "01" && s.Version != "02" {
		return nil, fmt.Errorf("%w: version %q", ErrStashFormat, s.Version)
	}

	le := binary.LittleEndian

	if s.Kind == StashShared {
		b, err := rd(4)
		if err != nil {
			return nil, err
		}

		s.Gold = le.Uint32(b)
	}

	if s.Kind == StashPersonal && s.Version == "02" {
		b, err := rd(4)
		if err != nil {
			return nil, err
		}

		s.Flags = le.Uint32(b)
	}

	b, err := rd(4)
	if err != nil {
		return nil, err
	}

	count := le.Uint32(b)
	if count > maxStashPages {
		return nil, fmt.Errorf("%w: implausible page count %d", ErrStashFormat, count)
	}

	for p := uint32(0); p < count; p++ {
		page, n, err := parseStashPage(data[pos:], s.Version, tables)
		if err != nil {
			return nil, fmt.Errorf("page %d at 0x%X: %w", p, pos, err)
		}

		pos += n

		s.Pages = append(s.Pages, page)
	}

	return s, nil
}

func parseStashPage(data []byte, version string, tables *ItemTables) (StashPage, int, error) {
	var page StashPage

	if len(data) < 2 || data[0] != 'S' || data[1] != 'T' {
		return page, 0, fmt.Errorf("%w: expected page tag 'ST'", ErrBadSection)
	}

	pos := 2

	if version == "02" {
		if len(data) < pos+4 {
			return page, 0, ErrStashTruncate
		}

		page.Flags = binary.LittleEndian.Uint32(data[pos:])
		pos += 4
	}

	if page.Flags&1 != 0 {
		end := bytes.IndexByte(data[pos:], 0)
		if end < 0 {
			return page, 0, fmt.Errorf("%w: unterminated page name", ErrStashTruncate)
		}

		page.Name = string(data[pos : pos+end])
		pos += end + 1
	}

	items, n, err := readItemSection(data, pos, tables)
	if err != nil {
		return page, 0, err
	}

	page.Items = items

	return page, n, nil
}
