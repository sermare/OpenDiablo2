package drlgpop

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadPopGold reads the preset population goldens: ORACLE_POP when set, else
// every testdata/pop_presets*.json[.gz] (several map seeds, all five acts).
func loadPopGold(t *testing.T) []goldLevel {
	t.Helper()

	files := []string{os.Getenv("ORACLE_POP")}
	if files[0] == "" {
		files, _ = filepath.Glob(filepath.Join("..", "testdata", "pop_presets*.json*"))
	}

	var all []goldLevel

	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			continue
		}

		var r io.Reader = fh

		if strings.HasSuffix(f, ".gz") {
			zr, err := gzip.NewReader(fh)
			if err != nil {
				fh.Close()
				t.Fatal(err)
			}

			r = zr
		}

		var part []goldLevel
		if err := json.NewDecoder(r).Decode(&part); err != nil {
			fh.Close()
			t.Fatalf("%s: %v", f, err)
		}

		fh.Close()

		all = append(all, part...)
	}

	if len(all) == 0 {
		t.Skip("no preset population golden")
	}

	return all
}
