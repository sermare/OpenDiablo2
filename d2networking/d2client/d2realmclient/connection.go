package d2realmclient

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gsnet"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2mp"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2realm"
)

const (
	logPrefix       = "Realm Client"
	defaultPort     = "6669"
	subtiles        = 5
	middleOfTile    = 3 // the offset the game server adds to a hero's start subtile
	defaultDummies  = 3
	defaultRetrySec = 30
)

// Enabled reports whether network games go through the realm: OD2_PROTO is
// unset or "realm". OD2_PROTO=d2gs (D2GS framing with the engine's JSON
// packets inside) and OD2_PROTO=json keep the older direct connection.
func Enabled() bool {
	p := os.Getenv("OD2_PROTO")

	return p == "" || p == "realm"
}

// StartPositioner is implemented by the game client: where the engine puts
// its heroes (known once the first level was generated).
type StartPositioner interface{ StartPosition() (x, y float64) }

// GameInfoListener is implemented by the game client: the game's map seed and
// difficulty, which every level of the game uses.
type GameInfoListener interface {
	SetGameInfo(mapSeed uint32, difficulty uint8)
}

var heroOfClass = map[d2s.Class]d2enum.Hero{
	d2s.Amazon: d2enum.HeroAmazon, d2s.Sorceress: d2enum.HeroSorceress, d2s.Necromancer: d2enum.HeroNecromancer,
	d2s.Paladin: d2enum.HeroPaladin, d2s.Barbarian: d2enum.HeroBarbarian, d2s.Druid: d2enum.HeroDruid,
	d2s.Assassin: d2enum.HeroAssassin,
}

func classOfHero(h d2enum.Hero) d2s.Class {
	for c, hh := range heroOfClass {
		if hh == h {
			return c
		}
	}

	return d2s.Sorceress
}

// Connection is the ServerConnection of a game played through the realm. The
// host runs the realm (authoritative simulation included) inside its own
// process and plays on it; a joiner plays on the host's realm.
type Connection struct {
	asset    *d2asset.AssetManager
	factory  *d2hero.HeroStateFactory
	listener d2networking.ClientListener
	host     bool
	uniqueID string
	state    *d2hero.HeroState
	bridge   *Bridge
	srv      *d2realm.Server

	*d2util.Logger
}

// Create constructs a connection; host says whether this process runs the realm.
func Create(l d2util.LogLevel, asset *d2asset.AssetManager, host bool) (*Connection, error) {
	factory, err := d2hero.NewHeroStateFactory(asset)
	if err != nil {
		return nil, err
	}

	c := &Connection{asset: asset, factory: factory, host: host, uniqueID: uuid.New().String()}
	c.Logger = d2util.NewLogger()
	c.Logger.SetPrefix(logPrefix)
	c.Logger.SetLevel(l)

	return c, nil
}

// SetClientListener sets the game client.
func (c *Connection) SetClientListener(l d2networking.ClientListener) { c.listener = l }

// GetUniqueID returns the engine player id of the local hero.
func (c *Connection) GetUniqueID() string { return c.uniqueID }

// GetPlayerState returns the local hero's state (the connection saves it).
func (c *Connection) GetPlayerState() *d2hero.HeroState { return c.state }

func realmPort() string {
	if p := os.Getenv("OD2_PORT"); p != "" {
		return p
	}

	return defaultPort
}

func retryFor() time.Duration {
	if v, err := strconv.Atoi(os.Getenv("OD2_JOIN_RETRY")); err == nil && v > 0 {
		return time.Duration(v) * time.Second
	}

	return 0
}

// dummies is the number of training dummies of the town (OD2_REALM_DUMMIES).
func dummies() int {
	if v, err := strconv.Atoi(os.Getenv("OD2_REALM_DUMMIES")); err == nil && v >= 0 {
		return v
	}

	return defaultDummies
}

