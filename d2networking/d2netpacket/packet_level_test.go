package d2netpacket

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

func TestChangeLevelRoundTrip(t *testing.T) {
	np, err := CreateChangeLevelPacket("p1", 35, 12.5, 7)
	if err != nil || np.PacketType != d2netpackettype.ChangeLevel {
		t.Fatal(np.PacketType, err)
	}

	got, err := UnmarshalChangeLevel(np.PacketData)
	if err != nil || got != (ChangeLevelPacket{PlayerID: "p1", Level: 35, X: 12.5, Y: 7}) {
		t.Errorf("%+v %v", got, err)
	}
}

func TestSetWaypointRoundTrip(t *testing.T) {
	np, err := CreateSetWaypointPacket("p1", 3, true)
	if err != nil || np.PacketType != d2netpackettype.SetWaypoint {
		t.Fatal(np.PacketType, err)
	}

	got, err := UnmarshalSetWaypoint(np.PacketData)
	if err != nil || got != (SetWaypointPacket{PlayerID: "p1", Level: 3, Active: true}) {
		t.Errorf("%+v %v", got, err)
	}
}

func TestAddPlayerCarriesProgress(t *testing.T) {
	prog := (&d2hero.HeroState{}).EnsureProgress()

	np, err := CreateAddPlayerPacket("id", "n", 1, 2, d2enum.HeroSorceress, nil, nil,
		d2hero.HeroState{}.Equipment, 0, 0, 5, prog, d2enum.DifficultyHell)
	if err != nil {
		t.Fatal(err)
	}

	got, err := UnmarshalAddPlayer(np.PacketData)
	if err != nil || got.Progress == nil || got.Difficulty != d2enum.DifficultyHell {
		t.Fatalf("%+v %v", got, err)
	}

	if got.Progress.Waypoints[2] != 1 {
		t.Errorf("fresh hero should have only the town waypoint, got %#x", got.Progress.Waypoints[2])
	}
}
