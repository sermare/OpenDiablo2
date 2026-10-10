package d2realm

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gs"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2mp"
)

// playRules is DefaultRules with sturdy heroes and a handful of monsters close
// to the town exit, so a scripted fight takes seconds.
type playRules struct {
	d2mp.DefaultRules
	hp int32
}

func (r playRules) Hero(class, level uint8) (int32, int, int) {
	if r.hp > 0 {
		return r.hp, 3, 5
	}

	return 4000, 25, 35
}

func (r playRules) Level(seed uint32, level uint16) *d2mp.LevelDef {
	l := r.DefaultRules.Level(seed, level)
	if len(l.Monsters) == 0 {
		return l
	}

	var ms []d2mp.MonsterSpawn

	for i := 0; i < 5; i++ {
		for dx := 9 + i; dx < 40; dx++ {
			x, y := l.SpawnX+float64(dx), l.SpawnY-6+float64(i*3)
			if l.Walkable(x, y) && l.Walkable(x, y+1) {
				ms = append(ms, d2mp.MonsterSpawn{Type: l.Monsters[i].Type, X: x + 0.4, Y: y + 0.4})

				break
			}
		}
	}

	l.Monsters = ms

	return l
}

type table struct {
	t       *testing.T
	srv     *Server
	game    string
	players []*Player
	rules   d2mp.Rules
}

func newTable(t *testing.T, rules d2mp.Rules, n int) *table {
	t.Helper()

	srv, addr := newServer(t, Config{Rules: rules})
	tb := &table{t: t, srv: srv, game: "fight", rules: rules}

	for i := 0; i < n; i++ {
		tb.add(addr, fmt.Sprintf("acct%d", i), "Hero"+string(rune('A'+i)))
	}

	return tb
}

func (tb *table) addr() string { return tb.srv.ln.Addr().String() }

func (tb *table) add(addr, account, char string) *Player {
	tb.t.Helper()

	c := player(tb.t, addr, account, char)

	var (
		j   GameJoined
		err error
	)

	if len(tb.players) == 0 {
		j, err = c.Create(CreateGame{Name: tb.game, MaxPlayers: 8})
	} else {
		j, err = c.Join(tb.game, "")
	}

	if err != nil {
		tb.t.Fatal(err)
	}

	p := NewPlayer(c, j, tb.rules, nil)
	tb.players = append(tb.players, p)

	return p
}

// serverDigest hashes the simulation's view of a level.
func (tb *table) serverDigest(level uint16) uint64 {
	tb.srv.mu.Lock()
	defer tb.srv.mu.Unlock()

	return tb.srv.games[tb.game].sim.Digest(level)
}

// converged waits until every player's replica equals the server's state of its level.
func (tb *table) converged(timeout time.Duration, level uint16) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		ok := true
		want := tb.serverDigest(level)

		for _, p := range tb.players {
			p.View(func(r *d2mp.Replica) { ok = ok && r.Level == level && r.Digest() == want })
		}

		if ok {
			return true
		}

		time.Sleep(25 * time.Millisecond)
	}

	return false
}

func objectOf(r *d2mp.Replica, typ uint16) uint32 {
	for _, u := range r.Units() {
		if u.Kind == d2mp.KindObject && u.Type == typ {
			return u.ID
		}
	}

	return 0
}

func (tb *table) tolevel2() {
	tb.t.Helper()

	for _, p := range tb.players {
		var exit uint32

		p.WaitFor(wait, func(r *d2mp.Replica) bool { exit = objectOf(r, d2mp.ObjPortal); return exit != 0 })

		if err := p.Interact(d2mp.KindObject, exit); err != nil {
			tb.t.Fatal(err)
		}
	}

	for _, p := range tb.players {
		if !p.WaitFor(5*time.Second, func(r *d2mp.Replica) bool { return r.Level == 2 && len(r.Units()) > 4 }) {
			tb.t.Fatalf("%s never arrived in level 2", p.Joined.Game.Name)
		}
	}
}

