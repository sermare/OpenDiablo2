package d2gamescreen

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// engineHost runs the OD2_AUTOQUEST scenario against the real game: quest
// events go through the runtime (effects applied, quest log refreshed), talks
// go through the Talk row code, and fights use the monster director.
type engineHost struct{ v *Game }

func (e engineHost) Infof(format string, args ...interface{})  { e.v.Infof(format, args...) }
func (e engineHost) Errorf(format string, args ...interface{}) { e.v.Errorf(format, args...) }
func (e engineHost) Q() *d2quest.Game                          { return e.v.questRT.g }
func (e engineHost) Move(area int)                             { e.v.questArea(area) }
func (e engineHost) Dispatch(ev d2quest.Event)                 { e.v.questDispatch(ev) }
func (e engineHost) Apply(effects []d2quest.Effect) {
	e.v.applyQuestEffects(effects)
	e.v.questRT.dirty = true
}
func (e engineHost) Area() int                     { return e.v.questRT.area }
func (e engineHost) SkillPoints() int              { return e.v.localPlayer.Stats.SkillPoints }
func (e engineHost) HeroLevel() int                { return e.v.localPlayer.Stats.Level }
func (e engineHost) RogueHire() bool               { return e.v.questRT.rogueHire }
func (e engineHost) ImbuePending() bool            { return e.v.questRT.imbuePending }
func (e engineHost) LogText(act, index int) string { return e.v.questLogText(act, index) }
func (e engineHost) Save()                         { _ = e.v.OnPlayerSave() }
func (e engineHost) Exit(pass bool)                { e.v.autoScriptExit(pass) }

func (e engineHost) Tick(frames int) {
	e.v.applyQuestEffects(e.v.questRT.g.Tick(frames))
	e.v.questRT.dirty = true
}

// Talk presses the Talk row of the NPC once.
func (e engineHost) Talk(class int) []int {
	r := e.v.questRT
	before := len(r.spoken)

	if !e.v.questTalkRef(e.v.refForClass(class)) {
		return nil
	}

	return r.spoken[before:]
}

// FoesAlive counts the scripted run's own monsters that still live.
func (e engineHost) FoesAlive() int {
	n := 0

	for m := range e.v.questRT.denFoes {
		if m.Alive() {
			n++
		}
	}

	return n
}

// Fight attacks the nearest scripted monster.
func (e engineHost) Fight() {
	v := e.v
	if v.attackTarget != nil && v.attackTarget.Alive() {
		return
	}

	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())

	var best *d2mapentity.Monster

	bestDist := 0

	for m := range v.questRT.denFoes {
		if !m.Alive() {
			continue
		}

		mx, my := m.SubtilePos()
		if d := d2monster.Distance(hx-mx, hy-my); best == nil || d < bestDist {
			best, bestDist = m, d
		}
	}

	if best != nil {
		v.OnPlayerAttack(best)
	}
}

// SpawnFoes puts a few real monsters next to the hero.
func (e engineHost) SpawnFoes(n int) {
	v := e.v
	d := v.monsterDirector()
	stat := d.FindStat("fallen1")

	if stat == nil {
		stat = d.FindStat("zombie1")
	}

	if stat == nil {
		v.Errorf("AUTOQUEST: no monster to spawn")
		return
	}

	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())

	for i := 0; i < n; i++ {
		m, err := d.SpawnNear(stat, hx+monsterSpawnOffset-4*i, hy+3*i, 2)
		if err != nil {
			v.Errorf("AUTOQUEST: spawn: %v", err)
			continue
		}

		if v.questRT.denFoes == nil {
			v.questRT.denFoes = map[*d2mapentity.Monster]bool{}
		}

		v.questRT.denFoes[m] = true
	}

	v.Infof("AUTOQUEST spawned %d x %s next to the hero", n, stat.Key)
}
