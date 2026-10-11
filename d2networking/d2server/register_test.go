package d2server

import (
	"errors"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
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

func TestSameCharName(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"Nokka", "nokka", true},
		{"NOKKA", "Nokka", true},
		{"Nokka", "Nokkb", false},
		{"abcdefghijklmnop", "ABCDEFGHIJKLMNOQ", true}, // only the first 15 are stored
		{"abcdefghijklmno", "abcdefghijklmnp", false},
		{"", "", true},
	}

	for _, tc := range tests {
		if got := sameCharName(tc.a, tc.b); got != tc.want {
			t.Errorf("sameCharName(%q,%q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestNameInUseRefusesDuplicate(t *testing.T) {
	conns := map[string]ClientConnection{
		"a": &fakeConn{id: "a", state: &d2hero.HeroState{HeroName: "Nokka"}},
		"b": &fakeConn{id: "b"},
	}

	if !nameInUse(conns, "NOKKA") {
		t.Error("a duplicate name must be in use")
	}

	if nameInUse(conns, "Other") || nameInUse(conns, "") {
		t.Error("a free or empty name must not be in use")
	}

	g := &GameServer{connections: conns, soc: newSocial(), Logger: d2util.NewLogger(), maxConnections: 8}

	req, err := d2netpacket.CreatePlayerConnectionRequestPacket("c", &d2hero.HeroState{HeroName: "nokka"})
	if err != nil {
		t.Fatal(err)
	}

	client, err := g.registerConnection(req.PacketData, nil, func(string) ClientConnection { return &fakeConn{} })
	if !errors.Is(err, errNameInUse) || client != nil {
		t.Fatalf("got client=%v err=%v, want errNameInUse", client, err)
	}

	if len(g.connections) != 2 {
		t.Fatalf("connections = %d, want 2", len(g.connections))
	}
}
