package d2tbl

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

// tblHeaderSize is the size of the fixed header Marshal writes (no index array).
const tblHeaderSize = 21

func TestLoadTextDictionaryUTF8Values(t *testing.T) {
	td := TextDictionary{
		"german": "Mitspielen nicht möglich",
		"korean": "취소",
		"colour": "ÿc4Unique",
	}

	got, err := LoadTextDictionary(td.Marshal())
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range td {
		if got[k] != v {
			t.Errorf("%s: got %q, want %q", k, got[k], v)
		}
	}
}

func TestLoadTextDictionaryEmptyValue(t *testing.T) {
	one := TextDictionary{"k": "v"}
	data := one.Marshal()
	// NameLength (uint16 at offset 15 of the single hash entry) = 0 used to
	// underflow to 65535 and fail the whole table.
	binary.LittleEndian.PutUint16(data[tblHeaderSize+15:], 0)

	td, err := LoadTextDictionary(data)
	if err != nil {
		t.Fatalf("zero NameLength: %v", err)
	}

	if v, ok := td["k"]; !ok || v != "" {
		t.Fatalf("got %q, %v; want empty value present", v, ok)
	}
}

func TestLoadTextDictionaryNumericKeys(t *testing.T) {
	// "x" keys become "#<hash slot>". Marshal writes slots in map order, so
	// the numbers are not preserved (known limitation); the values are.
	td := TextDictionary{"#0": "zero", "#1": "one", "dup": "first"}

	got, err := LoadTextDictionary(td.Marshal())
	if err != nil {
		t.Fatal(err)
	}

	vals := map[string]bool{}

	for k, v := range got {
		if k != "dup" && k[0] != '#' {
			t.Errorf("unexpected key %q", k)
		}

		vals[v] = true
	}

	if len(got) != 3 || !vals["zero"] || !vals["one"] || got["dup"] != "first" {
		t.Fatalf("round trip lost entries: %v", got)
	}
}

func TestLoadTextDictionaryTruncated(t *testing.T) {
	one := TextDictionary{"abc": "def"}
	data := one.Marshal()

	for _, n := range []int{0, 1, 5, tblHeaderSize, len(data) - 3} {
		if _, err := LoadTextDictionary(data[:n]); err == nil {
			t.Errorf("truncated to %d bytes: expected an error", n)
		}
	}
}

// loadRealTable reads $D2_TBL_DIR/<lang>/<name>.tbl, or returns nil when absent.
// The directory is laid out by the tester from the player's own MPQs, for
// example ENG/string.tbl (d2data), ENG/expansionstring.tbl (d2exp) and
// ENG/patchstring.tbl (patch_d2). Nothing from it is committed.
func loadRealTable(t *testing.T, lang, name string) TextDictionary {
	t.Helper()

	dir := os.Getenv("D2_TBL_DIR")
	if dir == "" {
		t.Skip("D2_TBL_DIR not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, lang, name+".tbl"))
	if err != nil {
		return nil
	}

	td, err := LoadTextDictionary(data)
	if err != nil {
		t.Fatalf("%s/%s: %v", lang, name, err)
	}

	return td
}

var realLangs = []string{"ENG", "DEU", "FRA", "POL", "KOR"}

func TestRealTablesAreUTF8(t *testing.T) {
	for _, lang := range realLangs {
		for _, name := range []string{"string", "expansionstring", "patchstring"} {
			td := loadRealTable(t, lang, name)
			if td == nil {
				continue
			}

			for k, v := range td {
				if !utf8.ValidString(k) || !utf8.ValidString(v) {
					t.Errorf("%s/%s key %q: value is not valid UTF-8", lang, name, k)
				}

				if strings.Contains(v, "\xffc") {
					t.Errorf("%s/%s key %q: raw Latin-1 colour intro in a UTF-8 table", lang, name, k)
				}
			}
		}
	}
}

// Every numeric label the menus use must resolve, through patch > expansion >
// base, to a non-empty string in the base English install.
func TestRealEnglishLabelsResolve(t *testing.T) {
	patch := loadRealTable(t, "ENG", "patchstring")
	exp := loadRealTable(t, "ENG", "expansionstring")
	base := loadRealTable(t, "ENG", "string")

	if base == nil {
		t.Skip("ENG/string.tbl missing")
	}

	lookup := func(key string) string {
		for _, td := range []TextDictionary{patch, exp, base} {
			if v, ok := td[key]; ok {
				return v
			}
		}

		return ""
	}

	// Labels that exist in the shipped tables; 1 and the 'not used' padding
	// entries are allowed to be empty, so only check the used enum values.
	for _, idx := range []int{
		d2enum.RepairAll, d2enum.CancelLabel, d2enum.CopyrightLabel, d2enum.AllRightsReservedLabel,
		d2enum.SinglePlayerLabel, d2enum.ExitGameLabel, d2enum.CreditsLabel, d2enum.CinematicsLabel,
		d2enum.TCPIPGameLabel, d2enum.HellLabel, d2enum.NightmareLabel, d2enum.NormalLabel,
		d2enum.YesLabel, d2enum.NoLabel, d2enum.ExitLabel, d2enum.OKLabel,
	} {
		key := "#" + strconv.Itoa(d2enum.BaseLabelNumbers(idx))
		if lookup(key) == "" {
			t.Errorf("label index %d (%s) resolves to nothing", idx, key)
		}
	}

	// Specific strings the character screens ask for by literal key.
	for key, prefix := range map[string]string{
		"#304": "Commanding the forces of nature",
		"#305": "Schooled in the Martial Arts",
		"#803": "EXPANSION CHARACTER",
		"#825": "CONVERT TO EXPANSION",
		"#831": "CREATE NEW CHARACTER",
	} {
		if got := lookup(key); !strings.HasPrefix(got, prefix) {
			t.Errorf("%s = %q, want prefix %q", key, got, prefix)
		}
	}
}

// Documents a divergence: "#N" is the hash slot, so patch and expansion tables
// can both define the same "#N" for different strings. Precedence then hides the
// expansion string. The calibrated keys above are not affected.
func TestRealNumericKeyCollisionsAreKnown(t *testing.T) {
	patch := loadRealTable(t, "ENG", "patchstring")
	exp := loadRealTable(t, "ENG", "expansionstring")

	if patch == nil || exp == nil {
		t.Skip("ENG patch/expansion tables missing")
	}

	n := 0

	for k, v := range patch {
		if k[0] == '#' && exp[k] != "" && exp[k] != v {
			n++
		}
	}

	t.Logf("%d numeric keys defined differently in patchstring and expansionstring", n)
}

func TestRealColourCodesPresent(t *testing.T) {
	base := loadRealTable(t, "ENG", "string")
	if base == nil {
		t.Skip("ENG/string.tbl missing")
	}

	n := 0

	for _, v := range base {
		if strings.Contains(v, "ÿc") {
			n++
		}
	}

	if n == 0 {
		t.Fatal("no ÿc colour codes found in ENG string.tbl; encoding assumption broken")
	}
}
