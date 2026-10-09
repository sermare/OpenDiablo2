package d2vendor

import (
	"sort"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2trade"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Vendor describes one town vendor.
type Vendor struct {
	// Name is the display name and the key of the column prefix in the item
	// tables ("Akara" for AkaraMin, AkaraMax, ...).
	Name string
	// NPC is the row of npc.txt (lower case monstats id).
	NPC string
	// ClassID is the monstats class id.
	ClassID int
	// Repairs: the vendor repairs items (price code: Charsi 154, Fara 178,
	// Hratli 253, Halbu 257, Larzuk 511; VERIFIED in the notes).
	Repairs bool
}

// Act1 lists the Act 1 vendors that have a trade window. Gheed's gamble stock
// is not implemented (the gamble row of his menu still logs "not implemented").
//
//nolint:gochecknoglobals // static lookup data
var Act1 = []Vendor{
	{Name: "Akara", NPC: "akara", ClassID: 148},
	{Name: "Charsi", NPC: "charsi", ClassID: 154, Repairs: true},
	{Name: "Gheed", NPC: "gheed", ClassID: 147},
}

// ByClassID finds a vendor by monstats class id.
func ByClassID(id int) (Vendor, bool) {
	for _, v := range Act1 {
		if v.ClassID == id {
			return v, true
		}
	}

	return Vendor{}, false
}

// ByName finds a vendor by display name (case insensitive).
func ByName(name string) (Vendor, bool) {
	for _, v := range Act1 {
		if strings.EqualFold(v.Name, name) {
			return v, true
		}
	}

	return Vendor{}, false
}

// BasesFor extracts the vendor's base items from the item tables, sorted by
// code (the original's table order is UNVERIFIED). Items the vendor has no
// columns for are left out.
func BasesFor(rec *d2records.RecordManager, v Vendor) []Base {
	codes := make([]string, 0, len(rec.Item.All))
	for code := range rec.Item.All {
		codes = append(codes, code)
	}

	sort.Strings(codes)

	bases := make([]Base, 0)

	for _, code := range codes {
		icr := rec.Item.All[code]

		p := icr.Vendors[v.Name]
		if p == nil || (p.Max == 0 && p.MagicMax == 0) {
			continue
		}

		tr := rec.Item.Types[icr.Type]
		gear := icr.Source == d2enum.InventoryItemTypeWeapon || icr.Source == d2enum.InventoryItemTypeArmor

		bases = append(bases, Base{
			Code:       code,
			ReqLevel:   icr.RequiredLevel,
			W:          icr.InventoryWidth,
			H:          icr.InventoryHeight,
			Vendor:     Params{Min: p.Min, Max: p.Max, MagicMin: p.MagicMin, MagicMax: p.MagicMax, MagicLevel: p.MagicLevel},
			Permanent:  icr.PermStoreItem,
			Gear:       gear,
			CanBeMagic: tr != nil && !tr.Normal && gear,
			Ammo:       tr != nil && tr.Quiver != "",
			MaxStack:   icr.MaxStack,
		})
	}

	return bases
}

const fixedPoint = 1024.0

// NPCPricing converts an npc.txt row into the multipliers of d2trade. The
// loader keeps the file's "sell mult" column as what the player pays and
// "buy mult" as what the vendor pays (VERIFIED in the notes: loader order is
// swapped relative to the file). quests may be nil (no quest overrides then;
// the engine does not yet hold the quest record in the game screen).
// difficulty indexes MaxBuy.
func NPCPricing(rec *d2records.RecordManager, v Vendor, quests *d2s.QuestRecord) d2trade.NPC {
	row := rec.NPCs[v.NPC]
	if row == nil {
		return d2trade.NPC{PlayerPays: 1024, VendorPays: 1024, Repair: 1024}
	}

	n := d2trade.NPC{
		PlayerPays: int(row.Multipliers.Sell*fixedPoint + 0.5),
		VendorPays: int(row.Multipliers.Buy*fixedPoint + 0.5),
		Repair:     int(row.Multipliers.Repair*fixedPoint + 0.5),
		MaxBuy:     [3]int{row.MaxBuy.Normal, row.MaxBuy.Nightmare, row.MaxBuy.Hell},
	}

	// Quest group overrides, keyed by quest slot. The record loader keeps
	// "questbuymult" as Buy and "questsellmult" as Sell; by the same swap as
	// above the player-pays side uses "questsellmult" (UNVERIFIED for the
	// quest columns: the notes only state it for the main columns).
	flags := make([]int, 0, len(row.QuestMultipliers))
	for f := range row.QuestMultipliers {
		flags = append(flags, f)
	}

	sort.Ints(flags)

	for i, f := range flags {
		if i >= len(n.Quest) {
			break
		}

		m := row.QuestMultipliers[f]
		n.Quest[i] = d2trade.Quest{
			Active: quests != nil && (quests.Get(f, 0) || quests.Get(f, 1)),
			Buy:    int(m.Sell*fixedPoint + 0.5),
			Sell:   int(m.Buy*fixedPoint + 0.5),
			Repair: int(m.Repair*fixedPoint + 0.5),
		}
	}

	return n
}
