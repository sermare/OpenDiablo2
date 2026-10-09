package d2drop

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

func TestRunewordWalk(t *testing.T) {
	c := cLoadCreator(t)
	c.Runes = loadRuneTablesFrom(skipTB{t}, testSource(t), c.Props)

	// Ancient's Pledge: Ral Ort Tal in a shield; the type must fit and the order matter
	w := c.FindRuneword("lrg", []string{"r08", "r09", "r07"})
	if w == nil || w.Name != "Runeword1" {
		t.Fatalf("runeword %+v", w)
	}

	if c.FindRuneword("lrg", []string{"r09", "r08", "r07"}) != nil {
		t.Error("runes in the wrong order make a runeword")
	}

	if c.FindRuneword("lsd", []string{"r08", "r09", "r07"}) != nil {
		t.Error("a sword takes a shield runeword")
	}

	req := Request{
		Code: "lrg", ILvl: 40, Quality: QualityNormal, Version: 100, Expansion: true,
		GameSeed: d2rand.Seed{Lo: 17, Hi: 0x29a},
	}

	base, err := c.Create(req)
	if err != nil {
		t.Fatal(err)
	}

	got := c.ApplyRuneword(base, req, w)
	if got.Flags&FlagRuneword == 0 {
		t.Errorf("flags %#x", got.Flags)
	}

	// cold resist 30 (stat 43), all resist 13 (stat 39 and its siblings), 50% defense
	want := map[int]int{43: 43, 16: 50} // cold resist 30 + all resist 13
	have := map[int]int{}

	for _, wr := range got.Writes[len(base.Writes):] {
		if wr.Kind == 'S' { // the defense bonus changes the item's base defense
			continue
		}

		if wr.List != ListRuneword {
			t.Errorf("write %+v not in the runeword list", wr)
		}

		have[wr.Stat] += wr.Value >> uint(c.Props.ValShift[wr.Stat])
	}

	for stat, v := range want {
		if have[stat] != v {
			t.Errorf("stat %d = %d, want %d (have %v)", stat, have[stat], v, have)
		}
	}

	// walking again changes nothing
	if again := c.ApplyRuneword(got, req, w); len(again.Writes) != len(got.Writes) {
		t.Error("the runeword was applied twice")
	}
}

func TestSetBonuses(t *testing.T) {
	c := cLoadCreator(t)

	set := -1

	for i, s := range c.Uniques.Sets {
		if s.Name == "Tancred's Battlegear" {
			set = i
		}
	}

	if set < 0 {
		t.Fatal("no Tancred's Battlegear")
	}

	b, ok := c.SetBonuses(set)
	if !ok || b.Pieces != 5 {
		t.Fatalf("bonus %+v %v", b, ok)
	}

	if len(b.Partial[0]) == 0 || len(b.Full) == 0 {
		t.Errorf("no bonus writes: 2 pieces %d, full %d", len(b.Partial[0]), len(b.Full))
	}
}
