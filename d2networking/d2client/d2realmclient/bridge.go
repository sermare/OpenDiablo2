package d2realmclient

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gs"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2mp"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2realm"
)

// DefaultGame is the name of the game the menu's host creates and join looks
// for: the TCP/IP screens of the original have no game list.
const DefaultGame = "od2"

const pumpEvery = 10 * time.Millisecond

// Hero is what the realm needs to know about the local hero.
type Hero struct {
	Name       string
	Class      d2s.Class
	Level      int
	Hardcore   bool
	Difficulty byte
	// D2S is the hero's real .d2s, uploaded as is; nil uploads a new
	// level 1 character of the class (the realm rebuilds nothing from the
	// engine's own save format).
	D2S []byte
}

// PlayerAdd describes a hero the engine must create.
type PlayerAdd struct {
	ID    string
	Name  string
	Class d2s.Class
	Level int
	// X and Y are the hero's position in tiles; negative means the engine's
	// start position (the local hero).
	X, Y  float64
	Local bool
}

// Config configures a Bridge.
type Config struct {
	// Rules must be the rules of the server (EngineRules with the same spawn).
	Rules d2mp.Rules
	Hero  Hero
	// Sink receives the engine packets (the game client's OnPacketReceived).
	Sink func(d2netpacket.NetPacket) error
	// NewPlayer builds the AddPlayer packet of a hero.
	NewPlayer func(PlayerAdd) (d2netpacket.NetPacket, error)
	// Other receives the engine packets the realm has no use for (the hero
	// save).
	Other func(d2netpacket.NetPacket)
	// Logf receives log lines (nil = silent).
	Logf func(format string, args ...interface{})
}

// Bridge connects the engine's packet model to a realm: engine packets
// (move, cast, chat, disconnect) become d2gs requests, the realm's events
// (World, chat, joins, leaves) become the engine packets the game client
// already understands, plus RealmUnit packets for monsters.
type Bridge struct {
	cfg Config

	client *d2realm.Client
	joined d2realm.GameJoined
	self   string

	mu      sync.Mutex
	rep     *d2mp.Replica
	known   map[uint32]string // unit id -> engine player id, heroes already announced
	mons    map[uint32]bool
	sched   []due // remote movement waiting for its server time
	skill   uint32
	hasSkil bool

	out []d2netpacket.NetPacket // packets for the engine, built under mu

	started bool
	stop    chan struct{}
	done    chan struct{}
	closing bool
	seen    Counts
}

// Counts are what the bridge has translated, for logs and tests.
type Counts struct {
	Spawns, Kills, Hits, Moves, Casts, Chats, Leaves, Events int
}

type due struct {
	at float64
	np d2netpacket.NetPacket
}

// New returns a bridge that is not connected yet.
func New(cfg Config) *Bridge {
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...interface{}) {}
	}

	return &Bridge{cfg: cfg, known: map[uint32]string{}, mons: map[uint32]bool{}, stop: make(chan struct{}), done: make(chan struct{})}
}

// PeerID is the engine player id of a hero of the realm.
func PeerID(unit uint32) string { return fmt.Sprintf("u%d", unit) }

// Connect dials the realm (retrying for up to retry, the host may still be
// starting), logs in and uploads the hero.
func (b *Bridge) Connect(addr string, retry time.Duration) error {
	deadline := time.Now().Add(retry)

	var err error

	for {
		if b.client, err = d2realm.Dial(addr); err == nil {
			break
		}

		if time.Now().After(deadline) {
			return err
		}

		time.Sleep(300 * time.Millisecond)
	}

	if _, err = b.client.Hello(AccountName(b.cfg.Hero.Name)); err != nil {
		return err
	}

	if len(b.cfg.Hero.D2S) > 0 {
		if err = b.client.Upload(b.cfg.Hero.D2S); err == nil {
			return nil
		}

		b.cfg.Logf("realm: the hero's own .d2s was refused (%v); playing as a new character", err)
	}

	cls := b.cfg.Hero.Class

	data, err := d2s.NewCharacter(CharName(b.cfg.Hero.Name), cls,
		d2s.NewCharacterFlags{Expansion: true, Hardcore: b.cfg.Hero.Hardcore}, d2s.DefaultAppearance(cls))
	if err != nil {
		return err
	}

	return b.client.Upload(data)
}

// SetRules sets the rules of the replica (the server's rules, known once the
// engine's start position is).
func (b *Bridge) SetRules(r d2mp.Rules) { b.cfg.Rules = r }

