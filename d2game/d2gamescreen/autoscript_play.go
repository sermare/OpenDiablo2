package d2gamescreen

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
)

// The play steps of OD2_AUTOSCRIPT (d2autoscript.PlayHost): walk to exits and
// objects, fight, choose NPC menu rows. They drive the same code paths the
// player's clicks reach.

const (
	killRepathSeconds = 0.4
	// killStuckSeconds is how long the hero may fail to get closer to a target
	// before the target is given up as unreachable.
	killStuckSeconds = 6.0
	// killCastRange is the distance (tiles) up to which a spell is preferred over the staff.
	killCastRange = 12.0
	// killPotionLife is the life fraction below which the hero drinks a belt potion.
	killPotionLife = 0.4
	fireBoltName   = "Fire Bolt"
)

// killState is a running kill: step of the script.
type killState struct {
	radius   float64 // tiles, <= 0: the whole level
	deadline float64
	elapsed  float64
	kills    int
	start    int // monsters alive at the start
	target   *d2mapentity.Monster
	since    float64 // seconds spent on the current target
	bestDist float64
	skip     map[*d2mapentity.Monster]bool
	castAcc  float64
	potions  int
}

// WalkToExit implements d2autoscript.PlayHost.
func (h autoScriptHost) WalkToExit(level int) error { return h.v.walkToExit(level) }

// WalkToObject walks to the nearest object of that name and operates it like a click does.
func (h autoScriptHost) WalkToObject(name string) error {
	ob := h.v.findObject(name)
	if ob == nil {
		return fmt.Errorf("no object %q on this map", name)
	}

	h.v.walkToObject(ob)

	return nil
}

// Menu chooses a row of the open NPC menu by its label or action name.
func (h autoScriptHost) Menu(row string) error {
	menu := h.v.gameControls.NPCMenu
	if !menu.IsOpen() {
		return fmt.Errorf("no NPC menu is open")
	}

	var have []string

	for i, r := range menu.Rows() {
		label := menu.RowLabel(r)
		have = append(have, label)

		if strings.EqualFold(label, row) || strings.EqualFold(r.Action.String(), row) || strings.Contains(strings.ToLower(label), strings.ToLower(row)) {
			h.v.Infof("AUTOSCRIPT menu row %d %q", i, label)
			menu.Choose(i)

			return nil
		}
	}

	return fmt.Errorf("the NPC menu has no row %q (rows: %v)", row, have)
}

// Kill starts a fight.
func (h autoScriptHost) Kill(radius, seconds float64) error {
	v := h.v
	if v.monsterDirector() == nil {
		return fmt.Errorf("no monster director")
	}

	k := &killState{radius: radius, deadline: seconds, skip: map[*d2mapentity.Monster]bool{}}
	k.start = len(v.killCandidates(k))
	v.levels.kill = k

	v.Infof("KILL start radius=%.0f seconds=%.0f candidates=%d level=%d", radius, seconds, k.start, v.currentLevel())

	return nil
}

// killCandidates lists the living hostile monsters the fight may go after.
func (v *Game) killCandidates(k *killState) []*d2mapentity.Monster {
	var out []*d2mapentity.Monster

	hx, hy := v.heroTilePos()

	for _, m := range v.monsters.Monsters() {
		if !m.Alive() || k.skip[m] || m.Stat == nil || !d2monsters.IsHostile(m.Stat) {
			continue
		}

		if k.radius > 0 {
			mx, my := m.GetPositionF()
			if math.Hypot(mx-hx, my-hy) > k.radius {
				continue
			}
		}

		out = append(out, m)
	}

	return out
}

