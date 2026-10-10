package d2automap

// Unit markers (AUTOMAP_DrawUnitMarker 0x455f70, AUTOMAP_GetUnitMarkerColor 0x455470,
// AUTOMAP_DrawRosterMarker 0x456350, AUTOMAP_DrawUnitMarkers 0x4563d0 of Game.exe 1.14b).
// The decision logic is kept here, free of any engine type, so that it can be tested.

// UnitKind is the engine's unit type of a marker candidate.
type UnitKind int

// Unit kinds (the game's unit types 0, 1 and 2).
const (
	UnitPlayer UnitKind = iota
	UnitMonster
	UnitObject
)

// ColorID names one of the marker colours (the engine keeps them in the palette
// indexes DAT_0079d238..240).
type ColorID int

// Marker colours.
const (
	ColorNone ColorID = iota
	ColorSelf
	ColorOther
	ColorParty
	ColorMinion
	ColorNPC
	ColorPortal
	ColorBlack // palette index 0: the stash cross
)

// Object ids (objects.txt Id) with a special marker. VERIFIED (class compares in 0x455f70/0x455470).
const (
	ObjectTownPortal      = 59
	ObjectPermanentPortal = 60
	ObjectStash           = 267 // 0x10b, "bank": gets a text label instead of a cross
)

// portalHiddenLevels are the levels in which a permanent portal (60) is not
// drawn (0x7d..0x7f, 0x6f, 0x70, 0x75). VERIFIED in 0x455470; which levels they
// are (the Act 5 Worldstone area and the Uber levels) is inferred.
var portalHiddenLevels = map[int]bool{0x7d: true, 0x7e: true, 0x7f: true, 0x6f: true, 0x70: true, 0x75: true}

// Unit is what the marker logic needs to know about a unit.
type Unit struct {
	Kind        UnitKind
	Self        bool // the client's own player
	Party       bool // a player in the hero's party
	Dead        bool // dead or dying
	FriendlyNPC bool // a monster with the "automap" monstats flag (town folk)
	Minion      bool // the hero's own mercenary or summon
	ObjectID    int  // objects.txt Id for objects
	Level       int  // level id the unit stands in
}

// Options are the automap options that change markers.
type Options struct {
	ShowParty bool // DAT_00710f38: "Show Party"
	ShowNames bool // DAT_00710f3c: "Show Names"
}

// Label tells which text, if any, is drawn at a marker.
type Label int

// Labels.
const (
	LabelNone  Label = iota
	LabelName        // the unit's name
	LabelStash       // the stash text, drawn instead of a cross
)

// Marker is the drawing decision for one unit.
type Marker struct {
	Cross bool
	Color ColorID
	Label Label
}

// Classify decides what is drawn for a unit.
func Classify(u Unit, o Options) Marker {
	switch u.Kind {
	case UnitPlayer:
		if u.Dead && !u.Self {
			return Marker{}
		}

		c := ColorOther

		switch {
		case u.Self:
			c = ColorSelf
		case u.Party:
			c = ColorParty
		}

		m := Marker{Color: c, Cross: o.ShowParty || c != ColorParty}
		if o.ShowNames && o.ShowParty && !u.Self {
			m.Label = LabelName
		}

		return m
	case UnitMonster:
		if u.Dead {
			return Marker{}
		}

		if u.Minion {
			return Marker{Cross: true, Color: ColorMinion}
		}

		if !u.FriendlyNPC {
			return Marker{} // hostile monsters are not on the map
		}

		m := Marker{Cross: true, Color: ColorNPC}
		if o.ShowNames {
			m.Label = LabelName
		}

		return m
	case UnitObject:
		switch u.ObjectID {
		case ObjectTownPortal:
			return Marker{Cross: true, Color: ColorPortal}
		case ObjectPermanentPortal:
			if portalHiddenLevels[u.Level] {
				return Marker{}
			}

			return Marker{Cross: true, Color: ColorPortal}
		case ObjectStash:
			if o.ShowNames {
				return Marker{Label: LabelStash, Color: ColorBlack}
			}

			return Marker{Cross: true, Color: ColorBlack}
		}
	}

	return Marker{}
}

// RosterMarker reports whether a party member who has no unit on the client
// gets a marker: the option is on and he is in the same act (the screen test is
// the caller's).
func RosterMarker(o Options, sameAct bool) bool { return o.ShowParty && sameAct }

// CrossOffset is the extra pixel offset of a cross on the mini map (-1, +5);
// the full map has none. VERIFIED (AUTOMAP_DrawPlayerCross).
func CrossOffset(size Size) (dx, dy int) {
	if size == SizeMini {
		return -1, 5
	}

	return 0, 0
}

// Text offsets above the marker position: names of players are drawn at y-10
// (AUTOMAP_DrawMarkerNameText), NPC names and the stash text at y-0x12. VERIFIED.
const (
	NameOffsetY  = -10
	LabelOffsetY = -0x12
)