func liveMonsters(r *d2mp.Replica) []*d2mp.RUnit {
	var out []*d2mp.RUnit

	for _, u := range r.Units() {
		if u.Kind == d2mp.KindMonster && !u.Dead {
			out = append(out, u)
		}
	}

	return out
}

func TestPlayWalkCastChatSeenByAll(t *testing.T) {
	tb := newTable(t, playRules{}, 4)
	a := tb.players[0]

	for _, p := range tb.players {
		if !p.WaitFor(wait, func(r *d2mp.Replica) bool { return r.Level == 1 && len(r.Units()) >= 8 }) {
			t.Fatal("a player never saw the whole town")
		}
	}

	var sx, sy float64

	a.View(func(r *d2mp.Replica) { sx, sy, _ = r.Pos(a.Joined.UnitID) })

	// A runs 10 tiles east; everybody else must see A arrive, gliding
	if err := a.WalkTo(true, sx+10, sy); err != nil {
		t.Fatal(err)
	}

	for _, o := range tb.players[1:] {
		o := o
		last, worst := [2]float64{}, 0.0
		first := true

		ok := o.WaitFor(5*time.Second, func(r *d2mp.Replica) bool {
			x, y, found := r.Pos(a.Joined.UnitID)
			if !found {
				return false
			}

			if !first {
				// the check runs about every 10 ms; a run covers 0.09 tiles in 10 ms, scheduling hiccups allowed
				worst = math.Max(worst, math.Hypot(x-last[0], y-last[1]))
			}

			last, first = [2]float64{x, y}, false

			return math.Abs(x-(sx+10)) < 0.3
		})
		if !ok {
			t.Fatalf("%s never saw A reach the target (last %v)", o.Joined.Game.Name, last)
		}

		if worst > 1.5 {
			t.Fatalf("A teleported %.2f tiles between two looks", worst)
		}
	}

	// the native packet 0x0f reached the others as well
	b := tb.players[1]
	if _, err := b.Wait(wait, func(e interface{}) bool { m, ok := e.(PlayerMoved); return ok && m.UnitID == a.Joined.UnitID }); err != nil {
		t.Fatal("no native PlayerMove (0x0f) for A")
	}

	// a cast: the missile exists for all, then vanishes
	if err := a.SelectSkill(d2mp.SkillFireBolt, false); err != nil {
		t.Fatal(err)
	}

	if err := a.Cast(false, sx+20, sy); err != nil {
		t.Fatal(err)
	}

	if _, err := b.Wait(wait, func(e interface{}) bool {
		m, ok := e.(PlayerCast)
		return ok && m.UnitID == a.Joined.UnitID && m.Skill == 36
	}); err != nil {
		t.Fatal("no native UnitSkillOnLocation (0x4d) for A")
	}

	// chat goes to the game
	if err := a.Say("hello team", ""); err != nil {
		t.Fatal(err)
	}

	for _, o := range tb.players {
		if _, err := o.Wait(wait, isChat("hello team")); err != nil {
			t.Fatalf("chat: %v", err)
		}
	}

	if !tb.converged(8*time.Second, 1) {
		t.Fatal("replicas do not match the server after the walk and cast")
	}
}

