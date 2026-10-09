package d2tbl

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2locale"
)

// Synthetic tables in every locale's code page: encode, load with the locale decoder, compare.
func TestTBL_LocaleRoundTrip(t *testing.T) {
	values := map[string]map[string]string{
		"deDE": {"Sturdy": "Grüße äöü", "x1": "ÿc3Rot"},
		"plPL": {"Sturdy": "£ód¿ ê³"},
		"ruRU": {"Sturdy": "Ïðèâåò"},
		"jaJP": {"Sturdy": "芠A±荁"},
		"koKR": {"Sturdy": "낡 x 쟑"},
		"zhCN": {"Sturdy": "쓣뫃!"},
		"zhTW": {"Sturdy": "ꝁꙮ"},
	}

	for code, vals := range values {
		loc, _ := d2locale.Parse(code)
		td := TextDictionary{}

		for k, v := range vals {
			td[k] = v
		}

		data := td.MarshalEncoded(loc.Encode)

		got, err := LoadTextDictionaryDecoded(data, loc.Decode)
		if err != nil {
			t.Fatalf("%s: %v", code, err)
		}

		for k, v := range vals {
			if got[k] != v {
				t.Errorf("%s %s: got %q want %q", code, k, got[k], v)
			}
		}

		// without a decoder the raw bytes are returned unchanged
		raw, err := LoadTextDictionary(data)
		if err != nil || len(raw) != len(vals) {
			t.Errorf("%s: raw load %v %d", code, err, len(raw))
		}
	}
}

// With D2_TBL_DIR pointing at a folder with the user's own string.tbl, patchstring.tbl and
// expansionstring.tbl (any depth) the real enUS tables must load and contain well known keys.
func TestTBL_RealEnUS(t *testing.T) {
	dir := os.Getenv("D2_TBL_DIR")
	if dir == "" {
		t.Skip("D2_TBL_DIR not set")
	}

	found := 0

	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(p) != ".tbl" {
			return nil
		}

		data, rerr := ioutil.ReadFile(p)
		if rerr != nil {
			t.Error(rerr)
			return nil
		}

		td, lerr := LoadTextDictionaryDecoded(data, d2locale.Default().Decode)
		if lerr != nil {
			t.Errorf("%s: %v", p, lerr)
			return nil
		}

		found++

		if len(td) < 100 {
			t.Errorf("%s: only %d strings", p, len(td))
		}

		if filepath.Base(p) == "string.tbl" {
			if td["strGold"] == "" && td["Gold"] == "" {
				t.Errorf("%s: no gold string", p)
			}
		}

		return nil
	})

	if found == 0 {
		t.Skip("no .tbl under D2_TBL_DIR")
	}
}