// advanceKill runs the scripted fight: the hero goes for the nearest monster,
// casting Fire Bolt when he has it, and drinks a belt potion when hurt.
func (v *Game) advanceKill(elapsed float64) {
	k := v.levels.kill
	if k == nil || v.monsters == nil || v.localPlayer == nil {
		return
	}

	k.elapsed += elapsed

	if v.localPlayer.IsDead() {
		v.Infof("KILL stopped: the hero died (kills=%d)", k.kills)
		v.levels.kill = nil

		return
	}

	cands := v.killCandidates(k)
	if len(cands) == 0 || k.elapsed >= k.deadline {
		v.Infof("KILL done: kills=%d of %d remaining=%d skipped=%d elapsed=%.1fs potions=%d",
			k.kills, k.start, len(cands), len(k.skip), k.elapsed, k.potions)

		if len(cands) > 0 {
			v.Warningf("KILL ran out of time with %d monster(s) left", len(cands))
		}

		v.levels.kill = nil
		v.attackTarget = nil

		return
	}

	v.drinkIfHurt(k)

	if k.target != nil && !k.target.Alive() {
		k.kills++
		k.target = nil
		v.attackTarget = nil
	}

	hx, hy := v.heroTilePos()

	if k.target == nil {
		best, bd := (*d2mapentity.Monster)(nil), math.MaxFloat64

		for _, m := range cands {
			mx, my := m.GetPositionF()
			if d := math.Hypot(mx-hx, my-hy); d < bd {
				best, bd = m, d
			}
		}

		k.target, k.since, k.bestDist = best, 0, bd
		v.OnPlayerAttack(best)

		return
	}

	mx, my := k.target.GetPositionF()
	dist := math.Hypot(mx-hx, my-hy)

	// give up on a target the hero cannot get closer to
	if dist < k.bestDist-0.5 {
		k.bestDist, k.since = dist, 0
	} else if k.since += elapsed; k.since > killStuckSeconds {
		v.Warningf("KILL giving up on %q at (%.1f,%.1f): the hero cannot get closer than %.1f tiles", k.target.Label(), mx, my, k.bestDist)
		k.skip[k.target] = true
		k.target = nil
		v.attackTarget = nil

		return
	}

	v.castAtTarget(k, dist)

	if v.attackTarget != k.target {
		v.OnPlayerAttack(k.target) // melee takes over again
	}
}

// castAtTarget throws Fire Bolt at a target that is in range but not yet
// in reach, like a sorceress does.
func (v *Game) castAtTarget(k *killState, dist float64) {
	eng := v.skillEngine()
	if eng == nil || dist > killCastRange || dist < 2.5 || v.localPlayer.IsCasting() {
		return
	}

	id := eng.SkillID(fireBoltName)
	if id < 0 {
		return
	}

	if s := v.localPlayer.Skills[id]; s == nil || s.SkillPoints < 1 {
		return
	}

	if v.localPlayer.Stats.Mana < fireBoltManaFloor {
		return
	}

	mx, my := k.target.SubtilePos()
	if eng.CastAt(v.localPlayer, id, mx, my) {
		k.castAcc++
	}
}

// fireBoltManaFloor is a cheap guard; the skill pipeline checks the exact cost.
const fireBoltManaFloor = 3

// drinkIfHurt uses a belt potion when life is low (hotkey columns 1..4).
func (v *Game) drinkIfHurt(k *killState) {
	st := v.localPlayer.Stats
	if st.MaxHealth <= 0 || float64(st.Health) >= killPotionLife*float64(st.MaxHealth) {
		return
	}

	for col := 0; col < 4; col++ {
		if v.gameControls.UseBeltColumnNoSave(col) {
			k.potions++
			v.Infof("KILL drank the belt potion of column %d at %d/%d life", col+1, st.Health, st.MaxHealth)

			return
		}
	}
}

// playBusy reports that a scripted play step still runs.
func (v *Game) playBusy() bool {
	g := v.ground

	return v.levels.kill != nil || g.item != nil || g.chest != nil || g.stash != nil || g.questObj != nil
}

// autoPlayEnabled says whether the world is populated for a scripted playthrough.
func autoPlayPopulate() bool {
	for _, name := range []string{"OD2_AUTOMONSTER", "OD2_AUTOMERC", "OD2_AUTODEATH", "OD2_AUTOQUEST", "OD2_NOPOPULATE"} {
		if os.Getenv(name) != "" {
			return false
		}
	}

	return true
}