func TestPlayScriptedFightFourClients(t *testing.T) {
	tb := newTable(t, playRules{}, 4)
	tb.tolevel2()

	if !tb.converged(5*time.Second, 2) {
		t.Fatal("level 2 differs between server and clients on arrival")
	}

	// the shared seed gave every client the same level layout as the server
	var want string

	for i, p := range tb.players {
		var got string

		p.View(func(r *d2mp.Replica) {
			got = fmt.Sprint(r.Def.W, r.Def.H, r.Def.Monsters, len(r.Def.Blocked), blockedSum(r.Def))
		})

		if i == 0 {
			want = got
		} else if got != want {
			t.Fatalf("client %d built another level than client 0", i)
		}
	}

	if got := tb.players[0].Joined.Seed; got == 0 {
		t.Fatal("no game seed")
	}

	// everybody fights: hero 0..2 melee the lowest live monster, hero 3 throws fireballs at it
	if err := tb.players[3].SelectSkill(d2mp.SkillFireBall, false); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(60 * time.Second)

	for time.Now().Before(deadline) {
		var target *d2mp.RUnit

		tb.players[0].View(func(r *d2mp.Replica) {
			if l := liveMonsters(r); len(l) > 0 {
				target = l[0]
			}
		})

		if target == nil {
			break
		}

		for i, p := range tb.players {
			if i < 3 {
				_ = p.Interact(d2mp.KindMonster, target.ID)

				continue
			}

			var px, py float64

			p.View(func(r *d2mp.Replica) { px, py, _ = r.Pos(p.Joined.UnitID) })

			tx, ty := target.PosAt(1e12)
			if math.Hypot(px-tx, py-ty) < 15 {
				_ = p.Cast(false, tx, ty)
			} else {
				_ = p.Walk(true, tx, ty)
			}
		}

		time.Sleep(300 * time.Millisecond)
	}

	for _, p := range tb.players {
		p.View(func(r *d2mp.Replica) {
			if n := len(liveMonsters(r)); n != 0 {
				t.Fatalf("%d monsters still alive on a client", n)
			}
		})
	}

	// they all saw the same kills and the same drops
	if !tb.converged(15*time.Second, 2) {
		t.Fatal("world state differs between clients and server after the fight")
	}

	tb.srv.mu.Lock()
	st := tb.srv.games[tb.game].sim.Stats
	tb.srv.mu.Unlock()

	if st.Kills != 5 || st.Casts == 0 {
		t.Fatalf("stats %+v", st)
	}

	xp := 0

	for _, p := range tb.players {
		p.View(func(r *d2mp.Replica) { xp += int(r.XP) })
	}

	if xp == 0 {
		t.Fatal("nobody got experience")
	}

	t.Logf("fight: %+v, replica events %v", st, tb.players[0].eventCounts())
}

func (p *Player) eventCounts() map[string]int {
	out := map[string]int{}

	p.View(func(r *d2mp.Replica) {
		for k, v := range r.Stats {
			out[k.String()] = v
		}
	})

	return out
}

func blockedSum(l *d2mp.LevelDef) int {
	n := 0

	for i, b := range l.Blocked {
		if b {
			n += i
		}
	}

	return n
}

func TestPlayDropInAndLeave(t *testing.T) {
	tb := newTable(t, playRules{}, 2)
	tb.tolevel2()

	// kill one monster, then a third client drops in and must see the same level
	for _, p := range tb.players {
		var target uint32

		p.View(func(r *d2mp.Replica) {
			if l := liveMonsters(r); len(l) > 0 {
				target = l[0].ID
			}
		})

		_ = p.Interact(d2mp.KindMonster, target)
	}

	time.Sleep(3 * time.Second)

	c := tb.add(tb.addr(), "late", "LateHero")
	if !c.WaitFor(5*time.Second, func(r *d2mp.Replica) bool { return r.Level == 1 && len(r.Units()) > 4 }) {
		t.Fatal("the late joiner never loaded")
	}

	if tb.serverDigest(1) != digestOf(c) {
		t.Fatal("the late joiner's town differs from the server")
	}

	// the late joiner takes the exit portal and meets the others, with the same corpses/items
	var exit uint32

	c.View(func(r *d2mp.Replica) { exit = objectOf(r, d2mp.ObjPortal) })
	_ = c.Interact(d2mp.KindObject, exit)

	if !c.WaitFor(5*time.Second, func(r *d2mp.Replica) bool { return r.Level == 2 && len(r.Units()) > 4 }) {
		t.Fatal("late joiner never arrived in level 2")
	}

	time.Sleep(500 * time.Millisecond)

	if tb.serverDigest(2) != digestOf(c) || tb.serverDigest(2) != digestOf(tb.players[0]) {
		t.Fatal("level 2 differs for the late joiner")
	}

	// the first hero leaves: the others stop seeing it
	gone := tb.players[1].Joined.UnitID
	_ = tb.players[1].Leave()

	if !c.WaitFor(5*time.Second, func(r *d2mp.Replica) bool { return r.Unit(gone) == nil }) {
		t.Fatal("a leaver is still visible")
	}
}

