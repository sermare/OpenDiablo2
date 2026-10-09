package d2equip

// Mercenary equipment rules. Source: Game.exe 1.14b, SERVER_HandleC2S61_GiveItemToMercenary
// 0x54b230 -> NPCSRV_ValidateMercenaryItemEquip 0x54b040 (the type rules, hard coded per
// monster class), INV_CheckItemRequirements 0x62ebf0 (stat / level / class) and 0x54ace0
// (slot choice and swap). Item type ids in the exe are ItemTypes rows with the
// "Expansion" marker row dropped, so ids from 0x3a on are the file row minus one:
// exe 0x47 is Primal Helm (phlm), 0x4c Healing Potion (hpot), 0x50 Antidote (apot) and
// 0x51 Thawing (wpot). All of this is VERIFIED from the decompile unless a comment says
// UNVERIFIED.

// Monster class ids (monstats rows) of the hireling classes.
const (
	MercRogue      = 0x10f // 271, hireling.txt Class of the Rogue Scout
	MercGuard      = 0x152 // 338, Desert Mercenary
	MercIronWolf   = 0x167 // 359, Eastern Sorceror
	MercBarbarian  = 0x230 // 560, Act 5 Hireling 1hs (hireling.txt uses 561 only)
	MercBarbarian2 = 0x231 // 561, Act 5 Hireling 2hs
)

// MercLocs are the body locations a mercenary has (inventory.txt record 13: head,
// torso, right hand, left hand). No neck, rings, belt, feet or gloves.
var MercLocs = []Loc{LocHead, LocTorso, LocRightHand, LocLeftHand} //nolint:gochecknoglobals // static

// MercHasLoc reports whether a mercenary has the body location.
func MercHasLoc(l Loc) bool {
	for _, m := range MercLocs {
		if m == l {
			return true
		}
	}

	return false
}

// Merc is the mercenary that receives an item: its class and its own stats (it has
// no hero class; strength, dexterity and level are those of hireling.txt, plus
// whatever its gear already gives, see MercGive).
type Merc struct {
	Class    int // monstats class id
	Level    int
	Str, Dex int
}

func isBarbarian(class int) bool { return class == MercBarbarian || class == MercBarbarian2 }

// IsMercClass reports whether class is one of the hireling monster classes.
func IsMercClass(class int) bool {
	switch class {
	case MercRogue, MercGuard, MercIronWolf, MercBarbarian, MercBarbarian2:
		return true
	}

	return false
}

// MercConsumes reports whether the item is a potion a merc drinks when given it
// (healing, which includes rejuvenation by its Equiv chain, antidote, thawing). The
// item is consumed and nothing is equipped. The healing effect itself is UNVERIFIED
// (FUN_005bce10); mana and stamina potions are not accepted.
func (r Rules) MercConsumes(it *Item) bool {
	t := r.Types

	return t.IsA(it.Type, "hpot") || t.IsA(it.Type, "apot") || t.IsA(it.Type, "wpot")
}

// MercAccepts is the per-class type rule of 0x54b040: body armor (incl. cloaks) and helms
// (incl. circlets) for every class, plus
//   - Rogue: bows (not crossbows);
//   - Desert Guard: spears and polearms;
//   - Iron Wolf: shields (the plain "shie" type, not class shields) and one-handed swords;
//   - Barbarian 1hs (0x230): one-handed axes, Barbarian 2hs (0x231): any sword; both
//     take primal helms (the class rule lets only them through).
//
// It does not look at stats or at the item's class restriction.
func (r Rules) MercAccepts(class int, it *Item) bool {
	t := r.Types
	if t.IsA(it.Type, "tors") || t.IsA(it.Type, "helm") {
		return true
	}

	switch class {
	case MercRogue:
		return t.IsA(it.Type, "bow")
	case MercGuard:
		return t.IsA(it.Type, "spea") || t.IsA(it.Type, "pole")
	case MercIronWolf:
		return t.IsA(it.Type, "shie") || (t.IsA(it.Type, "swor") && !it.TwoHanded)
	case MercBarbarian:
		return t.IsA(it.Type, "phlm") || (t.IsA(it.Type, "axe") && !it.TwoHanded)
	case MercBarbarian2:
		return t.IsA(it.Type, "phlm") || t.IsA(it.Type, "swor")
	}

	return false
}

