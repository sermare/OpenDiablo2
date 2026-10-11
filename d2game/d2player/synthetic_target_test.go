package d2player

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

func TestParseEntityTarget(t *testing.T) {
	tests := []struct {
		spec    string
		ok      bool
		wantErr bool
		kind    string
		name    string
		frac    float64
		start   int
	}{
		{spec: "left", ok: false},
		{spec: "left@400,300", ok: false},
		{spec: "left@hero:10,20", ok: false},
		{spec: "left@ui:392,560", ok: false},
		{spec: "left@monster", ok: true, kind: "monster", frac: 1, start: 4},
		{spec: "left+shift@monster", ok: true, kind: "monster", frac: 1, start: 10},
		{spec: "left@npc", ok: true, kind: "npc", frac: 1, start: 4},
		{spec: "left@npc:Akara", ok: true, kind: "npc", name: "Akara", frac: 1, start: 4},
		{spec: "hold:left@npc:Warriv*0.5", ok: true, kind: "npc", name: "Warriv", frac: 0.5, start: 9},
		{spec: "left@object:156", ok: true, kind: "object", name: "156", frac: 1, start: 4},
		{spec: "left@object:Waypoint", ok: true, kind: "object", name: "Waypoint", frac: 1, start: 4},
		{spec: "left@item", ok: true, kind: "item", frac: 1, start: 4},
		{spec: "left@item*0", ok: true, kind: "item", frac: 0, start: 4},
		{spec: "left@object", wantErr: true},
		{spec: "left@npc*2", wantErr: true},
		{spec: "left@npc*x", wantErr: true},
	}

	for _, tt := range tests {
		got, ok, err := parseEntityTarget(tt.spec)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q: err %v, wantErr %v", tt.spec, err, tt.wantErr)
			continue
		}

		if tt.wantErr {
			continue
		}

		if ok != tt.ok {
			t.Errorf("%q: ok %v, want %v", tt.spec, ok, tt.ok)
			continue
		}

		if ok && (got.kind != tt.kind || got.name != tt.name || got.frac != tt.frac || got.start != tt.start) {
			t.Errorf("%q: got %+v, want kind=%s name=%s frac=%v start=%d", tt.spec, got, tt.kind, tt.name, tt.frac, tt.start)
		}
	}
}

func TestParseKeySpec(t *testing.T) {
	tests := []struct {
		spec    string
		key     d2enum.Key
		mod     d2enum.KeyMod
		wantErr bool
	}{
		{spec: "I", key: d2enum.KeyI},
		{spec: "Escape", key: d2enum.KeyEscape},
		{spec: "F1", key: d2enum.KeyF1},
		{spec: "shift+F1", key: d2enum.KeyF1, mod: d2enum.KeyModShift},
		{spec: "ctrl+shift+Tab", key: d2enum.KeyTab, mod: d2enum.KeyModControl | d2enum.KeyModShift},
		{spec: "cmd+I", key: d2enum.KeyI, mod: d2enum.KeyModSuper},
		{spec: "hyper+I", wantErr: true},
		{spec: "NoSuchKey", wantErr: true},
	}

	for _, tt := range tests {
		k, m, err := parseKeySpec(tt.spec)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q: err %v, wantErr %v", tt.spec, err, tt.wantErr)
			continue
		}

		if !tt.wantErr && (k != tt.key || m != tt.mod) {
			t.Errorf("%q: got %v %v, want %v %v", tt.spec, k, m, tt.key, tt.mod)
		}
	}
}

func TestHoverRectContains(t *testing.T) {
	b := hoverRect{l: 10, r: 20, t: 30, b: 50}

	for _, tt := range []struct {
		x, y int
		in   bool
	}{{10, 30, true}, {20, 50, true}, {15, 40, true}, {9, 40, false}, {21, 40, false}, {15, 29, false}, {15, 51, false}} {
		if got := b.contains(tt.x, tt.y); got != tt.in {
			t.Errorf("contains(%d,%d) = %v, want %v", tt.x, tt.y, got, tt.in)
		}
	}
}
