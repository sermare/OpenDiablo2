package d2object

// FramesPerSecond converts the frame counts of shrines.txt and objects.txt to
// seconds (the game runs 25 frames per second).
const FramesPerSecond = 25.0

// Class is what an OperateFn does when the player clicks the object.
type Class int

// Operate classes.
const (
	ClassNone     Class = iota // nothing happens (decoration, dummy, clientside only)
	ClassLoot                  // chest, casket, barrel, urn, crate, stash: opens or breaks and drops a treasure class
	ClassExplode               // exploding barrel / chest: loot plus an explosion
	ClassShrine                // shrines.txt effect
	ClassWell                  // restores life and mana, then empties for a while
	ClassRack                  // weapon rack / armor stand: one drop, then used up
	ClassDoor                  // handled by the door code (d2level)
	ClassPortal                // handled by the portal code
	ClassWaypoint              // handled by the waypoint code
	ClassStash                 // the town stash (handled by the stash code)
	ClassQuest                 // quest object: documented stub, no behaviour yet
	ClassTrap                  // trap / switch / lever: documented stub
	ClassDecor                 // light source or scenery that needs no operation
	ClassTeleport              // act 2 teleport pads, stairs, special warps: stub
	numClasses
)

var classNames = [...]string{"none", "loot", "explode", "shrine", "well", "rack", "door", "portal",
	"waypoint", "stash", "quest", "trap", "decor", "teleport"}

func (c Class) String() string {
	if c < 0 || c >= numClasses {
		return "?"
	}

	return classNames[c]
}

// Info describes one OperateFn.
type Info struct {
	Fn    int
	Class Class
	Name  string // what the objects using it are
	// Sound is the Sounds.txt handle (lower case, "ESOUND_" removed) played
	// when the object is operated, "" for none. objects.txt 1.14b has no sound
	// columns; the original plays them from the operate/client function. The
	// choice per function is UNVERIFIED.
	Sound string
	// Stub marks classes that are recognised but have no behaviour yet; Note
	// says why.
	Stub bool
	Note string
}

