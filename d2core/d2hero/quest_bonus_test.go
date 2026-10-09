package d2hero

import (
	"bytes"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

func TestResistScrollBonusCountsReadRecords(t *testing.T) {
	var none *HeroProgress

	p := &HeroProgress{}
	if none.ResistScrollBonus() != 0 || p.ResistScrollBonus() != 0 {
		t.Fatal("a hero without the scroll must have no bonus")
	}

	p.Quests[0].Set(prisonSlot, 8) // scroll given, not read: nothing yet
	if p.ResistScrollBonus() != 0 {
		t.Fatal("an unread scroll must not count")
	}

	for want, d := range []int{0, 2, 1} {
		p.Quests[d].Set(prisonSlot, scrollReadBit)

		if got := p.ResistScrollBonus(); got != 10*(want+1) {
			t.Errorf("after %d read scrolls: %d", want+1, got)
		}
	}
}

func TestPrisonSlotMatchesQuestPackage(t *testing.T) {
	var rec d2s.QuestRecord

	g := d2quest.New(&rec, &d2s.NPCBlock{}, 0)
	if q := g.Quest(d2quest.QuestPrison); q == nil || q.Slot != prisonSlot || d2quest.FlagCustom3 != scrollReadBit {
		t.Fatal("prison slot or the scroll-read bit differ from d2quest")
	}
}

func TestResistScrollItemAddsToAllFourResists(t *testing.T) {
	base := d2statlist.Compute(d2statlist.Hero{Level: 1, Classic: true}, nil, nil)
	if base.Resist != [4]int{} {
		t.Fatalf("plain hero has resists %v", base.Resist)
	}

	if resistScrollItem(0) != nil {
		t.Fatal("no bonus must add no item")
	}

	tot := d2statlist.Compute(d2statlist.Hero{Level: 1, Classic: true}, resistScrollItem(20), nil)
	if tot.Resist != [4]int{20, 20, 20, 20} {
		t.Errorf("resists %v, want 20 each", tot.Resist)
	}
}

// reading the scroll through the quest game sets the record bit once, and the derived bonus follows the record.
func TestReadScrollThroughQuestGame(t *testing.T) {
	p := &HeroProgress{}
	p.Quests[0].Set(prisonSlot, 8) // Malah handed the scroll over

	g := d2quest.New(&p.Quests[0], &d2s.NPCBlock{}, 0)
	if eff := g.ReadScrollOfResistance(); len(eff) == 0 || p.ResistScrollBonus() != 10 {
		t.Fatalf("first read: effects %v bonus %d", eff, p.ResistScrollBonus())
	}

	if eff := g.ReadScrollOfResistance(); len(eff) != 0 || p.ResistScrollBonus() != 10 {
		t.Fatalf("second read must do nothing: effects %v bonus %d", eff, p.ResistScrollBonus())
	}
}

// fakeStats stands in for HeroStateFactory.RecalcStats: max life = class base 100 + LifeBonus.
func fakeStats(life int) *HeroStatsState {
	s := &HeroStatsState{Health: life, MaxHealth: life, BaseMaxHealth: life}
	s.Recalc = func() {
		s.StatsBonusInit = true
		s.BaseMaxHealth = 100 + s.LifeBonus
		s.MaxHealth = s.BaseMaxHealth

		if s.Health > s.MaxHealth {
			s.Health = s.MaxHealth
		}
	}

	return s
}

func TestAddLifeBonus(t *testing.T) {
	s := fakeStats(100)
	s.AddLifeBonus(20)

	if s.MaxHealth != 120 || s.Health != 120 || s.LifeBonus != 20 || s.BaseMaxHealth != 120 {
		t.Fatalf("after one potion: %+v", s)
	}

	s.Recalc() // a later recalculation (level up, equipment) must keep it
	if s.MaxHealth != 120 {
		t.Fatalf("recalc lost the bonus: %d", s.MaxHealth)
	}

	// no factory hook: the maxima move directly
	n := &HeroStatsState{Health: 50, MaxHealth: 100, BaseMaxHealth: 90}
	n.AddLifeBonus(20)

	if n.MaxHealth != 120 || n.BaseMaxHealth != 110 || n.Health != 70 {
		t.Fatalf("without hook: %+v", n)
	}

	var nilStats *HeroStatsState
	nilStats.AddLifeBonus(20) // must not panic
}

func TestUnchangedHeroIsUnaffected(t *testing.T) {
	s := fakeStats(100)
	before := *s
	s.AddLifeBonus(0)

	if s.MaxHealth != before.MaxHealth || s.LifeBonus != 0 {
		t.Fatal("a zero bonus changed the hero")
	}

	if (&HeroProgress{}).ResistScrollBonus() != 0 || resistScrollItem((&HeroProgress{}).ResistScrollBonus()) != nil {
		t.Fatal("a hero without the scroll got a resist item")
	}
}

// With the real save: an untouched hero still exports byte-exact, a drunk potion raises the stored maximum by
// exactly 20 and is stable when exported again from a state rebuilt out of the result (no double counting), and a
// read scroll changes only the quest record.
func TestRealSavePotionAndScrollRoundTrip(t *testing.T) {
	data, tables, state := realSave(t)

	out, _, err := ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil || !bytes.Equal(out, data) {
		t.Fatalf("baseline export differs: %v", err)
	}

	orig, err := d2s.Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	state.Stats.AddLifeBonus(20)

	out, _, err = ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}

	c, err := d2s.Parse(out, tables)
	if err != nil {
		t.Fatal(err)
	}

	if c.Body.Attributes.MaxHP != orig.Body.Attributes.MaxHP+20 {
		t.Fatalf("stored max life %d, want %d", c.Body.Attributes.MaxHP, orig.Body.Attributes.MaxHP+20)
	}

	// reload: a state built from the exported file exports to the same bytes
	a := c.Body.Attributes
	again := *state
	st := *state.Stats
	st.MaxHealth, st.BaseMaxHealth, st.Health = int(a.MaxHP), 0, int(a.CurrentHP)
	again.Stats = &st

	out2, _, err := ExportD2SWithOptions(&again, out, tables, ExportOptions{})
	if err != nil || !bytes.Equal(out2, out) {
		t.Fatalf("reload changed the file: %v", err)
	}

	// the scroll: the quest record carries it
	state2 := *state
	p := *state.Progress
	state2.Progress = &p
	st2 := *state.Stats
	state2.Stats = &st2
	p.Quests[2].Set(prisonSlot, scrollReadBit)

	out3, _, err := ExportD2SWithOptions(&state2, out, tables, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}

	c3, err := d2s.Parse(out3, tables)
	if err != nil || !c3.Body.QuestRecord(2).Get(prisonSlot, scrollReadBit) {
		t.Fatalf("scroll bit not saved: %v", err)
	}
}
