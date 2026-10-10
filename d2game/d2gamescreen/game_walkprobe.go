package d2gamescreen

import "errors"

// commandWalkProbe logs the walkability of the lava and water tiles of the
// level (the WALKPROBE lines are what scripts/verify.d/9l-walkability.sh reads).
func (v *Game) commandWalkProbe(_ []string) error {
	if v.localPlayer == nil {
		return errors.New("no hero")
	}

	me := v.gameClient.MapEngine
	sx, sy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())
	r := me.WalkProbe(sx, sy)
	tx, ty, ends, found := me.TryWalkOntoLava(sx, sy)

	v.Infof("WALKPROBE level=%d lava=%d lavaReachable=%d water=%d waterReachable=%d bridge=%d bridgeReachable=%d",
		v.currentLevel(), r.Lava, r.LavaReachable, r.Water, r.WaterReachable, r.Bridge, r.BridgeReachable)
	v.Infof("WALKPROBE level=%d orderOntoLava found=%v tile=%d,%d routeEndsOnLava=%v", v.currentLevel(), found, tx, ty, ends)

	return nil
}
