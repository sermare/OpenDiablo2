package d2monster

import "strings"

// Mode is a monster unit mode (VERIFIED numbering, monster-ai-2.md section 0).
type Mode int

// Monster modes.
const (
	ModeDying   Mode = 0 // DT
	ModeNeutral Mode = 1 // NU
	ModeWalk    Mode = 2 // WL
	ModeAttack1 Mode = 4 // A1
	ModeAttack2 Mode = 5 // A2
	ModeGetHit  Mode = 6 // GH
	ModeSkill1  Mode = 8 // S1
	ModeSkill2  Mode = 9 // S2 ("shout" in the Fallen rally)
	ModeSkill3  Mode = 10
	ModeSkill4  Mode = 11
	ModeDead    Mode = 12 // DD
	ModeCast    Mode = 14 // SQ
	ModeRun     Mode = 15 // RN
)

var modeNames = map[string]Mode{
	"DT": ModeDying, "NU": ModeNeutral, "WL": ModeWalk, "A1": ModeAttack1,
	"A2": ModeAttack2, "GH": ModeGetHit, "S1": ModeSkill1, "S2": ModeSkill2,
	"S3": ModeSkill3, "S4": ModeSkill4, "DD": ModeDead, "SQ": ModeCast,
	"RN": ModeRun,
}

// ParseMode converts a monstats mode token ("A1", "S2", "SQ"...) to a Mode.
func ParseMode(s string) (Mode, bool) {
	m, ok := modeNames[strings.ToUpper(strings.TrimSpace(s))]

	return m, ok
}

// IsAlive is false for the dying and dead modes.
func (m Mode) IsAlive() bool { return m != ModeDying && m != ModeDead }
