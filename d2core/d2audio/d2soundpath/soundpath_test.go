package d2soundpath

import "testing"

func TestResolve(t *testing.T) {
	archive := map[string]bool{
		"data/global/sfx/cursor/pass.wav":                  true,
		"data/local/sfx/act1/akara/aka_act1_intro_sor.wav": true,
		"data/local/sfx/Act1/Monster/andarieltaunt1.wav":   true,
		"data/global/music/act1/town1.wav":                 true,
	}
	exists := func(p string) bool { return archive[p] }

	tests := []struct {
		name, want string
		ok         bool
	}{
		{`cursor\pass.wav`, "data/global/sfx/cursor/pass.wav", true},
		{`act1\akara\aka_act1_intro_sor.wav`, "data/local/sfx/act1/akara/aka_act1_intro_sor.wav", true},
		{`/act1/town1.wav`, "data/global/music/act1/town1.wav", true},
		{"data/local/sfx/Act1/Monster/andarieltaunt1.wav", "data/local/sfx/Act1/Monster/andarieltaunt1.wav", true},
		{`data\local\sfx\Act1\Monster\andarieltaunt1.wav`, "data/local/sfx/Act1/Monster/andarieltaunt1.wav", true},
		{`act9\nobody.wav`, "", false},
		{"", "", false},
	}
	for _, tc := range tests {
		got, ok := Resolve(tc.name, exists)
		if got != tc.want || ok != tc.ok {
			t.Errorf("Resolve(%q) = %q,%v want %q,%v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}
