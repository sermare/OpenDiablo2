package diablo2item

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Charsi's imbue: a magic cap becomes a rare cap of the item level the rule
// gives, keeping ethereal and personalisation; jewellery and rares are refused.
func TestImbue(t *testing.T) {
	f := realCreatorFactory(t)

	f.asset.Records.Item.Types["helm"] = &d2records.ItemTypeRecord{Code: "helm", Equiv1: "armo"}
	f.asset.Records.Item.Types["armo"] = &d2records.ItemTypeRecord{Code: "armo"}

	old, err := f.Create(CreateParams{Code: "cap", ILvl: 20, Quality: d2drop.QualityMagic, Seed: 5})
	if err != nil {
		t.Fatal(err)
	}

	old.TypeCode = "helm"
	old.SetPersonalName("Hero")

	if info := old.ImbueInfo(); !info.WeaponOrArmor || info.Quality != int(d2drop.QualityMagic) {
		t.Fatalf("info %+v", info)
	}

	rare, err := f.Imbue(old, 34, 99)
	if err != nil {
		t.Fatal(err)
	}

	if rare.Quality() != d2drop.QualityRare || rare.Rolled() == nil {
		t.Errorf("not rare: quality %v", rare.Quality())
	}

	if rare.CommonCode != "cap" || !rare.IsIdentified() || rare.PersonalName() != "Hero" {
		t.Errorf("rare %q identified=%v personal=%q", rare.CommonCode, rare.IsIdentified(), rare.PersonalName())
	}

	if rare.ItemLevel() != 34 {
		t.Errorf("ilvl %d", rare.ItemLevel())
	}

	// a rare cannot be imbued again
	if _, err := f.Imbue(rare, 34, 100); err == nil {
		t.Error("a rare item was imbued")
	}

	// jewellery is not weapon or armour
	old.TypeCode = "ring"
	f.asset.Records.Item.Types["ring"] = &d2records.ItemTypeRecord{Code: "ring"}

	if _, err := f.Imbue(old, 34, 101); err == nil {
		t.Error("a ring was imbued")
	}
}
