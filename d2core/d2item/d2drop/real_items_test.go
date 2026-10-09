package d2drop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helpers shared by the real-data (D2_TABLES) tests of the item creation
// slices. D2_TABLES holds the extracted game tables:
//
//	$D2_TABLES/{weapons,armor,misc,ItemTypes,ItemStatCost}.txt
//	$D2_TABLES/itemgen/{patch_d2,d2exp,d2data}/*.txt
//
// The game loads each table from the first of patch_d2.mpq, d2exp.mpq,
// d2data.mpq that has it (tablePath implements that order).

func d2Tables(t *testing.T) string {
	t.Helper()

	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	return root
}

// tablePath returns the path of the effective version of a table under
// itemgen/ (patch_d2, then d2exp, then d2data), or skips the test.
func tablePath(t *testing.T, name string) string {
	t.Helper()

	root := d2Tables(t)

	for _, mpq := range []string{"patch_d2", "d2exp", "d2data"} {
		p := filepath.Join(root, "itemgen", mpq, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	t.Skipf("table %s not found under %s/itemgen", name, root)

	return ""
}

var heroIndex = map[string]int{"ama": 0, "sor": 1, "nec": 2, "pal": 3, "bar": 4, "dru": 5, "ass": 6}

// loadItemTables reads weapons.txt, armor.txt, misc.txt and ItemTypes.txt.
func loadItemTables(t *testing.T) *ItemTables {
	t.Helper()

	root := d2Tables(t)
	it := &ItemTables{ByCode: map[string]*BaseItem{}, Types: map[string]*ItemType{}}

	types := readTSV(t, filepath.Join(root, "ItemTypes.txt"))

	skipped := 0 // the "Expansion" separator row is not part of the game's table

	for i, r := range types.rows {
		if types.s(r, "ItemType") == "Expansion" {
			skipped++

			continue
		}

		i -= skipped

		code := types.s(r, "Code")
		if code == "" && i != 0 {
			continue
		}

		ty := &ItemType{
			Index: i, Code: code, Equiv1: types.s(r, "Equiv1"), Equiv2: types.s(r, "Equiv2"),
			Normal: types.n(r, "Normal") == 1, Magic: types.n(r, "Magic") == 1, Rare: types.n(r, "Rare") == 1,
			Charm: types.n(r, "Charm") == 1, Gem: types.n(r, "Gem") == 1, Beltable: types.n(r, "Beltable") == 1,
			MaxSock1: types.n(r, "MaxSock1"), MaxSock25: types.n(r, "MaxSock25"), MaxSock40: types.n(r, "MaxSock40"),
			TreasureClass: types.n(r, "TreasureClass") == 1, Rarity: types.n(r, "Rarity"),
			Class: -1, VarInvGfx: types.n(r, "VarInvGfx"), Throwable: types.n(r, "Throwable") == 1,
			Quiver: types.s(r, "Quiver") != "", AutoStack: types.n(r, "AutoStack") == 1,
			StaffMods: types.s(r, "StaffMods"), CostFormula: types.n(r, "CostFormula"),
		}

		if c, ok := heroIndex[strings.ToLower(types.s(r, "Class"))]; ok {
			ty.Class = c
		}

		it.Types[code] = ty
	}

	for _, ty := range it.Types {
		seen := map[string]bool{}

		var walk func(c string)

		walk = func(c string) {
			if c == "" || seen[c] {
				return
			}

			seen[c] = true
			ty.Ancestors = append(ty.Ancestors, c)

			if p := it.Types[c]; p != nil {
				walk(p.Equiv1)
				walk(p.Equiv2)
			}
		}

		walk(ty.Code)
	}

	for kind, file := range []string{"weapons", "armor", "misc"} {
		tab := readTSV(t, filepath.Join(root, file+".txt"))

		for _, r := range tab.rows {
			code := tab.s(r, "code")
			if code == "" {
				continue
			}

			b := &BaseItem{
				Class: len(it.Items), Kind: BaseKind(kind), Code: code,
				NormCode: tab.s(r, "normcode"), UberCode: tab.s(r, "ubercode"), UltraCode: tab.s(r, "ultracode"),
				Type: tab.s(r, "type"), Type2: tab.s(r, "type2"), Version: tab.n(r, "version"),
				Level: tab.n(r, "level"), LevelReq: tab.n(r, "levelreq"), Rarity: tab.n(r, "rarity"),
				Spawnable: tab.n(r, "spawnable") == 1, Quest: tab.n(r, "quest"),
				QuestDiffCheck: tab.n(r, "questdiffcheck"), Unique: tab.n(r, "unique") == 1,
				MagicLevel: tab.n(r, "magic lvl"), AutoPrefix: tab.n(r, "auto prefix"),
				MinAC: tab.n(r, "minac"), MaxAC: tab.n(r, "maxac"), Block: tab.n(r, "block"),
				Absorbs: tab.n(r, "absorbs"), Speed: tab.n(r, "speed"), Durability: tab.n(r, "durability"),
				NoDurability: tab.n(r, "nodurability") == 1,
				MinDam:       tab.n(r, "mindam"), MaxDam: tab.n(r, "maxdam"),
				TwoHandMinDam: tab.n(r, "2handmindam"), TwoHandMaxDam: tab.n(r, "2handmaxdam"),
				MinMisDam: tab.n(r, "minmisdam"), MaxMisDam: tab.n(r, "maxmisdam"),
				StrBonus: tab.n(r, "StrBonus"), DexBonus: tab.n(r, "DexBonus"),
				ReqStr: tab.n(r, "reqstr"), ReqDex: tab.n(r, "reqdex"),
				Stackable: tab.n(r, "stackable") == 1, MinStack: tab.n(r, "minstack"),
				MaxStack: tab.n(r, "maxstack"), SpawnStack: tab.n(r, "spawnstack"),
				HasInv: tab.n(r, "hasinv") == 1, GemSockets: tab.n(r, "gemsockets"),
				Throwable: tab.n(r, "throwable") == 1, Useable: tab.n(r, "useable") == 1,
				TwoHanded: tab.n(r, "2handed") == 1, OneOrTwoHanded: tab.n(r, "1or2handed") == 1,
				Cost: tab.n(r, "cost"), InvWidth: tab.n(r, "invwidth"), InvHeight: tab.n(r, "invheight"),
			}

			it.Items = append(it.Items, b)
			it.ByCode[code] = b
		}
	}

	return it
}

func TestLoadItemTables(t *testing.T) {
	it := loadItemTables(t)

	// 659 base items in the 1.14b tables: weapons, armor, misc.
	if len(it.Items) != 659 {
		t.Errorf("got %d base items, want 659", len(it.Items))
	}

	if b := it.ByCode["hax"]; b == nil || b.Class != 0 || b.Kind != KindWeapon {
		t.Errorf("hax = %+v", b)
	}

	if ty := it.Types["armo"]; ty == nil || ty.Index != 0x32 {
		t.Errorf("armo = %+v", ty)
	}

	if ty := it.Types["weap"]; ty == nil || ty.Index != 0x2d {
		t.Errorf("weap = %+v", ty)
	}

	if !it.IsA(it.ByCode["cap"], "armo") || it.IsA(it.ByCode["cap"], "weap") {
		t.Error("type ancestry of cap")
	}
}
