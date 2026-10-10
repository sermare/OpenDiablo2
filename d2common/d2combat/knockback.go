package d2combat

// Knockback item event (callback 0x5bd580, item stat knockback), verified
// against Game.exe in the emulator (golden knock_golden.json). Observations only.

// Knockback thresholds out of 128 (the roll is the attacker's new seed low
// word & 0x7f; the stat value only gates the roll, it is NOT the chance).
const (
	KnockbackBase  = 0x40 // 50 percent
	KnockbackBoss  = 0x20 // monsters whose record byte +5 has bit 8: 25 percent
	KnockbackEasy  = 0x80 // monsters whose record byte +5 has bit 4: always
	knockbackMonB8 = 8
	knockbackMonB4 = 4
)

// KnockbackThreshold is the threshold of the roll for a defender: 0x40, or for
// a monster whose monstats record byte +5 has bit 8 (0x20) or, if not, bit 4
// (0x80). rec5 is that byte; hasRec is false when the class record is missing.
func KnockbackThreshold(defenderKind int, hasRec bool, rec5 byte) int {
	thr := KnockbackBase

	if defenderKind == 1 && hasRec {
		switch {
		case rec5&knockbackMonB8 != 0:
			thr = KnockbackBoss
		case rec5&knockbackMonB4 != 0:
			thr = KnockbackEasy
		}
	}

	return thr
}

// RollKnockback rolls the knockback event: nothing is rolled (no step) unless
// the attacker's knockback stat is positive; the hit sets result bit 8 on the
// damage struct. step advances the attacker's seed and returns the new low word.
func RollKnockback(step func() uint32, statPositive bool, defenderKind int, hasRec bool, rec5 byte) bool {
	if !statPositive {
		return false
	}

	return int(step()&0x7f) < KnockbackThreshold(defenderKind, hasRec, rec5)
}