var operateInfos = []Info{
	{Fn: 0, Class: ClassNone, Name: "dummy / decoration"},
	{Fn: 1, Class: ClassLoot, Name: "casket", Sound: "object_casket"},
	{Fn: 2, Class: ClassShrine, Name: "shrine"},
	{Fn: 3, Class: ClassLoot, Name: "urn / basket / jar", Sound: "object_urn_break_1"},
	{Fn: 4, Class: ClassLoot, Name: "chest", Sound: "object_chest_large"},
	{Fn: 5, Class: ClassLoot, Name: "barrel", Sound: "object_wood_break_1"},
	{Fn: 6, Class: ClassQuest, Name: "tower tome (Forgotten Tower)", Sound: "object_tome", Stub: true,
		Note: "quest object: QUEST_OnObjectOperated needed"},
	{Fn: 7, Class: ClassExplode, Name: "exploding barrel", Sound: "object_barrel_explode"},
	{Fn: 8, Class: ClassDoor, Name: "door"},
	{Fn: 9, Class: ClassQuest, Name: "cairn stone", Stub: true, Note: "Stones of Jordan / Cairn quest"},
	{Fn: 10, Class: ClassQuest, Name: "Cain's gibbet", Sound: "object_gibbet_drop", Stub: true,
		Note: "Rescue Cain quest object (object 26), needs the quest state"},
	{Fn: 11, Class: ClassDecor, Name: "fire / torch / brazier / trap emitter"},
	{Fn: 12, Class: ClassQuest, Name: "Tree of Inifuss", Stub: true, Note: "Tree of Inifuss quest (object 30)"},
	{Fn: 13, Class: ClassDecor, Name: "dummy"},
	{Fn: 14, Class: ClassLoot, Name: "crate / corpse / hidden stash / skull pile", Sound: "object_corpse_loot"},
	{Fn: 15, Class: ClassPortal, Name: "portal"},
	{Fn: 16, Class: ClassTrap, Name: "trap door", Stub: true, Note: "Act 2 trap door"},
	{Fn: 17, Class: ClassQuest, Name: "obelisk", Stub: true, Note: "obelisks are scenery in Act 1/2"},
	{Fn: 18, Class: ClassDoor, Name: "secret jail door"},
	{Fn: 19, Class: ClassRack, Name: "armor stand", Sound: "object_armorstand"},
	{Fn: 20, Class: ClassRack, Name: "weapon rack", Sound: "object_weaponrack"},
	{Fn: 21, Class: ClassQuest, Name: "Malus", Sound: "object_malus", Stub: true, Note: "Charsi's Malus quest"},
	{Fn: 22, Class: ClassWell, Name: "well", Sound: "object_well"},
	{Fn: 23, Class: ClassWaypoint, Name: "waypoint"},
	{Fn: 24, Class: ClassQuest, Name: "tainted sun altar", Sound: "object_taintedsunaltar", Stub: true,
		Note: "Tainted Sun quest"},
	{Fn: 25, Class: ClassQuest, Name: "orifice (Horadric staff)", Stub: true, Note: "Horadric Staff quest"},
	{Fn: 26, Class: ClassLoot, Name: "bookshelf", Sound: "object_bookshelf"},
	{Fn: 27, Class: ClassTeleport, Name: "teleport pad", Sound: "object_teleportpad", Stub: true,
		Note: "Arcane Sanctuary pads need the sanctuary layout"},
	{Fn: 28, Class: ClassQuest, Name: "Lam Esen's tome", Stub: true, Note: "Lam Esen's Tome quest (Act 3)"},
	{Fn: 29, Class: ClassDoor, Name: "door (act 2 variant)"},
	{Fn: 30, Class: ClassExplode, Name: "trap / exploding chest", Sound: "object_barrel_explode"},
	{Fn: 31, Class: ClassQuest, Name: "Gidbinn altar", Stub: true, Note: "Blade of the Old Religion (Act 3)"},
	{Fn: 32, Class: ClassStash, Name: "stash"},
	{Fn: 33, Class: ClassQuest, Name: "Wirt's body", Stub: true, Note: "Wirt's leg is a Tristram quest drop"},
	{Fn: 34, Class: ClassTeleport, Name: "arcane sanctuary portal", Stub: true, Note: "Act 2 arcane portal"},
	{Fn: 42, Class: ClassQuest, Name: "tome", Stub: true},
	{Fn: 43, Class: ClassTeleport, Name: "Duriel's lair / guild portal", Stub: true},
	{Fn: 47, Class: ClassTeleport, Name: "stair", Stub: true},
}

var byFn = func() map[int]Info {
	m := make(map[int]Info, len(operateInfos))
	for _, i := range operateInfos {
		m[i.Fn] = i
	}

	return m
}()

// Lookup returns the description of an OperateFn. OperateFn values above the
// table (Act 3-5 and expansion special objects) are returned as quest stubs.
func Lookup(fn int) Info {
	if i, ok := byFn[fn]; ok {
		return i
	}

	return Info{Fn: fn, Class: ClassQuest, Name: "special object", Stub: true,
		Note: "OperateFn not mapped (Act 3-5 / expansion quest or special object)"}
}

// Implemented lists the classes the engine operates by clicking.
func (c Class) Implemented() bool {
	switch c {
	case ClassLoot, ClassExplode, ClassShrine, ClassWell, ClassRack:
		return true
	}

	return false
}

// Breakable reports whether the loot container breaks apart (barrels, urns,
// baskets, crates) instead of opening (chests, caskets), from its objects.txt
// name. The animation is the same Operating mode; only the sound differs.
func Breakable(name string) bool {
	switch lower(name) {
	case "barrel", "urn", "largeurn", "basket", "jug", "crate", "jar1", "jar2", "jar3", "barrel wilderness",
		"rockpile", "loose rock", "loose boulder", "goo pile", "ratnest", "skullpile", "skull pile":
		return true
	}

	return false
}

func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}

	return string(b)
}

// SoundFor returns the Sounds.txt handle for an object: breakables use the
// wood/urn break sounds, others the sound of their OperateFn.
func SoundFor(fn int, name string) string {
	info := Lookup(fn)
	if info.Class == ClassLoot && Breakable(name) {
		switch lower(name) {
		case "urn", "largeurn", "jar1", "jar2", "jar3", "jug":
			return "object_urn_break_1"
		case "basket":
			return "object_basket_1"
		case "barrel", "crate", "barrel wilderness":
			return "object_wood_break_1"
		}

		return "object_skullpile_1"
	}

	return info.Sound
}
