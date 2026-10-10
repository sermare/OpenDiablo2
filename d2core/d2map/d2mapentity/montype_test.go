package d2mapentity

import "testing"

// The type mask values are the masks the exe's predicates pass to MONSTER_TestTypeFlags (0x4a8f50):
// IsSuperUnique 2, IsChampion 4, IsUnique 8, IsMinion 0x10, IsGhostly 0x40.
func TestMonsterTypeMasksAreTheExeValues(t *testing.T) {
	for _, c := range []struct {
		name string
		got  uint16
		want uint16
	}{
		{"super unique", MonTypeSuperUnique, 0x2},
		{"champion", MonTypeChampion, 0x4},
		{"unique", MonTypeUnique, 0x8},
		{"minion", MonTypeMinion, 0x10},
		{"ghostly", MonTypeGhostly, 0x40},
	} {
		if c.got != c.want {
			t.Errorf("%s mask = %#x, want %#x", c.name, c.got, c.want)
		}
	}
}
