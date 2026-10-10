package d2realmclient

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2mp"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2realm"
)

func TestPeerIDRoundTrip(t *testing.T) {
	for _, u := range []uint32{1, 7, 4000000000} {
		got, ok := UnitOfPeer(PeerID(u))
		if !ok || got != u {
			t.Errorf("UnitOfPeer(PeerID(%d)) = %d, %v", u, got, ok)
		}
	}

	for _, bad := range []string{"", "u", "ux", "host-id", "7", "u-1"} {
		if _, ok := UnitOfPeer(bad); ok {
			t.Errorf("UnitOfPeer(%q) should fail", bad)
		}
	}
}

func TestRealmCommand(t *testing.T) {
	for _, tc := range []struct {
		op, target string
		want       d2mp.Command
		ok         bool
	}{
		{d2netpacket.PartyInvite, "u12", d2mp.Command{Type: d2mp.CmdPartyInvite, Target: 12}, true},
		{d2netpacket.PartyInvite, "host-id", d2mp.Command{}, false},
		{d2netpacket.PartyAccept, "", d2mp.Command{Type: d2mp.CmdPartyAccept}, true},
		{d2netpacket.PartyLeave, "", d2mp.Command{Type: d2mp.CmdPartyLeave}, true},
		{d2netpacket.PartyDecline, "", d2mp.Command{}, false},
		{d2netpacket.PartyHostile, "u3", d2mp.Command{}, false},
		{"nonsense", "", d2mp.Command{}, false},
	} {
		got, ok, reason := RealmCommand(tc.op, tc.target)
		if ok != tc.ok {
			t.Errorf("RealmCommand(%q, %q) ok=%v", tc.op, tc.target, ok)
		}

		if ok && (got.Type != tc.want.Type || got.Target != tc.want.Target) {
			t.Errorf("RealmCommand(%q, %q) = %+v", tc.op, tc.target, got)
		}

		if !ok && reason == "" {
			t.Errorf("RealmCommand(%q, %q): refused without a reason", tc.op, tc.target)
		}
	}
}

func TestLevelMessage(t *testing.T) {
	for _, tc := range []struct {
		level   int
		act     byte
		wantErr bool
	}{
		{1, 0, false},   // Rogue Encampment
		{40, 1, false},  // Lut Gholein
		{75, 2, false},  // Kurast Docks
		{103, 3, false}, // Pandemonium Fortress
		{109, 4, false}, // Harrogath
		{0, 0, true},
		{-5, 0, true},
	} {
		m, err := LevelMessage(tc.level)
		if (err != nil) != tc.wantErr || err == nil && (m.Act != tc.act || int(m.Level) != tc.level) {
			t.Errorf("LevelMessage(%d) = %+v, %v", tc.level, m, err)
		}
	}
}

func TestExplain(t *testing.T) {
	for code, want := range map[d2realm.Code]string{
		d2realm.CodeGameNotFound: "no game named",
		d2realm.CodeGameFull:     "full",
		d2realm.CodeBadPassword:  "password",
		d2realm.CodeModeMismatch: "hardcore",
		d2realm.CodeLevelTooLow:  "too low",
		d2realm.CodeLevelTooHigh: "too high",
	} {
		got := Explain(&d2realm.RequestError{Code: code, Message: "x"})
		if !strings.Contains(got, want) {
			t.Errorf("Explain(%v) = %q, want it to mention %q", code, got, want)
		}
	}

	if got := Explain(errors.New("connection refused")); !strings.Contains(got, "could not reach") {
		t.Errorf("Explain(net error) = %q", got)
	}

	if Explain(nil) != "" {
		t.Error("Explain(nil) must be empty")
	}
}

// The joiner retries while the host has not created the game, but a refusal
// that waiting cannot change ends the wait at once.
func TestRetryJoin(t *testing.T) {
	notFound := &d2realm.RequestError{Code: d2realm.CodeGameNotFound}
	full := &d2realm.RequestError{Code: d2realm.CodeGameFull}
	start := time.Unix(1000, 0)

	run := func(errs []error, deadlineIn time.Duration) (calls int, err error) {
		now := start
		_, err = retryJoin(func() (d2realm.GameJoined, error) {
			e := errs[calls]
			calls++

			return d2realm.GameJoined{UnitID: 9}, e
		}, start.Add(deadlineIn), func() { now = now.Add(time.Second) }, func() time.Time { return now })

		return calls, err
	}

	if calls, err := run([]error{notFound, notFound, nil}, time.Minute); err != nil || calls != 3 {
		t.Errorf("not-found then success: calls=%d err=%v", calls, err)
	}

	if calls, err := run([]error{notFound, full, nil}, time.Minute); err == nil || calls != 2 {
		t.Errorf("full must stop the retry: calls=%d err=%v", calls, err)
	}

	errs := make([]error, 50)
	for i := range errs {
		errs[i] = notFound
	}

	// attempts at t=0..3 s fail with time left or just reached; the first one after the deadline ends it
	if calls, err := run(errs, 3*time.Second); err == nil || calls != 5 {
		t.Errorf("deadline: calls=%d err=%v", calls, err)
	}
}

