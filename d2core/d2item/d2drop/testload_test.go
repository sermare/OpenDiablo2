package d2drop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testSource skips the test when D2_TABLES is not set.
func testSource(t *testing.T) TableSource {
	t.Helper()

	return DirTables{Root: d2Tables(t)}
}

// skipTB turns a missing table into a skipped test.
type skipTB struct{ *testing.T }

func (s skipTB) Fatalf(format string, args ...interface{}) {
	if strings.HasPrefix(format, "missing table") {
		s.T.Skipf(format, args...)
	}

	s.T.Fatalf(format, args...)
}

func readTSV(t *testing.T, path string) *tsv {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("missing table: %v", err)
	}

	return readTSV2(t, raw)
}

func loadItemTables(t *testing.T) *ItemTables {
	t.Helper()

	return loadItemTablesFrom(skipTB{t}, testSource(t))
}

func loadPropTables(t *testing.T) *PropTables {
	t.Helper()

	return loadPropTablesFrom(skipTB{t}, testSource(t))
}

func loadUniqueTables(t *testing.T, pt *PropTables) *UniqueTables {
	t.Helper()

	return loadUniqueTablesFrom(skipTB{t}, testSource(t), pt)
}

func loadTestAffixes(t *testing.T, pt *PropTables) [][3]PropInst {
	t.Helper()

	return loadTestAffixesFrom(skipTB{t}, testSource(t), pt)
}

func loadTestQuality(t *testing.T, pt *PropTables) [][2]PropInst {
	t.Helper()

	return loadTestQualityFrom(skipTB{t}, testSource(t), pt)
}

func loadQualityTables(t *testing.T) *QualityTables {
	t.Helper()

	return loadQualityTablesFrom(skipTB{t}, testSource(t))
}

func loadAffixTables(t *testing.T) *AffixTables {
	t.Helper()

	return loadAffixTablesFrom(skipTB{t}, testSource(t))
}

// tablePath returns the path of the effective version of a table under
// itemgen/ (patch_d2, then d2exp, then d2data), or skips the test.
func tablePath(t *testing.T, name string) string {
	t.Helper()

	root := d2Tables(t)

	for _, mpq := range []string{"patch_d2", "d2exp", "d2data"} {
		p := filepath.Join(root, "itemgen", mpq, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	t.Skipf("table %s not found under %s/itemgen", name, root)

	return ""
}