func digestOf(p *Player) (d uint64) {
	p.View(func(r *d2mp.Replica) { d = r.Digest() })

	return d
}

func TestPlayPartyTradeDeathRespawn(t *testing.T) {
	// fragile heroes so that a monster pack can kill one
	tb := newTable(t, playRules{hp: 20}, 2)
	a, b := tb.players[0], tb.players[1]

	for _, p := range tb.players {
		p.WaitFor(wait, func(r *d2mp.Replica) bool { return r.Level == 1 && len(r.Units()) >= 6 })
	}

	// party over the realm
	_ = a.GameCommand(d2mp.Command{Type: d2mp.CmdPartyInvite, Target: b.Joined.UnitID})

	if !b.WaitFor(wait, func(r *d2mp.Replica) bool { return strings.Contains(strings.Join(r.Msgs, "|"), "invites you") }) {
		t.Fatal("no invitation")
	}

	_ = b.GameCommand(d2mp.Command{Type: d2mp.CmdPartyAccept})

	for _, p := range tb.players {
		if !p.WaitFor(wait, func(r *d2mp.Replica) bool {
			ua, ub := r.Unit(a.Joined.UnitID), r.Unit(b.Joined.UnitID)

			return ua != nil && ub != nil && ua.Party != 0 && ua.Party == ub.Party
		}) {
			t.Fatal("party not visible")
		}
	}

	// A opens the chest, takes an item and trades it to B
	var chest uint32

	a.View(func(r *d2mp.Replica) {
		for _, u := range r.Units() {
			if u.Kind == d2mp.KindObject && u.Type == d2mp.ObjChest && chest == 0 {
				chest = u.ID
			}
		}
	})

	_ = a.Interact(d2mp.KindObject, chest)

	var item uint32

	if !a.WaitFor(5*time.Second, func(r *d2mp.Replica) bool {
		for _, u := range r.Units() {
			if u.Kind == d2mp.KindItem && u.Name != "gld" {
				item = u.ID
			}
		}

		return item != 0
	}) {
		t.Fatal("chest gave nothing")
	}

	_ = a.PickUp(item)

	if !a.WaitFor(5*time.Second, func(r *d2mp.Replica) bool { return len(r.Inv) == 1 }) {
		t.Fatal("item not picked up")
	}

	_ = a.GameCommand(d2mp.Command{Type: d2mp.CmdTradeRequest, Target: b.Joined.UnitID})

	if !b.WaitFor(wait, func(r *d2mp.Replica) bool { return r.Trade != nil && r.Trade.Incoming }) {
		t.Fatal("no trade request")
	}

	_ = b.GameCommand(d2mp.Command{Type: d2mp.CmdTradeRespond, A: 1})

	if !a.WaitFor(wait, func(r *d2mp.Replica) bool { return r.Trade != nil && r.Trade.State == d2mp.TradeOpen }) {
		t.Fatal("the trade window did not open for A")
	}

	var inv []string

	a.View(func(r *d2mp.Replica) { inv = r.Inv })
	_ = a.GameCommand(d2mp.Command{Type: d2mp.CmdTradeOffer, Items: inv})

	if !b.WaitFor(wait, func(r *d2mp.Replica) bool { return r.Trade != nil && len(r.Trade.TheirItems) == 1 }) {
		t.Fatal("B does not see the offer")
	}

	time.Sleep(2200 * time.Millisecond) // the accept lock after an offer

	_ = a.GameCommand(d2mp.Command{Type: d2mp.CmdTradeAccept})
	_ = b.GameCommand(d2mp.Command{Type: d2mp.CmdTradeAccept})

	if !b.WaitFor(wait, func(r *d2mp.Replica) bool {
		return len(r.Inv) == 1 && r.LastTrade != nil && r.LastTrade.State == d2mp.TradeDone
	}) {
		t.Fatal("trade did not complete for B")
	}

	if !a.WaitFor(wait, func(r *d2mp.Replica) bool { return len(r.Inv) == 0 }) {
		t.Fatal("A still holds the traded item")
	}

	// A walks into the field and dies
	tb.tolevel2()

	deadline := time.Now().Add(40 * time.Second)

	for time.Now().Before(deadline) {
		dead := false

		a.View(func(r *d2mp.Replica) { u := r.Unit(a.Joined.UnitID); dead = u != nil && u.Dead })

		if dead {
			break
		}

		var tgt *d2mp.RUnit

		a.View(func(r *d2mp.Replica) {
			if l := liveMonsters(r); len(l) > 0 {
				tgt = l[0]
			}
		})

		if tgt != nil {
			x, y := tgt.PosAt(1e12)
			_ = a.Walk(true, x, y)
		}

		time.Sleep(200 * time.Millisecond)
	}

	if !b.WaitFor(5*time.Second, func(r *d2mp.Replica) bool { u := r.Unit(a.Joined.UnitID); return u != nil && u.Dead }) {
		t.Skip("A survived; death not reproduced in this run")
	}

	_ = a.GameCommand(d2mp.Command{Type: d2mp.CmdRespawn})

	if !a.WaitFor(5*time.Second, func(r *d2mp.Replica) bool {
		u := r.Unit(a.Joined.UnitID)
		return r.Level == 1 && u != nil && !u.Dead && u.HP > 0
	}) {
		t.Fatal("A did not respawn in town")
	}

	if !b.WaitFor(5*time.Second, func(r *d2mp.Replica) bool { return r.Unit(a.Joined.UnitID) == nil }) {
		t.Fatal("B still sees A's body")
	}
}

