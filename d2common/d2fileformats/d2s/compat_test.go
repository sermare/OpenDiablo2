package d2s

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// realSaveFiles lists real saves found on this machine (never modified: tests
// work on in-memory copies). D2S_SAVE_DIRS is a path list of extra folders.
func realSaveFiles() []string {
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(home, "git", "d2s-test"),
		filepath.Join(home, "git", "nokka-d2s-ref", "examples"),
		filepath.Join(home, ".wine-d2classic", "drive_c", "users", filepath.Base(home), "Saved Games", "Diablo II"),
		filepath.Join(home, ".wine-d2classic", "drive_c", "Program Files (x86)", "Diablo II", "Save"),
	}
	dirs = append(dirs, filepath.SplitList(os.Getenv("D2S_SAVE_DIRS"))...)

	var out []string

	if p := os.Getenv("D2S_SAMPLE_BODY"); p != "" {
		out = append(out, p)
	}

	for _, d := range dirs {
		if d == "" {
			continue
		}

		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}

		for _, e := range entries {
			n := strings.ToLower(e.Name())
			if e.IsDir() {
				continue
			}

			// the sample of nokka-d2s-ref has no extension
			if strings.HasSuffix(n, ".d2s") || n == "nokkasorc" {
				out = append(out, filepath.Join(d, e.Name()))
			}
		}
	}

	return out
}

// TestRealSavesRoundTrip: parse -> write is byte-identical for every real save
// found (compared by hash), and the unsupported parts are listed.
func TestRealSavesRoundTrip(t *testing.T) {
	files := realSaveFiles()
	if len(files) == 0 {
		t.Skip("no real saves found")
	}

	var tables *ItemTables
	if os.Getenv("D2_TABLES") != "" {
		tables = loadRealTables(t, false)
	}

	seen := map[[32]byte]bool{}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}

		sum := sha256.Sum256(data)
		if seen[sum] {
			continue
		}

		seen[sum] = true

		t.Run(filepath.Base(f), func(t *testing.T) {
			h, err := ParseHeader(data)
			if err != nil {
				t.Fatalf("header: %v", err)
			}

			if !h.IsNewCharacter() && tables == nil {
				t.Skip("D2_TABLES not set")
			}

			c, err := Parse(data, tables)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			out, err := Write(c, tables)
			if err != nil {
				t.Fatalf("write: %v", err)
			}

			if sha256.Sum256(out) != sum {
				t.Fatalf("re-written bytes differ (len %d vs %d, first diff 0x%X)", len(out), len(data), firstDiff(out, data))
			}

			// re-parse of the output is stable too
			c2, err := Parse(out, tables)
			if err != nil {
				t.Fatalf("re-parse: %v", err)
			}

			if c2.Header.Name != c.Header.Name || len(c2.Items) != len(c.Items) {
				t.Fatalf("re-parse differs")
			}

			t.Logf("%s v=0x%X (%s) class=%v hc=%v dead=%v exp=%v items=%d merc=%d unsupported=%v", h.Name, h.Version,
				VersionName(h.Version), h.Class, h.IsHardcore(), h.IsDead(), h.IsExpansion(), len(c.Items), len(c.MercItems), Unsupported(c))
		})
	}
}

