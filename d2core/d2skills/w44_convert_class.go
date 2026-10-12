package d2skills

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Conversion special case (wave 44, VERIFIED SKILL_IsValidMonsterSkillTarget 0x56c0a0): before the shared
// candidate test (MONAI_IsCorpseReviveCandidate 0x5dc1e0) the function reads the monstats base-id word of the
// target; when it is class 0x1ec (492) and the unit holds state 0x8f (143, states.txt row "attached") the
// target is refused. Row 492 of the 1.14 monstats is imp1 and row 143 of states.txt is "attached".
const (
	convertRefusedBaseClass = 0x1ec
	convertRefusedState     = "attached"
)

// convertRefusedByClass reports the 0x56c0a0 special case for a monster whose base key is the monstats row
// convertRefusedBaseClass and that holds the state.
func convertRefusedByClass(stats d2records.MonStats, baseKey string, holdsState bool) bool {
	if !holdsState {
		return false
	}

	for _, st := range stats {
		if st != nil && st.ID == convertRefusedBaseClass {
			return strings.EqualFold(st.Key, baseKey)
		}
	}

	return false
}
