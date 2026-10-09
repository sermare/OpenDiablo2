package diablo2item

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func sourceTestFactory() *ItemFactory {
	f := testDropFactory()
	rec := f.asset.Records

	rec.Item.Treasure.Expansion["Gold Box"] = &d2records.TreasureClassRecord{
		Name: "Gold Box", Index: 10, NumPicks: 1, Treasures: []*d2records.Treasure{{Code: "gld,mul=512", Probability: 1}},
	}
	rec.Item.Treasure.Expansion["Act 2 Chest B"] = &d2records.TreasureClassRecord{
		Name: "Act 2 Chest B", Index: 11, NumPicks: 1, Treasures: []*d2records.Treasure{{Code: "cap", Probability: 1}},
	}
	rec.Item.Treasure.Expansion["Act 2 Chest A"] = &d2records.TreasureClassRecord{
		Name: "Act 2 Chest A", Index: 12, NumPicks: 1, Treasures: []*d2records.Treasure{{Code: "cap", Probability: 1}},
	}

	rec.Monster.Stats = d2records.MonStats{}
	rec.Monster.Stats["fallen"] = &d2records.MonStatRecord{}
	rec.Monster.Stats["fallen"].TreasureClassNormal = "Gold Box"
	rec.Monster.Stats["fallen"].TreasureClassNightmare = "Boss"

	rec.Level.Details = d2records.LevelDetails{
		40: {Act: 1, MonsterLevelNormal: 20, MonsterLevelNormalEx: 22},
		41: {Act: 1, MonsterLevelNormal: 30, MonsterLevelNormalEx: 32},
		73: {Act: 1, MonsterLevelNormal: 60, MonsterLevelNormalEx: 62},
	}

	return f
}

func TestMonsterTreasureClassChoice(t *testing.T) {
	f := sourceTestFactory()

	for _, c := range []struct {
		diff int
		want string
	}{{0, "Gold Box"}, {1, "Boss"}, {2, ""}} {
		got, err := f.MonsterTreasureClass(MonsterDropOptions{Monster: "fallen", Difficulty: c.diff})
		if err != nil || got != c.want {
			t.Errorf("difficulty %d: %q, %v; want %q", c.diff, got, err, c.want)
		}
	}

	if _, err := f.MonsterTreasureClass(MonsterDropOptions{Monster: "nobody"}); err == nil {
		t.Error("unknown monster must fail")
	}
}

func TestDropAllGold(t *testing.T) {
	f := sourceTestFactory()

	res, err := f.MonsterDrops(MonsterDropOptions{
		DropOptions: DropOptions{Seed: 9}, Monster: "fallen", Level: 30, Expansion: true,
	})
	if err != nil || len(res.Items) != 0 || len(res.Gold) != 1 {
		t.Fatalf("result %+v, %v", res, err)
	}

	// mul=512 doubles an amount of rand(5*30)+30 (30..179)
	if res.Gold[0] < 60 || res.Gold[0] > 358 {
		t.Errorf("gold %d out of range", res.Gold[0])
	}

	gf, _ := f.MonsterDrops(MonsterDropOptions{
		DropOptions: DropOptions{Seed: 9, GoldFind: 100}, Monster: "fallen", Level: 30, Expansion: true,
	})
	if gf.Gold[0] != res.Gold[0]*2 {
		t.Errorf("100%% gold find: %d, want %d", gf.Gold[0], res.Gold[0]*2)
	}
}

func TestChestDrops(t *testing.T) {
	f := sourceTestFactory()

	// act 2 (index 1): first level 41 -> 32, last 73 -> 62 in an expansion game;
	// level 41 (32) is in the first third of 32..62, tier 0 (class A).
	res, err := f.ChestDrops(ChestDropOptions{
		DropOptions: DropOptions{Seed: 3}, LevelID: 41, Difficulty: 0, Expansion: true,
	})
	if err != nil || len(res.Items) != 1 || res.Items[0].CommonCode != "cap" {
		t.Fatalf("result %+v, %v", res, err)
	}

	if _, err := f.ChestDrops(ChestDropOptions{LevelID: 999}); err == nil {
		t.Error("unknown level must fail")
	}

	// the class name follows from the tier
	if got := d2drop.ChestTier(32, 22, 62); got != 0 {
		t.Errorf("tier %d", got)
	}
}
