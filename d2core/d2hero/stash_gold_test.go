package d2hero

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
)

func TestLoadedStashGold(t *testing.T) {
	tests := []struct {
		stored uint64
		want   int
	}{
		{0, 0},
		{1234, 1234},
		{d2inventory.StashGoldLimit, d2inventory.StashGoldLimit},
		{d2inventory.StashGoldLimit + 1, 0},
		{1 << 33, 0},
	}

	for _, tc := range tests {
		if got := loadedStashGold(tc.stored); got != tc.want {
			t.Errorf("loadedStashGold(%d) = %d, want %d", tc.stored, got, tc.want)
		}
	}
}

// Every real save in ~/git/d2s-test (and D2S_SAMPLE_BODY) must round-trip
// byte-exact with the stash gold applied through the loader cap, and with it
// unknown (nil).
func TestStashGoldRoundTripRealSaves(t *testing.T) {
	home, _ := os.UserHomeDir()

	files, _ := filepath.Glob(filepath.Join(home, "git", "d2s-test", "*.d2s"))
	if env := os.Getenv("D2S_SAMPLE_BODY"); env != "" {
		files = append(files, env)
	}

	if os.Getenv("D2_TABLES") == "" || len(files) == 0 {
		t.Skip("set D2_TABLES and provide saves")
	}

	for _, f := range files {
		if st, err := os.Stat(f); err != nil || st.Size() < 1000 { // header-only stubs have no body
			continue
		}

		t.Run(filepath.Base(f), func(t *testing.T) {
			t.Setenv("D2S_SAMPLE_BODY", f)

			for _, known := range []bool{true, false} {
				data, tables, state := realSave(t)
				if !known {
					state.StashGold = nil
				}

				out, _, err := ExportD2SWithOptions(state, data, tables, ExportOptions{})
				if err != nil {
					t.Fatal(err)
				}

				if !bytes.Equal(out, data) {
					t.Errorf("stash known=%v: export differs from the original", known)
				}
			}
		})
	}
}

func TestExportStashGoldChanged(t *testing.T) {
	data, tables, state := realSave(t)

	v := 777000
	state.StashGold = &v

	out, _, err := ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}

	got, err := d2s.Parse(out, tables)
	if err != nil {
		t.Fatal(err)
	}

	if got.Body.Attributes.StashedGold != 777000 {
		t.Errorf("stashed gold = %d", got.Body.Attributes.StashedGold)
	}

	over := d2inventory.StashGoldLimit + 5
	state.StashGold = &over

	out, warns, err := ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil || len(warns) == 0 {
		t.Fatalf("err %v warnings %v", err, warns)
	}

	got, _ = d2s.Parse(out, tables)
	if got.Body.Attributes.StashedGold != d2inventory.StashGoldLimit {
		t.Errorf("clamped stash = %d", got.Body.Attributes.StashedGold)
	}
}
