package drlgpop

// Room population from the preset units of a room: FUN_00553a70 (which nodes
// are created and in which order) and FUN_0054c470 (what a monster node
// means). Verified by reading the code (no oracle for the leaf creators, see
// the notes of each function); the node lists themselves are verified against
// the emulator (oracle_test.go).

// Ctx carries the game state the room population reads.
type Ctx struct {
	// LevelID is the level of the room.
	LevelID int
	// Difficulty is the game difficulty 0..2 (game+0x6d).
	Difficulty int
	// Quest31 is FUN_00542360(game, 0x1f): quest flag 31 of the game, which
	// swaps two Act 5 monster classes in level 110 (Worldstone Keep... the
	// level with id 0x6e).
	Quest31 bool
	// Quest32 is FUN_00542360(game, 0x20) (class 0x1b2 only).
	Quest32 bool
	// Counter is game+0x1de4 and GameWord game+0x7c, used by level 0x86 only.
	Counter  *int
	GameWord uint32
}

// Request is one unit the population asks for, in creation order. X and Y are
// level subtile coordinates.
type Request struct {
	Node    Node
	X, Y    int
	Monster Monster // set for Node.Kind == KindMonster
}

// Levels with special rules in FUN_00553a70 (ids 0x85..0x88).
const (
	levelRules85 = 0x85
	levelRules86 = 0x86
	levelRules87 = 0x87
	levelRules88 = 0x88
)

// Requests returns what FUN_00553a70 creates for a room whose preset nodes
// (room relative, in the room's list order, i.e. reverse file order) are
// given. ox, oy is the room origin in tiles. Non-monsters come first, then
// the monsters (not at all on level 0x88); nodes with DS1 flag 1 are never
// created here.
func Requests(nodes []Node, ox, oy int, c *Ctx, nm *Names) []Request {
	var out []Request

	create := func(n Node) {
		r := Request{Node: n, X: n.X + ox*Subtile, Y: n.Y + oy*Subtile}
		if n.Kind == KindMonster {
			r.Monster = ResolveMonster(n.ID, c, nm)
		}

		out = append(out, r)
	}

	for _, n := range nodes {
		if n.Kind == KindMonster || n.Flags&1 != 0 {
			continue
		}

		switch c.LevelID {
		case levelRules88:
			if !(n.Kind == KindObject && (n.ID == 0x10c || n.ID == 0x1a || n.ID == 0x10d)) {
				create(n)
			}
		case levelRules85, levelRules87:
			if !(n.Kind == KindObject && n.ID == 0x18d) {
				create(n)
			}
		case levelRules86:
			if n.Kind == KindObject {
				if n.ID != 0x192 {
					*c.Counter++

					if uint32(*c.Counter) != c.GameWord%3+3 {
						create(n)
					}
				}

				continue
			}

			create(n)
		default:
			create(n)
		}
	}

	if c.LevelID != levelRules88 {
		for _, n := range nodes {
			if n.Kind == KindMonster && n.Flags&1 == 0 {
				create(n)
			}
		}
	}

	return out
}

// MonKind says what a monster node turns into.
type MonKind int

// Monster node results.
const (
	// MonNothing: the node creates nothing (markers like place_nothing,
	// place_group*, nests, talking npcs; the group markers only exist for the
	// DS1 filter).
	MonNothing MonKind = iota
	// MonClass: one monster of Class.
	MonClass
	// MonSuper: super unique Super (superuniques.txt row).
	MonSuper
	// MonUniquePack (place_unique_pack): a unique of a class drawn from the
	// level's monster palette (FUN_005bba00 mode 1) with its minions.
	MonUniquePack
	// MonChampion (place_champion): a monster of a palette class, champion
	// flag 0x10 set, plus minions (FUN_0054c050).
	MonChampion
	// MonGroup (place_impgroup / place_miniongroup, normal difficulty only):
	// FUN_0054bf10 spawns a group of Class.
	MonGroup
)

// Monster is the result of FUN_0054c470 for one node.
type Monster struct {
	Kind  MonKind
	Class int
	Super int
	// Mode is the collision mode handed to the placement (0, 4, 8 or 0xc) and
	// Retry4 says the placement is retried with mode 4 when it fails.
	Mode   int
	Retry4 bool
	// Late marks classes whose unit gets flag 0x2000000 after creation.
	Late bool
}

// ResolveMonster is FUN_0054c470 for a monster node id (the in-memory id
// from ResolveDS1).
func ResolveMonster(id int, c *Ctx, nm *Names) Monster {
	if id < 0 {
		return Monster{}
	}

	if id < nm.MonstatsCount {
		return resolveClass(id, c, nm)
	}

	rel := id - nm.MonstatsCount
	if rel < nm.SuperCount {
		return Monster{Kind: MonSuper, Super: rel}
	}

	return resolvePlace(rel-nm.SuperCount, c, nm)
}

