package d2drop

// Slice B: affixes. Magic (554710), rare (5bff00 -> 5bf8c0 / 5bfbd0), crafted
// (5bff40), tempered (5bf890), automagic (5bf5f0), charm path (554b60) and the
// pickers behind them (affix.go). Everything here is compared with the real
// game in create_b_test.go.
//
// Not ported: the random skill bonus of class items (staffmods: 5beea0, run at
// the end of the magic, rare and crafted rolls; it consumes the item
// generator) and the forced affix ids of cube recipes.

const (
	flagIdentified uint32 = 0x10
	jewelTypeIndex        = 0x3a // ItemTypes row of jewels (62b590 == 0x3a)
	maxRetries            = 250  // a clashing classic pick is retried 251 times
)

func (c *Creator) isJewel(st *itemState) bool {
	ty := c.Items.Type(st.base)

	return ty != nil && ty.Index == jewelTypeIndex
}

// HasStaffMods reports whether the game would give the item its random class
// skill bonus after the affixes (ItemTypes.StaffMods), which this port does
// not roll.
func (c *Creator) HasStaffMods(st *itemState) bool {
	ty := c.Items.Type(st.base)

	return ty != nil && ty.StaffMods != ""
}

// staffMods is ITEMGEN_RollStaffMods (5beea0). NOT PORTED.
func (c *Creator) staffMods(st *itemState) {}

// applyAll applies the properties of the affixes in the order of the rare
// and crafted rolls: prefix 0, suffix 0, prefix 1, ...
func (c *Creator) applyAll(st *itemState) {
	for i := 0; i < 3; i++ {
		c.Affixes.apply(st, st.prefix[i])
		c.Affixes.apply(st, st.suffix[i])
	}
}

// rollMagic is ITEMGEN_RollMagicAffixes (554710) for quality 4: a prefix
// behind the 50% gate, then a suffix behind the gate that is forced if there
// is no prefix. It returns false if the item got neither, in which case the
// caller falls back (and restores the generator).
func (c *Creator) rollMagic(st *itemState) bool {
	pre := c.pickAffix(st, pickReq{spawnCheck: true, apply: true, prefix: true})
	st.prefix[0] = pre

	suf := c.pickAffix(st, pickReq{spawnCheck: true, apply: true, force: pre == 0})
	st.suffix[0] = suf

	if pre == 0 && suf == 0 {
		return false
	}

	st.flags &^= flagIdentified
	c.staffMods(st)

	return true
}

// rollCharm is ITEMGEN_RollCharmAffixes (554b60), the path of normal quality
// charms: the same prefix and suffix roll as rollMagic.
func (c *Creator) rollCharm(st *itemState) bool {
	pre := c.pickAffix(st, pickReq{spawnCheck: true, apply: true, prefix: true})
	st.prefix[0] = pre

	suf := c.pickAffix(st, pickReq{spawnCheck: true, apply: true, force: pre == 0})
	st.suffix[0] = suf

	if pre == 0 && suf == 0 {
		return false // the game aborts here
	}

	st.flags &^= flagIdentified

	return true
}

// rollRare is ITEMGEN_RollRareAffixes (5bff00) for quality 6: false if the
// item's type cannot be rare or the roll found no names or no affixes.
func (c *Creator) rollRare(st *itemState) bool {
	if ty := c.Items.Type(st.base); ty == nil || !ty.Rare {
		return false
	}

	if st.req.Version >= 1 {
		return c.rollRareLoD(st)
	}

	return c.rollRareClassic(st)
}

// rollNames sets the two rare names (prefix name first, picked first).
func (c *Creator) rollNames(st *itemState) bool {
	p := c.pickRareName(st, true)
	s := c.pickRareName(st, false)

	if p == 0 || s == 0 {
		return false
	}

	st.rareNames = [2]int{p, s}

	return true
}

// rollRareLoD is ITEMGEN_RollRareAffixesLod (5bf8c0): 3 to 6 affixes (the
// table above; 3 or 4 on a jewel), each on a side chosen by a coin until one
// side is full (3) or has nothing left to pick; a failed pick costs no
// affix.
func (c *Creator) rollRareLoD(st *itemState) bool {
	if !c.rollNames(st) {
		return false
	}

	var count int

	if c.isJewel(st) {
		count = 3 + int(st.item.Roll(2))
	} else {
		count = rareAffixCounts[st.item.Roll(8)]
	}

	var np, ns int

	prefixDone, suffixDone := false, false

loop:
	for i := 0; i < count; i++ {
		var suffix bool

		switch {
		case prefixDone && suffixDone:
			break loop
		case prefixDone:
			suffix = true
		case suffixDone:
		default:
			suffix = st.item.Chance()
		}

		id := c.pickAffix(st, pickReq{spawnCheck: true, force: true, prefix: !suffix})

		switch {
		case id == 0 && suffix:
			suffixDone = true
			i--
		case id == 0:
			prefixDone = true
			i--
		case suffix:
			st.suffix[ns] = id
			ns++
			suffixDone = ns >= 3
		default:
			st.prefix[np] = id
			np++
			prefixDone = np >= 3
		}
	}

	if np == 0 && ns == 0 {
		return false
	}

	st.flags &^= flagIdentified
	c.applyAll(st)
	c.staffMods(st)

	return true
}

