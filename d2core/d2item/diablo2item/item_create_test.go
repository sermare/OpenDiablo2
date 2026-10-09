package diablo2item

import (
	"os"
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// realCreatorFactory is a factory with the real item creator, loaded from the
// extracted game tables (D2_TABLES), and the few records the item model needs
// for the test items.
func realCreatorFactory(t *testing.T) *ItemFactory {
	t.Helper()

	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	c, err := d2drop.LoadCreator(d2drop.DirTables{Root: root})
	if err != nil {
		t.Skipf("tables: %v", err)
	}

	f := testDropFactory()
	rec := f.asset.Records
	rec.Item.All["cap"].Cost = 10

	for _, code := range []string{"bhm", "uap", "lsd", "lrg", "r07", "r08", "r09", "ear"} {
		b := c.Items.ByCode[code]
		rec.Item.All[code] = &d2records.ItemCommonRecord{Code: code, Type: b.Type, Level: b.Level, Spawnable: true}
	}

	rec.Item.Types["swor"] = &d2records.ItemTypeRecord{Code: "swor"}
	rec.Item.Unique = d2records.UniqueItems{"Harlequin Crest": {Name: "Harlequin Crest", Code: "uap"}}
	rec.Item.SetItems = d2records.SetItems{"Tancred's Skull": {SetItemKey: "Tancred's Skull", SetKey: "Tancred's Battlegear", ItemCode: "bhm"}}

	f.SetCreator(c)

	return f
}

func TestCreateMagicAndRare(t *testing.T) {
	f := realCreatorFactory(t)
	ac := f.creator.Items.ByCode["cap"]

	magic, err := f.Create(CreateParams{Code: "cap", ILvl: 40, Quality: d2drop.QualityMagic, Seed: 7})
	if err != nil {
		t.Fatal(err)
	}

	if magic.Quality() != d2drop.QualityMagic || magic.Rolled() == nil {
		t.Fatalf("quality %v rolled %v", magic.Quality(), magic.Rolled())
	}

	if n := len(magic.PrefixCodes) + len(magic.SuffixCodes); n < 1 || n > 2 {
		t.Errorf("magic item has %d affixes: %v %v", n, magic.PrefixCodes, magic.SuffixCodes)
	}

	if d := magic.attributes.defense; d < ac.MinAC || d > ac.MaxAC+ac.MaxAC/2+1 {
		t.Errorf("defense %d outside %d..%d", d, ac.MinAC, ac.MaxAC)
	}

	if magic.ItemLevel() != 40 {
		t.Errorf("ilvl %d", magic.ItemLevel())
	}

	rare, err := f.Create(CreateParams{Code: "cap", ILvl: 80, Quality: d2drop.QualityRare, Seed: 11})
	if err != nil {
		t.Fatal(err)
	}

	if rare.rareName == "" || rare.Rolled().RareNames[0] == 0 || rare.Rolled().RareNames[1] == 0 {
		t.Errorf("rare item without names: %+v", rare.Rolled().RareNames)
	}

	if rare.IsIdentified() {
		t.Error("a rare item is created unidentified")
	}

	// equal parameters give equal items
	again, _ := f.Create(CreateParams{Code: "cap", ILvl: 80, Quality: d2drop.QualityRare, Seed: 11})
	if !reflect.DeepEqual(rare.Rolled(), again.Rolled()) {
		t.Error("creation is not reproducible")
	}
}

func TestCreateUniqueAndSet(t *testing.T) {
	f := realCreatorFactory(t)

	shako, err := f.Create(CreateParams{
		Code: "uap", ILvl: 99, Quality: d2drop.QualityUnique, Name: "Harlequin Crest", Seed: 5, FreshGame: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if shako.Quality() != d2drop.QualityUnique || shako.UniqueCode != "Harlequin Crest" {
		t.Fatalf("quality %v unique %q", shako.Quality(), shako.UniqueCode)
	}

	// +2 to all skills (item_allskills, stat 127) is a fixed property of the shako
	found := false

	for _, p := range shako.StatItem().Props {
		if p.ID == 127 && p.Value == 2 {
			found = true
		}
	}

	if !found {
		t.Errorf("no +2 all skills in %+v", shako.StatItem().Props)
	}

	skull, err := f.Create(CreateParams{
		Code: "bhm", ILvl: 99, Quality: d2drop.QualitySet, Name: "Tancred's Skull", Seed: 5, FreshGame: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if skull.SetItemCode != "Tancred's Skull" || skull.setRow == 0 {
		t.Fatalf("set item %q set row %d", skull.SetItemCode, skull.setRow)
	}

	si := skull.StatItem()
	if si.SetID != skull.setRow || len(si.SetLists) == 0 {
		t.Errorf("stat item: set %d, %d set lists", si.SetID, len(si.SetLists))
	}
}

func TestSpecKeepsRolledItem(t *testing.T) {
	f := realCreatorFactory(t)

	item, err := f.Create(CreateParams{Code: "cap", ILvl: 60, Quality: d2drop.QualityRare, Seed: 3})
	if err != nil {
		t.Fatal(err)
	}

	item.Identify()

	back, err := f.ItemFromSpec(item.Spec())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(item.Rolled(), back.Rolled()) || back.rareName != item.rareName || !back.IsIdentified() {
		t.Errorf("rebuilt item differs: %q vs %q", back.rareName, item.rareName)
	}

	if !reflect.DeepEqual(item.StatItem(), back.StatItem()) {
		t.Error("rebuilt item has other stats")
	}
}

func TestDropsUseCreator(t *testing.T) {
	f := realCreatorFactory(t)

	f.ResetGame()
	res, err := f.DropAll("Boss", DropOptions{Seed: 4, ILvl: 85, Players: 1})
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Items) == 0 {
		t.Fatal("no drop")
	}

	for _, it := range res.Items {
		if it.Rolled() == nil || it.ItemLevel() != 85 {
			t.Errorf("%s: rolled %v ilvl %d", it.CommonCode, it.Rolled(), it.ItemLevel())
		}
	}

	// a classic game never creates expansion items, and the same seed gives the same drop
	f.ResetGame()
	again, _ := f.DropAll("Boss", DropOptions{Seed: 4, ILvl: 85, Players: 1})
	if len(again.Items) != len(res.Items) || !reflect.DeepEqual(again.Items[0].Rolled(), res.Items[0].Rolled()) {
		t.Error("drop is not reproducible")
	}
}

func TestSocketRunewordSetsEarsCube(t *testing.T) {
	f := realCreatorFactory(t)

	var shield *Item

	for seed := uint32(1); seed < 400 && shield == nil; seed++ {
		it, err := f.Create(CreateParams{
			Code: "lrg", ILvl: 50, Quality: d2drop.QualityNormal, Seed: seed, Flags: d2drop.FlagForceSockets,
		})
		if err == nil && it.NumSockets() == 3 {
			shield = it
		}
	}

	if shield == nil {
		t.Fatal("no shield with three sockets")
	}

	for _, code := range []string{"r08", "r09", "r07"} {
		rune, err := f.Create(CreateParams{Code: code, ILvl: 1, Quality: d2drop.QualityNormal, Seed: 1})
		if err != nil {
			t.Fatal(err)
		}

		if err := shield.Socket(rune); err != nil {
			t.Fatal(err)
		}
	}

	if shield.RolledRuneword() != "Runeword1" {
		t.Fatalf("runeword %q", shield.RolledRuneword())
	}

	si := shield.StatItem()
	if !si.Runeword || len(si.RunewordProps) == 0 || len(si.Sockets) != 3 {
		t.Errorf("stat item %+v", si)
	}

	back, err := f.ItemFromSpec(shield.Spec())
	if err != nil || back.RolledRuneword() != "Runeword1" || len(back.Socketed()) != 3 ||
		!reflect.DeepEqual(back.StatItem(), si) {
		t.Errorf("rebuilt runeword: %v %q\n%+v\n%+v", err, back.RolledRuneword(), back.StatItem(), si)
	}

	if err := shield.Socket(shield); err == nil {
		t.Error("a full item took another item")
	}

	// the first piece of a set: the set table has the bonuses of its set
	table, err := f.SetTable()
	if err != nil {
		t.Fatal(err)
	}

	skull, _ := f.Create(CreateParams{
		Code: "bhm", ILvl: 99, Quality: d2drop.QualitySet, Name: "Tancred's Skull", Seed: 5, FreshGame: true,
	})
	if def := table[skull.StatItem().SetID]; def.Pieces != 5 || len(def.Full) == 0 || len(def.Partial) != 4 {
		t.Errorf("set definition %+v", def)
	}

	id := skull.StatItem().SetID
	if two, five := table.SetBonus(id, 2), table.SetBonus(id, 5); len(two) == 0 || len(five) <= len(two) || len(table.SetBonus(id, 1)) != 0 {
		t.Errorf("set bonus by pieces worn: 1 -> %d, 2 -> %d, 5 -> %d", len(table.SetBonus(id, 1)), len(two), len(five))
	}

	if f.SetOfSetItem(0) == 0 {
		t.Error("no set for the first set item")
	}

	ear, err := f.NewEar("Doomguy", 3, 41)
	if err != nil || ear.Ear() == nil || ear.Ear().Label() != "Doomguy (Level 41 Paladin)" {
		t.Fatalf("ear %v %+v", err, ear)
	}

	if b, err := f.ItemFromSpec(ear.Spec()); err != nil || b.Ear() == nil || b.Ear().Name != "Doomguy" {
		t.Errorf("rebuilt ear %v", err)
	}

	cube, err := f.CubeResult("cap", []string{"rar", "eth"}, 60, 9)
	if err != nil || cube.Quality() != d2drop.QualityRare {
		t.Fatalf("cube result %v %v", err, cube)
	}

	if _, err := f.CubeResult("nope", nil, 1, 1); err == nil {
		t.Error("unknown cube output")
	}

	mod, err := f.NewItem("cap", "q=magic", "ilvl=50")
	if err != nil || mod.Quality() != d2drop.QualityMagic || mod.ItemLevel() != 50 {
		t.Errorf("NewItem options: %v %v", err, mod)
	}
}
