// Package d2mapentity implements map entities
package d2mapentity

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
)

// Object represents a composite of animations that can be projected onto the map.
type Object struct {
	uuid      string
	Position  d2vector.Position
	composite *d2asset.Composite
	highlight bool
	// nameLabel    d2ui.Label
	objectRecord *d2records.ObjectDetailRecord
	drawLayer    int
	name         string
	opening      bool
	opened       bool

	// PortalDest is the level a portal object leads to (the original keeps it
	// in the object data, byte +4) and PortalOwner the player who opened it.
	PortalDest  int
	PortalOwner string
}

// Kind returns the interaction class of the object from its objects.txt row.
func (ob *Object) Kind() d2level.ObjectKind {
	r := ob.objectRecord

	return d2level.ClassifyObject(r.IsDoor, r.SubClass, r.OperateFn)
}

// IsDoor reports whether the object is a door (objects.txt IsDoor, or one of
// the operate functions of the door classes).
func (ob *Object) IsDoor() bool { return ob.Kind() == d2level.ObjectDoor }

// Blocking reports whether the object currently blocks walking: a door that
// is not open. (Other objects keep their behaviour of not blocking; only doors
// are put into the engine's collision overlay.)
func (ob *Object) Blocking() bool { return ob.IsDoor() && !ob.opened }

// Footprint returns the sub-tile rectangle the object occupies when it blocks:
// objects.txt SizeX x SizeY sub-tiles starting at the object's sub-tile. The
// exact anchor inside the rectangle is UNVERIFIED.
func (ob *Object) Footprint() (x, y, w, h int) {
	w, h = ob.objectRecord.SizeX, ob.objectRecord.SizeY
	if w < 1 {
		w = 1
	}

	if h < 1 {
		h = 1
	}

	return int(math.Floor(ob.Position.X())), int(math.Floor(ob.Position.Y())), w, h
}

// Close puts an opened object (a door) back into its neutral mode. It returns
// false if the object was not open.
func (ob *Object) Close() (bool, error) {
	if !ob.opened {
		return false, nil
	}

	ob.opened, ob.opening = false, false

	return true, ob.setMode(d2enum.ObjectAnimationModeNeutral, 0, false)
}

// Record returns the objects.txt row of the object.
func (ob *Object) Record() *d2records.ObjectDetailRecord { return ob.objectRecord }

// IsOpened reports whether Open was called on the object.
func (ob *Object) IsOpened() bool { return ob.opened }

// Open plays the object's operating animation (chest lid, barrel breaking) and
// leaves it in its opened mode afterwards. It returns false if the object was
// already opened.
func (ob *Object) Open() (bool, error) {
	if ob.opened {
		return false, nil
	}

	ob.opened = true

	if !ob.objectRecord.HasAnimationMode[d2enum.ObjectAnimationModeOperating] {
		return true, ob.setOpenedMode()
	}

	ob.opening = true

	return true, ob.setMode(d2enum.ObjectAnimationModeOperating, 0, false)
}

func (ob *Object) setOpenedMode() error {
	if ob.objectRecord.HasAnimationMode[d2enum.ObjectAnimationModeOpened] {
		return ob.setMode(d2enum.ObjectAnimationModeOpened, 0, false)
	}

	return nil
}

// setMode changes the graphical mode of this animated entity
// nolint:unparam // direction may not always be passed 0 in the future
func (ob *Object) setMode(animationMode d2enum.ObjectAnimationMode, direction int, randomFrame bool) error {
	err := ob.composite.SetMode(animationMode, "HTH")
	if err != nil {
		return err
	}

	ob.composite.SetDirection(direction)

	ob.drawLayer = ob.objectRecord.OrderFlag[d2enum.ObjectAnimationModeNeutral]

	// For objects their txt record entry overrides animationdata
	speed := ob.objectRecord.FrameDelta[animationMode]
	if speed != 0 {
		ob.composite.SetAnimSpeed(speed)
	}

	frameCount := ob.objectRecord.FrameCount[animationMode]

	if frameCount != 0 {
		ob.composite.SetSubLoop(0, frameCount)
	}

	ob.composite.SetPlayLoop(ob.objectRecord.CycleAnimation[animationMode])
	ob.composite.SetCurrentFrame(ob.objectRecord.StartFrame[animationMode])

	if randomFrame {
		// nolint:gosec // not concerned with crypto-strong randomness
		n := rand.Intn(frameCount)
		ob.composite.SetCurrentFrame(n)
	}

	return err
}

// ID returns the object uuid
func (ob *Object) ID() string {
	return ob.uuid
}

// Highlight sets the entity highlighted flag to true.
func (ob *Object) Highlight() {
	ob.highlight = true
}

// Selectable returns if the object is selectable or not
func (ob *Object) Selectable() bool {
	if ob.opened && !ob.IsDoor() { // an open door can be closed again
		return false
	}

	mode := ob.composite.ObjectAnimationMode()

	return ob.objectRecord.Selectable[mode]
}

// Render draws this animated entity onto the target
func (ob *Object) Render(target d2interface.Surface) {
	renderOffset := ob.Position.RenderOffset()
	target.PushTranslation(
		int((renderOffset.X()-renderOffset.Y())*subtileWidth),
		int((renderOffset.X()+renderOffset.Y())*subtileHeight),
	)

	if ob.highlight {
		target.PushBrightness(highlightBrightness)
		defer target.Pop()
	}

	defer target.Pop()

	if err := ob.composite.Render(target); err != nil {
		fmt.Printf("failed to render composite animation, err: %v\n", err)
	}

	ob.highlight = false
}

// Advance updates the animation
func (ob *Object) Advance(elapsed float64) {
	if err := ob.composite.Advance(elapsed); err != nil {
		fmt.Printf("failed to advance composiste animation, err: %v\n", err)
	}

	if ob.opening && ob.composite.GetPlayedCount() > 0 {
		ob.opening = false

		if err := ob.setOpenedMode(); err != nil {
			fmt.Printf("failed to set the opened mode, err: %v\n", err)
		}
	}
}

// GetLayer returns which layer of the map the object is drawn
func (ob *Object) GetLayer() int {
	return ob.drawLayer
}

// GetPositionF of the object but differently
func (ob *Object) GetPositionF() (x, y float64) {
	w := ob.Position.World()
	return w.X(), w.Y()
}

// Label gets the name of the object
func (ob *Object) Label() string {
	return ob.name
}

// GetPosition returns the object's position
func (ob *Object) GetPosition() d2vector.Position {
	return ob.Position
}

// GetVelocity returns the object's velocity vector
func (ob *Object) GetVelocity() d2vector.Vector {
	return *d2vector.VectorZero()
}

// GetSize returns the current frame size
func (ob *Object) GetSize() (width, height int) {
	return ob.composite.GetSize()
}