// Class ids the placement rules know by number.
const (
	classFallen1       = 0x13
	classFallen2       = 0x14
	classFallen3       = 0x15
	classFallenShaman1 = 0x3a
	classFallenShaman2 = 0x3b
	classFallenShaman3 = 0x3c
	classFetish        = 0x8d
	classFetishShaman  = 0x116
)

func resolveClass(class int, c *Ctx, nm *Names) Monster {
	const level110 = 0x6e

	// Act 5 swaps once quest flag 31 is set (level 110 only).
	if c.Quest31 && c.LevelID == level110 {
		switch class {
		case 0x1f2:
			class = pickIfExists(0x1f3, nm)
		case 0x205:
			class = pickIfExists(0x206, nm)
		}
	}

	if c.Difficulty > 0 && c.LevelID == level110 && (class == 0x1c5 || class == 0x211) {
		return Monster{}
	}

	if class < 0 {
		return Monster{}
	}

	m := Monster{Kind: MonClass, Class: class, Retry4: true}

	// 0x1b2 is placed one subtile to the left and uses mode 0xc until quest
	// flag 32 is set (FUN_0054c300).
	if class == 0x1b2 && !c.Quest32 {
		m.Mode = 0xc
	}

	return m
}

func pickIfExists(class int, nm *Names) int {
	if class < nm.MonstatsCount {
		return class
	}

	return -1
}

// tier is FUN_0054c110: fallen and fallen shamans get a stronger class in a
// few Act 1 levels (levels 6..16, indexed by level-6).
func tier(class, level int, nm *Names) int {
	if level < 6 || level > 16 {
		return class
	}

	i := level - 6

	switch class {
	case classFallen1:
		switch [11]int{0, 1, 2, 2, 2, 2, 1, 2, 2, 2, 1}[i] {
		case 0:
			return pickOrSame(classFallen2, class, nm)
		case 1:
			return pickOrSame(classFallen3, class, nm)
		}
	case classFallenShaman1:
		switch [11]int{0, 0, 2, 2, 2, 2, 1, 2, 2, 2, 1}[i] {
		case 0:
			return pickOrSame(classFallenShaman2, class, nm)
		case 1:
			return pickOrSame(classFallenShaman3, class, nm)
		}
	}

	return class
}

func pickOrSame(c, same int, nm *Names) int {
	if c < nm.MonstatsCount {
		return c
	}

	return same
}

// resolvePlace handles the monplace.txt codes (the switch of FUN_0054c470).
// i is the index into monplace.txt.
func resolvePlace(i int, c *Ctx, nm *Names) Monster {
	switch i {
	case 2: // place_unique_pack
		return Monster{Kind: MonUniquePack}
	case 3: // place_champion
		return Monster{Kind: MonChampion}
	case 4: // place_rogue_warner
		return Monster{Kind: MonClass, Class: 0x10a}
	case 5: // place_bloodraven
		return Monster{Kind: MonClass, Class: pickIfExists(0x10b, nm)}
	case 8: // place_tightspotboss
		return Monster{Kind: MonClass, Class: 0x11c, Mode: 8}
	case 17, 18, 22, 23, 29, 30, 31, 32: // fallen, fallenshaman, fetish, fetishshaman, dead*
		return resolveSpawnClass(i, c, nm)
	case 24, 26: // impgroup, miniongroup: normal difficulty only
		if c.Difficulty > 0 {
			return Monster{}
		}

		class := 0
		if i == 24 {
			class = 0x1ec
			if c.LevelID == 0x6e {
				class = 0x211
			}
		} else {
			class = 0x1c5
		}

		return Monster{Kind: MonGroup, Class: class}
	}

	return Monster{}
}

// resolveSpawnClass is the fallen / fetish / dead* cases of the switch: the
// class comes from a constant, fallen and shamans go through the level tier,
// the dead* classes use mode 0xc and flag the unit afterwards.
func resolveSpawnClass(i int, c *Ctx, nm *Names) Monster {
	m := Monster{Kind: MonClass, Retry4: true}

	switch i {
	case 17:
		m.Class = classFallen1
	case 18:
		m.Class = classFallenShaman1
	case 22:
		m.Class = classFetish
	case 23:
		m.Class = classFetishShaman
	case 29:
		m.Class, m.Mode, m.Late = 0x1c5, 0xc, true
	case 30:
		m.Class, m.Mode, m.Late = 0x1ec, 0xc, true

		if c.LevelID == 0x6e {
			m.Class = 0x211
		}
	case 31:
		m.Class, m.Mode, m.Late = 0x20a, 0xc, true
	case 32:
		m.Class, m.Mode, m.Late = 0x1b6, 0xc, true
	}

	if m.Class >= nm.MonstatsCount {
		return Monster{}
	}

	m.Class = tier(m.Class, c.LevelID, nm)

	return m
}
