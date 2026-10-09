package d2client

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapgen"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
)

// CanLoadLevel reports whether a level provider can build the level now.
func (g *GameClient) CanLoadLevel(levelID int) bool {
	return g.mapGen.CanLoadLevel(levelID)
}

// RegisterLevelProvider plugs in an additional level provider (see
// d2mapgen.LevelProvider); it is asked before the built-in ones.
func (g *GameClient) RegisterLevelProvider(p d2mapgen.LevelProvider) {
	g.mapGen.RegisterProvider(p)
}

// ArrivalFunc lets the caller choose the arrival point in the freshly built
// level (in tiles), for example next to a waypoint object. ok=false keeps the
// provider's default.
type ArrivalFunc func(m *d2mapengine.MapEngine) (x, y float64, ok bool)

const arrivalRadius = 0x32 // sub-tiles searched for a free cell

// ChangeLevel builds the level into the map engine, moves every player to its
// arrival point and tells the server. The caller has already validated the
// move (cooldown, waypoint rules) and does the fade. It returns the arrival
// point in tiles.
func (g *GameClient) ChangeLevel(levelID int, prefer ArrivalFunc) (d2mapgen.Arrival, error) {
	arrival, err := g.mapGen.LoadLevel(levelID, d2mapgen.LoadRequest{
		Seed:       d2mapgen.HeroMapSeed,
		Difficulty: d2drlg.Difficulty(g.LevelDifficulty()),
	})
	if err != nil {
		return arrival, err
	}

	if prefer != nil {
		if x, y, ok := prefer(g.MapEngine); ok {
			sx, sy, found := g.MapEngine.NearestWalkable(int(x*numSubtilesPerTile), int(y*numSubtilesPerTile), arrivalRadius)
			if found {
				arrival = d2mapgen.Arrival{X: (float64(sx) + 0.5) / numSubtilesPerTile, Y: (float64(sy) + 0.5) / numSubtilesPerTile}
			}
		}
	}

	// ResetMap dropped every entity, the players included
	for _, p := range g.Players {
		g.MapEngine.AddEntity(p)

		p.StopMoving()
		p.Position.Set(arrival.X*numSubtilesPerTile, arrival.Y*numSubtilesPerTile)
		p.Target = p.Position
		p.SetIsInTown(d2level.IsTown(levelID))

		if err := p.SetAnimationMode(p.GetAnimationMode()); err != nil {
			g.Errorf("level change: animation mode of %s: %v", p.ID(), err)
		}
	}

	g.Level = levelID
	g.RegenMap = true
	g.MapEngine.IsLoading = false

	if pkt, err := d2netpacket.CreateChangeLevelPacket(g.PlayerID, levelID, arrival.X, arrival.Y); err == nil {
		if err := g.SendPacketToServer(pkt); err != nil {
			g.Errorf("could not report the level change to the server: %v", err)
		}
	}

	return arrival, nil
}

// HasWaypoint reports whether the local hero activated the waypoint of a level
// in his difficulty.
func (g *GameClient) HasWaypoint(levelID int) bool {
	bit, ok := d2level.WaypointBit(levelID)
	if !ok || g.Progress == nil {
		return false
	}

	return g.Progress.Waypoints.Has(int(g.Difficulty), d2s.Waypoint(bit))
}

// SetWaypoint activates (or clears) a waypoint of the local hero, updates the
// local copy and sends it to the server, which saves the hero. It reports
// whether the bit changed.
func (g *GameClient) SetWaypoint(levelID int, active bool) (bool, error) {
	if _, ok := d2level.WaypointBit(levelID); !ok {
		return false, fmt.Errorf("level %d has no waypoint", levelID)
	}

	if g.HasWaypoint(levelID) == active {
		return false, nil
	}

	bit, _ := d2level.WaypointBit(levelID)
	g.Progress.Waypoints.Set(int(g.Difficulty), d2s.Waypoint(bit), active)

	pkt, err := d2netpacket.CreateSetWaypointPacket(g.PlayerID, levelID, active)
	if err != nil {
		return true, err
	}

	return true, g.SendPacketToServer(pkt)
}
