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

func (t *Tree) propagate(id, param int, delta int64) {
	for n := t; n != nil; n = n.parent {
		n.agg.Add(id, param, delta)

		if n.parent != nil && n.skip != nil && n.skip(id) {
			return // the stat stays below this list
		}
	}
}

// Attach links the list under a parent and adds its aggregate to the parents.
// skip names the stats that do not apply to the parent (nil: all apply).
// Attaching an attached list is a no-op.
func (t *Tree) Attach(parent *Tree, skip func(id int) bool) {
	if t.parent != nil || parent == nil || parent == t {
		return
	}

	t.parent, t.skip = parent, skip
	t.Flags |= FlagApplied
	parent.children = append(parent.children, t)

	for _, p := range t.agg.Props() {
		parent.fold(t, p)
	}
}

// fold adds a child's stat to this list and its parents.
func (t *Tree) fold(child *Tree, p Prop) {
	if child.skip != nil && child.skip(p.ID) {
		return
	}

	t.agg.Add(p.ID, p.Param, p.Value)

	if t.parent != nil {
		t.parent.fold(t, p)
	}
}

// Detach removes the list from its parent and takes its stats out of the
// parents' aggregates again.
func (t *Tree) Detach() {
	parent := t.parent
	if parent == nil {
		return
	}

	for _, p := range t.agg.Props() {
		if t.skip != nil && t.skip(p.ID) {
			continue
		}

		p.Value = -p.Value
		parent.fold(&Tree{skip: nil}, p)
	}

	for i, c := range parent.children {
		if c == t {
			parent.children = append(parent.children[:i], parent.children[i+1:]...)

			break
		}
	}

	t.parent, t.skip = nil, nil
	t.Flags &^= FlagApplied
}

// Get reads a stat with parameter 0: the own value plus the attached children
// (the unit stat read).
func (t *Tree) Get(id int) int64 { return t.agg.Get(id) }

// GetParam reads a stat with a parameter.
func (t *Tree) GetParam(id, param int) int64 { return t.agg.GetParam(id, param) }

// Children is the number of attached lists.
func (t *Tree) Children() int { return len(t.children) }

// List returns a copy of the aggregate.
func (t *Tree) List() *List {
	out := NewList()
	out.Merge(t.agg)

	return out
}
