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

// TestMonsterResistImmunityAndPierce: a monster defender sets the one ignore
// flag (VERIFIED 0x579b10 ctx[5]): no cap, no difficulty penalty, and pierce
// cannot lower a resist of 100 or more; below 100 it works normally.
func TestMonsterResistImmunityAndPierce(t *testing.T) {
	tests := []struct {
		name string
		in   ResistInput
		want int
	}{
		{"immune", ResistInput{Resist: 100, Ignore: true, NoDifficultyPenalty: true}, 100},
		{"above 100 stays", ResistInput{Resist: 120, Ignore: true, NoDifficultyPenalty: true}, 120},
		{"85 not capped to 75", ResistInput{Resist: 85, Ignore: true}, 85},
		{"pierce 50 on immune does nothing", ResistInput{Resist: 100, Ignore: true, HasPierce: true, Pierce: 50}, 100},
		{"pierce 300 on 120 does nothing", ResistInput{Resist: 120, Ignore: true, HasPierce: true, Pierce: 300}, 120},
		{"pierce 50 on 99", ResistInput{Resist: 99, Ignore: true, HasPierce: true, Pierce: 50}, 49},
		{"pierce past floor below immunity", ResistInput{Resist: 90, Ignore: true, HasPierce: true, Pierce: 300}, -100},
		{"difficulty penalty skipped", ResistInput{Resist: 50, Ignore: true, DifficultyPenalty: -100}, 50},
		{"player still capped", ResistInput{Resist: 100}, 75},
	}

	for _, tt := range tests {
		if got := EffectiveResist(tt.in); got != tt.want {
			t.Errorf("%s: got %d want %d", tt.name, got, tt.want)
		}
	}

	if ApplyResist(1000, EffectiveResist(ResistInput{Resist: 100, Ignore: true})) != 0 {
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

// TestResistPlayerPierceAndPenaltyOrder pins pierce before the difficulty
// penalty before the cap for a non-ignored defender (VERIFIED 0x579b10), and
// that a mercenary (not ignored) is capped like a player.
func TestResistPlayerPierceAndPenaltyOrder(t *testing.T) {
	tests := []struct {
		name string
		in   ResistInput
		want int
	}{
		{"pierce applies to a player at 100", ResistInput{Resist: 100, HasPierce: true, Pierce: 30}, 70},
		{"pierce then penalty", ResistInput{Resist: 100, HasPierce: true, Pierce: 30, DifficultyPenalty: -40}, 30},
		{"magic exempt from penalty", ResistInput{Resist: 60, NoDifficultyPenalty: true, DifficultyPenalty: -100}, 60},
		{"no max stat: cap 75", ResistInput{Resist: 90}, 75},
		{"negative max stat lowers cap", ResistInput{Resist: 90, HasMaxResist: true, MaxResistBonus: -10}, 65},
		{"physical cap 50", ResistInput{Resist: 90, IsPhysical: true}, 50},
		{"no pierce stat (physical) ignores Pierce", ResistInput{Resist: 40, Pierce: 30}, 40},
	}

	for _, tt := range tests {
		if got := EffectiveResist(tt.in); got != tt.want {
			t.Errorf("%s: got %d want %d", tt.name, got, tt.want)
		}
	}
}

// TestResistZeroPhysicalState pins the state 0x2f special case: only physical
// resist, only when the resist is positive (VERIFIED 0x579b10).
func TestResistZeroPhysicalState(t *testing.T) {
	tests := []struct {
		name string
		in   ResistInput
		want int
	}{
		{"positive physical zeroed", ResistInput{Resist: 40, IsPhysical: true, ZeroPhysical: true, Ignore: true}, 0},
		{"zeroed without ignore", ResistInput{Resist: 40, IsPhysical: true, ZeroPhysical: true}, 0},
		{"negative physical kept", ResistInput{Resist: -30, IsPhysical: true, ZeroPhysical: true, Ignore: true}, -30},
		{"not physical unaffected", ResistInput{Resist: 40, ZeroPhysical: true, Ignore: true}, 40},
	}

	for _, tt := range tests {
		if got := EffectiveResist(tt.in); got != tt.want {
			t.Errorf("%s: got %d want %d", tt.name, got, tt.want)
		}
	}
}

// TestApplyResistFull pins the MulDiv application of 0x579c90: flat reduction
// first, truncating division, resist clamped at 100, unresistable flag.
func TestApplyResistFull(t *testing.T) {
	tests := []struct {
		name           string
		dmg, flat, res int
		unresistable   bool
		want           int
	}{
		{"truncates toward zero", 1001, 0, 50, false, 500},
		{"75 percent", 1000, 0, 75, false, 250},
		{"flat before percent", 1000, 200, 50, false, 400},
		{"flat exceeds damage", 100, 200, 50, false, -100},
		{"res 0 keeps flat subtraction", 1000, 300, 0, false, 700},
		{"over 100 clamps to immune", 1000, 0, 250, false, 0},
		{"negative resist amplifies", 1000, 0, -50, false, 1500},
		{"unresistable ignores positive resist and flat", 1000, 300, 60, true, 1000},
		{"unresistable still amplifies", 1000, 300, -50, true, 1500},
		{"non positive damage", 0, 0, 10, false, 0},
	}

	for _, tt := range tests {
		if got := ApplyResistFull(tt.dmg, tt.flat, tt.res, tt.unresistable); got != tt.want {
			t.Errorf("%s: got %d want %d", tt.name, got, tt.want)
		}
	}
}

// TestAbsorbCapAndOrder pins 0x579c20: percent capped at 40, then flat*256
// limited to what is left; types without an absorb stat pass through.
func TestAbsorbCapAndOrder(t *testing.T) {
	if rem, heal := Absorb(1000, true, 90, 0); rem != 600 || heal != 400 {
		t.Errorf("percent cap: rem %d heal %d", rem, heal)
	}

	if rem, heal := Absorb(1000, true, 10, 2); rem != 1000-100-512 || heal != 612 {
		t.Errorf("percent then flat: rem %d heal %d", rem, heal)
	}

	if rem, heal := Absorb(300, true, 0, 5); rem != 0 || heal != 300 {
		t.Errorf("flat capped by remainder: rem %d heal %d", rem, heal)
	}

	if rem, heal := Absorb(300, true, -5, 0); rem != 300 || heal != 0 {
		t.Errorf("negative percent: rem %d heal %d", rem, heal)
	}

	if rem, heal := Absorb(300, false, 50, 5); rem != 300 || heal != 0 {
		t.Errorf("no absorb stat: rem %d heal %d", rem, heal)
	}
}
