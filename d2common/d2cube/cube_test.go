package d2cube

import "testing"

func TestMatch(t *testing.T) {
	tests := []struct {
		name      string
		in        []string
		expansion bool
		want      string
		ok        bool
	}{
		{"staff", []string{"vip", "msf"}, false, CodeStaff, true},
		{"will", []string{"qbr", "qey", "qhr", "qf1"}, false, CodeWill, true},
		{"will missing part", []string{"qey", "qhr", "qf1"}, false, "", false},
		{"extra item refused", []string{"msf", "vip", "gcv"}, false, "", false},
		{"keys", []string{"pk3", "pk1", "pk2"}, true, "", true},
		{"keys need LoD", []string{"pk3", "pk1", "pk2"}, false, "", false},
		{"organs", []string{"mbr", "dhn", "bey"}, true, "", true},
		{"same key twice", []string{"pk1", "pk1", "pk2"}, true, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, ok := Match(tt.in, tt.expansion)
			if ok != tt.ok || (ok && r.Code != tt.want) {
				t.Fatalf("Match(%v) = %+v, %v", tt.in, r, ok)
			}
		})
	}

	if r, _ := Match([]string{"pk1", "pk2", "pk3"}, true); r.Portal != PortalPandemonium {
		t.Fatalf("keys open %v", r.Portal)
	}

	if r, _ := Match([]string{"dhn", "bey", "mbr"}, true); r.Portal != PortalFinale {
		t.Fatalf("organs open %v", r.Portal)
	}
}
