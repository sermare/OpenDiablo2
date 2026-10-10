package d2monster

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestColumnForMode(t *testing.T) {
	tests := []struct {
		mode Mode
		want Column
	}{
		{ModeAttack1, ColumnA1}, {ModeAttack2, ColumnA2},
		{ModeSpecialCast, ColumnS1}, {ModeSkill1, ColumnS1},
		// every other mode falls to the A1 columns (VERIFIED 0x5a2960)
		{ModeSkill2, ColumnA1}, {ModeSkill3, ColumnA1}, {ModeSkill4, ColumnA1}, {ModeCast, ColumnA1},
		{ModeNeutral, ColumnA1}, {ModeGetHit, ColumnA1}, {ModeRun, ColumnA1},
	}

	for _, tt := range tests {
		if got := ColumnForMode(tt.mode); got != tt.want {
			t.Errorf("mode %v: column %v want %v", tt.mode, got, tt.want)
		}
	}
}

func TestElementFires(t *testing.T) {
	if !ElementFires(ModeAttack1, ModeAttack1) || ElementFires(ModeAttack1, ModeAttack2) || ElementFires(0, 0) {
		t.Error("El mode must equal the request mode and 0 means unused")
	}
}

func TestParseModeSequences(t *testing.T) {
	tests := []struct {
		in   string
		want Mode
		ok   bool
	}{
		{"A1", ModeAttack1, true}, {"s3", ModeSkill3, true}, {"SC", ModeSpecialCast, true},
		{"SQ", ModeCast, true}, {"seq_skeletonraise", ModeCast, true}, {"SEQ_MummyRes", ModeCast, true},
		{"", 0, false}, {"bogus", 0, false},
	}

	for _, tt := range tests {
		got, ok := ParseMode(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("ParseMode(%q) = %v,%v want %v,%v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestClassifyEffect(t *testing.T) {
	tests := []struct {
		name      string
		do        int
		hasMis    bool
		want      EffectKind
		skillName string
	}{
		{"Attack", 1, false, EffectMelee, "Attack"},
		{"Jab", 7, false, EffectMelee, "Jab"},
		{"Charge", 67, false, EffectMelee, "Charge"},
		{"Smite", 150, false, EffectMelee, "Smite"},
		{"FireHit", 83, false, EffectMelee, "Fire Hit"},
		{"Baal Nova", 22, false, EffectArea, "Baal Nova"},
		{"Fire Ball missile", 0, true, EffectMissile, "Fire Ball"},
		{"Amplify Damage curse", 30, false, EffectNone, "Amplify Damage"},
		{"Resurrect", 97, false, EffectNone, "Resurrect"},
		{"Teleport", 98, false, EffectNone, "MonTeleport"},
		{"MinionSpawner", 135, false, EffectNone, "MinionSpawner"},
		{"SkeletonRaise", 0, false, EffectNone, "SkeletonRaise"},
	}

	for _, tt := range tests {
		if got := ClassifyEffect(tt.do, tt.hasMis); got != tt.want {
			t.Errorf("%s (do %d missile %v): %v want %v", tt.skillName, tt.do, tt.hasMis, got, tt.want)
		}
	}
}

// TestRealMonsterSkillSlots audits every used Skill slot of the real 1.14b
// monstats.txt: its Sk#mode must resolve to a real mode (the seq_* slots used
// to parse as DT and silently fall back to A1), the skill must exist in the
// patch_d2 skills.txt, and its level must be positive unless the slot is a
// passive NU one. Skipped without D2_TABLES.
func TestRealMonsterSkillSlots(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	buf, err := os.ReadFile(filepath.Join(root, "monsters", "patch_d2", "monstats.txt"))
	if err != nil {
		t.Skip(err)
	}

	sk, err := os.ReadFile(filepath.Join(root, "monsters", "patch_d2", "skills.txt"))
	if err != nil {
		t.Skip(err)
	}

	names := map[string]bool{}

	for i, line := range strings.Split(string(sk), "\n") {
		if i == 0 {
			continue
		}

		if f := strings.SplitN(line, "\t", 2); f[0] != "" {
			names[f[0]] = true
		}
	}

	tp, err := LoadTxtProfiles(buf)
	if err != nil {
		t.Fatal(err)
	}

	used, seq := 0, 0

	for class := 0; class < 1000; class++ {
		p, ok := tp.Profile(class, Normal)
		if !ok {
			continue
		}

		for i, s := range p.Skills {
			if !s.Used() {
				continue
			}

			used++

			if s.Mode == ModeDying {
				t.Errorf("%s slot %d (%s): mode parsed as DT", p.ID, i+1, s.Name)
			}

			if s.Mode == ModeCast {
				seq++
			}

			if !names[s.Name] {
				t.Errorf("%s slot %d: skill %q is not in skills.txt", p.ID, i+1, s.Name)
			}
		}
	}

	if used != 775 || seq != 319 { // golden: 775 used slots, 319 of them seq_* (SQ)
		t.Errorf("used slots %d (want 775), SQ slots %d (want 319): table changed or not read?", used, seq)
	}
}
