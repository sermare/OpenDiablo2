package d2records

import (
	"strings"
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calculation"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
)

// pipelineTables caches the pure-package views of skills.txt and missiles.txt.
type pipelineTables struct {
	once     sync.Once
	skills   *d2skill.Registry
	missiles *MissileTable
}

func (r *RecordManager) buildPipelineTables() {
	r.pipeline.once.Do(func() {
		reg := d2skill.NewRegistry()
		for _, rec := range r.Skill.Details {
			reg.Add(rec.PipelineSkill())
		}

		r.pipeline.skills = reg

		mt := &MissileTable{byID: map[int]*d2missile.Spec{}, byName: map[string]*d2missile.Spec{}}
		for _, rec := range r.Missiles {
			sp := rec.PipelineSpec()
			mt.byID[sp.ID] = sp
			mt.byName[sanitizeMissilesKey(sp.Name)] = sp
		}

		r.pipeline.missiles = mt
	})
}

// SkillTable returns the skills as d2skill.Skill, ready for the cast pipeline.
// Calc columns are compiled d2calc programs; evaluate them with a d2skill.Env.
func (r *RecordManager) SkillTable() *d2skill.Registry {
	r.buildPipelineTables()

	return r.pipeline.skills
}

// MissileTable returns the missiles as d2missile.Spec; it implements
// d2missile.Table.
func (r *RecordManager) MissileTable() *MissileTable {
	r.buildPipelineTables()

	return r.pipeline.missiles
}

// MissileTable is the missiles.txt lookup used by d2missile.Sim.
type MissileTable struct {
	byID   map[int]*d2missile.Spec
	byName map[string]*d2missile.Spec
}

// ByID implements d2missile.Table.
func (t *MissileTable) ByID(id int) *d2missile.Spec { return t.byID[id] }

// ByName implements d2missile.Table; names ignore case and spaces.
func (t *MissileTable) ByName(name string) *d2missile.Spec {
	return t.byName[sanitizeMissilesKey(name)]
}

// PipelineSpec converts a missiles.txt row for d2missile.
func (m *MissileRecord) PipelineSpec() *d2missile.Spec {
	sp := &d2missile.Spec{
		ID: m.Id, Name: m.Name,
		SrvDoFunc: m.ServerMovementFunc, SrvHitFunc: m.ServerCollisionFunc, SrvDmgFunc: m.ServerDamageFunc,
		Vel: m.Velocity, VelLev: m.LevelVelocityBonus, MaxVel: m.MaxVelocity, Accel: m.Accel,
		Range: m.Range, LevRange: m.LevelRangeBonus, Activate: m.Animation.StepsBeforeActive,
		CollideType: m.Collision.CollisionType, CollideKill: m.Collision.DestroyedUponCollision,
		CollideFriend: m.Collision.FriendlyFire, LastCollide: m.Collision.LastCollide, Collision: m.Collision.Collision,
		Pierce: m.AffectedByPierce, Explosion: m.ClientExplosion, AlwaysExplode: m.AlwaysExplode,
		ToHit: m.UseAttackRating, CanSlow: m.CanBeSlowed, NextHit: m.Collision.UseCollisionTimer,
		NextDelay: m.Collision.TimerFrames, Size: m.Size,
		SubLoop: m.Animation.HasSubLoop, SubStart: m.Animation.SubStartingFrame, SubStop: m.Animation.SubEndingFrame,
		ExplosionMissile: m.ExplosionMissile, SubMissile: m.SubMissile, HitSubMissile: m.HitSubMissile,
		SkillName: m.SkillName, HitClass: m.HitClass, SrcDam: m.SourceDamage, ResultFlags: m.ResultFlags, HitFlags: m.HitFlags,
		SrvCalc1: m.ServerMovementCalc.Program, DmgCalc1: m.ServerDamageCalc.Program,
	}

	sp.SHitCalc1 = m.ServerCollisionCalc.Program
	sp.ApplyMastery = m.ApplyMastery

	// A missile without a Skill column but with damage columns carries damage
	// of its own (burning ground: meteorfire, immolationfire, molten boulder
	// path): the pipeline builds it from these columns.
	if m.SkillName == "" && (m.ElementalDamage.ElementType != "" || m.Damage.MinDamage > 0 || m.Damage.MaxDamage > 0) {
		ed, pd := m.ElementalDamage.Damage, m.Damage
		sym := func(c d2calculation.CalcString) *d2calc.Program { return d2calc.Compile(string(c), d2calc.KindSkill) }

		sp.Own = &d2skill.DamageSpec{
			HitShift: m.HitShift, SrcDam: m.SourceDamage,
			MinDam: pd.MinDamage, MaxDam: pd.MaxDamage, MinLevDam: pd.MinLevelDamage, MaxLevDam: pd.MaxLevelDamage,
			DmgSymPer: sym(pd.DamageSynergyPerCalc),
			EType:     strings.ToLower(m.ElementalDamage.ElementType),
			EMin:      ed.MinDamage, EMax: ed.MaxDamage, EMinLev: ed.MinLevelDamage, EMaxLev: ed.MaxLevelDamage,
			EDmgSymPer: sym(ed.DamageSynergyPerCalc),
			ELen:       m.ElementalDamage.Duration, ELevLen: m.ElementalDamage.LevelDuration,
		}
	}

	for i, p := range m.ServerMovementCalc.Params {
		if i < len(sp.Param) {
			sp.Param[i] = p.Param
		}
	}

	for i, p := range m.ServerCollisionCalc.Params {
		if i < len(sp.SHitPar) {
			sp.SHitPar[i] = p.Param
		}
	}

	for i, p := range m.ServerDamageCalc.Params {
		if i < len(sp.DParam) {
			sp.DParam[i] = p.Param
		}
	}

	return sp
}

