package d2audio

import "testing"

func TestSwingHandle(t *testing.T) {
	for _, c := range []struct {
		class, want string
		ok          bool
	}{
		{"", "weapon_punch_1", true}, {"1hsl", "weapon_1hs_large_1", true}, {"club", "weapon_1hs_large_1", true},
		{"staf", "weapon_staff_1", true}, {"bow", "", true}, {"zzz", "", false},
	} {
		if h, ok := SwingHandle(c.class); h != c.want || ok != c.ok {
			t.Errorf("%q: got %q %v", c.class, h, ok)
		}
	}
}

func TestVocalAllowed(t *testing.T) {
	for _, c := range []struct {
		prob          int
		now, last     int64
		speech, allow bool
	}{
		{100, 5, 4, true, true}, {50, 100, 90, false, false}, {50, 100, 85, false, true}, {50, 100, 85, true, false},
	} {
		if got := VocalAllowed(c.prob, c.now, c.last, c.speech); got != c.allow {
			t.Errorf("%+v: got %v", c, got)
		}
	}
}

func TestRefusalVoice(t *testing.T) {
	want := map[string]int{"mana": VoiceNeedMana, "town": VoiceNotInTown, "restricted": VoiceImpossible,
		"no_ammo": VoiceImpossible, "cooldown": 0, "no_skill": 0, "passive": 0}
	for r, w := range want {
		if got := RefusalVoice(r); got != w {
			t.Errorf("%s: %#x want %#x", r, got, w)
		}
	}
}
