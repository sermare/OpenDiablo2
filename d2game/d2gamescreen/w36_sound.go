package d2gamescreen

import (
	"math/rand"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2audio"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Wave 36 sound wiring: objects, the town portal, the Druid forms. Rules and
// their UNVERIFIED parts are in d2audio/w36_sound.go.

// formSoundRecord is the MonSounds.txt row of the form the hero is in, nil in human shape.
func (v *Game) formSoundRecord() *d2records.MonsterSoundRecord {
	if v.localPlayer == nil || !v.localPlayer.InForm() {
		return nil
	}

	id := d2audio.FormMonSoundsID(v.localPlayer.FormToken())
	if id == "" {
		return nil
	}

	return v.asset.Records.Monster.Sounds[id]
}

// playFormSwing plays the attack sounds of the Werewolf / Werebear form and
// reports whether the hero is in a form (then no weapon swing is played).
func (v *Game) playFormSwing() bool {
	return v.playFormMode(d2monster.ModeAttack1)
}

// playFormMode plays the MonSounds sounds of a hero form entering a mode (VERIFIED wave 44,
// UNITSND_HandleUnitModeSound 0x4c9030 monster branch): A1 / A2 -> UNITSND_PlayHitSoundForUnit 0x4c8150 (the
// Attack1 / Weapon1 fields for A1, the Attack2 / Weapon2 fields otherwise; the vocal obeys its probability and, below
// 100, the 15 tick spacing and the no-speech test; the weapon sound always plays); S1..S4 ->
// UNITSND_PlayMonsterEventSound 0x4c8340 (Skill1..4 fields, row 0xa84 at most once per 25 ticks).
// It reports whether the hero is in a form (then no weapon swing is played).
func (v *Game) playFormMode(mode d2monster.Mode) bool {
	if v.localPlayer == nil || !v.localPlayer.InForm() {
		return false
	}

	rec := v.formSoundRecord()
	if rec == nil {
		return true
	}

	now := v.soundEngine.Ticks()

	prob := rec.Attack1Probability
	if mode == d2monster.ModeAttack2 {
		prob = rec.Attack2Probability
	}

	if (mode == d2monster.ModeAttack1 || mode == d2monster.ModeAttack2) &&
		!d2audio.VocalAllowed(prob, now, v.formSoundTick, v.soundEngine.SpeechPlaying()) {
		rec = withoutVocal(rec, mode)
	}

	indexOf, _ := v.soundIndexLookups()

	// nolint:gosec // sound variation only
	for _, s := range d2monsters.FormAttackSounds(rec, mode, rand.Intn) {
		if s.Kind == "skill" {
			if idx, ok := indexOf(s.Handle); ok && idx == d2audio.ThrottledSkillSoundRow &&
				now-v.heroSoundTick < d2audio.SkillRepeatTicks {
				continue
			}
		}

		v.formSoundTick = now
		v.heroSoundTick = now
		v.Infof("HEROVOICE form-%s mode=%v handle=%s", s.Kind, mode, s.Handle)
		v.playSoundAtPos(s.Handle, v.localPlayer.GetPosition(), d2audio.PlayOpts{
			Emitter: heroEmitter, Hero: true, Kind: "form-" + s.Kind, Who: v.localPlayer.Name(),
			Delay: s.DelayTicks, Volume: s.Volume,
		})
	}

	return true
}

// withoutVocal copies a MonSounds row with the vocal of the attack mode removed.
func withoutVocal(r *d2records.MonsterSoundRecord, mode d2monster.Mode) *d2records.MonsterSoundRecord {
	c := *r

	if mode == d2monster.ModeAttack2 {
		c.Attack2 = ""
	} else {
		c.Attack1 = ""
	}

	return &c
}

// playHeroAnimSounds is the skill engine's hook for the mode a hero skill enters: the form sounds by mode, or the
// weapon swing for the A1 / A2 modes of a human hero (the plain Attack skill 0 swings through advanceHeroAttack).
func (v *Game) playHeroAnimSounds(skillID int, mode d2enum.PlayerAnimationMode) {
	if v.localPlayer == nil {
		return
	}

	var m d2monster.Mode

	switch mode {
	case d2enum.PlayerAnimationModeAttack1:
		m = d2monster.ModeAttack1
	case d2enum.PlayerAnimationModeAttack2:
		m = d2monster.ModeAttack2
	case d2enum.PlayerAnimationModeSkill1:
		m = d2monster.ModeSkill1
	case d2enum.PlayerAnimationModeSkill2:
		m = d2monster.ModeSkill2
	case d2enum.PlayerAnimationModeSkill3:
		m = d2monster.ModeSkill3
	case d2enum.PlayerAnimationModeSkill4:
		m = d2monster.ModeSkill4
	case d2enum.PlayerAnimationModeKick:
		v.playHeroKick()

		return
	default:
		return
	}

	if v.playFormMode(m) {
		return
	}

	if skillID != 0 && (m == d2monster.ModeAttack1 || m == d2monster.ModeAttack2) {
		v.playWeaponSwing()

		return
	}

	// the Barbarian's S3 and the Assassin's S4 modes swing the weapon like an attack (VERIFIED w48,
	// UNITSND_HandleUnitModeSound 0x4c9030 modes 0xf and 0x10 for classes 4 and 6)
	if (m == d2monster.ModeSkill3 || m == d2monster.ModeSkill4) &&
		d2audio.SwingInSkillMode(skillModeNumber(m), v.localPlayer.ClassNumber()) {
		v.playWeaponSwing()
	}
}

// skillModeNumber is 3 for S3 and 4 for S4 (0 for any other mode).
func skillModeNumber(m d2monster.Mode) int {
	switch m {
	case d2monster.ModeSkill3:
		return 3
	case d2monster.ModeSkill4:
		return 4
	}

	return 0
}

// playHeroKick plays the kick sound of the KK mode (row 0xff; the Assassin delays it 4 ticks, VERIFIED w48).
func (v *Game) playHeroKick() {
	if v.localPlayer == nil || v.localPlayer.InForm() {
		return
	}

	v.playSoundAtPos(d2audio.KickHandle, v.localPlayer.GetPosition(), d2audio.PlayOpts{
		Emitter: heroEmitter, Hero: true, Kind: "hero-kick", Who: v.localPlayer.Name(),
		Delay: d2audio.KickDelay(v.localPlayer.ClassNumber()),
	})
}

// onHeroBlockMode plays the block sound when the local hero's block animation starts: row 0x18e for a weapon hit
// class 1..9 (an unarmed hero included), row 0x191 for bows and crossbows (VERIFIED w48, UNITSND_GetHitClassSoundId
// 0x4c7310). Which item's hit class the exe reads is UNVERIFIED (the right hand weapon here).
func (v *Game) onHeroBlockMode(p *d2mapentity.Player) {
	if p != v.localPlayer || v.localPlayer == nil || v.localPlayer.InForm() {
		return
	}

	hitClass := ""

	if eq := v.localPlayer.Equipment; eq != nil && eq.RightHand != nil {
		if rec := v.asset.Records.Item.Weapons[eq.RightHand.GetItemCode()]; rec != nil {
			hitClass = rec.HitClass
		}
	}

	h := d2audio.BlockHandle(hitClass)
	if h == "" {
		return
	}

	v.Infof("HEROVOICE block handle=%s hitclass=%q", h, hitClass)
	v.heroSoundTick = v.soundEngine.Ticks()
	v.playSoundAtPos(h, v.localPlayer.GetPosition(),
		d2audio.PlayOpts{Emitter: heroEmitter, Hero: true, Kind: "hero-block", Who: v.localPlayer.Name()})
}

// playObjectSound plays a Sounds.txt handle at an object.
func (v *Game) playObjectSound(handle string, ob *d2mapentity.Object, kind string) {
	if handle == "" || ob == nil {
		return
	}

	v.playSoundAtPos(handle, ob.GetPosition(), d2audio.PlayOpts{Kind: kind, Who: ob.Label()})
}

// playHeroSound plays a Sounds.txt handle at the hero (the town portal cast).
func (v *Game) playHeroSound(handle, kind string) {
	if handle == "" || v.localPlayer == nil {
		return
	}

	v.playSoundAtPos(handle, v.localPlayer.GetPosition(),
		d2audio.PlayOpts{Emitter: heroEmitter, Hero: true, Kind: kind, Who: v.localPlayer.Name()})
}
