package d2netpacket

import (
	"encoding/json"

	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

// ChangeLevelPacket tells the server that the hero arrived in another level.
// In the original this is server-authoritative (SERVER_ChangePlayerLevel);
// the local single-player client decides and reports, so the server keeps the
// hero's level and position for saving.
type ChangeLevelPacket struct {
	PlayerID string  `json:"playerId"`
	Level    int     `json:"level"`
	X        float64 `json:"x"` // arrival position in tiles
	Y        float64 `json:"y"`
	// ActFinished is set for a forward act trip made through the travel NPC
	// or talk: the server then marks the act it leaves as finished
	// (QUESTS_OnActChangeNpcTravel, quest slots 7/15/28).
	ActFinished bool `json:"actFinished,omitempty"`
}

// CreateChangeLevelPacket returns a NetPacket for the server.
func CreateChangeLevelPacket(playerID string, level int, x, y float64) (NetPacket, error) {
	return CreateChangeLevelPacketAct(playerID, level, x, y, false)
}

// CreateChangeLevelPacketAct is CreateChangeLevelPacket with the act-finished flag.
func CreateChangeLevelPacketAct(playerID string, level int, x, y float64, actFinished bool) (NetPacket, error) {
	b, err := json.Marshal(ChangeLevelPacket{PlayerID: playerID, Level: level, X: x, Y: y, ActFinished: actFinished})

	return NetPacket{PacketType: d2netpackettype.ChangeLevel, PacketData: b}, err
}

// UnmarshalChangeLevel unmarshals the packet data.
func UnmarshalChangeLevel(packet []byte) (ChangeLevelPacket, error) {
	var p ChangeLevelPacket

	return p, json.Unmarshal(packet, &p)
}

// SetWaypointPacket activates (or clears) the waypoint of a level in the
// hero's record for his current difficulty.
type SetWaypointPacket struct {
	PlayerID string `json:"playerId"`
	Level    int    `json:"level"`
	Active   bool   `json:"active"`
}

// CreateSetWaypointPacket returns a NetPacket for the server.
func CreateSetWaypointPacket(playerID string, level int, active bool) (NetPacket, error) {
	b, err := json.Marshal(SetWaypointPacket{PlayerID: playerID, Level: level, Active: active})

	return NetPacket{PacketType: d2netpackettype.SetWaypoint, PacketData: b}, err
}

// UnmarshalSetWaypoint unmarshals the packet data.
func UnmarshalSetWaypoint(packet []byte) (SetWaypointPacket, error) {
	var p SetWaypointPacket

	return p, json.Unmarshal(packet, &p)
}
