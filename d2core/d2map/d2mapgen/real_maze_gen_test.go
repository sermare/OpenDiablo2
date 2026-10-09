package d2mapgen

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func TestEntryRoomNames(t *testing.T) {
	tests := []struct {
		file string
		want bool
	}{
		// rooms whose DS1 holds the entry marker (found by scanning the Act 1 DS1 files)
		{"Act1/Caves/CaveEPre2.ds1", true},
		{"Act1/Caves/CaveSPre1.ds1", true},
		{"Act1/Caves/DenEnt.ds1", true},
		{"Act1/Crypt/CryptEWarpPrev.ds1", true},
		{"Act1/Catacomb/CatNSUp3.ds1", true},
		{"Act1/Catacomb/CatNSEWExit.ds1", true},
		{"Act1/Barracks/JailSWarpPrev.ds1", true},
		// exits further down, filler and theme rooms
		{"Act1/Caves/CaveSDown2.ds1", false},
		{"Act1/Caves/CaveNext1.ds1", false},
		{"Act1/Crypt/CryptNWarpNext.ds1", false},
		{"Act1/Caves/CaveWCrow.ds1", false},
		{"Act1/Catacomb/CatNSETheme1.ds1", false},
	}

	for _, tt := range tests {
		if got := entryRoom.MatchString(tt.file); got != tt.want {
			t.Errorf("%s: got %v want %v", tt.file, got, tt.want)
		}
	}
}

func TestLevelMonsters(t *testing.T) {
	d := &d2records.LevelDetailRecord{
		NumMonsterTypes:      2,
		MonsterID1Normal:     "skeleton1",
		MonsterID2Normal:     "zombie2",
		MonsterID3Normal:     "ignored",
		MonsterID1Nightmare:  "skeleton2",
		MonsterID2Nightmare:  "0",
		MonsterID3Nightmare:  "x",
		MonsterID10Nightmare: "y",
	}

	tests := []struct {
		name string
		d    *d2records.LevelDetailRecord
		diff d2drlg.Difficulty
		want []string
	}{
		{"normal, limited by NumMon", d, 0, []string{"skeleton1", "zombie2"}},
		{"nightmare skips the 0 placeholder", d, 1, []string{"skeleton2"}},
		{"no record", nil, 0, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := levelMonsters(tt.d, tt.diff)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestRealLevel(t *testing.T) {
	tests := []struct {
		name                       string
		realmaps, autolevel, autom string
		want                       int
	}{
		{"off without OD2_REALMAPS", "", "9", "34", 0},
		{"autolevel wins", "1", "18", "34", 18},
		{"falls back to automap", "1", "", "34", 34},
		{"nothing selected", "1", "", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OD2_REALMAPS", tt.realmaps)
			t.Setenv("OD2_AUTOLEVEL", tt.autolevel)
			t.Setenv("OD2_AUTOMAP", tt.autom)

			if got := RealLevel(); got != tt.want {
				t.Fatalf("got %d want %d", got, tt.want)
			}
		})
	}
}