// PipelineSkill converts a skills.txt row for d2skill.
func (s *SkillRecord) PipelineSkill() *d2skill.Skill {
	sk := &d2skill.Skill{
		ID: s.ID, Name: s.Skill, CharClass: s.Charclass,
		InTown: s.InTown, UseManaOnDo: s.Usemanaondo, DecQuant: s.Decquant, Lob: s.Lob, Passive: s.Passive,
		Aura: s.Aura, Progressive: s.Progressive, Kick: s.Kick, NoAmmo: s.Noammo, AttackNoMana: s.AttackNoMana,
		SrvStFunc: s.Srvstfunc, SrvDoFunc: s.Srvdofunc,
		SrvMissile: s.Srvmissile, SrvMissileA: s.Srvmissilea, SrvMissileB: s.Srvmissileb, SrvMissileC: s.Srvmissilec,
		LineOfSight: s.LineOfSight, MaxLvl: s.Maxlvl,
		StartMana: s.Startmana, MinMana: s.Minmana, ManaShift: s.Manashift, Mana: s.Mana, LvlMana: s.Lvlmana,
		Delay: s.Delay, ToHit: s.ToHit, LevToHit: s.LevToHit, ToHitCalc: s.ToHitCalc,
		ResultFlags: s.ResultFlags, HitFlags: s.HitFlags, HitClass: s.HitClass,
		Params:     [9]int{0, s.Param1, s.Param2, s.Param3, s.Param4, s.Param5, s.Param6, s.Param7, s.Param8},
		Calc:       [5]*d2calc.Program{nil, s.Calc1, s.Calc2, s.Calc3, s.Calc4},
		AuraFilter: s.Aurafilter, AuraState: s.Aurastate, AuraTargetState: s.Auratargetstate,
		AuraLenCalc: s.Auralencalc, AuraRangeCalc: s.Aurarangecalc,
		AuraStat: [7]string{"", s.Aurastat1, s.Aurastat2, s.Aurastat3, s.Aurastat4, s.Aurastat5, s.Aurastat6},
		AuraStatCalc: [7]*d2calc.Program{nil, s.Aurastatcalc1, s.Aurastatcalc2, s.Aurastatcalc3, s.Aurastatcalc4,
			s.Aurastatcalc5, s.Aurastatcalc6},
		PassiveState: s.Passivestate,
		PassiveIType: s.Passiveitype,
		IType1:       s.Itypea1,
		PassiveStat:  [6]string{"", s.Passivestat1, s.Passivestat2, s.Passivestat3, s.Passivestat4, s.Passivestat5},
		PassiveCalc: [6]*d2calc.Program{nil, s.Passivecalc1, s.Passivecalc2, s.Passivecalc3, s.Passivecalc4,
			s.Passivecalc5},
		PetMax: s.Petmax, Skpoints: s.Skpoints,
		Summon: s.Summon, PetType: s.Pettype, SumMode: s.Summode,
		SumSkill:     [6]string{"", s.Sumskill1, s.Sumskill2, s.Sumskill3, s.Sumskill4, s.Sumskill5},
		TargetCorpse: s.TargetCorpse, Periodic: s.Periodic, PerDelay: s.Perdelay, Range: s.Range,
	}

	sk.HitShift, sk.SrcDam = s.HitShift, s.SrcDam
	sk.MinDam, sk.MaxDam = s.MinDam, s.MaxDam
	sk.MinLevDam = [5]int{s.MinLevDam1, s.MinLevDam2, s.MinLevDam3, s.MinLevDam4, s.MinLevDam5}
	sk.MaxLevDam = [5]int{s.MaxLevDam1, s.MaxLevDam2, s.MaxLevDam3, s.MaxLevDam4, s.MaxLevDam5}
	sk.DmgSymPer = s.DmgSymPerCalc
	sk.EType = strings.ToLower(s.EType)
	sk.EMin, sk.EMax = s.EMin, s.EMax
	sk.EMinLev = [5]int{s.EMinLev1, s.EMinLev2, s.EMinLev3, s.EMinLev4, s.EMinLev5}
	sk.EMaxLev = [5]int{s.EMaxLev1, s.EMaxLev2, s.EMaxLev3, s.EMaxLev4, s.EMaxLev5}
	sk.EDmgSymPer = s.EDmgSymPerCalc
	sk.ELen = s.ELen
	sk.ELevLen = [3]int{s.ELevLen1, s.ELevLen2, s.ELevLen3}
	sk.ELenSymPer = s.ELenSymPerCalc

	return sk
}
