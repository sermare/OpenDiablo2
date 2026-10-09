package d2setup

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2config"
)

func touch(t *testing.T, parts ...string) string {
	t.Helper()

	p := filepath.Join(parts...)
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	return p
}

func fillGame(t *testing.T, dir string, names []string) {
	t.Helper()

	for _, n := range names {
		touch(t, dir, n)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		files       []string
		wantOK      bool
		wantMissing int
		wantExpMiss int
		wantOrder   []string
	}{
		{"empty", nil, false, 7, 4, nil},
		{"core only", CoreMPQs, true, 0, 4,
			[]string{"patch_d2.mpq", "d2data.mpq", "d2char.mpq", "d2music.mpq", "d2sfx.mpq", "d2video.mpq", "d2speech.mpq"}},
		{"missing patch", []string{"d2data.mpq", "d2char.mpq", "d2music.mpq", "d2sfx.mpq", "d2speech.mpq", "d2video.mpq"}, false, 1, 4, nil},
		{"all, odd case", []string{"D2DATA.MPQ", "d2char.mpq", "d2music.mpq", "d2sfx.mpq", "d2speech.mpq", "d2video.mpq",
			"Patch_D2.mpq", "d2exp.mpq", "d2xmusic.mpq", "d2xtalk.mpq", "d2xvideo.mpq"}, true, 0, 0,
			[]string{"Patch_D2.mpq", "d2exp.mpq", "d2xmusic.mpq", "d2xtalk.mpq", "d2xvideo.mpq", "D2DATA.MPQ",
				"d2char.mpq", "d2music.mpq", "d2sfx.mpq", "d2video.mpq", "d2speech.mpq"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			fillGame(t, dir, tc.files)

			v := Validate(dir)
			if v.OK() != tc.wantOK || len(v.MissingCore) != tc.wantMissing || len(v.MissingExpansion) != tc.wantExpMiss {
				t.Fatalf("got ok=%v missing=%v exp=%v", v.OK(), v.MissingCore, v.MissingExpansion)
			}

			if tc.wantOK && !reflect.DeepEqual(v.LoadOrder, tc.wantOrder) {
				t.Fatalf("order %v want %v", v.LoadOrder, tc.wantOrder)
			}
		})
	}
}

func TestResolveSubfolder(t *testing.T) {
	parent := t.TempDir()
	fillGame(t, filepath.Join(parent, "Diablo II"), CoreMPQs)

	if v := Resolve(parent); !v.OK() || v.Dir != filepath.Join(parent, "Diablo II") {
		t.Fatalf("got %+v", v)
	}
}

func TestFindInstallsAndSaves(t *testing.T) {
	home := t.TempDir()
	wine := filepath.Join(home, ".wine-d2", "drive_c", "Program Files (x86)", "Diablo II")
	fillGame(t, wine, CoreMPQs)

	partial := filepath.Join(home, "Library", "Application Support", "Diablo II")
	fillGame(t, partial, []string{"d2data.mpq"})

	got := FindInstalls(home)
	if len(got) != 1 || got[0].Dir != wine {
		t.Fatalf("installs: %+v", got)
	}

	saves := filepath.Join(home, ".wine-d2", "drive_c", "users", "me", "Saved Games", "Diablo II")
	touch(t, saves, "Hero.d2s")
	touch(t, home, "Documents", "Diablo II", "notes.txt") // no .d2s: ignored
	touch(t, home, "Library", "Application Support", "Diablo II Saves", "Other.D2S")

	dirs := FindSaveDirs(home, wine)
	want := []string{filepath.Join(home, "Library", "Application Support", "Diablo II Saves"), saves}

	if !reflect.DeepEqual(dirs, want) {
		t.Fatalf("save dirs %v want %v", dirs, want)
	}
}

type fakeUI struct {
	pick    string
	answers []bool
	asked   int
	alerts  int
}

func (f *fakeUI) ChooseFolder(string, string) (string, error) {
	if f.pick == "" {
		return "", ErrCancelled
	}

	return f.pick, nil
}

func (f *fakeUI) Ask(string, string, string, string) bool {
	f.asked++

	if len(f.answers) == 0 {
		return true
	}

	a := f.answers[0]
	f.answers = f.answers[1:]

	return a
}

func (f *fakeUI) Alert(string, string) { f.alerts++ }

func newCfg(t *testing.T, mpqPath string) *d2config.Configuration {
	t.Helper()
	t.Setenv(d2config.ConfigDirEnv, t.TempDir())

	cfg := d2config.DefaultConfig()
	cfg.MpqPath = mpqPath

	return cfg
}

func TestEnsureGameFiles(t *testing.T) {
	t.Run("valid config untouched", func(t *testing.T) {
		game := t.TempDir()
		fillGame(t, game, LoadOrder)
		cfg := newCfg(t, game)
		ui := &fakeUI{}

		if err := EnsureGameFiles(cfg, ui, t.TempDir()); err != nil || ui.asked != 0 {
			t.Fatalf("err=%v asked=%d", err, ui.asked)
		}
	})

	t.Run("autodetect accepted", func(t *testing.T) {
		home := t.TempDir()
		game := filepath.Join(home, ".wine", "drive_c", "Program Files (x86)", "Diablo II")
		fillGame(t, game, CoreMPQs)

		cfg := newCfg(t, "/nonexistent")
		if err := EnsureGameFiles(cfg, &fakeUI{}, home); err != nil {
			t.Fatal(err)
		}

		if cfg.MpqPath != game || len(cfg.MpqLoadOrder) != 7 {
			t.Fatalf("cfg %q %v", cfg.MpqPath, cfg.MpqLoadOrder)
		}

		if _, err := os.Stat(cfg.Path()); err != nil {
			t.Fatalf("config not written: %v", err)
		}
	})

	t.Run("picker with a bad folder then a good one", func(t *testing.T) {
		pick := t.TempDir()
		fillGame(t, pick, CoreMPQs)

		cfg := newCfg(t, "/nonexistent")
		ui := &fakeUI{pick: pick}

		if err := EnsureGameFiles(cfg, ui, t.TempDir()); err != nil || cfg.MpqPath != pick {
			t.Fatalf("err=%v path=%q", err, cfg.MpqPath)
		}
	})

	t.Run("incomplete folder, user cancels", func(t *testing.T) {
		pick := t.TempDir()
		fillGame(t, pick, []string{"d2data.mpq"})

		ui := &fakeUI{pick: pick, answers: []bool{false}}
		if err := EnsureGameFiles(newCfg(t, "/nonexistent"), ui, t.TempDir()); err != ErrNoGameFiles {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("picker cancelled", func(t *testing.T) {
		if err := EnsureGameFiles(newCfg(t, ""), &fakeUI{}, t.TempDir()); err != ErrNoGameFiles {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestAsQuote(t *testing.T) {
	if got := asQuote("a\"b\\c\nd"); got != `"a\"b\\c\nd"` {
		t.Fatal(got)
	}
}
