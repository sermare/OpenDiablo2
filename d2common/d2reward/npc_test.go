package d2reward

import "testing"

func TestItemKindOf(t *testing.T) {
	for class, want := range map[int]ItemKind{511: KindSocket, 512: KindPersonalize, 154: KindImbue} {
		k, ok := ItemKindOf(class)
		if !ok || k != want || NPCOf(k) != class || k.Verb() == "" {
			t.Errorf("class %d: %v %v", class, k, ok)
		}
	}

	if _, ok := ItemKindOf(148); ok { // Akara takes no item
		t.Error("Akara does not take items")
	}
}

func TestOwed(t *testing.T) {
	s := &State{SocketPending: 1}

	o := OwedFrom(s, true)
	if !o.Of(KindSocket) || o.Of(KindPersonalize) || !o.Of(KindImbue) {
		t.Errorf("%+v", o)
	}

	if OwedFrom(nil, false).Of(KindSocket) {
		t.Error("nil state owes nothing")
	}
}

func TestRespec(t *testing.T) {
	base := Allocation{Strength: 10, Dexterity: 25, Vitality: 10, Energy: 35}
	now := Allocation{Strength: 20, Dexterity: 25, Vitality: 40, Energy: 35, SkillPoints: 17}

	if stat, skill := Respec(now, base); stat != 40 || skill != 17 {
		t.Errorf("stat %d skill %d", stat, skill)
	}

	// a stat below the base (a bad save) gives nothing back, never a debt
	if stat, _ := Respec(Allocation{Strength: 5}, base); stat != 0 {
		t.Errorf("negative refund %d", stat)
	}
}
