package d2combat

import "testing"

// TestResistPenaltyMatrix pins the difficulty penalty per game mode: LoD reads
// -40/-100 from DifficultyLevels.txt, classic -20/-50 (VERIFIED, 0x579b10).
func TestResistPenaltyMatrix(t *testing.T) {
	for _, tt := range []struct {
		diff            int
		classic, expans int
	}{{0, 0, 0}, {1, -20, -40}, {2, -50, -100}, {3, 0, 0}} {
		if got := ClassicResistPenalty(tt.diff); got != tt.classic {
			t.Errorf("classic diff %d: %d want %d", tt.diff, got, tt.classic)
		}

		if got := LoDResistPenalty(tt.diff); got != tt.expans {
			t.Errorf("lod diff %d: %d want %d", tt.diff, got, tt.expans)
		}
	}
}

// TestMonsterResistImmunityAndPierce: monsters are not capped (NoCap), so a
// monstats resist of 100 stays an immunity and pierce lowers it from 100.
// The no-cap rule for monster defenders is UNVERIFIED in the exe.
func TestMonsterResistImmunityAndPierce(t *testing.T) {
	tests := []struct {
		name string
		in   ResistInput
		want int
	}{
		{"immune", ResistInput{Resist: 100, NoCap: true, NoDifficultyPenalty: true}, 100},
		{"above 100 stays", ResistInput{Resist: 120, NoCap: true, NoDifficultyPenalty: true}, 120},
		{"85 not capped to 75", ResistInput{Resist: 85, NoCap: true}, 85},
		{"pierce 50 on immune", ResistInput{Resist: 100, NoCap: true, HasPierce: true, Pierce: 50}, 50},
		{"pierce 150 below immune goes negative", ResistInput{Resist: 100, NoCap: true, HasPierce: true, Pierce: 150}, -50},
		{"pierce past floor", ResistInput{Resist: 100, NoCap: true, HasPierce: true, Pierce: 300}, -100},
		{"ignore keeps immunity", ResistInput{Resist: 100, NoCap: true, HasPierce: true, Pierce: 50, Ignore: true}, 100},
		{"player still capped", ResistInput{Resist: 100}, 75},
	}

	for _, tt := range tests {
		if got := EffectiveResist(tt.in); got != tt.want {
			t.Errorf("%s: got %d want %d", tt.name, got, tt.want)
		}
	}

	if ApplyResist(1000, EffectiveResist(ResistInput{Resist: 100, NoCap: true})) != 0 {
		t.Error("immune monster takes damage")
	}
}

// TestPlayerResistCaps pins max resist rules for the player.
func TestPlayerResistCaps(t *testing.T) {
	tests := []struct {
		name string
		in   ResistInput
		want int
	}{
		{"penalty before cap", ResistInput{Resist: 110, DifficultyPenalty: -100, HasMaxResist: true}, 10},
		{"max stat +20 caps 95", ResistInput{Resist: 200, HasMaxResist: true, MaxResistBonus: 20}, 95},
		{"max stat +50 still 95", ResistInput{Resist: 200, HasMaxResist: true, MaxResistBonus: 50}, 95},
		{"hell classic -50", ResistInput{Resist: 75, DifficultyPenalty: ClassicResistPenalty(2)}, 25},
		{"physical never penalised nor over 50", ResistInput{Resist: 70, IsPhysical: true, NoDifficultyPenalty: true, DifficultyPenalty: -100}, 50},
		{"min -100", ResistInput{Resist: -80, DifficultyPenalty: -100}, -100},
	}

	for _, tt := range tests {
		if got := EffectiveResist(tt.in); got != tt.want {
			t.Errorf("%s: got %d want %d", tt.name, got, tt.want)
		}
	}
}
