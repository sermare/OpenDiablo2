package d2hero

import "testing"

func TestCycleSkill(t *testing.T) {
	skills := map[int]*HeroSkill{
		0:  heroSkill(0, 1, true, false),  // Attack
		36: heroSkill(36, 3, true, false), // usable on both buttons
		40: heroSkill(40, 0, true, false), // not learned
		47: heroSkill(47, 2, true, false),
		54: heroSkill(54, 1, false, false), // right button only
		61: heroSkill(61, 1, true, true),   // passive
	}

	tests := []struct {
		name    string
		current int
		left    bool
		dir     int
		want    int
	}{
		{"next on the right", 0, false, 1, 36},
		{"skips unlearned", 36, false, 1, 47},
		{"right-only skill is reachable on the right", 47, false, 1, 54},
		{"wraps forward", 54, false, 1, 0},
		{"left skips right-only skills", 47, true, 1, 0},
		{"previous", 47, false, -1, 36},
		{"wraps backward", 0, false, -1, 54},
		{"unknown current goes to the next higher", 40, false, 1, 47},
		{"unknown current goes to the previous lower", 40, false, -1, 36},
		{"past the end wraps to the first", 99, true, 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := CycleSkill(skills, tt.current, tt.left, tt.dir)
			if !ok || got != tt.want {
				t.Fatalf("CycleSkill = %d,%v want %d", got, ok, tt.want)
			}
		})
	}

	if _, ok := CycleSkill(map[int]*HeroSkill{61: heroSkill(61, 1, true, true)}, 0, false, 1); ok {
		t.Fatal("a hero with only passives has nothing to cycle")
	}
}
