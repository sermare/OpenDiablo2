package d2hireling

// Presentation rules of the hire dialog and of the mercenary panel: which sellers
// offer which hirelings, what a row of the hire list shows, what the hero must
// pay, which skills a merc has at a level, and what replacing or reviving does.
// Everything is derived from hireling.txt and the rules of hirelings.md
// (UI_OpenMercHireDialog 0x4b24c0, HIRE_ProcessHireOffer 0x574f00,
// HIRE_ServerHandleReviveMercenary 0x577a10); no game file is embedded.

// SellerFirstAct and the other sellers: the hire NPC of each act, taken from
// the Seller and Act columns of hireling.txt (Kashya act 1, Greiz act 2,
// Asheara act 3, Qual-Kehk act 5; act 4 has no hireling vendor: Tyrael only
// revives).
const (
	SellerGreiz   = 198 // 0xc6
	SellerAsheara = 252 // 0xfc
)

// SellerOfAct is the class id of the act's hire NPC (0 for act 4 and unknown acts).
func SellerOfAct(act int) int {
	switch act {
	case 1:
		return SellerKashya
	case 2:
		return SellerGreiz
	case 3:
		return SellerAsheara
	case 5:
		return SellerQualKehk
	}

	return 0
}

// ActOfSeller is the inverse of SellerOfAct (0 when the NPC hires nobody).
func ActOfSeller(seller int) int {
	for act := 1; act <= 5; act++ {
		if SellerOfAct(act) == seller && seller != 0 {
			return act
		}
	}

	return 0
}

// HireList is the set of rows a seller can offer in a difficulty (1-based): the
// first band of the (seller, difficulty) pair with its SubType variants. It is
// the pool the ten offers are rolled from.
func (t *Table) HireList(seller, difficulty int) []*Record { return t.Candidates(seller, difficulty) }

// SkillAt is one skill of a merc at a level.
type SkillAt struct {
	Name  string
	Level int
	Mode  int
}

// SkillsAt lists the skills a merc of the row has at level mlvl: the used
// slots whose skills.txt reqlevel (reqLevel, nil = no restriction) is reached
// by the merc, with their level (MERC_SetLevelStats 0x570690, V).
func (r *Record) SkillsAt(mlvl int, reqLevel func(name string) int) []SkillAt {
	var out []SkillAt

	for i, s := range r.Skills {
		if !s.Used() {
			break
		}

		if reqLevel != nil && reqLevel(s.Name) > mlvl {
			continue
		}

		out = append(out, SkillAt{Name: s.Name, Level: r.SkillLevel(i, mlvl), Mode: s.Mode})
	}

	return out
}

// MenuRow is one row of the hire list: the numbers of the four columns of the
// original dialog (which number sits in which column is UNVERIFIED there, so the
// row carries all of them) plus the label.
type MenuRow struct {
	Slot           int // index into OfferTable.Slots
	NameID         int
	NameKey        string
	Level          int
	HireDesc       string // the subtype label key (fire/cold/ltng/comb/def/off/...)
	SubType        string
	HP, Defense    int
	DmgMin, DmgMax int
	Resist         int
	Price          int
	Skills         []SkillAt
	HireID         int
	Class, Act     int
}

// HireMenu builds the rows of a seller's hire dialog for an owner level: one row
// per offered, not yet hired slot, in slot order (at most OffersShown).
func (t *Table) HireMenu(o *OfferTable, ownerLevel int, reqLevel func(string) int) []MenuRow {
	if o == nil {
		return nil
	}

	var rows []MenuRow

	for _, slot := range o.Offered() {
		of, ok := t.MakeOffer(o, slot, ownerLevel)
		if !ok {
			continue
		}

		rows = append(rows, MenuRow{
			Slot: slot, NameID: of.NameID, NameKey: NameKey(of.Rec, of.NameID), Level: of.Stats.Level,
			HireDesc: of.Rec.HireDesc, SubType: of.Rec.SubType,
			HP: of.Stats.MaxHP, Defense: of.Stats.Defense, DmgMin: of.Stats.DmgMin, DmgMax: of.Stats.DmgMax,
			Resist: of.Stats.Resist, Price: of.Cost,
			Skills: of.Rec.SkillsAt(of.Stats.Level, reqLevel),
			HireID: of.Rec.ID, Class: of.Rec.Class, Act: of.Rec.Act,
		})
	}

	return rows
}

// CanAfford reports whether gold covers a price.
func CanAfford(gold, price int) bool { return price >= 0 && gold >= price }

// Replacement describes what hiring a second merc does to the first (V, notes
// "Hiring" step 5): NPCSRV_PlaceHirelingForPlayer dismisses the old merc and the
// items it wore are discarded; nothing is refunded and there is no stash.
type Replacement struct {
	DiscardsItems bool
	Refund        int
}

// ReplaceTerms is the rule for dismissing or replacing a merc.
func ReplaceTerms() Replacement { return Replacement{DiscardsItems: true, Refund: 0} }

// ReviveQuote is what a dead merc costs to bring back and whether the hero can
// pay it and the NPC may do it.
type ReviveQuote struct {
	Cost    int
	Afford  bool
	Allowed bool
}

// QuoteRevive prices the revive of a level-mlvl merc at an NPC class for a hero
// with gold (cost min(50000, (lvl*lvl/2)*15), allow-list of 0x577a10).
func QuoteRevive(mlvl, gold, npcClass int) ReviveQuote {
	c := ReviveCost(mlvl)

	return ReviveQuote{Cost: c, Afford: CanAfford(gold, c), Allowed: CanRevive(npcClass)}
}

// Progress is the experience bar of the panel: the experience owned, the
// threshold of the current and of the next level and the percent into the level.
type Progress struct {
	Exp, Current, Next int
	Percent            int
	Max                bool
}

// ProgressOf computes the bar for a merc of a row at a level with exp points
// (Next is 0 at MaxLevel, as the stat 0x1e of MERC_SetLevelStats).
func (r *Record) ProgressOf(level int, exp uint32) Progress {
	p := Progress{Exp: int(exp), Current: int(ExpThreshold(level, r.ExpPerLvl))}
	if level >= MaxLevel {
		p.Max = true

		return p
	}

	p.Next = int(ExpThreshold(level+1, r.ExpPerLvl))
	if span := p.Next - p.Current; span > 0 {
		pc := (p.Exp - p.Current) * 100 / span
		switch {
		case pc < 0:
			pc = 0
		case pc > 100:
			pc = 100
		}

		p.Percent = pc
	}

	return p
}