// TestRealSaveVariants flips header fields of a real save (every supported
// version, classic/expansion, softcore/hardcore/dead/ladder and every
// difficulty/act state) and checks the round trip stays byte-identical.
func TestRealSaveVariants(t *testing.T) {
	data, tables := realSample(t)

	le := binary.LittleEndian
	n := 0

	for v := MinVersion; v <= MaxVersion; v++ {
		for _, flags := range []uint32{0, StatusHardcore, StatusHardcore | StatusDied, StatusLadder, StatusHardcore | StatusLadder | StatusDied} {
			for _, exp := range []bool{true, false} {
				for diff := 0; diff < 3; diff++ {
					for act := 0; act < 5; act++ {
						d := append([]byte(nil), data...)
						le.PutUint32(d[4:], v)

						st := le.Uint32(d[statusOffset:]) &^ (StatusHardcore | StatusDied | StatusLadder | StatusExpansion)
						st |= flags

						if exp {
							st |= StatusExpansion
						}

						le.PutUint32(d[statusOffset:], st)

						for i := 0; i < 3; i++ {
							d[difficultyOffset+i] = 0
						}

						d[difficultyOffset+diff] = activeFlag | byte(act)

						refix(d)

						c, err := Parse(d, tables)
						if err != nil {
							t.Fatalf("v=%X flags=%X exp=%v diff=%d act=%d: %v", v, flags, exp, diff, act, err)
						}

						out, err := Write(c, tables)
						if err != nil {
							t.Fatalf("write: %v", err)
						}

						if sha256.Sum256(out) != sha256.Sum256(d) {
							t.Fatalf("v=%X flags=%X exp=%v diff=%d act=%d: differs at 0x%X", v, flags, exp, diff, act, firstDiff(out, d))
						}

						n++
					}
				}
			}
		}
	}

	t.Logf("%d variants round-tripped", n)
}

// TestFuzzTruncationAndCorruption: no input makes Parse panic (a recovered
// panic comes back as ErrCorrupt and fails here: the parser must reject
// damage with a regular error). Errors must be non-empty and, whenever the
// header is intact, parse results must re-write without panicking.
func TestFuzzTruncationAndCorruption(t *testing.T) {
	data, tables := realSample(t)

	check := func(what string, d []byte) {
		t.Helper()

		c, err := Parse(d, tables)
		if err != nil {
			if errors.Is(err, ErrCorrupt) {
				t.Errorf("%s: parser panicked: %v", what, err)
			}

			if strings.TrimSpace(err.Error()) == "" {
				t.Errorf("%s: empty error", what)
			}

			return
		}

		if _, err := Write(c, tables); err != nil {
			t.Logf("%s: parsed but not writable: %v", what, err)
		}
	}

	// every truncation length, with and without a repaired size/checksum
	for n := 0; n < len(data); n++ {
		d := append([]byte(nil), data[:n]...)
		check("truncated raw", d)

		if n >= HeaderSize {
			refix(d)
			check("truncated refixed", d)
		}
	}

	// random byte corruption behind a repaired checksum, so the parser reaches it
	rng := rand.New(rand.NewSource(1))

	for i := 0; i < 20000; i++ {
		d := append([]byte(nil), data...)

		for k := rng.Intn(4) + 1; k > 0; k-- {
			d[HeaderSize+rng.Intn(len(d)-HeaderSize)] = byte(rng.Intn(256))
		}

		if rng.Intn(4) == 0 {
			d = d[:HeaderSize+rng.Intn(len(d)-HeaderSize)]
		}

		refix(d)
		check("corrupt", d)
	}
}

func TestRepairAndParseRepair(t *testing.T) {
	data, tables := realSample(t)

	bad := append([]byte(nil), data...)
	bad[HeaderSize+3] ^= 0x01 // stale checksum, no longer parses
	binary.LittleEndian.PutUint32(bad[8:], 5)

	if _, err := Parse(bad, tables); err == nil {
		t.Fatal("damaged save parsed")
	}

	fixed, fixes, err := Repair(bad)
	if err != nil || len(fixes) != 2 {
		t.Fatalf("repair: %v %v", fixes, err)
	}

	if Checksum(fixed) != binary.LittleEndian.Uint32(fixed[checksumOffset:]) {
		t.Fatal("checksum not repaired")
	}

	// the undamaged file needs no repair and is returned unchanged
	same, fixes, err := Repair(data)
	if err != nil || len(fixes) != 0 || !bytes.Equal(same, data) {
		t.Fatalf("clean save changed: %v %v", fixes, err)
	}

	// a stale checksum only: ParseRepair reads the save
	stale := append([]byte(nil), data...)
	binary.LittleEndian.PutUint32(stale[checksumOffset:], 1)

	if _, fixes, err := ParseRepair(stale, tables); err != nil || len(fixes) != 1 {
		t.Fatalf("parse repair: %v %v", fixes, err)
	}

	if _, _, err := Repair(data[:10]); !errors.Is(err, ErrTooShort) {
		t.Fatalf("short: %v", err)
	}
}

