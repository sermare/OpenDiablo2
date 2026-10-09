package d2level

// ObjectKind is the interaction class of an object of objects.txt.
type ObjectKind int

// Object kinds.
const (
	ObjectOther ObjectKind = iota
	ObjectDoor
	ObjectWaypoint
	ObjectPortal
	ObjectShrine
	ObjectContainer
)

func (k ObjectKind) String() string {
	return [...]string{"other", "door", "waypoint", "portal", "shrine", "container"}[k]
}

// objects.txt SubClass bits.
const (
	subClassShrine    = 1
	subClassPortal    = 4
	subClassContainer = 8
	subClassWaypoint  = 64
)

// objects.txt OperateFn values seen in the 1.14b table (the "OperateFn"
// column): 8 and 29 are the doors, 15 the town/permanent portals, 23 the
// waypoints. Other functions are not mapped.
const (
	operateDoor     = 8
	operateDoorAlt  = 29
	operateJailDoor = 18 // the (secret) jail cell doors, SubClass 128, IsDoor=0
	operatePortal   = 15
	operateWaypoint = 23
)

// ClassifyObject maps the objects.txt columns IsDoor, SubClass and OperateFn
// to an interaction class. The waypoint objects have SubClass 64 and
// OperateFn 23; doors have IsDoor=1 (OperateFn 8, or 29 for a few act 2
// doors; the jail cell doors use OperateFn 18); portals SubClass 4 with OperateFn 15.
func ClassifyObject(isDoor bool, subClass, operateFn int) ObjectKind {
	switch {
	case operateFn == operateWaypoint || subClass&subClassWaypoint != 0:
		return ObjectWaypoint
	case isDoor || operateFn == operateDoor || operateFn == operateDoorAlt || operateFn == operateJailDoor:
		return ObjectDoor
	case operateFn == operatePortal || subClass&subClassPortal != 0:
		return ObjectPortal
	case subClass&subClassShrine != 0:
		return ObjectShrine
	case subClass&subClassContainer != 0:
		return ObjectContainer
	}

	return ObjectOther
}
