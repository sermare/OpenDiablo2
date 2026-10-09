package d2hero

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// robustSave defaults the real-save env vars to the developer's local copies
// (never committed) and skips when they are absent.
func robustSave(t *testing.T) ([]byte, *d2s.ItemTables, *HeroState) {
	home, _ := os.UserHomeDir()

	if os.Getenv("D2S_SAMPLE_BODY") == "" {
		t.Setenv("D2S_SAMPLE_BODY", filepath.Join(home, "git", "d2s-test", "NokkaSorc.d2s"))
	}

	if os.Getenv("D2_TABLES") == "" {
		t.Setenv("D2_TABLES", filepath.Join(home, "git", "d2-tables"))
	}

	if _, err := os.Stat(os.Getenv("D2S_SAMPLE_BODY")); err != nil {
		t.Skip("real save not available")
	}

	return realSave(t)
}

func refix(d []byte) []byte {
	if len(d) >= d2s.HeaderSize {
		binary.LittleEndian.PutUint32(d[8:], uint32(len(d)))
		binary.LittleEndian.PutUint32(d[0x0C:], 0)
		binary.LittleEndian.PutUint32(d[0x0C:], d2s.Checksum(d))
	}

	return d
}

// Export must reject (not panic on) a corrupt or truncated original save.
func TestExportCorruptOriginal(t *testing.T) {
	data, tables, state := robustSave(t)

	check := func(d []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic: %v", r)
			}
		}()

		_, _, _ = ExportD2SWithOptions(state, d, tables, ExportOptions{})
	}

	for n := 0; n < len(data); n += 5 {
		check(append([]byte(nil), data[:n]...))
		check(refix(append([]byte(nil), data[:n]...)))
	}

	for i := d2s.HeaderSize; i < len(data); i++ {
		d := append([]byte(nil), data...)
		d[i] ^= 0xFF
		check(refix(d))
	}
}
