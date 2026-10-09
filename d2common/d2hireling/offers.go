package d2hireling

import (
	"strconv"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

const (
	// OffersShown is how many distinct names are offered at once (V).
	OffersShown = 10
	maxSlots    = 0x45
)

// Slot is one entry of the per-game offer table (16 bytes in the exe).
type Slot struct {
	NameID  int    // offset into NameFirst..NameLast: the value the d2s stores as name id
	Seed    uint32 // per-slot LCG seed, also the merc's d2s "id"
	Hired   bool
	Offered bool
}

// OfferTable is NPCSRV_AllocHireOfferTable's result for one vendor.
type OfferTable struct {
	Seller     int
	Difficulty int // 1-based
	Slots      []Slot
}

// nameRange splits "merc41" into prefix "merc" and number 41.
func splitKey(k string) (prefix string, n int, width int) {
	i := len(k)
	for i > 0 && k[i-1] >= '0' && k[i-1] <= '9' {
		i--
	}

	n, _ = strconv.Atoi(k[i:])

	return k[:i], n, len(k) - i
}

// NameCount is the number of names in the record's NameFirst..NameLast range
// (rogue 41, desert 21, sorceress 20, barbarian 67).
func NameCount(r *Record) int {
	_, a, _ := splitKey(r.NameFirst)
	_, b, _ := splitKey(r.NameLast)

	if b < a {
		return 0
	}

	return b - a + 1
}

// NameKey is the string.tbl key of name id nameID (the d2s offset); ids past
// the range wrap to the first name (V, header note).
func NameKey(r *Record, nameID int) string {
	prefix, first, width := splitKey(r.NameFirst)

	n := NameCount(r)
	if n > 0 {
		nameID %= n
	}

	num := strconv.Itoa(first + nameID)
	for len(num) < width {
		num = "0" + num
	}

	return prefix + num
}

// NewOfferTable builds the offer table of a seller: one slot per name with a
// per-slot seed from the game seed's LCG, then OffersShown distinct slots are
// marked offered by repeated bounded rolls walking forward to the next unused
// slot (V, hirelings.md). gameSeed is advanced.
func (t *Table) NewOfferTable(gameSeed *d2rand.Seed, seller, difficulty int) *OfferTable {
	cands := t.Candidates(seller, difficulty)
	if len(cands) == 0 {
		return nil
	}

	count := NameCount(cands[0])
	if count > maxSlots {
		count = maxSlots
	}

	o := &OfferTable{Seller: seller, Difficulty: difficulty, Slots: make([]Slot, count)}
	for i := range o.Slots {
		o.Slots[i] = Slot{NameID: i, Seed: gameSeed.Step()}
	}

	o.pick(gameSeed)

	return o
}

func (o *OfferTable) pick(gameSeed *d2rand.Seed) {
	n := len(o.Slots)
	if n == 0 {
		return
	}

	want := OffersShown
	if want > n {
		want = n
	}

	for i := range o.Slots {
		o.Slots[i].Offered = false
	}

	for picked := 0; picked < want; picked++ {
		i := int(gameSeed.Roll(int32(n)))
		for o.Slots[i].Offered {
			i = (i + 1) % n
		}

		o.Slots[i].Offered = true
	}
}

// Regenerate is HIRE_RegenerateOffersIfExhausted: once every offered slot is
// hired the table is rebuilt (new seeds and a new random ten). It reports
// whether it did.
func (o *OfferTable) Regenerate(t *Table, gameSeed *d2rand.Seed) bool {
	for _, s := range o.Slots {
		if s.Offered && !s.Hired {
			return false
		}
	}

	if n := t.NewOfferTable(gameSeed, o.Seller, o.Difficulty); n != nil {
		*o = *n
	}

	return true
}

// Offer is a priced offer for a slot (HIRE_GenerateOfferStats, V except where
// marked).
type Offer struct {
	Slot   int // index into OfferTable.Slots
	Rec    *Record
	Stats  Stats
	Cost   int
	NameID int
	Seed   uint32
}

// MakeOffer rolls the offer of a slot for an owner level: the slot seed seeds
// a private LCG; first a roll picks the SubType variant among the candidate
// rows, then a roll of 5 gives the level (owner level - 5..-1, min 2). The
// stats use the picked first-band row even when the level is higher (V per
// notes; the band is re-selected when the merc is created).
func (t *Table) MakeOffer(o *OfferTable, slot, ownerLevel int) (Offer, bool) {
	cands := t.Candidates(o.Seller, o.Difficulty)
	if len(cands) == 0 || slot < 0 || slot >= len(o.Slots) {
		return Offer{}, false
	}

	s := o.Slots[slot]
	rng := d2rand.New(s.Seed)
	rec := cands[rng.Roll(int32(len(cands)))]
	level := OfferLevel(ownerLevel, int(rng.Roll(5)))

	return Offer{
		Slot: slot, Rec: rec, Stats: rec.StatsAt(level), Cost: HireCost(rec, level),
		NameID: s.NameID, Seed: s.Seed,
	}, true
}

// Offered returns the slot indexes currently offered and not yet hired.
func (o *OfferTable) Offered() []int {
	var out []int

	for i, s := range o.Slots {
		if s.Offered && !s.Hired {
			out = append(out, i)
		}
	}

	return out
}
