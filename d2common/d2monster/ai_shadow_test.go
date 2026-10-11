package d2monster

import "testing"

type shadowFake struct {
	*fb1World
	skills   []ShadowSkill
	scan     ShadowScan
	resist   int
	hasState map[int]bool
	other    bool
	castMask bool
	progStat int
	progOK   bool
	threat   int
	isMon    bool
	selected []int
	leftSet  bool
	cast     []int
	castOK   map[int]bool
}

func newShadowFake() *shadowFake {
	return &shadowFake{
		fb1World: newFB1(10, false),
		scan:     ShadowScan{NearestDist2: 0x7fffffff},
		hasState: map[int]bool{}, castOK: map[int]bool{}, isMon: true, threat: 30,
	}
}

func (f *shadowFake) ShadowSkills(*Brain) []ShadowSkill          { return f.skills }
func (f *shadowFake) ShadowScan(*Brain, *Target) ShadowScan      { return f.scan }
func (f *shadowFake) TargetStat(Target, int) int                 { return f.resist }
func (f *shadowFake) SelfHasState(_ *Brain, s int) bool          { return f.hasState[s] }
func (f *shadowFake) TargetHasState(Target, int) bool            { return false }
func (f *shadowFake) SelfInCastMask(*Brain) bool                 { return f.castMask }
func (f *shadowFake) OtherInStateGroup(*Brain, int) bool         { return f.other }
func (f *shadowFake) SelfStateStat(*Brain, int, int) (int, bool) { return f.progStat, f.progOK }
func (f *shadowFake) SelectActiveSkill(_ *Brain, id int)         { f.selected = append(f.selected, id) }
func (f *shadowFake) LeftSkillSet(*Brain) bool                   { return f.leftSet }
func (f *shadowFake) TargetMonsterThreat(Target, Difficulty) (bool, int) {
	return f.isMon, f.threat
}

func (f *shadowFake) CastQueued(_ *Brain, id int, _ Target, _ bool) bool {
	f.cast = append(f.cast, id)

	return f.castOK[id]
}

// the fake world is the book, the env and the world at once
func (f *shadowFake) HasSkills(*Brain) bool                            { return true }
func (f *shadowFake) Upkeep(*Brain, *Target, bool) bool                { return false }
func (f *shadowFake) Choose(*Brain, Target, *Target, bool) (int, bool) { return 0, false }
func (f *shadowFake) Copy(*Brain, Target, Target, bool, bool, bool) (int, bool) {
	return 0, false
}

func shadowInput(f *shadowFake, b *Brain) *ShadowScoreInput {
	prog := 0

	return &ShadowScoreInput{
		B: b, Env: f, Target: Target{ID: 9, X: 110, Y: 100, Size: 1}, Scan: f.scan,
		Dist2: 100, Life: 100, Progress: &prog,
	}
}

// pickOne makes the score roll bound 1 so every roll is 0 (it still advances the seed)
func pickOne(b *Brain) { b.Profile.AIPRaw[2] = [3]int{1, 36, 36} }

func TestShadowResistTerm(t *testing.T) {
	for _, tc := range []struct{ resist, want int }{{0, 0}, {9, 0}, {10, -1}, {50, -5}, {75, -7}, {-50, 5}, {-9, 0}} {
		if got := shadowResistTerm(tc.resist); got != tc.want {
			t.Errorf("term(%d) = %d, want %d", tc.resist, got, tc.want)
		}
	}
}

