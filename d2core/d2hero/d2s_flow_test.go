package d2hero

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupOriginalKeepsFirstCopy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Hero.d2s")

	if err := backupOriginal(path); err != nil { // nothing to back up yet
		t.Fatal(err)
	}

	if _, err := os.Stat(path + BackupSuffix); !os.IsNotExist(err) {
		t.Fatalf("backup made for a file that does not exist: %v", err)
	}

	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	// three saves in a row: the backup stays the untouched original
	for i, write := range []string{"first save", "second save", "third save"} {
		if err := backupOriginal(path); err != nil {
			t.Fatal(err)
		}

		if err := writeFileAtomic(path, []byte(write)); err != nil {
			t.Fatal(err)
		}

		got, err := os.ReadFile(path + BackupSuffix)
		if err != nil || string(got) != "original" {
			t.Fatalf("save %d: backup = %q, %v; want the original", i, got, err)
		}

		if cur, _ := os.ReadFile(path); string(cur) != write {
			t.Fatalf("save %d: file = %q, want %q", i, cur, write)
		}

		if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
			t.Fatalf("save %d: temporary file left behind", i)
		}
	}
}

func TestD2SPath(t *testing.T) {
	wb := t.TempDir()
	home := t.TempDir()

	tests := []struct {
		name      string
		writeback string
		state     *HeroState
		want      string
	}{
		{"imported goes back to its file", "", &HeroState{HeroName: "A", FilePath: "/od2/0.od2",
			Imported: &ImportedInfo{Source: "/saves/A.d2s"}}, "/saves/A.d2s"},
		{"no source: next to the od2", "", &HeroState{HeroName: "A", FilePath: "/od2/0.od2"}, "/od2/A.d2s"},
		{"writeback wins", wb, &HeroState{HeroName: "A", FilePath: "/od2/0.od2",
			Imported: &ImportedInfo{Source: "/saves/A.d2s"}}, filepath.Join(wb, "A.d2s")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(EnvD2SWriteback, tt.writeback)
			t.Setenv("OD2_D2S_DIR", "")

			if got := D2SPath(tt.state); got != tt.want {
				t.Fatalf("D2SPath = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("new character goes to the first save folder", func(t *testing.T) {
		t.Setenv(EnvD2SWriteback, "")
		t.Setenv("OD2_D2S_DIR", home)

		got := newD2SPath(&HeroState{HeroName: "Fresh", FilePath: "/od2/0.od2"})
		if want := filepath.Join(home, "Fresh.d2s"); got != want {
			t.Fatalf("newD2SPath = %q, want %q", got, want)
		}
	})
}

func TestDeleteHero(t *testing.T) {
	dir := t.TempDir()
	od2 := filepath.Join(dir, "0.od2")
	d2sFile := filepath.Join(dir, "Gone.d2s")
	bak := d2sFile + BackupSuffix

	for _, p := range []string{od2, d2sFile, bak} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	f := &HeroStateFactory{}
	hero := &HeroState{HeroName: "Gone", FilePath: od2, Imported: &ImportedInfo{Source: d2sFile}}

	if err := f.DeleteHero(hero); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{od2, d2sFile} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s still there", p)
		}
	}

	for _, p := range []string{d2sFile + DeletedSuffix, bak} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s should remain: %v", p, err)
		}
	}

	// deleting again (files already gone) is not an error
	if err := f.DeleteHero(hero); err != nil {
		t.Fatal(err)
	}
}
