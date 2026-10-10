package d2gamescreen

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2audio"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2skills"
)

const (
	heroEmitter = 1 << 24 // emitter id of the local hero (monsters use their brain ids)

	heroWalkStepSeconds = 0.32 // time between hero footsteps while walking (engine choice)
	heroRunStepSeconds  = 0.22

	// beyond the largest Falloff radius nothing can be heard: skip the call
	farthestAudible = 2000.0
)

// surfaces are the footstep materials of Sounds.txt (light|medium|heavy_walk_<surface>_n).
// SoundEnviron.txt Material 1/2 index them from 1; the order is inferred from
// the rows (Act 2 town 3/4 = outdoor stone/sand, Act 5 snow barricade 6 = snow,
// town 5 = wood) and is UNVERIFIED.
var surfaces = []string{"dirt", "istone", "ostone", "sand", "wood", "snow"}

// weaponSwings maps the hero's weapon class (COF code) to the swing sound.
// Which Sounds.txt row the real game plays per item type is not in the notes
// (UNVERIFIED); these are the rows named after each class.
var weaponSwings = map[string]string{
	"hth": "weapon_punch_1", "1hs": "weapon_1hs_small_1", "1ht": "weapon_1ht_1",
	"2hs": "weapon_2hs_small_1", "2ht": "weapon_2ht_1", "stf": "weapon_staff_1",
	"bow": "weapon_bow_1", "xbw": "weapon_xbow_1",
}

// soundTraceWanted turns on the SOUNDAT lines when an autotest that looks at
// sounds is running.
func soundTraceWanted() bool {
	for _, name := range []string{"OD2_AUTOAMBIENT", "OD2_AUTOMONSTER", "OD2_AUTOGROUND", "OD2_AUTOSOUND_TRACE"} {
		if os.Getenv(name) != "" {
			return true
		}
	}

	return false
}

// advanceSound feeds the sound engine each frame: the hero is the listener,
// the day phase picks the day or night ambience, and the hero's footsteps play.
func (v *Game) advanceSound(elapsed float64) {
	if v.localPlayer == nil {
		return
	}

	if !v.soundTraceSet {
		v.soundTraceSet = true
		v.soundEngine.SetTrace(soundTraceWanted())
	}

	pos := v.localPlayer.GetPosition()
	v.soundEngine.SetListenerSubtile(pos.X(), pos.Y())
	v.soundEnv.SetDayPhase(v.currentDayPhase().Phase())
	v.advanceHeroFootsteps(elapsed)
}

// soundPos converts a map position to sound units.
func soundPos(pos d2vector.Position) (x, y float64) {
	return d2audio.SubtileToSound(pos.X(), pos.Y())
}

// playSoundAtPos plays a Sounds.txt handle at a map position through the
// positional voice bank. Unknown handles are skipped silently.
func (v *Game) playSoundAtPos(handle string, pos d2vector.Position, o d2audio.PlayOpts) {
	v.playSoundAtPosSound(handle, pos, o)
}

// playSoundAtPosSound is playSoundAtPos returning the started sound (nil when
// nothing started).
func (v *Game) playSoundAtPosSound(handle string, pos d2vector.Position, o d2audio.PlayOpts) *d2audio.Sound {
	if handle == "" {
		return nil
	}

	o.X, o.Y = soundPos(pos)

	if lx, ly := v.soundEngine.Listener(); farther(o.X-lx, o.Y-ly) {
		return nil
	}

	return v.soundEngine.PlayHandleAt(handle, o)
}

// farther reports whether a sound-unit delta is beyond every Falloff radius.
func farther(dx, dy float64) bool {
	dy *= 2

	return dx*dx+dy*dy > farthestAudible*farthestAudible
}

// onMonsterSound plays a monster sound from the director at the monster.
func (v *Game) onMonsterSound(ev d2monsters.SoundEvent) {
	v.playSoundAtPos(ev.Handle, d2vector.NewPosition(ev.X, ev.Y),
		d2audio.PlayOpts{Emitter: ev.Emitter, Delay: ev.DelayTicks, Volume: ev.Volume, Kind: ev.Kind, Who: ev.Who})
}

// advanceHeroFootsteps plays the hero's footsteps while the walk or run
// animation is showing: <weight>_walk_<surface>_1 with the surface from the
// area's SoundEnviron Material 1. The armor weight class is not modelled;
// medium is used (UNVERIFIED which class the real game picks).
func (v *Game) advanceHeroFootsteps(elapsed float64) {
	mode := v.localPlayer.GetAnimationMode()

	step := heroWalkStepSeconds

	switch mode {
	case d2enum.PlayerAnimationModeWalk, d2enum.PlayerAnimationModeTownWalk:
	case d2enum.PlayerAnimationModeRun:
		step = heroRunStepSeconds
	default:
		v.heroStepAcc = 0
		return
	}

	v.heroStepAcc += elapsed
	if v.heroStepAcc < step {
		return
	}

	v.heroStepAcc = 0

	handle := heroFootstepHandle("medium", v.soundEnv.Environment().Material1)
	v.playSoundAtPos(handle, v.localPlayer.GetPosition(),
		d2audio.PlayOpts{Emitter: heroEmitter, Hero: true, Kind: "hero-footstep", Who: v.localPlayer.Name()})
}

// heroFootstepHandle names the footstep sound for an armor weight and a
// SoundEnviron Material column (0 or unknown falls back to dirt).
func heroFootstepHandle(weight string, material int) string {
	surface := surfaces[0]
	if material >= 1 && material <= len(surfaces) {
		surface = surfaces[material-1]
	}

	return fmt.Sprintf("%s_walk_%s_1", weight, surface)
}

// playHeroSwing plays the weapon swing of the hero's melee attack.
func (v *Game) playHeroSwing() {
	class := strings.ToLower(v.localPlayer.WeaponClass())

	handle, ok := weaponSwings[class]
	if !ok {
		handle = weaponSwings["hth"]
	}

	v.playSoundAtPos(handle, v.localPlayer.GetPosition(),
		d2audio.PlayOpts{Emitter: heroEmitter, Hero: true, Kind: "hero-swing", Who: v.localPlayer.Name()})
}

// soundSummary describes the sounds the engine traced, for autotest summaries.
func (v *Game) soundSummary() string {
	st := v.soundEngine.TraceStats()

	kinds := make([]string, 0, len(st.ByKind))
	for k, n := range st.ByKind {
		kinds = append(kinds, fmt.Sprintf("%s:%d", k, n))
	}

	sort.Strings(kinds)

	return fmt.Sprintf("sounds=%d audible=%d inaudible=%d nearest=%.0f farthest=%.0f voices=%d kinds=[%s]",
		st.Total, st.Audible, st.Inaudible, st.Nearest, st.Farthest, v.soundEngine.Bank().ActiveVoices(),
		strings.Join(kinds, " "))
}

// onSkillSound plays a skill or missile sound from the skill engine and
// returns the function that stops it (looping travel sounds).
func (v *Game) onSkillSound(ev d2skills.SoundEvent) func() {
	snd := v.playSoundAtPosSound(ev.Handle, d2vector.NewPosition(ev.X, ev.Y),
		d2audio.PlayOpts{Hero: ev.Hero, Kind: ev.Kind, Who: ev.Who})
	if snd == nil {
		return nil
	}

	return snd.Stop
}
