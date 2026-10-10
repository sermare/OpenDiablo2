package d2gsnet

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gs"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

// seedStream returns a few well-formed tunnelled packets (synthetic) as one plain byte stream.
func seedStream() []byte {
	var out []byte

	for _, np := range []d2netpacket.NetPacket{
		{PacketType: d2netpackettype.Ping, PacketData: []byte(`{"TS":1}`)},
		{PacketType: d2netpackettype.PlayerConnectionRequest, PacketData: []byte(`{"id":"x","playerName":"p"}`)},
		{PacketType: d2netpackettype.MovePlayer, PacketData: []byte(`{"playerId":"x","startX":1,"startY":2,"destX":3,"destY":4}`)},
	} {
		for _, p := range tunnelNP(d2gs.CtlTunnel, np) {
			out = append(out, p...)
		}
	}

	return out
}

// FuzzServerSideDecode feeds arbitrary client streams to the server translation layer.
func FuzzServerSideDecode(f *testing.F) {
	f.Add(seedStream())
	f.Add([]byte{})
	f.Add([]byte{d2gs.CtlTunnel, 0xff, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, data []byte) {
		s := &ServerSide{IDs: NewIDs(), PlayerID: "p", Pos: func() (float64, float64) { return 1, 1 }}

		_, _, _ = s.Decode(data)
		_, _, _ = s.Decode(data) // state carried over (tunnel assembler, joined)
	})
}

// FuzzClientSideDecode feeds arbitrary server streams to the client translation layer.
func FuzzClientSideDecode(f *testing.F) {
	f.Add(seedStream())
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x01, 0x02})

	f.Fuzz(func(t *testing.T, data []byte) {
		c := &ClientSide{IDs: NewIDs(), PlayerID: "p"}

		_, _, _ = c.Decode(data)
		_, _, _ = c.Decode(data)
	})
}

// FuzzNetPacketJSON feeds arbitrary bytes to the engine packet decoders.
func FuzzNetPacketJSON(f *testing.F) {
	f.Add([]byte(`{"packetType":1,"packetData":{}}`))
	f.Add([]byte(`[]`))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = d2netpacket.InspectPacketType(data)

		if np, err := d2netpacket.UnmarshalNetPacket(data); err == nil {
			_, _ = d2netpacket.UnmarshalChangeLevel(np.PacketData)
			_, _ = d2netpacket.UnmarshalSetWaypoint(np.PacketData)
		}
	})
}
