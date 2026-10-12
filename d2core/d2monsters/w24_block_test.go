package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Wave 24: the monster shield block (Director.MonsterBlock) for every hit that goes through the attack outcome.

func blockUnit(class, block int, seed uint32) *unit {
	m := &d2mapentity.Monster{Stat: &d2records.MonStatRecord{ID: class}, StatEx: &d2records.MonStat2Record{}}
	m.Vitals.Block = block
	m.Vitals.HP, m.Vitals.MaxHP = 50, 50

	b := &d2monster.Brain{ID: 1, Mode: d2monster.ModeNeutral, Seed: d2rand.New(seed)}

	return &unit{m: m, b: b}
}

func blockDirector(u *unit) *Director {
	d := testDirector(u)
	d.byEntity[u.m.ID()] = u

	return d
}

func TestMonsterBlockChanceAndSeed(t *testing.T) {
	// chance 0: no block and no seed step (COMBAT_RollShieldBlock returns early below 1)
	u := blockUnit(1, 0, 7)
	ref := d2rand.New(7)

	d := blockDirector(u)
	if d.MonsterBlock(u.m, false) || u.b.Seed.Step() != ref.Step() {
		t.Fatal("chance 0 must neither block nor consume a step")
	}

	// chance 75 (the cap): over many rolls about 75 percent block, and each roll takes one step of the unit's seed
	u = blockUnit(1, 75, 99)
	d = blockDirector(u)
	ref = d2rand.New(99)
	got, want := 0, 0

	for i := 0; i < 2000; i++ {
		if d.MonsterBlock(u.m, i%2 == 0) {
			got++
		}

		if ref.Roll(100) < 75 {
			want++
		}
	}

	if got != want {
		t.Fatalf("blocks %d, the defender's own seed predicts %d", got, want)
	}

	if got < 1400 || got > 1600 {
		t.Fatalf("75 percent chance blocked %d of 2000", got)
	}

	if d.Counters.MonsterBlocks != got {
		t.Fatalf("counter %d != %d", d.Counters.MonsterBlocks, got)
	}
}

// The classes without a BL animation (0xf3, 0x14d, 0x2c1) and classes lacking the mode block silently; a dying
// monster never blocks; a monster outside the director never blocks.
func TestMonsterBlockSilentAndDead(t *testing.T) {
	for _, class := range []int{0xf3, 0x14d, 0x2c1, 5} { // class 5: StatEx has no mBL
		u := blockUnit(class, 100, 3)
		d := blockDirector(u)

		if !d.MonsterBlock(u.m, false) {
			t.Fatalf("class %#x: chance 100 must block", class)
		}

		if u.b.Mode == d2monster.ModeBlock || d.Counters.BlockAnimsMonster != 0 {
			t.Fatalf("class %#x: no block animation expected", class)
		}
	}

	u := blockUnit(5, 100, 3)
	u.b.Mode = d2monster.ModeDying

	if blockDirector(u).MonsterBlock(u.m, false) {
		t.Fatal("a dead monster does not block")
	}

	lone := blockUnit(5, 100, 3)
	if testDirector().MonsterBlock(lone.m, false) {
		t.Fatal("an unknown monster does not block")
	}
}

// The block animation gate (0x57ceb0): the cooldown stamp is taken whether or not the hero may enter the mode,
// the animation counter only counts when she may.
func TestBlockAnimGate(t *testing.T) {
	d := testDirector()
	p := &d2mapentity.Player{}
	allow := false
	d.HeroBlockAnim = func(*d2mapentity.Player) bool { return allow }

	if note := d.noteBlockAnim(p); d.Counters.BlockAnims != 0 || note != " block_anim=false gated=true" {
		t.Fatalf("gated: %q anims=%d", note, d.Counters.BlockAnims)
	}

	// the stamp was taken: within 15 frames no block animation, gated or not
	allow = true
	d.frame += 5

	if note := d.noteBlockAnim(p); d.Counters.BlockAnims != 0 || note != " block_anim=false" {
		t.Fatalf("cooldown: %q anims=%d", note, d.Counters.BlockAnims)
	}

	// wave 48: the block sound hook runs exactly when the block mode is entered
	entered := 0
	d.OnHeroBlockMode = func(*d2mapentity.Player) { entered++ }

	d.frame += 40

	if note := d.noteBlockAnim(p); d.Counters.BlockAnims != 1 || note != " block_anim=true" || entered != 1 {
		t.Fatalf("allowed: %q anims=%d hook=%d", note, d.Counters.BlockAnims, entered)
	}

	d.frame += 3 // within the cooldown: no mode, no sound

	if d.noteBlockAnim(p); entered != 1 {
		t.Fatalf("hook ran %d times, want 1", entered)
	}
}

func TestBlockModeToken(t *testing.T) {
	if m, ok := d2monster.ParseMode("bl"); !ok || m != d2monster.ModeBlock || m.String() != "BL" {
		t.Fatalf("BL token: %v %v", m, ok)
	}
}
