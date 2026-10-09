package d2difficulty

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"

// Progression: the d2s header keeps, in the status word bits 8..12 (byte
// 0x25), how far the character got. UNVERIFIED encoding (agrees with the real
// level-94 sample, progression 13): one step per act boss killed, 5 steps per
// difficulty in the expansion (4 in classic). The menus gate a difficulty on a
// minimum progression (error codes 0xd/0xe in SAVE_ValidateD2sHeader, verified
// to exist; the thresholds are this reading).

// Progression reads the progression out of the header status word.
func Progression(status uint32) int { return int(status>>8) & 0x1f }

// WithProgression returns the status word with the progression replaced.
func WithProgression(status uint32, p int) uint32 {
	if p < 0 {
		p = 0
	}

	if p > 0x1f {
		p = 0x1f
	}

	return status&^(0x1f<<8) | uint32(p)<<8
}

// ActsPerDifficulty is 5 in the expansion, 4 in classic.
func ActsPerDifficulty(expansion bool) int {
	if expansion {
		return 5
	}

	return 4
}

// bossQuest is the (act, quest) whose completion is the end of an act: Sisters
// to the Slaughter (Andariel), The Seven Tombs (Duriel), The Guardian
// (Mephisto), Terror's End (Diablo), Eve of Destruction (Baal). The "Terror's
// End" quest is Diablo, it ends classic Normal; in the expansion the last act
// boss is Baal (A5Q6).
var bossQuest = [d2s.NumActs][2]int{{1, 6}, {2, 6}, {3, 6}, {4, 2}, {5, 6}}

// ActBossDefeated reports whether the end-of-act quest of act (1..5) is
// complete or has its reward pending in a difficulty's quest record
// (UNVERIFIED that the reward-pending state already counts).
func ActBossDefeated(q *d2s.QuestRecord, act int) bool {
	if q == nil || act < 1 || act > d2s.NumActs {
		return false
	}

	a, n := bossQuest[act-1][0], bossQuest[act-1][1]

	return q.Completed(a, n) || q.RewardPending(a, n)
}

// ActsDefeated counts the acts whose boss is defeated, in order: counting stops
// at the first act that is not.
func ActsDefeated(q *d2s.QuestRecord, expansion bool) int {
	n := 0

	for act := 1; act <= ActsPerDifficulty(expansion); act++ {
		if !ActBossDefeated(q, act) {
			break
		}

		n++
	}

	return n
}

// ProgressionOf computes the progression byte of a hero from its three quest
// records: the finished difficulties count fully, the first unfinished one by
// the acts done.
func ProgressionOf(quests [3]d2s.QuestRecord, expansion bool) int {
	per := ActsPerDifficulty(expansion)
	total := 0

	for d := 0; d < int(Count); d++ {
		n := ActsDefeated(&quests[d], expansion)
		total += n

		if n < per {
			break
		}
	}

	return total
}

// Finished reports whether a difficulty has been completed: its last act boss
// is defeated in that difficulty's quest record, or the stored progression
// already reaches the end of it.
func Finished(l Level, quests [3]d2s.QuestRecord, progression int, expansion bool) bool {
	l = Clamp(int(l))
	per := ActsPerDifficulty(expansion)

	if progression >= per*(int(l)+1) {
		return true
	}

	return ActsDefeated(&quests[l], expansion) >= per
}

// Unlocked lists whether each difficulty may be started: Normal always,
// Nightmare after Normal is finished, Hell after Nightmare is finished.
func Unlocked(quests [3]d2s.QuestRecord, progression int, expansion bool) [3]bool {
	u := [3]bool{true, false, false}
	u[Nightmare] = Finished(Normal, quests, progression, expansion)
	u[Hell] = u[Nightmare] && Finished(Nightmare, quests, progression, expansion)

	return u
}

// Allowed reports whether one difficulty may be started.
func Allowed(l Level, quests [3]d2s.QuestRecord, progression int, expansion bool) bool {
	return Unlocked(quests, progression, expansion)[Clamp(int(l))]
}

// Highest returns the highest unlocked difficulty.
func Highest(u [3]bool) Level {
	for l := Hell; l > Normal; l-- {
		if u[l] {
			return l
		}
	}

	return Normal
}