func TestShadowScoreKinds(t *testing.T) {
	f := newShadowFake()
	b := fb1Brain("ShadowMaster", 20, 20, 50)
	b.Profile.AIPRaw[1] = [3]int{20, 10, 5}
	pickOne(b)

	// kind 3: within*3 - 9 + roll + base (base = aibonus + req/4 + level + resist term)
	f.scan.WithinCount = 4
	in := shadowInput(f, b)

	got, tg := ShadowScore(in, ShadowSkill{ID: 40, AIType: 3, AIBonus: 2, ReqLevel: 9, Level: 5})
	if want := 4*3 - 9 + 0 + (2 + 2 + 5); got != want || tg == nil || tg.ID != 9 {
		t.Errorf("kind 3: %d, want %d", got, want)
	}

	// kind 3 penalties: nearest hostile close, few units
	f.scan.WithinCount, f.scan.NearestDist2 = 2, 10
	in = shadowInput(f, b)
	got, _ = ShadowScore(in, ShadowSkill{AIType: 3})

	if want := 2*3 - 9 - 7 - 10; got != want {
		t.Errorf("kind 3 penalties: %d, want %d", got, want)
	}

	// kind 4, plain melee: target out of the approach range (dist2 500 > 400), not in reach,
	// bonus word 10, progressive word 5 with the cast mask off: -10 (range) + 10 (bonus) - 10
	f.scan = ShadowScan{NearestDist2: 0x7fffffff}
	in = shadowInput(f, b)
	in.Dist2 = 500
	got, _ = ShadowScore(in, ShadowSkill{AIType: 4})

	if want := -10 + 10 - 10; got != want {
		t.Errorf("kind 4: %d, want %d", got, want)
	}

	// with the cast mask on: +3 + progress*4
	in.CastMask = true
	*in.Progress = 2
	got, _ = ShadowScore(in, ShadowSkill{AIType: 4})

	if want := -10 + 10 + 3 + 8; got != want {
		t.Errorf("kind 4 mask: %d, want %d", got, want)
	}

	// progressive kind 4 (bits4 & 4): a state stat above 2 silences it, else + progressive word
	f.hasState[7], f.progOK, f.progStat = true, true, 3
	in = shadowInput(f, b)
	in.Dist2 = 500
	got, _ = ShadowScore(in, ShadowSkill{AIType: 4, Bits4: 4, AuraState: 7, AuraStat1: 1})

	if got != 0 {
		t.Errorf("progressive over 2: %d", got)
	}

	f.progStat = 2
	in = shadowInput(f, b)
	in.Dist2 = 500
	got, _ = ShadowScore(in, ShadowSkill{AIType: 4, Bits4: 4, AuraState: 7, AuraStat1: 1})

	if want := -10 + 10 + 5; got != want || *in.Progress != 2 {
		t.Errorf("progressive: %d (acc %d), want %d", got, *in.Progress, want)
	}

	// kind 5: no clear line scores nothing
	in = shadowInput(f, b)
	in.Blocked = true

	if got, _ = ShadowScore(in, ShadowSkill{AIType: 5, SrvMissile: 5}); got != 0 {
		t.Errorf("blocked missile: %d", got)
	}

	// kind 5 in the clear: the close-range penalties
	in = shadowInput(f, b)
	in.Dist2 = 10
	in.CastMask = true
	f.scan.NearestDist2 = 5
	in.Scan = f.scan
	got, _ = ShadowScore(in, ShadowSkill{AIType: 5, SrvMissile: 5})

	if want := -15; got != want {
		t.Errorf("kind 5 penalties: %d, want %d", got, want)
	}

	// kinds 7 and 8 only when wounded; kind 8 targets the shadow itself
	f.scan = ShadowScan{NearestDist2: 0x7fffffff}
	in = shadowInput(f, b)
	in.Life = 80

	if got, _ = ShadowScore(in, ShadowSkill{AIType: 8}); got != 0 {
		t.Errorf("kind 8 healthy: %d", got)
	}

	in.Life = 60
	got, tg = ShadowScore(in, ShadowSkill{AIType: 8, AIBonus: 3})

	if got != 6 || tg == nil || tg.ID != b.ID {
		t.Errorf("kind 8 wounded: %d %v", got, tg)
	}

	in.Life = 40
	if got, _ = ShadowScore(in, ShadowSkill{AIType: 8, AIBonus: 3}); got != 12 {
		t.Errorf("kind 8 badly wounded: %d", got)
	}

	in.Life = 60
	got, tg = ShadowScore(in, ShadowSkill{AIType: 7, AIBonus: 1})

	if got != 11 || tg != nil {
		t.Errorf("kind 7: %d %v", got, tg)
	}

	// kind 0xc ignores weak monsters
	in = shadowInput(f, b)
	f.threat = 10

	if got, _ = ShadowScore(in, ShadowSkill{AIType: 0xc}); got != 0 {
		t.Errorf("kind 0xc weak monster: %d", got)
	}

	// kind 0xd: far target gets +10 above dist2 0x144, too close is dropped
	in = shadowInput(f, b)
	in.Dist2 = 400

	if got, _ = ShadowScore(in, ShadowSkill{AIType: 0xd}); got != 10-5+10 {
		// bonus word 10 (+0x58), -5 (progressive word set, mask off), +10 distance
		t.Errorf("kind 0xd far: %d", got)
	}

	in.Dist2 = 20

	if got, _ = ShadowScore(in, ShadowSkill{AIType: 0xd}); got != 0 {
		t.Errorf("kind 0xd close: %d", got)
	}
}

