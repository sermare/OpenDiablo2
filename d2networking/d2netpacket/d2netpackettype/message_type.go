package d2netpackettype

import (
	"encoding/json"
)

// NetPacketType is an enum referring to all packet types in package
// d2netpacket.
type NetPacketType uint32

// (Except NetPacket which declares a NetPacketType to specify the packet body
// type. See d2netpackettype.NetPacket.)
//
// # Warning
//
// Do NOT re-arrange the order of these packet values unless you want to
// break compatibility between clients of slightly different versions.
// Also note that the packet id is a byte, so if we use more than 256 of
// these then we are doing something very wrong.
const (
	UpdateServerInfo                NetPacketType = iota // Sent by the server, client sets the given player ID and map seed
	GenerateMap                                          // Sent by the server, client generates a map
	AddPlayer                                            // Sent by the server, client adds a player
	MovePlayer                                           // Sent by client or server, moves a player entity
	PlayerConnectionRequest                              // Sent by the remote client when connecting
	PlayerDisconnectionNotification                      // Sent by the remote client when disconnecting
	Ping                                                 // Requests a Pong packet
	Pong                                                 // Responds to a Ping packet
	ServerClosed                                         // Sent by the local host when it has closed the server
	CastSkill                                            // Sent by client or server, indicates entity casting skill
	SpawnItem                                            // Sent by server
	SavePlayer                                           // Sent by the client, saves the player
	ServerFull                                           // Sent by server when server has reached max connections
	ChangeLevel                                          // Sent by the client, the hero arrived in another level
	SetWaypoint                                          // Sent by the client, a waypoint bit was activated or cleared
	Chat                                                 // Sent by a client, the server relays it to everybody (with the sender's name)
	PartyCommand                                         // Sent by a client: invite, accept, decline, leave, hostile, peace
	RosterUpdate                                         // Sent by the server: the roster (players, parties, hostility, invitations)
	TradeCommand                                         // Sent by a client: request, respond, offer, accept, cancel
	TradeUpdate                                          // Sent by the server to both traders: the trade window state or its result
	PvPHit                                               // Sent by a client (the attacker), relayed by the server to the defender
	PartyXP                                              // Sent by a client (a kill), the server answers every sharing member with its part
	RealmUnit                                            // Sent by the realm connection: a monster of the authoritative simulation appeared, moved, was hit or died

	UnknownPacketType = 666
)

func (n NetPacketType) String() string {
	strings := map[NetPacketType]string{
		UpdateServerInfo:                "UpdateServerInfo",
		GenerateMap:                     "GenerateMap",
		AddPlayer:                       "AddPlayer",
		MovePlayer:                      "MovePlayer",
		PlayerConnectionRequest:         "PlayerConnectionRequest",
		PlayerDisconnectionNotification: "PlayerDisconnectionNotification",
		Ping:                            "Ping",
		Pong:                            "Pong",
		ServerClosed:                    "ServerClosed",
		CastSkill:                       "CastSkill",
		SpawnItem:                       "SpawnItem",
		SavePlayer:                      "SavePlayer",
		ServerFull:                      "ServerFull",
		ChangeLevel:                     "ChangeLevel",
		SetWaypoint:                     "SetWaypoint",
		Chat:                            "Chat",
		PartyCommand:                    "PartyCommand",
		RosterUpdate:                    "RosterUpdate",
		TradeCommand:                    "TradeCommand",
		TradeUpdate:                     "TradeUpdate",
		PvPHit:                          "PvPHit",
		PartyXP:                         "PartyXP",
		RealmUnit:                       "RealmUnit",
	}

	return strings[n]
}

// MarshalPacket marshals the packet to a byte slice
func (n NetPacketType) MarshalPacket() ([]byte, error) {
	p, err := json.Marshal(n)
	if err != nil {
		return p, err
	}

	return p, nil
}