// Create creates the game and enters it. A difficulty the character has not
// unlocked is replaced by Normal.
func (b *Bridge) Create(game string) (d2realm.GameJoined, error) {
	cg := d2realm.CreateGame{Name: game, MaxPlayers: d2realm.MaxPlayers, Difficulty: b.cfg.Hero.Difficulty}

	j, err := b.client.Create(cg)
	if re, ok := err.(*d2realm.RequestError); ok && re.Code == d2realm.CodeDifficultyLocked {
		cg.Difficulty = d2realm.Normal
		j, err = b.client.Create(cg)
	}

	if err == nil {
		b.joined = j
	}

	return j, err
}

// Join enters the game, retrying while it does not exist yet (the host is
// still creating it) for up to retry.
func (b *Bridge) Join(game string, retry time.Duration) (d2realm.GameJoined, error) {
	deadline := time.Now().Add(retry)

	for {
		j, err := b.client.Join(game, "")
		if err == nil {
			b.joined = j

			return j, nil
		}

		if re, ok := err.(*d2realm.RequestError); (ok && re.Code != d2realm.CodeGameNotFound) || time.Now().After(deadline) {
			return j, err
		}

		time.Sleep(300 * time.Millisecond)
	}
}

// Begin sets up the replica of the game that was entered, tells the engine
// about the local hero and prepares the event pump (Resume starts it).
func (b *Bridge) Begin(selfID string, self PlayerAdd) error {
	b.self = selfID

	start := time.Now()
	b.rep = d2mp.NewReplica(d2mp.ReplicaConfig{Self: b.joined.UnitID, Seed: b.joined.Seed, Rules: b.cfg.Rules,
		Clock: func() float64 { return float64(time.Since(start)) / float64(time.Millisecond) }})
	b.rep.Handler = b.handle

	self.ID, self.Local = selfID, true

	np, err := b.cfg.NewPlayer(self)
	if err != nil {
		return err
	}

	return b.cfg.Sink(np)
}

// Joined returns the realm's answer to the create or join.
func (b *Bridge) Joined() d2realm.GameJoined { return b.joined }

// Resume starts delivering the realm's events to the engine. The game client
// asks for it once it queues packets from other goroutines.
func (b *Bridge) Resume() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.started || b.rep == nil {
		return
	}

	b.started = true

	go b.pump()
}

// Close leaves the game and disconnects.
func (b *Bridge) Close() {
	b.mu.Lock()
	if b.closing {
		b.mu.Unlock()

		return
	}

	b.closing = true
	started := b.started
	b.mu.Unlock()

	if b.client != nil {
		_ = b.client.Leave()
		time.Sleep(50 * time.Millisecond) // the leave is on the wire before the socket goes
	}

	if started {
		close(b.stop)
		<-b.done
	}

	if b.client != nil {
		_ = b.client.Close()
	}
}

// Send handles an engine packet for the realm.
func (b *Bridge) Send(np d2netpacket.NetPacket) error {
	if b.client == nil {
		return nil
	}

	switch np.PacketType {
	case d2netpackettype.MovePlayer:
		p, err := d2netpacket.UnmarshalMovePlayer(np.PacketData)
		if err != nil {
			return err
		}

		if err = b.client.Walk(false, p.DestX, p.DestY); err != nil {
			return err
		}

		// a game server echoes a move to its sender, and the engine walks its own hero when the echo arrives
		return b.cfg.Sink(np)
	case d2netpackettype.CastSkill:
		p, err := d2netpacket.UnmarshalCast(np.PacketData)
		if err != nil {
			return err
		}

		b.mu.Lock()
		changed := !b.hasSkil || b.skill != uint32(p.SkillID)
		b.skill, b.hasSkil = uint32(p.SkillID), true
		b.mu.Unlock()

		if changed {
			if err := b.client.SelectSkill(uint16(p.SkillID), false); err != nil {
				return err
			}
		}

		if err = b.client.Cast(false, p.TargetX, p.TargetY); err != nil {
			return err
		}

		return b.cfg.Sink(np) // the echo the game client skips (it played the cast itself)
	case d2netpackettype.Chat:
		p, err := d2netpacket.UnmarshalChat(np.PacketData)
		if err != nil {
			return err
		}

		return b.client.Say(p.Text, "")
	case d2netpackettype.PlayerDisconnectionNotification:
		b.Close()

		return nil
	}

	if b.cfg.Other != nil {
		b.cfg.Other(np)
	}

	return nil
}

