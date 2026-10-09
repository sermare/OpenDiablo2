package d2records

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
)

func monsterLevelsLoader(r *RecordManager, d *d2txt.DataDictionary) error {
	records := make(map[int]*MonsterLevelRecord)

	for d.Next() {
		record := &MonsterLevelRecord{
			Level: d.Number("Level"),
			BattleNet: monsterDifficultyLevels{
				Normal: monsterLevelValues{
					Hitpoints:     d.Number("HP"),
					DefenseRating: d.Number("AC"),
					AttackRating:  d.Number("TH"),
					Damage:        d.Number("DM"),
					Experience:    d.Number("XP"),
				},
				Nightmare: monsterLevelValues{
					Hitpoints:     d.Number("HP(N)"),
					DefenseRating: d.Number("AC(N)"),
					AttackRating:  d.Number("TH(N)"),
					Damage:        d.Number("DM(N)"),
					Experience:    d.Number("XP(N)"),
				},
				Hell: monsterLevelValues{
					Hitpoints:     d.Number("HP(H)"),
					DefenseRating: d.Number("AC(H)"),
					AttackRating:  d.Number("TH(H)"),
					Damage:        d.Number("DM(H)"),
					Experience:    d.Number("XP(H)"),
				},
			},
			Ladder: monsterDifficultyLevels{
				Normal: monsterLevelValues{
					Hitpoints:     d.Number("L-HP"),
					DefenseRating: d.Number("L-AC"),
					AttackRating:  d.Number("L-TH"),
					Damage:        d.Number("L-DM"),
					Experience:    d.Number("L-XP"),
				},
				Nightmare: monsterLevelValues{
					Hitpoints:     d.Number("L-HP(N)"),
					DefenseRating: d.Number("L-AC(N)"),
					AttackRating:  d.Number("L-TH(N)"),
					Damage:        d.Number("L-DM(N)"),
					Experience:    d.Number("L-XP(N)"),
				},
				Hell: monsterLevelValues{
					Hitpoints:     d.Number("L-HP(H)"),
					DefenseRating: d.Number("L-AC(H)"),
					AttackRating:  d.Number("L-TH(H)"),
					Damage:        d.Number("L-DM(H)"),
					Experience:    d.Number("L-XP(H)"),
				},
			},
		}
		records[record.Level] = record
	}

	if d.Err != nil {
		return d.Err
	}

	r.Debugf("Loaded %d MonsterLevel records", len(records))

	r.Monster.Levels = records

	return nil
}