func TestShadowScoreDrawsItsRolls(t *testing.T) {
	// aitype 6 draws one roll(100); 0 and unknown kinds draw none
	f := newShadowFake()
	b := fb1Brain("ShadowMaster", 20, 20, 50)
	pickOne(b)
	orig := &Brain{Seed: shadow(b)}
	in := shadowInput(f, b)

	ShadowScore(in, ShadowSkill{AIType: 0})

	if !stepsUsed(b, orig, 0) {
		t.Error("kind 0 must not draw")
	}

	ShadowScore(in, ShadowSkill{ID: 3, AIType: 6})

	if !stepsUsed(b, orig, 1) {
		t.Error("kind 6 draws one roll")
	}
}

func TestShadowPickKeepsOnlyImprovements(t *testing.T) {
	f := newShadowFake()
	b := fb1Brain("ShadowMaster", 20, 20, 50)
	pickOne(b)
	f.scan.WithinCount = 5 // kind 3 base = 15 - 9 = 6 + bonus
	in := shadowInput(f, b)

	stack := ShadowPick(in, []ShadowSkill{
		{ID: 10, AIType: 3, AIBonus: 1}, // 7
		{ID: 11, AIType: 3, AIBonus: 0}, // 6: worse, not pushed
		{ID: 12, AIType: 3, AIBonus: 4}, // 10
		{ID: 13, AIType: 9},             // 0
	})

	if len(stack) != 3 || stack[0].Skill != 0 || stack[1].Skill != 10 || stack[2].Skill != 12 || stack[2].Score != 10 {
		t.Fatalf("stack %+v", stack)
	}
}

func TestShadowTailCastsBestFirstAndQueuesFollowUps(t *testing.T) {
	f := newShadowFake()
	f.scan.WithinCount = 5
	f.skills = []ShadowSkill{{ID: 10, AIType: 3, AIBonus: 1}, {ID: 12, AIType: 3, AIBonus: 4, DoFunc: 0x13}}
	f.castOK[12] = true

	// a brain whose cast-loop draws are all non-zero in the low two bits
	var b *Brain

	for id := uint32(1); id < 3000 && b == nil; id++ {
		c := fb1Brain("ShadowMaster", 20, 20, 50)
		pickOne(c)
		c.Seed.Init(id)
		s := shadow(c)
		s.Step() // the score rolls of the two skills: bound 1 advances once each
		s.Step()

		if s.Step()&3 != 0 {
			b = c
		}
	}

	if b == nil {
		t.Fatal("no seed")
	}

	w := &shadowWorld{shadowFake: f}
	c := &Ctx{B: b, W: w}
	tgt := Target{ID: 9, X: 110, Y: 100, Size: 1}
	c.Target, c.Dist = &tgt, 10

	if !shadowTail(c, w, w, nil, &tgt, nil) {
		t.Fatal("tail must end the tick")
	}

	if len(f.cast) != 1 || f.cast[0] != 12 || b.Scratch[0] != 0x19 || b.Scratch[1] != 12 {
		t.Errorf("cast %v scratch %v", f.cast, b.Scratch)
	}
}

// shadowWorld adapts the fake to the interfaces shadowTail takes.
type shadowWorld struct{ *shadowFake }
