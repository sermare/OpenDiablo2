package d2gamescreen

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2audio"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// heroClassPrefix is the Sounds.txt voice prefix of the local hero.
func (v *Game) heroClassPrefix() string {
	if v.localPlayer == nil {
		return ""
	}

	code := d2hero.ClassCodeOfHero(v.localPlayer.Class)
	if code == "" {
		return ""
	}

	for _, p := range d2audio.HeroClassPrefixes {
		if strings.HasPrefix(p, code) {
			return p
		}
	}

	return ""
}

// soundIndexLookups returns the handle<->index lookups of Sounds.txt.
func (v *Game) soundIndexLookups() (func(string) (int, bool), func(int) string) {
	byIndex := map[int]string{}
	for h, e := range v.asset.Records.Sound.Details {
		byIndex[e.Index] = h
	}

	return func(h string) (int, bool) {
			e, ok := v.asset.Records.Sound.Details[h]
			if !ok {
				return 0, false
			}

			return e.Index, true
		}, func(i int) string {
			return byIndex[i]
		}
}

// sayHero plays hero voice id (d2audio.Voice*) with the exe's suppression
// rules (UNITSND_PlayUnitSoundById 0x4c8470). "A unit sound is already playing" is the exe's
// UNITSND_HasPlayingEntry: any speech entry (ids 0xb76..0x1230) holds a voice (VERIFIED, wave 37).
// The quest lines that come with music (d2audio.QuestPlans) also stop the level music for a while and
// play their sting (DRAWLIST_AddSoundLinkedEffect 0x4d9b20).
func (v *Game) sayHero(id int) {
	prefix := v.heroClassPrefix()
	if prefix == "" || v.localPlayer == nil {
		return
	}

	dead := v.death.dead
	if !v.heroVoiceGate.Allow(id, v.soundEngine.Ticks(), v.soundEngine.SpeechPlaying(), dead) {
		v.Infof("HEROVOICE id=%#x suppressed", id)
		return
	}

	indexOf, handleAt := v.soundIndexLookups()

	h, ok := d2audio.VoiceHandle(prefix, id, indexOf, handleAt)
	if !ok {
		return
	}

	delay := d2audio.VoiceDelay(id)

	if plan, isPlan := d2audio.QuestPlans[id]; isPlan {
		sting, _ := indexOf(plan.Sting)
		v.soundEnv.QuestMusic(plan, sting)
		v.Infof("HEROVOICE quest music id=%#x sting=%s silence=%d line=%v in %d ticks", id, plan.Sting, plan.Silence,
			plan.Line, plan.LineDelay)

		if !plan.Line {
			return
		}

		delay = plan.LineDelay
	}

	v.Infof("HEROVOICE id=%#x handle=%s tick=%d", id, h, v.soundEngine.Ticks())
	v.heroSoundTick = v.soundEngine.Ticks()
	v.playSoundAtPos(h, v.localPlayer.GetPosition(), d2audio.PlayOpts{
		Emitter: heroEmitter, Hero: true, Kind: "hero-voice", Who: v.localPlayer.Name(), Delay: delay,
	})
}

// sayHeroVocal plays the hit (slot 0) or death (slot 1) vocal of the class record. The hit grunt plays
// when the hero starts a hit recovery and no speech entry is playing (UNITSND_PlayUnitModeStartSound
// 0x4c8e90, VERIFIED, wave 37): there is no time spacing in the exe.
func (v *Game) sayHeroVocal(death bool) {
	prefix := v.heroClassPrefix()
	if prefix == "" || v.localPlayer == nil {
		return
	}

	h, kind := prefix+"_hit_1", "hit"

	if death {
		h, kind = prefix+"_death_1", "death"

		// VERIFIED wave 40: a shapeshifted hero is a monster for the sound code, and the monster
		// branch of UNITSND_PlayUnitModeStartSound 0x4c8e90 plays the MonSounds DeathSound (record
		// +0x3c, delay +0x40) on mode 0, so the form dies with the wolf/bear cry, not the class one.
		if rec := v.formSoundRecord(); rec != nil {
			if h = rec.DeathSound; h == "" {
				return
			}

			kind = "form-death"
		}
	} else {
		if !d2audio.HitGruntPlays(v.soundEngine.SpeechPlaying(), v.death.dead) {
			return
		}

		if rec := v.formSoundRecord(); rec != nil {
			// the form's own hit sound (a shapeshifted hero is a monster for the sound code, VERIFIED);
			// a form row without one is silent
			if h = rec.HitSound; h == "" {
				return
			}

			kind = "form-hit"
		}
	}

	v.Infof("HEROVOICE %s handle=%s", kind, h)
	v.heroSoundTick = v.soundEngine.Ticks()

	opts := d2audio.PlayOpts{Emitter: heroEmitter, Hero: true, Kind: "hero-voice", Who: v.localPlayer.Name()}

	if rec := v.formSoundRecord(); rec != nil {
		// a form is a monster for the sound code: its sounds set the global monster sound tick (DAT_007c0944)
		// and carry the MonSounds delay (+0x38 hit, +0x40 death), UNITSND_PlayUnitModeStartSound 0x4c8e90
		v.formSoundTick = v.soundEngine.Ticks()
		opts.Delay = rec.HitDelay

		if death {
			opts.Delay = rec.DeaDelay
		}
	} else if death {
		opts.Delay = 1 // the player branch passes delay 1 for death and 2 for a hit (same function)
	} else {
		opts.Delay = 2
	}

	v.playSoundAtPos(h, v.localPlayer.GetPosition(), opts)
}

// onSkillRefused maps a refused cast to the hero's complaint (d2audio.RefusalVoice: needmana, impossible,
// not in town; the other refusals are silent, VERIFIED in PLAYER_PlayCastFailSound 0x45c830).
func (v *Game) onSkillRefused(reason string) {
	if id := d2audio.RefusalVoice(reason); id != 0 {
		v.sayHero(id)
	}
}