// Level changes and party commands of the engine reach the other bridge:
// PlayerLevel is recorded, the roster packet carries both heroes and, after an
// invite and an accept, the same party id.
func TestLevelAndPartyThroughRealm(t *testing.T) {
	rules := d2mp.NewEngineRules(126.9, 117.5, 0)
	srv := d2realm.New(d2realm.Config{Store: d2realm.NewMemStore(), Rules: rules})

	addr, err := srv.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	defer srv.Close()

	host, hs := testBridge(rules, "Alice")
	join, js := testBridge(rules, "Bobby")

	if err = host.Connect(addr.String(), time.Second); err != nil {
		t.Fatal(err)
	}

	hj, err := host.Create(DefaultGame)
	if err != nil {
		t.Fatal(err)
	}

	host.SetSelf("host-id")

	if err = host.Announce(hj.Seed, d2enum.RegionAct1Town); err != nil {
		t.Fatal(err)
	}

	if err = host.Begin("host-id", PlayerAdd{Name: "Alice", Class: d2s.Sorceress, X: -1, Y: -1}); err != nil {
		t.Fatal(err)
	}

	host.Resume()

	defer host.Close()

	if err = join.Connect(addr.String(), time.Second); err != nil {
		t.Fatal(err)
	}

	jj, err := join.Join(DefaultGame, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	join.SetSelf("join-id")

	if err = join.Announce(jj.Seed, d2enum.RegionAct1Town); err != nil {
		t.Fatal(err)
	}

	if err = join.Begin("join-id", PlayerAdd{Name: "Bobby", Class: d2s.Sorceress, X: -1, Y: -1}); err != nil {
		t.Fatal(err)
	}

	join.Resume()

	defer join.Close()

	hs.wait(t, "host sees the joiner", func() bool { return len(hs.find(d2netpackettype.AddPlayer)) == 2 })
	js.wait(t, "joiner sees the host", func() bool { return len(js.find(d2netpackettype.AddPlayer)) == 2 })

	peer := PeerID(jj.UnitID)

	// the joiner walks to the Blood Moor (level 2); the host learns it
	lc, _ := d2netpacket.CreateChangeLevelPacket("join-id", 2, 10, 10)
	if err = join.Send(lc); err != nil {
		t.Fatal(err)
	}

	hs.wait(t, "host learns the joiner's level", func() bool { l, ok := host.PeerLevel(peer); return ok && l == 2 })

	if !host.SameLevel(peer) {
		t.Error("the host has not reported a level yet: a peer counts as present")
	}

	lc, _ = d2netpacket.CreateChangeLevelPacket("host-id", 1, 0, 0)
	if err = host.Send(lc); err != nil {
		t.Fatal(err)
	}

	if host.SameLevel(peer) {
		t.Error("host in level 1 and joiner in level 2 must differ")
	}

	// party: the host invites the joiner, who accepts
	inv, _ := d2netpacket.CreatePartyCommandPacket(d2netpacket.PartyInvite, peer)
	if err = host.Send(inv); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	acc, _ := d2netpacket.CreatePartyCommandPacket(d2netpacket.PartyAccept, "")
	if err = join.Send(acc); err != nil {
		t.Fatal(err)
	}

	inParty := func(s *engineSink) bool {
		pk := s.find(d2netpackettype.RosterUpdate)
		if len(pk) == 0 {
			return false
		}

		u, err := d2netpacket.UnmarshalRosterUpdate(pk[len(pk)-1].PacketData)
		if err != nil || len(u.Roster.Players) != 2 {
			return false
		}

		return u.Roster.Players[0].Party != 0 && u.Roster.Players[0].Party == u.Roster.Players[1].Party
	}

	hs.wait(t, "host roster shows the party", func() bool { return inParty(hs) })
	js.wait(t, "joiner roster shows the party", func() bool { return inParty(js) })
}

// A join to a game that does not exist fails with a sentence for the player
// once the retry time is over.
func TestJoinMissingGame(t *testing.T) {
	rules := d2mp.NewEngineRules(126.9, 117.5, 0)
	srv := d2realm.New(d2realm.Config{Store: d2realm.NewMemStore(), Rules: rules})

	addr, err := srv.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	defer srv.Close()

	join, _ := testBridge(rules, "Bobby")
	if err = join.Connect(addr.String(), time.Second); err != nil {
		t.Fatal(err)
	}

	defer join.Close()

	_, err = join.Join(DefaultGame, 400*time.Millisecond)
	if err == nil || !strings.Contains(Explain(err), "no game named") {
		t.Fatalf("join of a missing game: %v / %q", err, Explain(err))
	}
}
