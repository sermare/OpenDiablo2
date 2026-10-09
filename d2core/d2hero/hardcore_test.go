package d2hero

import "testing"

func TestKindAndPlayRefusal(t *testing.T) {
	for _, tc := range []struct {
		name    string
		hero    *HeroState
		kind    CharacterKind
		refused bool
	}{
		{"nil", nil, KindSoftcore, false},
		{"softcore", &HeroState{HeroName: "a"}, KindSoftcore, false},
		{"softcore that died is playable", &HeroState{HeroName: "a", Death: &DeathState{Died: true}}, KindSoftcore, false},
		{"hardcore alive", &HeroState{HeroName: "b", Hardcore: true}, KindHardcore, false},
		{"hardcore with deaths but not dead", &HeroState{HeroName: "b", Hardcore: true, Death: &DeathState{}}, KindHardcore, false},
		{"hardcore dead", &HeroState{HeroName: "c", Hardcore: true, Death: &DeathState{Died: true}}, KindDeadHardcore, true},
	} {
		if got := tc.hero.Kind(); got != tc.kind {
			t.Errorf("%s: kind %v, want %v", tc.name, got, tc.kind)
		}

		if got := tc.hero.PlayRefusal() != ""; got != tc.refused {
			t.Errorf("%s: refused %v, want %v", tc.name, got, tc.refused)
		}
	}
}

// A hardcore death is permanent: Respawn refuses and the flag stays.
func TestHardcoreDeathPermanent(t *testing.T) {
	s := &DeathState{Died: true}
	if s.Respawn(true) {
		t.Fatal("a hardcore hero respawned")
	}

	if !s.Died {
		t.Fatal("the died flag was cleared for a hardcore hero")
	}

	soft := &DeathState{Died: true}
	if !soft.Respawn(false) || soft.Died {
		t.Fatal("a softcore hero must respawn and clear the flag")
	}
}
