package d2drop

import (
	"os"
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