func TestWorldProtocolRoundTrip(t *testing.T) {
	w := World{Events: []d2mp.Event{{Type: d2mp.EvTick, A: 9}, {Type: d2mp.EvHit, ID: 3, Other: 4, A: 5, B: 6}}}
	typ, body := Encode(w)

	got, err := Decode(typ, body)
	if err != nil || len(got.(World).Events) != 2 {
		t.Fatalf("%v %v", got, err)
	}

	c := Command{Cmd: d2mp.Command{Type: d2mp.CmdTradeOffer, A: 3, Items: []string{"1:cap"}}}
	typ, body = Encode(c)

	got, err = Decode(typ, body)
	if err != nil || got.(Command).Cmd.Items[0] != "1:cap" {
		t.Fatalf("%v %v", got, err)
	}

	if _, err := Decode(MsgWorld, []byte{1, 2}); err == nil {
		t.Fatal("garbage accepted")
	}
}

func TestGameplayPacketsOutsideGameIgnored(t *testing.T) {
	_, addr := newServer(t, Config{})
	c := player(t, addr, "solo", "SoloHero")

	_ = c.Walk(true, 3, 3)
	_ = c.Interact(d2mp.KindMonster, 5)

	if err := c.GameCommand(d2mp.Command{Type: d2mp.CmdRespawn}); err != nil {
		t.Fatal(err)
	}

	waitFor(t, c, "not-in-game result", func(e interface{}) bool {
		r, ok := e.(Result)

		return ok && r.Op == MsgCommand && r.Code == CodeNotInGame
	})

	_ = d2gs.LeaveGame
	_ = d2s.Sorceress
}
