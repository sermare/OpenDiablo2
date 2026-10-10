package d2monster

import "strings"

// Mode is a monster unit mode (VERIFIED numbering, monster-ai-2.md section 0).
type Mode int

// Monster modes.
const (
	ModeDying       Mode = 0 // DT
	ModeNeutral     Mode = 1 // NU
	ModeWalk        Mode = 2 // WL
	ModeAttack1     Mode = 4 // A1
	ModeAttack2     Mode = 5 // A2
	ModeGetHit      Mode = 6 // GH
	ModeSpecialCast Mode = 7 // SC (the cast animation of Summoner, Izual, Diablo...)
	ModeSkill1      Mode = 8 // S1
	ModeSkill2      Mode = 9 // S2 ("shout" in the Fallen rally)
	ModeSkill3      Mode = 10
	ModeSkill4      Mode = 11
	ModeDead        Mode = 12 // DD
	ModeCast        Mode = 14 // SQ
	ModeRun         Mode = 15 // RN
)

var modeNames = map[string]Mode{
	"DT": ModeDying, "NU": ModeNeutral, "WL": ModeWalk, "A1": ModeAttack1,
	"A2": ModeAttack2, "GH": ModeGetHit, "S1": ModeSkill1, "S2": ModeSkill2,
	"S3": ModeSkill3, "S4": ModeSkill4, "DD": ModeDead, "SQ": ModeCast, "SC": ModeSpecialCast,
	"RN": ModeRun,
}

// ParseMode converts a monstats mode token ("A1", "S2", "SQ"...) to a Mode.
//
// A monstats Sk#mode of the form "seq_<name>" names a monseq.txt sequence
// (321 of the 1.14b Skill slots, e.g. seq_skeletonraise, seq_mummyres). Such
// a skill is played in the SQ mode (14): MONAI_QueueSkillCast has an SQ
// variant (0x5dcde0, VERIFIED in monster-ai-2.md) and monstats2 carries the
// sequence name next to the mSQ flag. The token-to-mode mapping itself is
// UNVERIFIED (the CUSTOMLINK loader was not decompiled). Before this mapping
// those slots parsed as mode 0 (DT) and fell back to A1.
func ParseMode(s string) (Mode, bool) {
	t := strings.ToLower(strings.TrimSpace(s))
	if strings.HasPrefix(t, "seq_") {
		return ModeCast, true
	}

	m, ok := modeNames[strings.ToUpper(t)]

	return m, ok
}

// IsAlive is false for the dying and dead modes.
func (m Mode) IsAlive() bool { return m != ModeDying && m != ModeDead }

// String is the monstats mode token ("A1", "NU"...).
func (m Mode) String() string {
	for name, v := range modeNames {
		if v == m {
			return name
		}
	}

	return "??"
}
