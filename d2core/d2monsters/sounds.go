package d2monsters

import (
	"math/rand"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// SoundEvent is one Sounds.txt sound a monster wants played, positioned at the
// monster. The director only decides which sound and when (from MonSounds.txt
// via the MonSound column of monstats.txt); the owner plays it, for example
// through d2audio.SoundEngine.PlayHandleAt.
type SoundEvent struct {
	// Kind is attack, weapon, skill, hit, death, taunt, neutral or footstep.
	Kind string
	// Handle is a Sounds.txt handle (group variants are picked by the sound bank).
	Handle string
	// X, Y is the monster position in map subtiles.
	X, Y float64
	// Emitter is the brain id, for Defer/Stop Inst.
	Emitter int
	// DelayTicks is the MonSounds.txt delay in game frames (25/s).
	DelayTicks int
	// Volume overrides the Sounds.txt volume when > 0 (Wea1Vol / Wea2Vol).
	Volume int
	// Who names the monster, for logs.
	Who string
}

// soundPlan is a SoundEvent before it is positioned.
type soundPlan struct {
	kind, handle string
	delay, vol   int
}

// attackPlans returns the sounds of an attack or skill animation.
//
// MonSounds.txt columns (semantics from the d2mods documentation, UNVERIFIED
// in the binary): AttackN is a vocal played with probability AttNPrb percent
// after AttNDel frames; WeaponN is the swing/impact sound, always played after
// WeaNDel frames, with WeaNVol replacing its volume. SkillN plays for the
// monster's SN animation.
func attackPlans(r *d2records.MonsterSoundRecord, mode d2monster.Mode, rnd func(int) int) []soundPlan {
	if r == nil {
		return nil
	}

	var out []soundPlan

	add := func(kind, handle string, delay, vol int) {
		if handle != "" {
			out = append(out, soundPlan{kind, handle, delay, vol})
		}
	}

	switch mode {
	case d2monster.ModeAttack1:
		if rnd(100) < r.Attack1Probability {
			add("attack", r.Attack1, r.Attack1Delay, 0)
		}

		add("weapon", r.Weapon1, r.Weapon1Delay, r.Weapon1Volume)
	case d2monster.ModeAttack2:
		if rnd(100) < r.Attack2Probability {
			add("attack", r.Attack2, r.Attack2Delay, 0)
		}

		add("weapon", r.Weapon2, r.Weapon2Delay, r.Weapon2Volume)
	case d2monster.ModeSkill1, d2monster.ModeCast:
		add("skill", r.Skill1, 0, 0)
	case d2monster.ModeSkill2:
		add("skill", r.Skill2, 0, 0)
	case d2monster.ModeSkill3:
		add("skill", r.Skill3, 0, 0)
	case d2monster.ModeSkill4:
		add("skill", r.Skill4, 0, 0)
	}

	return out
}

func hitPlans(r *d2records.MonsterSoundRecord) []soundPlan {
	if r == nil || r.HitSound == "" {
		return nil
	}

	return []soundPlan{{"hit", r.HitSound, r.HitDelay, 0}}
}

func deathPlans(r *d2records.MonsterSoundRecord) []soundPlan {
	if r == nil || r.DeathSound == "" {
		return nil
	}

	return []soundPlan{{"death", r.DeathSound, r.DeaDelay, 0}}
}

func tauntPlans(r *d2records.MonsterSoundRecord) []soundPlan {
	if r == nil || r.Taunt == "" {
		return nil
	}

	return []soundPlan{{"taunt", r.Taunt, 0, 0}}
}

// neutralPlans is the idle vocal; NeuTime says how often (see nextNeutral).
func neutralPlans(r *d2records.MonsterSoundRecord) []soundPlan {
	if r == nil || r.Neutral == "" {
		return nil
	}

	return []soundPlan{{"neutral", r.Neutral, 0, 0}}
}

// nextNeutral is the wait in frames before the next idle vocal: NeuTime frames
// nominally, randomised to [NeuTime/2, 3*NeuTime/2) (the original's spread is
// UNVERIFIED). 0 means the monster has none.
func nextNeutral(r *d2records.MonsterSoundRecord, rnd func(int) int) int {
	if r == nil || r.Neutral == "" || r.NeutralTime <= 0 {
		return 0
	}

	return r.NeutralTime/2 + rnd(r.NeutralTime)
}

// footstepPlans is one footstep: Footstep and FootstepLayer together, with
// probability FsPrb percent.
func footstepPlans(r *d2records.MonsterSoundRecord, rnd func(int) int) []soundPlan {
	if r == nil || (r.Footstep == "" && r.FootstepLayer == "") || r.FootstepCount <= 0 ||
		rnd(100) >= r.FootstepProbability {
		return nil
	}

	var out []soundPlan

	for _, h := range []string{r.Footstep, r.FootstepLayer} {
		if h != "" {
			out = append(out, soundPlan{"footstep", h, 0, 0})
		}
	}

	return out
}

// footstepPeriod is the frames between footsteps: FsCnt steps per walk
// animation of animFrames frames, at one animation frame per game frame
// (UNVERIFIED), never faster than every 3 frames.
func footstepPeriod(r *d2records.MonsterSoundRecord, animFrames int) int {
	if r == nil || r.FootstepCount <= 0 {
		return 0
	}

	p := animFrames / r.FootstepCount
	if p < 3 {
		p = 3
	}

	return p
}

// soundRecord returns the MonSounds.txt row of a monster.
func (d *Director) soundRecord(u *unit) *d2records.MonsterSoundRecord {
	if u.m.Stat == nil {
		return nil
	}

	return d.asset.Records.Monster.Sounds[u.m.Stat.SoundKeyNormal]
}

// playPlans positions the plans at the monster and hands them to Options.OnSound.
func (d *Director) playPlans(u *unit, plans []soundPlan) {
	if d.opt.OnSound == nil {
		return
	}

	pos := u.m.GetPosition()

	for _, p := range plans {
		d.opt.OnSound(SoundEvent{Kind: p.kind, Handle: p.handle, X: pos.X(), Y: pos.Y(),
			Emitter: int(u.b.ID), DelayTicks: p.delay, Volume: p.vol, Who: u.m.Label()})
	}
}

// ambientSounds runs the per-frame idle and footstep sounds of a living monster.
func (d *Director) ambientSounds(u *unit) {
	if d.opt.OnSound == nil || !u.m.Alive() {
		return
	}

	rec := d.soundRecord(u)
	if rec == nil {
		return
	}

	mode := u.m.Mode()

	if mode == d2monster.ModeWalk || mode == d2monster.ModeRun {
		if period := footstepPeriod(rec, u.m.AnimationFrames()); period > 0 {
			if u.nextStep == 0 {
				u.nextStep = d.frame + rec.FootstepOffset + period
			}

			if d.frame >= u.nextStep {
				u.nextStep = d.frame + period
				d.playPlans(u, footstepPlans(rec, d.snd.Intn))
			}
		}
	} else {
		u.nextStep = 0
	}

	if mode == d2monster.ModeNeutral || mode == d2monster.ModeWalk || mode == d2monster.ModeRun {
		if u.nextIdle == 0 {
			u.nextIdle = d.frame + nextNeutral(rec, d.snd.Intn)
		}

		if u.nextIdle > 0 && d.frame >= u.nextIdle {
			u.nextIdle = d.frame + nextNeutral(rec, d.snd.Intn)
			d.playPlans(u, neutralPlans(rec))
		}
	}
}

func newSoundRand(seed uint32) *rand.Rand {
	// nolint:gosec // sound variation only, never feeds gameplay
	return rand.New(rand.NewSource(int64(seed) + 0x5d2))
}
