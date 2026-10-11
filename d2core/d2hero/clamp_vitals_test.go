package d2hero

import "testing"

func TestClampVitalsToMax(t *testing.T) {
	tests := []struct {
		name                 string
		in                   HeroStatsState
		wantHP, wantMP       int
		wantStamina          float64
		wantMaxHP, wantMaxMP int
	}{
		{"all above", HeroStatsState{Health: 500, MaxHealth: 300, Mana: 200, MaxMana: 150, Stamina: 120, MaxStamina: 100}, 300, 150, 100, 300, 150},
		{"all within", HeroStatsState{Health: 100, MaxHealth: 300, Mana: 50, MaxMana: 150, Stamina: 60, MaxStamina: 100}, 100, 50, 60, 300, 150},
		{"only life", HeroStatsState{Health: 301, MaxHealth: 300, Mana: 50, MaxMana: 150, Stamina: 60, MaxStamina: 100}, 300, 50, 60, 300, 150},
	}

	for _, tc := range tests {
		st := tc.in
		st.ClampVitalsToMax()

		if st.Health != tc.wantHP || st.Mana != tc.wantMP || st.Stamina != tc.wantStamina ||
			st.MaxHealth != tc.wantMaxHP || st.MaxMana != tc.wantMaxMP {
			t.Errorf("%s: %+v", tc.name, st)
		}
	}
}