// Attack tells the realm the hero attacks a monster (the realm walks the hero
// up to it and decides the damage).
func (b *Bridge) Attack(unit uint32) error {
	if b.client == nil {
		return nil
	}

	return b.client.Interact(d2mp.KindMonster, unit)
}

// UnitPos returns where the realm's unit is drawn now, in tiles.
func (b *Bridge) UnitPos(unit uint32) (x, y float64, ok bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.rep == nil {
		return 0, 0, false
	}

	return b.rep.Pos(unit)
}

// Digest hashes the units of the current level as this client sees them.
func (b *Bridge) Digest() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.rep == nil {
		return 0
	}

	return b.rep.Digest()
}

// Counts returns what the bridge has translated so far.
func (b *Bridge) Counts() Counts {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.seen
}

// StableDigest hashes what lasts: heroes, living monsters and objects, with
// their final positions and life. Corpses and ground items come and go on
// timers (a corpse is removed seconds after the death), so two clients that
// look at slightly different moments still agree on this.
func (b *Bridge) StableDigest() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.stableDigest()
}

func (b *Bridge) stableDigest() uint64 {
	var keep []*d2mp.Unit

	for _, u := range b.rep.Units() {
		if u.Kind == d2mp.KindItem || u.Kind == d2mp.KindMissile || u.Dead {
			continue
		}

		keep = append(keep, &u.Unit)
	}

	return d2mp.DigestUnits(keep)
}

// Summary describes the replica for logs and scenarios: heroes, living
// monsters, monsters seen to die (kills), the stable digest and the experience.
func (b *Bridge) Summary() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.rep == nil {
		return "no game"
	}

	var heroes, alive int

	for _, u := range b.rep.Units() {
		switch {
		case u.Kind == d2mp.KindPlayer && !u.Dead:
			heroes++
		case u.Kind == d2mp.KindMonster && !u.Dead:
			alive++
		}
	}

	return fmt.Sprintf("level=%d heroes=%d monsters_alive=%d kills=%d digest=%016x xp=%d", b.rep.Level, heroes, alive,
		b.seen.Kills, b.stableDigest(), b.rep.XP)
}

func (b *Bridge) pump() {
	defer close(b.done)

	t := time.NewTicker(pumpEvery)
	defer t.Stop()

	for {
		select {
		case <-b.stop:
			return
		case <-t.C:
			b.drain()
		}
	}
}

// drain applies everything the realm has sent.
func (b *Bridge) drain() {
	evs := b.client.Drain(func(interface{}) bool { return true })

	b.mu.Lock()

	for _, e := range evs {
		switch m := e.(type) {
		case d2realm.World:
			for _, ev := range m.Events {
				b.seen.Events++
				b.rep.Apply(ev)
			}
		case d2realm.Chat:
			b.seen.Chats++
			b.chat(m)
		case d2gs.UnitSkillOnLocation:
			if id, ok := b.known[m.UnitID]; ok && m.UnitID != b.joined.UnitID {
				b.seen.Casts++
				b.emit(d2netpacket.CreateCastPacket(id, int(m.Skill), d2mp.FromSub(m.X), d2mp.FromSub(m.Y)))
			}
		case d2realm.PlayerLeave:
			if id, ok := b.known[m.UnitID]; ok {
				b.seen.Leaves++
				delete(b.known, m.UnitID)
				b.emit(d2netpacket.CreatePlayerDisconnectRequestPacket(id))
			}
		case d2realm.Disconnected:
			if !b.closing {
				b.emit(d2netpacket.CreateServerClosedPacket())
			}
		}
	}

	// remote movement whose server time (plus the interpolation delay) has come
	now := b.rep.Now() - d2mp.DefaultInterpMs
	n := 0

	for n < len(b.sched) && b.sched[n].at <= now {
		b.out = append(b.out, b.sched[n].np)
		n++
	}

	b.sched = b.sched[n:]
	out := b.out
	b.out = nil
	b.mu.Unlock()

	for _, np := range out {
		if err := b.cfg.Sink(np); err != nil {
			b.cfg.Logf("realm: engine refused %s: %v", np.PacketType, err)
		}
	}
}

func (b *Bridge) emit(np d2netpacket.NetPacket, err error) {
	if err != nil {
		b.cfg.Logf("realm: building a packet: %v", err)

		return
	}

	b.out = append(b.out, np)
}

