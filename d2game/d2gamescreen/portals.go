package d2gamescreen

import (
	"errors"
	"math"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2portal"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2client"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
)

// Town portals (package d2portal): casting a scroll or tome of town portal
// opens a pair of linked portal objects, one next to the hero and one in the
// town of the act. The server keeps the pairs of the whole game
// (d2server/portals.go) and every client draws the ends that stand in the
// level it is in; the objects are rebuilt after each level change, so a pair
// persists as long as its owner keeps it (recasting replaces it, leaving the
// game closes it). The owner and the owner's party members may walk through.

const (
	portalCastOffset = 2.0 // tiles east of the hero, where the portal opens
	townSlotStep     = 2.0 // tiles between the town ends of several owners
	townSlotBase     = 3.0 // tiles from the waypoint to the first town end
	portalPlaceRange = 24  // sub-tiles searched for a walkable place
)

// portalState is the game screen's copy of the server's pairs and the objects
// built for the level on screen.
type portalState struct {
	pairs []d2portal.Pair
	ents  map[*d2mapentity.Object]d2portal.Portal
}

// hookPortals connects the portal list of the server.
func (v *Game) hookPortals() {
	v.gameClient.OnPortals = v.onPortals
}

// onPortals takes the server's list and redraws the ends in this level.
func (v *Game) onPortals(pairs []d2portal.Pair, notice string) {
	v.portal.pairs = pairs

	if notice != "" {
		v.Infof("PORTAL notice: %s", notice)
	}

	if v.gameClient.MapEngine != nil && v.localPlayer != nil && !v.gameClient.MapEngine.IsLoading {
		v.syncPortals()
	}
}

// ownPair is the local hero's open pair.
func (v *Game) ownPair() *d2portal.Pair {
	for i := range v.portal.pairs {
		if v.portal.pairs[i].Owner == v.me() {
			return &v.portal.pairs[i]
		}
	}

	return nil
}

// townSlot orders the auto placed town ends of a level, so equal lists give
// every client the same places.
func (v *Game) townSlot(pairID, level int) int {
	slot := 0

	for _, p := range v.portal.pairs {
		if p.Town.Level != level || !p.Town.Auto {
			continue
		}

		if p.ID == pairID {
			return slot
		}

		slot++
	}

	return slot
}

// portalSpot is where a portal end stands in the map of its level, in
// sub-tiles: the cast spot, or for a town end without one the place beside the
// town waypoint (UNVERIFIED: the original's town portal spot).
func (v *Game) portalSpot(m *d2mapengine.MapEngine, end d2portal.End, pairID int) (sx, sy int) {
	x, y := end.X, end.Y

	if end.Auto {
		if wx, wy, ok := nextToWaypoint(m); ok {
			x, y = wx+townSlotBase+float64(v.townSlot(pairID, end.Level))*townSlotStep, wy
		} else {
			x, y = v.heroTilePos()
		}
	}

	sx, sy = int(math.Floor(x*subtilesInTile)), int(math.Floor(y*subtilesInTile))

	if wx, wy, ok := m.NearestWalkable(sx, sy, portalPlaceRange); ok {
		sx, sy = wx, wy
	}

	return sx, sy
}

// syncPortals removes the portal objects of the old map and builds those of
// the pairs that stand in the level on screen.
func (v *Game) syncPortals() {
	m := v.gameClient.MapEngine
	for ob := range v.portal.ents {
		m.RemoveEntity(ob)
	}

	v.portal.ents = map[*d2mapentity.Object]d2portal.Portal{}
	v.spawnPortals()
}

// spawnPortals builds the objects for the current map (no earlier ones).
func (v *Game) spawnPortals() {
	if v.portal.ents == nil {
		v.portal.ents = map[*d2mapentity.Object]d2portal.Portal{}
	}

	m := v.gameClient.MapEngine
	level := v.currentLevel()

	rec := v.asset.Records.Object.Details[portalObjectID]
	if rec == nil {
		return
	}

	for _, p := range d2portal.PortalsIn(v.portal.pairs, level) {
		sx, sy := v.portalSpot(m, p.Here, p.PairID)

		ob, err := m.NewObject(sx, sy, rec, d2resource.PaletteUnits)
		if err != nil {
			v.Errorf("PORTAL object: %v", err)
			continue
		}

		ob.PortalDest, ob.PortalOwner = p.Dest.Level, p.Owner
		m.AddEntity(ob)
		v.portal.ents[ob] = p

		v.Infof("PORTAL object level=%d (%s) owner=%q dest=%d (%s) town=%v at=(%d,%d)", level, v.levelName(level), p.OwnerName,
			p.Dest.Level, v.levelName(p.Dest.Level), p.InTown, sx, sy)
	}
}