// MercClassOK is the class restriction of INV_CheckItemRequirements for a monster unit:
// an unrestricted item is fine; a barbarian class item only for the barbarian
// mercenary; every other class's item is refused.
func (r Rules) MercClassOK(class int, it *Item) bool {
	c := r.Types.ClassOf(it.Type)

	return c == "" || (c == ClassBarbarian && isBarbarian(class))
}

// MercSlot is the body location the item goes to (0x54ace0): the type's first body
// location, except that an Iron Wolf puts a shield into the second one (left hand).
// ok is false when the merc has no such location.
func (r Rules) MercSlot(class int, it *Item) (Loc, bool) {
	locs := r.Types.Locs(it.Type)
	if len(locs) == 0 {
		return LocNone, false
	}

	l := locs[0]
	if class == MercIronWolf && r.Types.IsA(it.Type, "shie") && len(locs) > 1 {
		l = locs[1]
	}

	l = l.Primary()

	return l, MercHasLoc(l)
}

// MercResult is the verdict of giving an item to a mercenary.
type MercResult struct {
	Decision
	// Loc is the body location the item took.
	Loc Loc
	// Consumed is set for a potion: it is used up, nothing changes in the body.
	Consumed bool
	// Returned is the item that was in the slot and goes to the player's cursor.
	Returned *Item
}

// MercGive follows the server path of giving the cursor item to a mercenary:
//  1. the item must be identified and not broken;
//  2. potions of the accepted kinds are consumed;
//  3. the class's type rule (MercAccepts) and the requirement check against the MERC's
//     own strength, dexterity and level (reqstr/reqdex reduced by the item's
//     requirement percent, ethereal -10; level via Item.ReqLevel) and the class rule;
//  4. the item to be replaced is taken off, the requirements are checked again with
//     the merc's stats as they are WITHOUT that item (statsWithout, may be nil to reuse
//     m), and when the new item fails the old one stays and nothing changes;
//     otherwise the old item is returned to the player.
//
// body is changed on success. Distance to the merc and quest items (checked by the
// caller of the packet handler) are not modelled.
func (r Rules) MercGive(m Merc, body map[Loc]*Item, it *Item, statsWithout func(removed *Item) Merc) MercResult {
	if it == nil {
		return MercResult{Decision: refuse(ReasonBodyLoc, "no item")}
	}

	if !it.Identified {
		return MercResult{Decision: refuse(ReasonUnidentified, "item is not identified")}
	}

	if it.Broken() {
		return MercResult{Decision: refuse(ReasonBroken, "item is broken")}
	}

	if r.MercConsumes(it) {
		return MercResult{Decision: Decision{OK: true, Reason: ReasonOK}, Consumed: true}
	}

	if !r.MercAccepts(m.Class, it) {
		return MercResult{Decision: refuse(ReasonBodyLoc, "a %s mercenary cannot use %s (%s)", mercName(m.Class), it.Code, it.Type)}
	}

	loc, ok := r.MercSlot(m.Class, it)
	if !ok {
		return MercResult{Decision: refuse(ReasonBodyLoc, "%s has no slot on a mercenary", it.Code)}
	}

	if !r.MercClassOK(m.Class, it) {
		return MercResult{Decision: Decision{Reason: ReasonClass, Detail: "item is restricted to another class"}}
	}

	if req := Check(Hero{Str: m.Str, Dex: m.Dex, Level: m.Level}, it, nil); !req.OK() {
		return MercResult{Decision: Decision{Reason: req.Reason(), Detail: req.Detail()}}
	}

	old := body[loc]
	if old != nil {
		m2 := m
		if statsWithout != nil {
			m2 = statsWithout(old)
		}

		if req := Check(Hero{Str: m2.Str, Dex: m2.Dex, Level: m2.Level}, it, nil); !req.OK() {
			return MercResult{Decision: Decision{Reason: req.Reason(), Detail: req.Detail()}, Loc: loc}
		}
	}

	body[loc] = it

	return MercResult{Decision: Decision{OK: true, Reason: ReasonOK}, Loc: loc, Returned: old}
}

func mercName(class int) string {
	switch class {
	case MercRogue:
		return "Rogue"
	case MercGuard:
		return "Desert"
	case MercIronWolf:
		return "Iron Wolf"
	case MercBarbarian, MercBarbarian2:
		return "Barbarian"
	}

	return "unknown"
}
