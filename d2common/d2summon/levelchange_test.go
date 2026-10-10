package d2summon

import "testing"

func TestLevelChangeFate(t *testing.T) {
	// flags as in the 1.14b PetType.txt
	skeleton := PetFlags{Warp: true}
	trap := PetFlags{Range: true}
	dopplezon := PetFlags{}

	tests := []struct {
		name string
		f    PetFlags
		dist int
		want Fate
	}{
		{"warp pet far away follows", skeleton, 100000, FateFollow},
		{"warp pet near follows", skeleton, 0, FateFollow},
		{"range pet at the limit stays", trap, 1600, FateStay},
		{"range pet just beyond is dropped", trap, 1601, FateDrop},
		{"range pet near stays", trap, 4, FateStay},
		{"neither flag frees the pet", dopplezon, 0, FateDrop},
		{"warp wins over range", PetFlags{Warp: true, Range: true}, 5000, FateFollow},
	}

	for _, tc := range tests {
		if got := LevelChangeFate(tc.f, tc.dist); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}
