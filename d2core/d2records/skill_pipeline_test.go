package d2records

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
)

// calcUnit is the minimal caster the calc evaluation needs.
type calcUnit struct {
	d2skill.Unit
	reg    *d2skill.Registry
	levels map[string]int
}

func (u calcUnit) Level() int { return 30 }
func (u calcUnit) SkillLevel(id int) int {
	return u.levels[u.reg.ByID(id).Name]
}
func (u calcUnit) BaseSkillLevel(id int) int { return u.levels[u.reg.ByID(id).Name] }
func (u calcUnit) Stat(string) int           { return 0 }
func (u calcUnit) Roller() d2combat.Roller   { return nil }

// loadRealRecords loads the real skills.txt and missiles.txt from D2_TABLES.
func loadRealRecords(t *testing.T) *RecordManager {
	t.Helper()

	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	rm := &RecordManager{Logger: d2util.NewLogger()}

	for _, f := range []struct {
		name   string
		loader recordLoader
	}{{"skills.txt", skillDetailsLoader}, {"missiles.txt", missilesLoader}} {
		buf, err := os.ReadFile(filepath.Join(root, "skills", "patch_d2", f.name))
		if err != nil {
			t.Skip(err)
		}

		if err = f.loader(rm, d2txt.LoadDataDictionary(buf)); err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}
	}

	return rm
}

func TestRealSkillCalcsEvaluate(t *testing.T) {
	rm := loadRealRecords(t)
	reg := rm.SkillTable()

	if reg.ByName("Fire Bolt") == nil || reg.ByName("Frost Nova") == nil {
		t.Fatalf("skills missing (%d loaded)", reg.Len())
	}

	u := calcUnit{reg: reg, levels: map[string]int{}}
	env := func(name string, lvl int) *d2skill.Env { return d2skill.NewEnv(reg.ByName(name), lvl, u, reg) }

	// Fire Bolt level 10, no synergy: 17.5 - 22.5 (HitShift 7)
	fb := reg.ByName("Fire Bolt")
	if got := fb.ElemMin(env("Fire Bolt", 10), 10); got != 4480 {
		t.Errorf("Fire Bolt L10 min = %d", got)
	}

	if got := fb.ElemMax(env("Fire Bolt", 10), 10); got != 5760 {
		t.Errorf("Fire Bolt L10 max = %d", got)
	}

	// synergy: Fire Ball 10 + Meteor 5, par8 16 -> +240%
	u.levels["Fire Ball"], u.levels["Meteor"] = 10, 5
	if got := fb.ElemMin(env("Fire Bolt", 10), 10); got != 4480+4480*240/100 {
		t.Errorf("Fire Bolt L10 with synergy min = %d", got)
	}

	// Charged Bolt: calc1 bolts = min(24, 3+(lvl-1))
	for lvl, want := range map[int]int{1: 3, 5: 7, 22: 24, 30: 24} {
		if got := env("Charged Bolt", lvl).Eval(reg.ByName("Charged Bolt").Calc[1]); got != want {
			t.Errorf("Charged Bolt L%d calc1 = %d, want %d", lvl, got, want)
		}
	}

	// Frost Nova: 22 - 28.5 cold at level 10, chill 200 + 7*25 + 2*25 frames
	fn := reg.ByName("Frost Nova")
	e := env("Frost Nova", 10)

	if fn.ElemMin(e, 10) != 5632 || fn.ElemMax(e, 10) != 7296 || fn.ElemLen(e, 10) != 425 {
		t.Errorf("Frost Nova L10: %d %d %d", fn.ElemMin(e, 10), fn.ElemMax(e, 10), fn.ElemLen(e, 10))
	}

	// Frozen Armor duration at level 3 with Shiver 2 + Chilling 4: 3000+2*300 + 6*250
	u.levels["Shiver Armor"], u.levels["Chilling Armor"] = 2, 4
	if got := env("Frozen Armor", 3).Eval(reg.ByName("Frozen Armor").AuraLenCalc); got != 3600+1500 {
		t.Errorf("Frozen Armor duration = %d", got)
	}

	// Bash damage bonus ln12 + Stun*par8 with Stun 3: 50+5*(4-1)+15
	u.levels["Stun"] = 3
	if got := env("Bash", 4).Eval(reg.ByName("Bash").Calc[1]); got != 80 {
		t.Errorf("Bash calc1 = %d", got)
	}

	// Warmth: 30 + 12(l-1)
	if got := env("Warmth", 8).Eval(reg.ByName("Warmth").PassiveCalc[1]); got != 114 {
		t.Errorf("Warmth passivecalc1 = %d", got)
	}

	// every skill's every calc evaluates without panic, at levels 0, 1 and 20
	for _, sk := range []int{0, 1, 20} {
		for _, rec := range rm.Skill.Details {
			s := rec.PipelineSkill()
			ev := d2skill.NewEnv(s, sk, u, reg)

			ev.Eval(s.Delay)
			ev.Eval(s.AuraLenCalc)
			ev.Eval(s.Calc[1])
			ev.Eval(s.Calc[2])
			ev.Eval(s.ToHitCalc)
			ev.Eval(s.EDmgSymPer)
		}
	}
}

func TestRealMissileSpecs(t *testing.T) {
	rm := loadRealRecords(t)
	mt := rm.MissileTable()

	fb := mt.ByName("firebolt")
	if fb == nil || fb.ID != 58 || fb.Vel != 20 || fb.Range != 50 || fb.CollideType != 3 || !fb.CollideKill ||
		!fb.LastCollide || fb.ExplosionMissile != "fireexplode" || fb.SkillName != "Fire Bolt" {
		t.Fatalf("firebolt spec %+v", fb)
	}

	if ib := mt.ByID(59); ib == nil || ib.Name != "icebolt" || ib.Vel != 12 {
		t.Fatalf("icebolt %+v", ib)
	}

	howl := mt.ByName("Howl")
	if howl == nil || howl.Accel != -1000 || howl.SrvHitFunc != 17 || howl.CollideKill {
		t.Fatalf("howl %+v", howl)
	}

	arrow := mt.ByName("firearrow")
	if arrow == nil || !arrow.ToHit || !arrow.Pierce || arrow.Vel != 24 {
		t.Fatalf("firearrow %+v", arrow)
	}
}