func TestLegacyVersionMessage(t *testing.T) {
	data, _ := realSample(t)
	d := append([]byte(nil), data...)
	binary.LittleEndian.PutUint32(d[4:], 0x59)
	refix(d)

	_, err := ParseHeader(d)
	if !errors.Is(err, ErrBadVersion) || !strings.Contains(err.Error(), "1.08-1.09") {
		t.Fatalf("legacy version error: %v", err)
	}

	if !IsLegacyVersion(0x59) || IsLegacyVersion(0x60) {
		t.Fatal("IsLegacyVersion")
	}
}

// buildStash makes a synthetic PlugY-style stash from the items of a real save.
func buildStash(t *testing.T, kind, version string, c *Character, tables *ItemTables) []byte {
	t.Helper()

	var b []byte

	if kind == StashShared {
		b = append(b, "SSS\x00"...)
	} else {
		b = append(b, "CSTM"...)
	}

	b = append(b, version...)

	u32 := func(v uint32) { b = binary.LittleEndian.AppendUint32(b, v) }

	if kind == StashShared {
		u32(1234)
	} else if version == "02" {
		u32(0)
	}

	u32(2)

	for p := 0; p < 2; p++ {
		b = append(b, 'S', 'T')

		if version == "02" {
			if p == 1 {
				u32(1)
				b = append(b, "second\x00"...)
			} else {
				u32(0)
			}
		}

		var err error
		if b, err = appendItemList(b, c.Items, tables); err != nil {
			t.Fatal(err)
		}
	}

	return b
}

func TestStashParse(t *testing.T) {
	data, tables := realSample(t)

	c, err := Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ kind, version string }{{StashShared, "01"}, {StashShared, "02"}, {StashPersonal, "01"}, {StashPersonal, "02"}} {
		raw := buildStash(t, tc.kind, tc.version, c, tables)

		s, err := ParseStash(raw, tables)
		if err != nil {
			t.Fatalf("%v: %v", tc, err)
		}

		if len(s.Pages) != 2 || len(s.Pages[0].Items) != len(c.Items) || len(s.Items()) != 2*len(c.Items) {
			t.Fatalf("%v: pages %d", tc, len(s.Pages))
		}

		if tc.kind == StashShared && s.Gold != 1234 {
			t.Fatalf("gold %d", s.Gold)
		}

		if tc.version == "02" && s.Pages[1].Name != "second" {
			t.Fatalf("page name %q", s.Pages[1].Name)
		}

		// every truncation is a clean error, never a panic
		for n := 0; n < len(raw); n++ {
			if _, err := ParseStash(raw[:n], tables); err == nil || errors.Is(err, ErrCorrupt) {
				t.Fatalf("%v truncated at %d: %v", tc, n, err)
			}
		}
	}

	if _, err := ParseStash(data, tables); !errors.Is(err, ErrStashFormat) {
		t.Fatalf("d2r-style header: %v", err)
	}

	if _, err := ParseStash([]byte("hello world"), tables); !errors.Is(err, ErrNotStash) {
		t.Fatalf("garbage: %v", err)
	}

	if _, err := ParseStash(nil, tables); !errors.Is(err, ErrNotStash) {
		t.Fatalf("empty: %v", err)
	}
}

// TestRealStashFiles parses any real stash found (D2S_STASH_FILES path list).
func TestRealStashFiles(t *testing.T) {
	list := filepath.SplitList(os.Getenv("D2S_STASH_FILES"))
	if len(list) == 0 || os.Getenv("D2_TABLES") == "" {
		t.Skip("set D2_TABLES and D2S_STASH_FILES")
	}

	tables := loadRealTables(t, false)

	for _, f := range list {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}

		s, err := ParseStash(raw, tables)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}

		t.Logf("%s: %s v%s gold=%d pages=%d items=%d", filepath.Base(f), s.Kind, s.Version, s.Gold, len(s.Pages), len(s.Items()))
	}
}
