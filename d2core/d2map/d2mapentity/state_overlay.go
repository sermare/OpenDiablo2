package d2mapentity

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Draw order of a state overlay relative to the unit it surrounds: the "back" half of an aura circle or a curse
// goes behind the unit, the "front" half and single overlays in front. The renderer draws the lower order first
// among entities of one subtile (d2maprenderer.renderEntitiesBySubtile).
const (
	DrawBehind = -1
	DrawNormal = 0
	DrawFront  = 1
)

// StateOverlay is the looping overlay a state keeps on a unit while it lasts (states.txt overlay1..4: aura circles,
// curse symbols, frozen armor, stun stars). It follows the unit; the owner removes it when the state ends.
type StateOverlay struct {
	*AnimatedEntity
	record *d2records.OverlayRecord
	order  int
}

// DrawOrder is the order among entities sharing a subtile.
func (so *StateOverlay) DrawOrder() int { return so.order }

// Record is the overlay.txt row.
func (so *StateOverlay) Record() *d2records.OverlayRecord { return so.record }

// Follow puts the overlay on a unit's position (subtile coordinates).
func (so *StateOverlay) Follow(x, y float64) {
	so.Position.Set(x, y)
	so.Target = so.Position
}

// Advance runs the animation.
func (so *StateOverlay) Advance(elapsed float64) { so.AnimatedEntity.Advance(elapsed) }

// StateOverlayOrder is the draw order of overlay slot n (0 based) of a state: the states with two overlays keep the
// front half in overlay1 and the back half in overlay2 (aura_might_front / aura_might_back), the Paladin auras and
// the shrines name them the same way; UNVERIFIED that the exe orders them by slot rather than by the PreDraw column.
func StateOverlayOrder(slot int, preDraw bool) int {
	if preDraw || slot == 1 {
		return DrawBehind
	}

	return DrawFront
}

// NewStateOverlay creates the looping overlay entity of a state at a subtile.
func (f *MapEntityFactory) NewStateOverlay(x, y float64, rec *d2records.OverlayRecord, order int) (*StateOverlay, error) {
	animation, err := f.asset.LoadAnimationWithEffect(
		fmt.Sprintf("/data/Global/Overlays/%s.dcc", rec.Filename),
		d2resource.PaletteUnits,
		d2enum.DrawEffectModulate,
	)
	if err != nil {
		return nil, err
	}

	animation.Rewind()
	animation.ResetPlayedCount()
	animation.SetPlayLength(float64(rec.AnimRate*retailFps) / millisecondsPerSecond)
	animation.SetPlayLoop(true)
	animation.PlayForward()

	so := &StateOverlay{AnimatedEntity: NewAnimatedEntity(0, 0, animation), record: rec, order: order}
	so.Follow(x, y)

	return so, nil
}

// ID returns the overlay uuid.
func (so *StateOverlay) ID() string { return so.AnimatedEntity.uuid }

// GetPosition returns the position of the overlay.
func (so *StateOverlay) GetPosition() d2vector.Position { return so.AnimatedEntity.Position }

// GetVelocity returns the velocity vector of the overlay.
func (so *StateOverlay) GetVelocity() d2vector.Vector { return so.AnimatedEntity.velocity }
