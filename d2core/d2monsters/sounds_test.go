package d2monsters

import (
	"fmt"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// fallen mirrors the real MonSounds.txt row of the Fallen.
func fallenSounds() *d2records.MonsterSoundRecord {
	return &d2records.MonsterSoundRecord{
		ID: "fallen", Attack1: "fallen_attack_1", Weapon1: "weapon_punch_1", Weapon1Delay: 5,
		Attack1Probability: 60, Weapon1Volume: 160,
		Attack2: "fallen_attack_1", Weapon2: "weapon_punch_1", Weapon2Delay: 5, Attack2Probability: 60, Weapon2Volume: 160,
		HitSound: "fallen_hit_1", DeathSound: "fallen_death_1", HitDelay: 2, DeaDelay: 1,
		Skill2: "fallen_warcry_1", Footstep: "light_walk_dirt_1", FootstepCount: 2, FootstepProbability: 100,
		Neutral: "fallen_neutral_1", NeutralTime: 300, Taunt: "fallen_warcry_1",
	}
}

func fixedRoll(n int) func(int) int { return func(m int) int { return n % m } }

func plansString(p []soundPlan) string {
	s := ""
	for _, x := range p {
		s += fmt.Sprintf("%s:%s@%d/%d ", x.kind, x.handle, x.delay, x.vol)
	}

	return s
}

func TestAttackPlans(t *testing.T) {
	skeleton := &d2records.MonsterSoundRecord{Weapon1: "weapon_1hs_large_1", Weapon1Delay: 6, Weapon1Volume: 190, Skill1: "skeleton_raise_1"}

	tests := []struct {
		name string
		rec  *d2records.MonsterSoundRecord
		mode d2monster.Mode
		roll int
		want string
	}{
		{"vocal and swing", fallenSounds(), d2monster.ModeAttack1, 10, "attack:fallen_attack_1@0/0 weapon:weapon_punch_1@5/160 "},
		{"vocal skipped by probability", fallenSounds(), d2monster.ModeAttack1, 75, "weapon:weapon_punch_1@5/160 "},
		{"attack 2", fallenSounds(), d2monster.ModeAttack2, 0, "attack:fallen_attack_1@0/0 weapon:weapon_punch_1@5/160 "},
		{"no vocal column", skeleton, d2monster.ModeAttack1, 0, "weapon:weapon_1hs_large_1@6/190 "},
		{"skill 1", skeleton, d2monster.ModeSkill1, 0, "skill:skeleton_raise_1@0/0 "},
		{"skill without sound", skeleton, d2monster.ModeSkill3, 0, ""},
		{"no record", nil, d2monster.ModeAttack1, 0, ""},
		{"walk makes no attack sound", fallenSounds(), d2monster.ModeWalk, 0, ""},
	}

	for _, tc := range tests {
		if got := plansString(attackPlans(tc.rec, tc.mode, fixedRoll(tc.roll))); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestHitDeathTauntPlans(t *testing.T) {
	r := fallenSounds()

	if got := plansString(hitPlans(r)); got != "hit:fallen_hit_1@2/0 " {
		t.Errorf("hit: %q", got)
	}

	if got := plansString(deathPlans(r)); got != "death:fallen_death_1@1/0 " {
		t.Errorf("death: %q", got)
	}

	if got := plansString(tauntPlans(r)); got != "taunt:fallen_warcry_1@0/0 " {
		t.Errorf("taunt: %q", got)
	}

	if hitPlans(nil) != nil || deathPlans(&d2records.MonsterSoundRecord{}) != nil || tauntPlans(nil) != nil {
		t.Error("missing columns must produce no sound")
	}
}

func TestNeutralTiming(t *testing.T) {
	r := fallenSounds()

	lo, hi := nextNeutral(r, fixedRoll(0)), nextNeutral(r, func(n int) int { return n - 1 })
	if lo != 150 || hi != 449 {
		t.Errorf("neutral wait range %d..%d, want 150..449", lo, hi)
	}

	if nextNeutral(&d2records.MonsterSoundRecord{Neutral: "x"}, fixedRoll(1)) != 0 {
		t.Error("NeuTime 0 must mean no idle sound")
	}

	if got := plansString(neutralPlans(r)); got != "neutral:fallen_neutral_1@0/0 " {
		t.Errorf("neutral: %q", got)
	}
}

func TestFootsteps(t *testing.T) {
	r := fallenSounds()
	r.FootstepLayer = "walk_flesh_1"

	if got := plansString(footstepPlans(r, fixedRoll(50))); got != "footstep:light_walk_dirt_1@0/0 footstep:walk_flesh_1@0/0 " {
		t.Errorf("footstep: %q", got)
	}

	r.FootstepProbability = 40
	if footstepPlans(r, fixedRoll(50)) != nil {
		t.Error("roll 50 must miss a 40% footstep")
	}

	tests := []struct{ frames, count, want int }{{12, 2, 6}, {8, 2, 4}, {4, 2, 3}, {12, 3, 4}}
	for _, tc := range tests {
		rec := &d2records.MonsterSoundRecord{FootstepCount: tc.count}
		if got := footstepPeriod(rec, tc.frames); got != tc.want {
			t.Errorf("period(%d frames, %d steps) = %d, want %d", tc.frames, tc.count, got, tc.want)
		}
	}

	if footstepPeriod(&d2records.MonsterSoundRecord{}, 12) != 0 {
		t.Error("FsCnt 0 means no footsteps")
	}
}
