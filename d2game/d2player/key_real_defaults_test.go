package d2player

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2key"
)

func TestVkToKey(t *testing.T) {
	tests := []struct {
		vk   uint16
		want d2enum.Key
		ok   bool
	}{
		{'A', d2enum.KeyA, true}, {'Z', d2enum.KeyZ, true}, {'0', d2enum.Key0, true}, {'4', d2enum.Key4, true},
		{0x70, d2enum.KeyF1, true}, {0x7b, d2enum.KeyF12, true}, {0x60, d2enum.KeyKP0, true},
		{0x67, d2enum.KeyKP7, true}, {0x09, d2enum.KeyTab, true}, {0xc0, d2enum.KeyTilde, true},
		{0x103, d2enum.KeyMouseWheelUp, true}, {d2key.Unbound, 0, false}, {0x100, 0, false},
	}

	for _, tc := range tests {
		got, ok := vkToKey(tc.vk)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("vkToKey(%#x) = %v,%v want %v,%v", tc.vk, got, ok, tc.want, tc.ok)
		}
	}
}

func TestRealCommandsAreNamed(t *testing.T) {
	for id := range realCommandEvents {
		if _, ok := d2key.Commands[id]; !ok {
			t.Errorf("command %d mapped without a d2key name", id)
		}
	}
}

// TestRealDefaultsVsCurrent compares the real default.key with the current
// OpenDiablo2 defaults. Differences are logged, not asserted, because the
// current defaults are deliberately unchanged.
func TestRealDefaultsVsCurrent(t *testing.T) {
	dir := os.Getenv("D2_INSTALL")
	if dir == "" {
		t.Skip("D2_INSTALL not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, "default.key"))
	if err != nil {
		t.Skip(err)
	}

	f, err := d2key.Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	km := &KeyMap{
		mapping:  make(map[d2enum.Key]d2enum.GameEvent),
		controls: make(map[d2enum.GameEvent]*KeyBinding),
	}
	km.ResetToDefault()

	real := RealDefaultBindings(f)
	if len(real) != len(realCommandEvents) {
		t.Fatalf("converted %d of %d commands", len(real), len(realCommandEvents))
	}

	for event, want := range real {
		got := km.GetKeysForGameEvent(event)
		if got == nil {
			t.Logf("event %d: not in OD2 defaults; real %v", event, want)
			continue
		}

		if *got != want {
			t.Logf("event %d differs: OD2 %v, real %v", event, *got, want)
		}
	}
}
