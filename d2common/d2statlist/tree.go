package d2statlist

// The exe keeps stats in a tree of lists (itemgen.md section 2, gap 10): an
// owner unit has a list, every item, set bonus and state attaches its own list
// under it, and attaching adds the child's values into the parent's aggregate
// (STATS_AttachStatList 0x627160), skipping the stats whose ItemStatCost row
// says they do not apply to the parent. A read of a unit stat (0x625620 ->
// 0x625140) is the unit's own value plus what its attached children added.
// VERIFIED from the notes: owner type and id, the flag word (0x80000000 base
// unit list, 0x2000 set bonus / state list, 0x40000000 attached), the
// parent / sibling links and the skip rule. UNVERIFIED: the exact order in
// which the exe folds percent conversions (done by Compute after the sums).

// ListFlags are the flag bits of a stat list.
type ListFlags uint32

// The flag bits of a stat list (VERIFIED, itemgen.md).
const (
	FlagApplied  ListFlags = 0x40000000 // attached to a parent
	FlagBaseUnit ListFlags = 0x80000000 // the base list of a unit
	FlagSetState ListFlags = 0x2000     // a set bonus or state list
)

// Owner says whose list it is (unit type and id as in the exe's list header).
type Owner struct {
	Type, ID int
}

// Tree is one stat list of the tree.
type Tree struct {
	Owner Owner
	Flags ListFlags

	own      *List // the list's own stats
	agg      *List // own plus the attached children
	parent   *Tree
	children []*Tree
	second   []*Tree // set bonus / state lists (the exe's second child list)
	// skip reports the stats that do not apply to the parent
	skip func(id int) bool
}

// NewTree makes an empty stat list for an owner.
func NewTree(owner Owner, flags ListFlags) *Tree {
	return &Tree{Owner: owner, Flags: flags, own: NewList(), agg: NewList()}
}

// Add adds to the list's own stat; an attached list passes the change up to
// the parents' aggregates.
func (t *Tree) Add(id, param int, delta int64) {
	t.own.Add(id, param, delta)
	t.propagate(id, param, delta)
}

// AddProps adds properties.
func (t *Tree) AddProps(props []Prop) {
	for _, p := range props {
		t.Add(p.ID, p.Param, p.Value)
	}
}

// skipFor reports whether the stat is one that does not apply to a parent,
// asking this list and then its parents for the rule.
func (t *Tree) skipFor(id int) bool {
	for n := t; n != nil; n = n.parent {
		if n.skip != nil {
			return n.skip(id)
		}
	}

	return false
}

// propagate passes a change of the list's own stat up (the exe walk of
// STATS_ApplyStatDeltaToList 0x626c80, VERIFIED reading): a set bonus / state
// list (flag 0x2000) never passes anything up; otherwise every parent gets the
// delta in turn, and the walk ends after a parent that is a set bonus / state
// list, or, for a stat that does not apply to the parent, after a parent that
// is itself attached.
func (t *Tree) propagate(id, param int, delta int64) {
	t.agg.Add(id, param, delta)

	if t.Flags&FlagSetState != 0 {
		return
	}

	t.walkUp(id, param, delta, t.skipFor(id))
}

func (t *Tree) walkUp(id, param int, delta int64, skipStat bool) {
	for n := t.parent; n != nil; n = n.parent {
		n.agg.Add(id, param, delta)

		if n.Flags&FlagSetState != 0 {
			return
		}

		if skipStat && n.Flags&FlagApplied != 0 {
			return
		}
	}
}

// Attach links the list under a parent and adds its stats to the parents
// (STATS_AttachStatList 0x627160, VERIFIED reading). The parent must be a unit
// list (flag 0x80000000). The list is first detached from any old parent;
// attaching a list under itself or under one of its own descendants does
// nothing. A set bonus / state list (0x2000) goes on the parent's second
// child list and adds nothing; other lists add every stat except those named
// by skip (the ItemStatCost "not applied to parent" bit), which is how the
// exe's attach loop treats them. nil skip: all stats apply.
func (t *Tree) Attach(parent *Tree, skip func(id int) bool) {
	if parent == nil || parent.Flags&FlagBaseUnit == 0 {
		return
	}

	for n := parent; n != nil; n = n.parent {
		if n == t {
			return
		}
	}

	t.Detach()
	t.skip = skip

	if t.Flags&FlagSetState != 0 {
		t.parent = parent
		parent.second = append(parent.second, t)

		return
	}

	t.parent = parent
	t.Flags |= FlagApplied
	parent.children = append(parent.children, t)

	for _, p := range t.agg.Props() {
		if skip != nil && skip(p.ID) {
			continue
		}

		t.walkUp(p.ID, p.Param, p.Value, false)
	}
}

// Detach removes the list from its parent and takes its stats out of the
// parents' aggregates again. UNVERIFIED: the exe's detach (0x627160 callee
// STATS_DetachStatList) was not read; this undoes what Attach added.
func (t *Tree) Detach() {
	if t.parent == nil {
		return
	}

	parent := t.parent

	if t.Flags&FlagSetState != 0 {
		parent.second = removeTree(parent.second, t)
	} else {
		for _, p := range t.agg.Props() {
			if t.skip != nil && t.skip(p.ID) {
				continue
			}

			t.walkUp(p.ID, p.Param, -p.Value, false)
		}

		parent.children = removeTree(parent.children, t)
	}

	t.parent, t.skip = nil, nil
	t.Flags &^= FlagApplied
}

func removeTree(l []*Tree, t *Tree) []*Tree {
	for i, c := range l {
		if c == t {
			return append(l[:i], l[i+1:]...)
		}
	}

	return l
}

// Get reads a stat with parameter 0: the own value plus the attached children
// (the unit stat read).
func (t *Tree) Get(id int) int64 { return t.agg.Get(id) }

// GetParam reads a stat with a parameter.
func (t *Tree) GetParam(id, param int) int64 { return t.agg.GetParam(id, param) }

// SecondChildren is the number of set bonus / state lists on the second list.
func (t *Tree) SecondChildren() int { return len(t.second) }

// Children is the number of attached lists.
func (t *Tree) Children() int { return len(t.children) }

// List returns a copy of the aggregate.
func (t *Tree) List() *List {
	out := NewList()
	out.Merge(t.agg)

	return out
}