// Open starts the realm (host) or finds it (joiner), enters the game and hands
// the engine the packets of a server that has just accepted a player.
func (c *Connection) Open(connectionString, saveFilePath string) error {
	c.state = c.factory.LoadHeroState(saveFilePath)
	if c.state == nil || c.state.HeroType == d2enum.HeroNone {
		return fmt.Errorf("realm: cannot load the hero %q", saveFilePath)
	}

	st := c.state
	level := 1

	if st.Stats != nil {
		level = st.Stats.Level
	}

	c.bridge = New(Config{
		Rules: d2mp.NewEngineRules(0, 0, dummies()),
		Hero: Hero{Name: st.HeroName, Class: classOfHero(st.HeroType), Level: level, Hardcore: st.Hardcore,
			Difficulty: byte(st.Difficulty), D2S: st.D2SBase},
		Sink:      c.listener.OnPacketReceived,
		NewPlayer: c.newPlayer,
		Other:     c.other,
		Logf:      func(f string, a ...interface{}) { c.Infof("REALM "+f, a...) },
	})
	c.bridge.SetSelf(c.uniqueID)

	if c.host {
		return c.openHost(saveFilePath)
	}

	return c.openJoin(connectionString)
}

func (c *Connection) openHost(string) error {
	seed := uint32(d2gsnet.GameSeed(c.state.MapSeed, rand.Int63()))

	// the engine builds the town first: its start position is the spawn of the
	// simulation, and the town depends on the seed the game will have
	if gi, ok := c.listener.(GameInfoListener); ok {
		gi.SetGameInfo(seed, byte(c.state.Difficulty))
	}

	if err := c.bridge.Announce(seed, d2enum.RegionAct1Town); err != nil {
		return err
	}

	rules := d2mp.NewEngineRules(0, 0, dummies())

	if sp, ok := c.listener.(StartPositioner); ok {
		x, y := sp.StartPosition()
		// the engine places the hero on subtile int(x*5)+3
		rules = d2mp.NewEngineRules(float64(int(x*subtiles)+middleOfTile)/subtiles, float64(int(y*subtiles)+middleOfTile)/subtiles, dummies())
	}

	c.bridge.SetRules(rules)

	c.srv = d2realm.New(d2realm.Config{Store: d2realm.NewMemStore(), Rules: rules, ServerName: "OpenDiablo2 host",
		Rand: func() uint32 { return seed }, Logf: func(f string, a ...interface{}) { c.Infof("REALM server: "+f, a...) }})

	bind := "0.0.0.0"
	if v := os.Getenv("OD2_BIND"); v != "" {
		bind = v
	}

	addr, err := c.srv.Listen(net.JoinHostPort(bind, realmPort()))
	if err != nil {
		return err
	}

	c.Infof("Starting realm @ %s", addr)

	if err = c.bridge.Connect(net.JoinHostPort("127.0.0.1", strconv.Itoa(addr.(*net.TCPAddr).Port)), 5*time.Second); err != nil {
		return err
	}

	j, err := c.bridge.Create(DefaultGame)
	if err != nil {
		return err
	}

	c.Infof("PLAYER JOIN name=%q id=%s level=%d players=1 proto=realm unit=%d seed=%#x", c.state.HeroName, c.uniqueID,
		c.bridge.cfg.Hero.Level, j.UnitID, j.Seed)

	return c.bridge.Begin(c.uniqueID, PlayerAdd{Name: c.state.HeroName, X: -1, Y: -1})
}

func (c *Connection) openJoin(addr string) error {
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, realmPort())
	}

	retry := retryFor()

	if err := c.bridge.Connect(addr, retry); err != nil {
		return err
	}

	j, err := c.bridge.Join(DefaultGame, retry)
	if err != nil {
		c.Errorf("JOIN refused: %s", Explain(err))

		return fmt.Errorf("%s: %w", Explain(err), err)
	}

	c.Infof("PLAYER JOIN name=%q id=%s level=%d proto=realm unit=%d seed=%#x", c.state.HeroName, c.uniqueID,
		c.bridge.cfg.Hero.Level, j.UnitID, j.Seed)

	if err = c.bridge.Announce(j.Seed, d2enum.RegionAct1Town); err != nil {
		return err
	}

	if gi, ok := c.listener.(GameInfoListener); ok {
		gi.SetGameInfo(j.Seed, j.Game.Difficulty)
	}

	return c.bridge.Begin(c.uniqueID, PlayerAdd{Name: c.state.HeroName, X: -1, Y: -1})
}

