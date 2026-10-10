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

func TestCarryOverList(t *testing.T) {
	warp, rng, none := PetFlags{Warp: true}, PetFlags{Range: true}, PetFlags{}

	tests := []struct {
		name string
		pets []PetRef
		act  bool
		want []int
	}{
		{"empty", nil, false, nil},
		{"warp follows at any distance", []PetRef{{warp, 0}, {warp, 99999}}, false, []int{0, 1}},
		{"range near stays behind, far dropped", []PetRef{{rng, 1600}, {rng, 1601}}, false, nil},
		{"neither released", []PetRef{{none, 0}}, false, nil},
		{"mixed keeps order", []PetRef{{none, 1}, {warp, 5}, {rng, 2}, {warp, 7}}, false, []int{1, 3}},
		{"act change releases all", []PetRef{{warp, 0}, {warp, 4}}, true, nil},
	}

	for _, tc := range tests {
		got := CarryOverList(tc.pets, tc.act)
		if len(got) != len(tc.want) {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}

		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
			}
		}
	}
}
