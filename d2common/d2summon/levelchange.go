package d2summon

// What happens to a player's pets when the owner changes level inside an act
// (MERC_RelocatePetsWithOwner 0x5732b0, VERIFIED against the exe). For every
// pet type the exe reads a flag byte of the pettype record (stride 0xe0,
// byte +4). The loader DATATBL_LoadPetTypesTable 0x618a30 fills bit 0 from the
// column "warp" and bit 1 from the column "range" (VERIFIED from the column
// table: both are bit columns of the same byte). Then:
//
//   - bit 0 (warp): all pets of the type are moved to the owner;
//   - else bit 1 (range): pets whose squared distance to the owner exceeds
//     1600 (40 subtiles; PETS_PruneFarPetsInList 0x573180, strictly greater)
//     are released, the others stay where they are;
//   - neither: the whole list of the type is freed (dopplezon).
//
// An act change releases every pet type instead (PETS_ReleaseAllPetsOnActChange
// 0x573980, hireable kept), which the engine models in minions.go.

// PetFlags are the two PetType.txt columns the level change reads.
type PetFlags struct {
	Warp, Range bool
}

// Fate is what a level change does to one pet.
type Fate int

// The fates.
const (
	// FateFollow moves the pet next to the owner.
	FateFollow Fate = iota
	// FateStay leaves the pet where it is (a range pet within 40 subtiles).
	FateStay
	// FateDrop releases the pet.
	FateDrop
)

// RangeDropDistSq is the squared distance above which a range pet is dropped.
const RangeDropDistSq = 1600

// LevelChangeFate decides a pet's fate from its type flags and its squared
// distance to the owner before the move.
func LevelChangeFate(f PetFlags, distSq int) Fate {
	switch {
	case f.Warp:
		return FateFollow
	case f.Range:
		if distSq > RangeDropDistSq {
			return FateDrop
		}

		return FateStay
	}

	return FateDrop
}

// PetRef is what the carry-over planner needs to know about one pet.
type PetRef struct {
	Flags  PetFlags
	DistSq int // squared subtile distance to the owner before the move
}

// CarryOverList plans a level change for a list of pets and returns the
// indices (into pets, in order) of those that travel with the owner. On an act
// change nothing travels (PETS_ReleaseAllPetsOnActChange 0x573980). Inside an
// act only FateFollow pets travel. A FateStay pet (range type within 40
// subtiles) keeps its place in the exe's list, but the engine rebuilds the map
// and does not park pets, so it cannot stay: it is released like a dropped
// one (engine limitation, not exe behaviour).
func CarryOverList(pets []PetRef, actChange bool) []int {
	if actChange {
		return nil
	}

	var out []int

	for i, p := range pets {
		if LevelChangeFate(p.Flags, p.DistSq) == FateFollow {
			out = append(out, i)
		}
	}

	return out
}