// Resume starts the delivery of the realm's events (the game client calls it
// once it queues packets).
func (c *Connection) Resume() {
	if c.bridge != nil {
		c.bridge.Resume()
	}
}

// Close leaves the game; the host also shuts the realm down.
func (c *Connection) Close() error {
	if c.bridge != nil {
		c.bridge.Close()
	}

	if c.srv != nil {
		c.srv.Close()
		c.srv = nil
	}

	return nil
}

// SendPacketToServer sends an engine packet to the realm.
func (c *Connection) SendPacketToServer(np d2netpacket.NetPacket) error {
	if c.bridge == nil {
		return nil
	}

	if np.PacketType == d2netpackettype.PlayerDisconnectionNotification {
		err := c.bridge.Send(np)

		if c.srv != nil {
			c.srv.Close()
			c.srv = nil
		}

		return err
	}

	return c.bridge.Send(np)
}

// RealmUnitPos returns where a unit of the simulation is drawn.
func (c *Connection) RealmUnitPos(unit uint32) (x, y float64, ok bool) {
	if c.bridge == nil {
		return 0, 0, false
	}

	return c.bridge.UnitPos(unit)
}

// RealmAttack tells the realm the local hero attacks a monster.
func (c *Connection) RealmAttack(unit uint32) error {
	if c.bridge == nil {
		return nil
	}

	return c.bridge.Attack(unit)
}

// RealmSummary describes the replica of the simulation (logs, scenarios).
func (c *Connection) RealmSummary() string {
	if c.bridge == nil {
		return "no realm"
	}

	return c.bridge.Summary()
}

// newPlayer builds the AddPlayer packet of the local hero (full state) or of
// another hero (the class defaults at the level the realm reports).
func (c *Connection) newPlayer(a PlayerAdd) (d2netpacket.NetPacket, error) {
	if a.Local {
		return c.localPlayer(a)
	}

	hero := heroOfClass[a.Class]
	if hero == d2enum.HeroNone {
		hero = d2enum.HeroSorceress
	}

	stats := c.factory.CreateHeroStatsState(hero, c.asset.Records.Character.Stats[hero])
	if a.Level > 0 {
		stats.Level = a.Level
	}

	st, err := c.factory.CreateHeroState(a.Name, hero, stats)
	if err != nil {
		return d2netpacket.NetPacket{}, err
	}

	d2hero.HydrateSkills(st.Skills, c.asset)

	return d2netpacket.CreateAddPlayerPacket(a.ID, a.Name, int(a.X*subtiles), int(a.Y*subtiles), hero, st.Stats, st.Skills,
		st.Equipment, st.LeftSkill, st.RightSkill, st.Gold, nil, d2enum.DifficultyNormal)
}

func (c *Connection) localPlayer(a PlayerAdd) (d2netpacket.NetPacket, error) {
	st := c.state

	d2hero.HydrateSkills(st.Skills, c.asset)

	// a negative position is "where the engine starts its heroes"; the game
	// client resolves it once its first level exists
	x, y := -1, -1
	if a.X >= 0 {
		x, y = int(a.X*subtiles)+middleOfTile, int(a.Y*subtiles)+middleOfTile
	}

	return d2netpacket.CreateAddPlayerPacket(a.ID, st.HeroName, x, y, st.HeroType, st.Stats, st.Skills, st.Equipment,
		st.LeftSkill, st.RightSkill, st.Gold, st.Progress, st.Difficulty,
		d2netpacket.WithContainers(st.Containers),
		d2netpacket.WithMerc(st.Merc),
		d2netpacket.WithSkillBar(st.SkillBar),
		d2netpacket.WithDeath(st.Death, st.Hardcore),
		d2netpacket.WithAct(st.Act, len(st.D2SBase) > 0, st.Expansion),
	)
}

// other handles the engine packets the realm has no use for: the hero save
// (the realm keeps no engine heroes; the connection writes the save, as the
// game server does for the other connection types).
func (c *Connection) other(np d2netpacket.NetPacket) {
	if np.PacketType != d2netpackettype.SavePlayer {
		return
	}

	sp, err := d2netpacket.UnmarshalSavePlayer(np.PacketData)
	if err != nil {
		c.Errorf("SavePlayer: %v", err)

		return
	}

	c.applySave(sp)
}