// addSlot picks an affix for the next prefix or suffix slot, retrying while
// it clashes with the same list (classic items and crafted items). It is the
// shared body of rollRareClassic and rollCrafted. skipEmpty says whether a
// failed pick leaves the slot alone (rare) or fills it with 0 (crafted).
func (c *Creator) addSlot(st *itemState, suffix, skipEmpty bool, n *int) {
	list := &st.prefix
	if suffix {
		list = &st.suffix
	}

	for tries := 0; ; tries++ {
		id := c.pickAffix(st, pickReq{spawnCheck: true, force: true, prefix: !suffix})
		if id == 0 && skipEmpty {
			return
		}

		clash := false

		for _, have := range list {
			old := c.Affixes.row(have)
			if old == nil {
				continue
			}

			if cur := c.Affixes.row(id); id == have || (cur != nil && cur.Group == old.Group) {
				clash = true

				break
			}
		}

		if !clash {
			list[*n] = id
			*n++

			return
		}

		if tries > maxRetries {
			list[*n] = 0

			return
		}
	}
}

// rollRareClassic is ITEMGEN_RollRareAffixesClassic (5bfbd0): 4 to 6 (3 or 4
// on a jewel) rolls, the side chosen by a coin (forced when the other side is
// full) and a clashing pick retried up to 251 times.
func (c *Creator) rollRareClassic(st *itemState) bool {
	if !c.rollNames(st) {
		return false
	}

	var count int

	if c.isJewel(st) {
		count = 3 + int(st.item.Roll(2))
	} else {
		count = 4 + int(st.item.Roll(3))
	}

	var np, ns int

	for ; count > 0; count-- {
		coin := st.item.Chance()

		switch {
		case np == 3:
			c.addSlot(st, true, true, &ns)
		case ns == 3, !coin:
			c.addSlot(st, false, true, &np)
		default:
			c.addSlot(st, true, true, &ns)
		}
	}

	if np == 0 && ns == 0 {
		return false
	}

	st.flags &^= flagIdentified
	c.applyAll(st)
	c.staffMods(st)

	return true
}

// rollCrafted is ITEMGEN_RollCraftedAffixes (5bff40) for quality 8: the item
// level of the request gives 1 to 4 rolls (1 up to level 30, 2 to 50, 3 to 70,
// else 4), at least rand(5) of them; otherwise as the classic rare, except
// that a failed pick still takes a slot.
func (c *Creator) rollCrafted(st *itemState) bool {
	if !c.rollNames(st) {
		return false
	}

	tier := 1

	switch il := st.req.ILvl; {
	case il > 70:
		tier = 4
	case il > 50:
		tier = 3
	case il > 30:
		tier = 2
	}

	count := maxInt(tier, int(st.item.Roll(5)))

	var np, ns int

	for ; count > 0; count-- {
		coin := st.item.Chance()

		switch {
		case np == 3:
			c.addSlot(st, true, false, &ns)
		case ns == 3, !coin:
			c.addSlot(st, false, false, &np)
		default:
			c.addSlot(st, true, false, &ns)
		}
	}

	st.flags &^= flagIdentified
	c.applyAll(st)
	c.staffMods(st)

	return true
}

// rollTempered is the tempered quality (9) of ApplyQualityToItem: only the two
// rare names are picked (5bf890), no affixes.
func (c *Creator) rollTempered(st *itemState) bool {
	st.rareNames = [2]int{}

	return c.rollNames(st)
}

// pickAutomagic is ITEMGEN_PickAutomagic (5bf5f0) for a LoD item: the affix of
// the automagic group named by the base item's Auto Prefix column, picked
// without the gate or the spawnable test. It stores the id in the item,
// applies its properties (see AffixTables.OnApply) and returns it, 0 if the
// base has no group or nothing fits.
func (c *Creator) pickAutomagic(st *itemState) int {
	group := st.base.AutoPrefix
	if group == 0 {
		return 0
	}

	id := c.pickAffix(st, pickReq{force: true, prefix: true, autoGroup: group})
	if id == 0 {
		return 0
	}

	st.auto = id
	c.Affixes.apply(st, id)

	return id
}
