package d2monster

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
)

// TxtProfiles is a ProfileSource built straight from a 1.14b monstats.txt
// (the patch_d2 / d2exp layout with AI, aip1..8, aidel, aidist; the old d2data
// layout lacks the AI column and must not be used).
type TxtProfiles struct {
	byClass map[int]*[3]Profile
	byID    map[string]int
}

var diffSuffix = [3]string{"", "(N)", "(H)"}

// LoadTxtProfiles parses monstats.txt contents.
func LoadTxtProfiles(monstats []byte) (*TxtProfiles, error) {
	d := d2txt.LoadDataDictionary(monstats)
	t := &TxtProfiles{byClass: map[int]*[3]Profile{}, byID: map[string]int{}}

	for d.Next() {
		class := d.Number("hcIdx")
		id := d.String("Id")

		var ps [3]Profile

		for diff := 0; diff < 3; diff++ {
			sfx := diffSuffix[diff]
			p := Profile{
				Class: class, ID: id, AI: d.String("AI"),
				AIDel:  d.Number("aidel" + sfx),
				AIDist: d.Number("aidist" + sfx),
				Threat: d.Number("threat"),
				Walk:   d.Number("Velocity"),
				Run:    d.Number("Run"),
			}

			for n := 1; n <= 8; n++ {
				p.AIP[n] = d.Number(fmt.Sprintf("aip%d%s", n, sfx))
			}

			for s := 0; s < NumSkills; s++ {
				p.Skills[s].Name = d.String(fmt.Sprintf("Skill%d", s+1))
				p.Skills[s].Level = d.Number(fmt.Sprintf("Sk%dlvl", s+1))
				p.Skills[s].Mode, _ = ParseMode(d.String(fmt.Sprintf("Sk%dmode", s+1)))
			}

			ps[diff] = p
		}

		t.byClass[class] = &ps
		t.byID[id] = class
	}

	if d.Err != nil {
		return nil, d.Err
	}

	return t, nil
}

// Profile implements ProfileSource.
func (t *TxtProfiles) Profile(class int, diff Difficulty) (*Profile, bool) {
	ps, ok := t.byClass[class]
	if !ok || diff < Normal || diff > Hell {
		return nil, false
	}

	p := ps[diff]

	return &p, true
}

// ByID resolves a monstats Id ("skeleton1") to its class.
func (t *TxtProfiles) ByID(id string) (int, bool) {
	c, ok := t.byID[id]

	return c, ok
}
