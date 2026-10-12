package d2gamescreen

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2audio"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2skills"
)

const (
	heroEmitter = 1 << 24 // emitter id of the local hero (monsters use their brain ids)

	// beyond the largest Falloff radius nothing can be heard: skip the call
	farthestAudible = 2000.0
)

// surfaces are the footstep materials of Sounds.txt (light|medium|heavy_walk_<surface>_n).
// the exe indexes them from 1 (a Sounds.txt row offset of 4 rows per surface, UNITSND_GetStepSoundId 0x4c7970,
// VERIFIED against the rows): dirt, istone, ostone, sand, snow, wood. SoundEnviron Material 1 is the fallback index.
var surfaces = []string{"dirt", "istone", "ostone", "sand", "snow", "wood"}

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
	v.soundEngine.SetPanelOpen(v.gameControls != nil && v.gameControls.IsEscapeMenuOpen())
	v.advanceHeroFootsteps(elapsed)
	v.advanceAreaVoice()
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
	handle := ev.Handle
	if handle == "" && ev.ID != 0 {
		handle = v.soundHandle(ev.ID)
	}

	pos := d2vector.NewPosition(ev.X, ev.Y)
	opts := d2audio.PlayOpts{Emitter: ev.Emitter, Delay: ev.DelayTicks, Volume: ev.Volume, Kind: ev.Kind, Who: ev.Who}

	// The Init vocal is often a loop (fire golem, hydra, swarm...): the exe removes the unit's
	// sounds when it dies (UNITSND_RemoveModeSoundsForUnit 0x4c7d90), so keep it to stop it then.
	if ev.Kind == "init" {
		if snd := v.playSoundAtPosSound(handle, pos, opts); snd != nil {
			if v.initSounds == nil {
				v.initSounds = map[int]*d2audio.Sound{}
			}

			v.initSounds[ev.Emitter] = snd
		}

		return
	}

	if ev.Kind == "death" {
		if snd := v.initSounds[ev.Emitter]; snd != nil {
			snd.Stop()
			delete(v.initSounds, ev.Emitter)
		}
	}

	v.playSoundAtPos(handle, pos, opts)
}

// advanceHeroFootsteps plays the hero's footsteps while the walk or run
// animation is showing: <weight>_walk|run_<surface>_1, the weight by hero class and the surface by the floor tile
// (see footstepSurface). The steps fall on animation frames like the exe's (UNITSND_PlayModeEndSound 0x4c7a20,
// VERIFIED wave 48, d2audio.FootstepTracker): 2 per cycle for a hero, FsCnt / FsOff of the MonSounds row for a form.
func (v *Game) advanceHeroFootsteps(_ float64) {
	mode := v.localPlayer.GetAnimationMode()

	switch mode {
	case d2enum.PlayerAnimationModeWalk, d2enum.PlayerAnimationModeTownWalk, d2enum.PlayerAnimationModeRun:
	default:
		v.heroSteps.Reset()
		return
	}

	steps, offset := 2, 0

	rec := v.formSoundRecord()
	if rec != nil {
		steps, offset = rec.FootstepCount, rec.FootstepOffset
	}

	frame, count := v.localPlayer.AnimationFrame()
	if count < 1 || !v.heroSteps.Observe(frame, count, steps, offset) {
		return
	}

	run := mode == d2enum.PlayerAnimationModeRun
	surface := v.footstepSurface()

	if rec != nil {
		v.playFormFootstep(rec, surface, run)
		return
	}

	handle := heroFootstepHandle(d2audio.HeroFootstepWeight(v.heroClassPrefix()), surface, run)
	v.playSoundAtPos(handle, v.localPlayer.GetPosition(),
		d2audio.PlayOpts{Emitter: heroEmitter, Hero: true, Kind: "hero-footstep", Who: v.localPlayer.Name()})
}

// footstepSurface is the surface index (1..6) of the ground under the hero: the floor tile's material flags, else the
// area's SoundEnviron Material 1 when it is 1..6, else 1 (UNITSND_GetFootstepSoundIndex 0x4c7870, VERIFIED order).
func (v *Game) footstepSurface() int {
	if v.gameClient != nil && v.gameClient.MapEngine != nil && v.localPlayer != nil {
		pos := v.localPlayer.Position
		if s := v.gameClient.MapEngine.FloorSurface(int(pos.X()), int(pos.Y())); s != 0 {
			return s
		}
	}

	if m := v.soundEnv.Environment().Material1; m >= 1 && m <= len(surfaces) {
		return m
	}

	return 1
}

// playFormFootstep plays the footstep of a hero form (a monster for the sound code): the MonSounds Footstep row plus
// 4 rows per surface, plus 24 rows for the run set (UNITSND_GetStepSoundId 0x4c7970, VERIFIED), then the footstep
// layer with its probability (UNITSND_PlayModeEndSound 0x4c7a20).
func (v *Game) playFormFootstep(rec *d2records.MonsterSoundRecord, surface int, run bool) {
	if rec.Footstep == "" {
		return
	}

	indexOf, handleAt := v.soundIndexLookups()

	base, ok := indexOf(rec.Footstep)
	if !ok {
		return
	}

	base += 4 * (surface - 1)
	if run {
		base += 24
	}

	if h := handleAt(base); h != "" {
		v.playSoundAtPos(h, v.localPlayer.GetPosition(),
			d2audio.PlayOpts{Emitter: heroEmitter, Hero: true, Kind: "form-footstep", Who: v.localPlayer.Name()})
	}

	// nolint:gosec // sound variation only
	if rec.FootstepLayer != "" && rand.Intn(100) < rec.FootstepProbability {
		v.playSoundAtPos(rec.FootstepLayer, v.localPlayer.GetPosition(),
			d2audio.PlayOpts{Emitter: heroEmitter, Hero: true, Kind: "form-footstep-layer", Who: v.localPlayer.Name()})
	}
}

// heroFootstepHandle names the footstep sound of a class weight (d2audio.HeroFootstepWeight) on a surface
// index 1..6 (anything else is dirt), walking or running.
func heroFootstepHandle(weight string, surface int, run bool) string {
	name := surfaces[0]
	if surface >= 1 && surface <= len(surfaces) {
		name = surfaces[surface-1]
	}

	gait := "walk"
	if run {
		gait = "run"
	}

	return fmt.Sprintf("%s_%s_%s_1", weight, gait, name)
}

// playHeroSwing plays the weapon swing of the hero's melee attack.
func (v *Game) playHeroSwing() {
	v.heroSoundTick = v.soundEngine.Ticks()

	if v.playFormSwing() { // the Werewolf / Werebear attack sounds replace the weapon swing
		return
	}

	v.playWeaponSwing()
}

// playWeaponSwing plays the swing of the right hand weapon's hit class (d2audio.SwingHandle: the exe table at
// 0x726bc8 indexed by weapons.txt "hit class"); a weapon whose class is not known falls back to the COF weapon class.
func (v *Game) playWeaponSwing() {
	hitClass := ""

	if eq := v.localPlayer.Equipment; eq != nil && eq.RightHand != nil {
		if rec := v.asset.Records.Item.Weapons[eq.RightHand.GetItemCode()]; rec != nil {
			hitClass = rec.HitClass
		}
	}

	handle, known := d2audio.SwingHandle(hitClass)
	if !known {
		var ok bool

		if handle, ok = weaponSwings[strings.ToLower(v.localPlayer.WeaponClass())]; !ok {
			handle = weaponSwings["hth"]
		}
	}

	if handle == "" {
		return
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
