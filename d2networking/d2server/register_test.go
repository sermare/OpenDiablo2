package d2server

import (
	"errors"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
)

// A joiner whose hero could not be loaded sends a connection request without a
// hero. The server must refuse it (and close the connection) instead of
// dereferencing the missing state in OnClientConnected.
func TestRegisterConnectionWithoutHeroIsRefused(t *testing.T) {
	g := &GameServer{connections: map[string]ClientConnection{}, soc: newSocial(), Logger: d2util.NewLogger(), maxConnections: 8}

	noHero, err := d2netpacket.CreatePlayerConnectionRequestPacket("joiner", nil)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		data []byte
		want error
	}{
		{"null hero", noHero.PacketData, errNoPlayerState},
		{"unparsable request", []byte("{"), nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, err := g.registerConnection(tc.data, nil, func(string) ClientConnection { return &fakeConn{} })
			if err == nil || client != nil {
				t.Fatalf("got client=%v err=%v, want a refusal", client, err)
			}

			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}

			if len(g.connections) != 0 {
				t.Fatalf("a refused request left %d connections", len(g.connections))
			}
		})
	}
}
