package d2gamescreen

import (
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2skills"
)

const (
	castTestInterval     = 0.6 // seconds between scripted casts
	castTestDefaultCount = 5
	castTestDefaultLevel = 10
	castTestMoveSeconds  = 0.4
)

// castTest is the state of the OD2_AUTOCAST scenario.
type castTest struct {
	skill    string
	id       int
	count    int
	level    int
	casts    int
	since    float64
	moveAcc  float64
	refused  int
	reported bool
}

// skillEngine returns the skill engine, creating it once the hero and the
// monster director exist.
func (v *Game) skillEngine() *d2skills.Engine {
	if v.skills != nil {
		return v.skills
	}

	if v.localPlayer == nil {
		return nil
	}

	md := v.monsterDirector()
	if md == nil {
		return nil
	}

	scenario := os.Getenv("OD2_AUTOCAST") != ""
	v.skills = d2skills.New(v.asset, v.gameClient.MapEngine, md, v.logLevel, d2skills.Options{
		Seed:         uint32(v.gameClient.MapEngine.Seed()),
		IgnoreTown:   scenario || os.Getenv("OD2_AUTOMONSTER") != "",
		InfiniteAmmo: scenario,
	})

	return v.skills
}

func (v *Game) advanceSkills(elapsed float64) {
	if eng := v.skillEngine(); eng != nil {
		eng.Advance(elapsed)
	}
}

// castWithPipeline runs a skill through the skill pipeline when it supports
// it; false means the caller should use the old client-only cast.
func (v *Game) castWithPipeline(skillID int, tileX, tileY float64) bool {
	eng := v.skillEngine()
	if eng == nil || !eng.Supported(skillID) {
		return false
	}

	eng.Cast(v.localPlayer, skillID, tileX, tileY)

	return true
}

// parseCastTest reads OD2_AUTOCAST=<skill name or id>[,count].
func (v *Game) parseCastTest(eng *d2skills.Engine) *castTest {
	ref := os.Getenv("OD2_AUTOCAST")
	t := &castTest{skill: ref, count: castTestDefaultCount, level: castTestDefaultLevel}

	if i := strings.LastIndex(ref, ","); i >= 0 {
		if n, err := strconv.Atoi(strings.TrimSpace(ref[i+1:])); err == nil && n > 0 {
			t.skill, t.count = ref[:i], n
		}
	}

	if n, err := strconv.Atoi(os.Getenv("OD2_AUTOCAST_LEVEL")); err == nil && n > 0 {
		t.level = n
	}

	t.id = eng.SkillID(t.skill)
	if t.id < 0 {
		v.Errorf("AUTOCAST: unknown skill %q", t.skill)
		v.autoTestExit()

		return t
	}

	// the scenario gives the hero the skill if the class cannot have it
	p := v.localPlayer
	if s := p.Skills[t.id]; s == nil || s.SkillPoints < 1 {
		if p.Skills == nil {
			p.Skills = map[int]*d2hero.HeroSkill{}
		}

		p.Skills[t.id] = &d2hero.HeroSkill{SkillRecord: v.asset.Records.Skill.Details[t.id], SkillPoints: t.level}
		v.Infof("AUTOCAST granted skill=%q level=%d (the hero did not have it)", t.skill, t.level)
	}

	if n, err := strconv.Atoi(os.Getenv("OD2_AUTOCAST_MANA")); err == nil && n > 0 {
		p.Stats.MaxMana, p.Stats.Mana = n, n
	}

	v.Infof("AUTOCAST start skill=%q id=%d level=%d count=%d hero_mana=%s", t.skill, t.id, p.Skills[t.id].SkillPoints,
		t.count, eng.Hero(p).ManaString())

	return t
}

// autoCast implements OD2_AUTOCAST=<skill>[,count]: the hero casts the skill
// at the nearest living monster (OD2_AUTOMONSTER spawns them) every
// castTestInterval seconds; melee range skills first walk up to the monster.
func (v *Game) autoCast(elapsed float64) {
	eng := v.skillEngine()
	if eng == nil {
		return
	}

	t := v.castTestState
	if t == nil {
		t = v.parseCastTest(eng)
		v.castTestState = t
	}

	if t.id < 0 || t.casts >= t.count || v.localPlayer.IsCasting() {
		return
	}

	t.since += elapsed
	if t.since < castTestInterval {
		return
	}

	m := v.nearestMonster()
	if m == nil {
		return
	}

	mx, my := m.SubtilePos()
	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())

	if rec := v.asset.Records.Skill.Details[t.id]; rec != nil && rec.Range == "h2h" &&
		d2monster.Distance(hx-mx, hy-my) > heroMeleeReach {
		t.moveAcc += elapsed
		if t.moveAcc >= castTestMoveSeconds {
			t.moveAcc = 0
			x, y := m.GetPositionF()
			v.OnPlayerMove(x, y)
		}

		return
	}

	t.since = 0
	t.casts++

	if !eng.CastAt(v.localPlayer, t.id, mx, my) {
		t.refused++
	}
}

func (v *Game) nearestMonster() *d2mapentity.Monster {
	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())

	var best *d2mapentity.Monster

	bestDist := 0

	for _, m := range v.monsters.Monsters() {
		if !m.Alive() {
			continue
		}

		mx, my := m.SubtilePos()
		if d := d2monster.Distance(hx-mx, hy-my); best == nil || d < bestDist {
			best, bestDist = m, d
		}
	}

	return best
}

// logCastSummary prints the OD2_AUTOCAST totals once.
func (v *Game) logCastSummary() {
	t := v.castTestState
	if t == nil || t.reported || v.skills == nil {
		return
	}

	t.reported = true
	c := v.skills.Counters
	v.Infof("AUTOCAST summary skill=%q casts=%d refused=%d missiles=%d hits=%d misses=%d walls=%d expired=%d melee=%d "+
		"kills=%d damage=%d mana_spent=%.2f hero_mana=%s", t.skill, c.Casts, c.Refused, c.Missiles, c.Hits, c.Misses,
		c.Walls, c.Expired, c.Melee, c.Kills, c.Damage, float64(c.ManaSpent)/256, v.skills.Hero(v.localPlayer).ManaString())
}