// openTownPortal casts the town portal at the hero's feet. It sends the pair
// to the server; the objects appear when the server's list comes back.
func (v *Game) openTownPortal() error {
	if v.localPlayer == nil {
		return errors.New("no hero")
	}

	px, py := v.heroTilePos()
	level := v.currentLevel()

	pair, err := d2portal.PlanOpen(v.me(), v.localPlayer.Name(), level, px+portalCastOffset, py, v.ownPair())
	if err != nil {
		return err
	}

	pkt, err := d2netpacket.CreatePortalOpenPacket(d2netpacket.PortalOpenPacket{Pair: pair})
	if err != nil {
		return err
	}

	v.Infof("PORTAL cast level=%d (%s) at=(%.1f,%.1f) town=%d (%s)", level, v.levelName(level), px+portalCastOffset, py,
		pair.Town.Level, v.levelName(pair.Town.Level))

	return v.gameClient.SendPacketToServer(pkt)
}

// useTownPortalItem is the right click on a scroll or tome of town portal.
func (v *Game) useTownPortalItem(src *diablo2item.Item) {
	if err := v.openTownPortal(); err != nil {
		v.Infof("PORTAL refused: %v", err)
		return
	}

	left, err := v.gameControls.ConsumeTownPortal(src)
	if err != nil {
		v.Warningf("PORTAL charge: %v", err)
		return
	}

	v.Infof("PORTAL paid with %s, left=%d", src.GetItemCode(), left)
}

// commandTownPortal is "townportal [free]": cast a town portal, paying with a
// scroll or a tome charge unless "free" is given (scenarios).
func (v *Game) commandTownPortal(args []string) error {
	free := len(args) > 0 && strings.EqualFold(args[0], "free")

	if free {
		return v.openTownPortal()
	}

	src := v.gameControls.TownPortalSource()
	if src == nil {
		return errors.New("no Scroll of Town Portal and no Tome of Town Portal with a scroll left")
	}

	v.useTownPortalItem(src)

	return nil
}

// commandClosePortal is "closeportal": the hero's pair is closed.
func (v *Game) commandClosePortal(_ []string) error {
	pkt, err := d2netpacket.CreatePortalOpenPacket(d2netpacket.PortalOpenPacket{Close: true})
	if err != nil {
		return err
	}

	return v.gameClient.SendPacketToServer(pkt)
}

// commandPortals logs the pairs and the objects on screen.
func (v *Game) commandPortals(_ []string) error {
	v.Infof("PORTAL list level=%d pairs=%d objects=%d", v.currentLevel(), len(v.portal.pairs), len(v.portal.ents))

	for _, p := range v.portal.pairs {
		v.Infof("PORTAL pair id=%d owner=%q field=%d town=%d", p.ID, p.OwnerName, p.Field.Level, p.Town.Level)
	}

	return nil
}

// operateTownPortal uses a portal object of a pair. false means the object is
// not one (a spawnportal or quest portal), and the caller handles it.
func (v *Game) operateTownPortal(ob *d2mapentity.Object) bool {
	p, ok := v.portal.ents[ob]
	if !ok {
		return false
	}

	if !v.levels.cooldown.Ready(v.levels.clock, d2level.PortalCooldownSeconds) {
		v.Infof("PORTAL refused: %v", d2level.ErrPortalCooldown)
		return true
	}

	inParty := v.gameClient.Roster.SameParty(p.Owner, v.me())
	if err := d2portal.CanUse(p, v.me(), inParty, !v.gameClient.IsSinglePlayer()); err != nil {
		v.Infof("PORTAL refused: %v (owner %q)", err, p.OwnerName)
		return true
	}

	if v.startLevelChange(p.Dest.Level, d2level.StartPortal, "portal") {
		dest := p.Dest
		v.levels.trans.portalDest, v.levels.trans.portalPair = &dest, p.PairID
		v.Infof("PORTAL used owner=%q from=%d dest=%d (%s) town=%v mine=%v", p.OwnerName, v.currentLevel(), p.Dest.Level,
			v.levelName(p.Dest.Level), p.InTown, p.Owner == v.me())
	}

	return true
}

// nextToPortalEnd picks the arrival next to the far end of the portal that was
// used (the original's arrival is next to the destination portal object).
func (v *Game) nextToPortalEnd(end d2portal.End, pairID int) d2client.ArrivalFunc {
	return func(m *d2mapengine.MapEngine) (x, y float64, ok bool) {
		sx, sy := v.portalSpot(m, end, pairID)

		return (float64(sx) + 0.5) / subtilesInTile, (float64(sy)+0.5)/subtilesInTile + 1, true
	}
}