func (b *Bridge) chat(m d2realm.Chat) {
	id := ""
	if m.UnitID == b.joined.UnitID {
		id = b.self
	} else if k, ok := b.known[m.UnitID]; ok {
		id = k
	}

	b.emit(d2netpacket.CreateChatPacket(id, m.Name, m.Text))
}

// handle is the replica's event hook; b.mu is held (it runs inside Apply).
func (b *Bridge) handle(e d2mp.Event) {
	switch e.Type {
	case d2mp.EvLevel:
		b.cfg.Logf("realm: level %d", e.Level)
	case d2mp.EvSpawn:
		b.spawn(e)
	case d2mp.EvSeg:
		b.seg(e)
	case d2mp.EvHit:
		if b.mons[e.ID] {
			b.seen.Hits++
			x, y, _ := b.rep.Pos(e.ID)

			if u := b.rep.Unit(e.ID); u != nil {
				b.emit(d2netpacket.CreateRealmUnitPacket(d2netpacket.RealmUnitPacket{Op: d2netpacket.RealmUnitHit, UnitID: e.ID,
					X: x, Y: y, HP: e.B, MaxHP: u.MaxHP}))
			}
		}
	case d2mp.EvDeath:
		if b.mons[e.ID] {
			b.seen.Kills++
			x, y, _ := b.rep.Pos(e.ID)

			killer := ""
			if e.Other == b.joined.UnitID {
				killer = b.self
			} else if k, ok := b.known[e.Other]; ok {
				killer = k
			}

			b.emit(d2netpacket.CreateRealmUnitPacket(d2netpacket.RealmUnitPacket{Op: d2netpacket.RealmUnitDeath, UnitID: e.ID,
				X: x, Y: y, Killer: killer}))
		}
	case d2mp.EvMsg:
		b.cfg.Logf("realm: %s", e.Text)
	}
}

func (b *Bridge) spawn(e d2mp.Event) {
	u := e.Unit

	switch u.Kind {
	case d2mp.KindPlayer:
		if u.ID == b.joined.UnitID {
			return
		}

		if _, ok := b.known[u.ID]; ok {
			return
		}

		id := PeerID(u.ID)
		b.known[u.ID] = id
		b.seen.Spawns++

		x, y, _ := b.rep.Pos(u.ID)
		np, err := b.cfg.NewPlayer(PlayerAdd{ID: id, Name: u.Name, Class: d2s.Class(u.Type), Level: 1, X: x, Y: y})
		b.emit(np, err)
	case d2mp.KindMonster:
		if b.mons[u.ID] {
			return
		}

		b.mons[u.ID] = true
		b.seen.Spawns++

		x, y, _ := b.rep.Pos(u.ID)
		b.emit(d2netpacket.CreateRealmUnitPacket(d2netpacket.RealmUnitPacket{Op: d2netpacket.RealmUnitSpawn, UnitID: u.ID,
			Type: u.Type, Name: u.Name, X: x, Y: y, HP: u.HP, MaxHP: u.MaxHP}))
	}
}

func (b *Bridge) seg(e d2mp.Event) {
	id, ok := b.known[e.ID]
	if !ok || e.ID == b.joined.UnitID || e.Seg.Speed <= 0 {
		return
	}

	b.seen.Moves++

	np, err := d2netpacket.CreateMovePlayerPacket(id, e.Seg.X0, e.Seg.Y0, e.Seg.X1, e.Seg.Y1)
	if err != nil {
		b.cfg.Logf("realm: move packet: %v", err)

		return
	}

	b.sched = append(b.sched, due{at: float64(e.Seg.T0), np: np})
	sort.SliceStable(b.sched, func(i, j int) bool { return b.sched[i].at < b.sched[j].at })
}

// Announce tells the engine the game's seed and difficulty and has it build
// the level: the packets a server sends first.
func (b *Bridge) Announce(seed uint32, region d2enum.RegionIdType) error {
	usi, err := d2netpacket.CreateUpdateServerInfoPacket(int64(seed), b.self)
	if err != nil {
		return err
	}

	if err = b.cfg.Sink(usi); err != nil {
		return err
	}

	gm, err := d2netpacket.CreateGenerateMapPacket(region)
	if err != nil {
		return err
	}

	return b.cfg.Sink(gm)
}

// SetSelf records the local engine player id before Announce.
func (b *Bridge) SetSelf(id string) { b.self = id }
