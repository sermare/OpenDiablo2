package d2hero

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSortByLastPlayed(t *testing.T) {
	dir := t.TempDir()
	base := time.Now().Add(-time.Hour)
	mk := func(name string, age int) *HeroState {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}

		ts := base.Add(time.Duration(age) * time.Minute)
		if err := os.Chtimes(p, ts, ts); err != nil {
			t.Fatal(err)
		}

		return &HeroState{HeroName: name, FilePath: p}
	}

	old, mid, newest := mk("a.od2", 1), mk("b.od2", 5), mk("c.od2", 9)
	// an imported hero counts its .d2s time when that is newer than the .od2
	imp := mk("d.od2", 0)
	imp.Imported = &ImportedInfo{Source: mk("d.d2s", 20).FilePath}
	gone := &HeroState{HeroName: "gone", FilePath: filepath.Join(dir, "missing.od2")}

	list := []*HeroState{old, mid, gone, newest, imp}
	SortByLastPlayed(list)

	want := []*HeroState{imp, newest, mid, old, gone}
	for i := range want {
		if list[i] != want[i] {
			t.Fatalf("slot %d: got %s want %s", i, list[i].HeroName, want[i].HeroName)
		}
	}
}
