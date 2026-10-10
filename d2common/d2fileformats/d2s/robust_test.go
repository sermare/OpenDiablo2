package d2s

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Robustness tests: malformed input must yield errors, never panics.
// Real saves are read from the developer's machine only (never committed);
// tests skip when they are absent.

func homePath(parts ...string) string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(append([]string{h}, parts...)...)
}

// realSaves returns the available real saves.
func realSaves() [][]byte {
	var out [][]byte

	paths, _ := filepath.Glob(homePath("git", "d2s-test", "*.d2s"))
	paths = append(paths, homePath("git", "nokka-d2s-ref", "examples", "nokkasorc"))

	for _, p := range paths {
		if b, err := os.ReadFile(p); err == nil {
			out = append(out, b)
		}
	}

	return out
}

// robustTables loads the real tables from $D2_TABLES or ~/git/d2-tables,
// or returns nil.
func robustTables() *ItemTables {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		dir = homePath("git", "d2-tables")
	}

	read := func(names ...string) []byte {
		for _, n := range names {
			if b, err := os.ReadFile(filepath.Join(dir, n)); err == nil {
				return b
			}
		}

		return nil
	}

	tb, err := NewItemTables(read("itemstatcost.bin", "ItemStatCost.txt"),
		read("armor.txt"), read("weapons.txt"), read("misc.txt"), read("ItemTypes.txt"))
	if err != nil {
		return nil
	}

	return tb
}

// fixup makes the size and checksum fields consistent so mutated input gets
// past the header checks and reaches the body/item parsers.
func fixup(d []byte) []byte {
	if len(d) < HeaderSize {
		return d
	}

	le := binary.LittleEndian
	le.PutUint32(d[8:], uint32(len(d)))
	le.PutUint32(d[checksumOffset:], 0)
	le.PutUint32(d[checksumOffset:], Checksum(d))

	return d
}

func mustNotPanic(t *testing.T, name string, f func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s: panic: %v", name, r)
		}
	}()

	f()
}

func TestParseTruncatedAndCorrupt(t *testing.T) {
	saves, tables := realSaves(), robustTables()
	if len(saves) == 0 || tables == nil {
		t.Skip("real saves or tables not available")
	}

	le := binary.LittleEndian

	for si, orig := range saves {
		clone := func() []byte { return append([]byte(nil), orig...) }

		cases := []struct {
			name string
			data []byte
			want error // nil: any error is acceptable
		}{
			{"empty", nil, ErrTooShort},
			{"short header", clone()[:HeaderSize-1], ErrTooShort},
		}

		bad := clone()
		bad[len(bad)-1] ^= 0xFF
		cases = append(cases, struct {
			name string
			data []byte
			want error
		}{"bad checksum", bad, ErrBadChecksum})

		wrongMagic := clone()
		wrongMagic[0] ^= 1
		cases = append(cases, struct {
			name string
			data []byte
			want error
		}{"bad magic", wrongMagic, ErrBadMagic})

		ver := clone()
		le.PutUint32(ver[4:], 0x7FFFFFFF)
		cases = append(cases, struct {
			name string
			data []byte
			want error
		}{"wrong version", fixup(ver), ErrBadVersion})

		cls := clone()
		cls[classOffset] = 0xEE
		cases = append(cases, struct {
			name string
			data []byte
			want error
		}{"invalid class", fixup(cls), ErrInvalidClass})

		// every truncation point with a valid size/checksum
		for n := HeaderSize; n < len(orig); n += 7 {
			cases = append(cases, struct {
				name string
				data []byte
				want error
			}{"truncated", fixup(clone()[:n]), nil})
		}

		for _, tc := range cases {
			mustNotPanic(t, tc.name, func() {
				c, err := Parse(tc.data, tables)
				if err == nil && tc.name != "truncated" {
					t.Errorf("save %d %s: expected error, got %v", si, tc.name, c.Header)
				}

				if tc.want != nil && !errors.Is(err, tc.want) {
					t.Errorf("save %d %s: err=%v, want %v", si, tc.name, err, tc.want)
				}
			})
		}
	}
}

func TestParseHugeItemCount(t *testing.T) {
	tables := robustTables()
	saves := realSaves()

	if tables == nil || len(saves) == 0 {
		t.Skip("real saves or tables not available")
	}

	for _, orig := range saves {
		d := append([]byte(nil), orig...)
		c, err := Parse(d, tables)

		if err != nil || c.Body == nil {
			continue
		}

		pos := c.Body.ItemsOffset
		d[pos+2], d[pos+3] = 0xFF, 0xFF // 65535 items claimed

		mustNotPanic(t, "huge count", func() {
			if _, err := Parse(fixup(d), tables); err == nil {
				t.Error("65535-item claim parsed without error")
			}
		})
	}
}

func TestParseCorruptSections(t *testing.T) {
	tables := robustTables()
	saves := realSaves()

	if tables == nil || len(saves) == 0 {
		t.Skip("real saves or tables not available")
	}

	// flip each byte after the header (with a repaired checksum): corrupt
	// bit-lengths, unknown item codes, merc/golem sections, section tags.
	for si, orig := range saves {
		for i := HeaderSize; i < len(orig); i++ {
			for _, x := range []byte{0xFF, 0x01, 0x80} {
				d := append([]byte(nil), orig...)
				d[i] ^= x

				mustNotPanic(t, "flip", func() {
					c, err := Parse(fixup(d), tables)
					if err != nil {
						return
					}

					// anything that parses must also be writable without panic
					_, _ = Write(c, tables)
				})

				if t.Failed() {
					t.Fatalf("save %d byte 0x%X ^0x%X", si, i, x)
				}
			}
		}
	}
}

func TestParseItemListUnknownCode(t *testing.T) {
	tb := miniTables(t)
	w := &bitWriter{}
	w.write('J'|'M'<<8, 16)
	w.write(1, 16) // count
	writeItemHeader(w, 0, 1, 1, 0, 0, 0)
	writeCode(w, "zzzz")
	w.write(0, 3)
	// extended part is missing/garbage: must error, not panic
	data := append([]byte{'J', 'M', 1, 0}, w.bytes()...)

	mustNotPanic(t, "unknown code", func() {
		if _, _, err := ParseItemList(data, tb); err == nil {
			t.Error("expected error")
		}
	})
}

func FuzzParse(f *testing.F) {
	tables := robustTables()
	if tables == nil {
		f.Skip("tables not available")
	}

	for _, s := range realSaves() {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		for _, d := range [][]byte{data, fixup(append([]byte(nil), data...))} {
			c, err := Parse(d, tables)
			if err != nil {
				continue
			}

			if _, err := Write(c, tables); err != nil {
				continue
			}
		}
	})
}

func FuzzParseItemList(f *testing.F) {
	tables := robustTables()
	if tables == nil {
		f.Skip("tables not available")
	}

	for _, s := range realSaves() {
		c, err := Parse(s, tables)
		if err == nil && c.Body != nil {
			f.Add(s[c.Body.ItemsOffset:])
		}
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _, _ = ParseItemList(data, tables)
	})
}
