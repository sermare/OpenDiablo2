package d2realm

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gs"
)

// ErrTimeout is returned when an expected event does not arrive.
var ErrTimeout = errors.New("d2realm: timed out")

// ErrClosed is returned once the connection is gone.
var ErrClosed = errors.New("d2realm: connection closed")

// Event types delivered by Client (besides the realm messages of
// protocol.go, which are delivered as their Go types).
type (
	// Chat is a 0x26 packet.
	Chat struct {
		Type   uint8
		UnitID uint32
		Name   string
		Text   string
	}
	// PlayerInGame is a 0x59 packet: a player is in the game.
	PlayerInGame = d2gs.AssignPlayer
	// PlayerLeave is a 0x5c packet: a player left the game.
	PlayerLeave = d2gs.PlayerLeave
	// GameFlags is a 0x01 packet.
	GameFlags = d2gs.GameFlags
	// LoadAct is a 0x03 packet; Seed is the game seed.
	LoadAct = d2gs.LoadAct
	// Disconnected is the last event of a client.
	Disconnected struct{ Err error }
)

// Client is a realm client. It is used by the tests and can back a game
// screen. Incoming packets are decoded into events kept in arrival order.
type Client struct {
	conn net.Conn

	mu     sync.Mutex
	cond   *sync.Cond
	events []interface{}
	err    error
	wmu    sync.Mutex
}

// Dial connects to a realm.
func Dial(addr string) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, err
	}

	c := &Client{conn: conn}
	c.cond = sync.NewCond(&c.mu)

	go c.readLoop()

	return c, nil
}

// Close disconnects.
func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) push(ev interface{}) {
	c.mu.Lock()
	c.events = append(c.events, ev)
	c.mu.Unlock()
	c.cond.Broadcast()
}

func (c *Client) readLoop() {
	r := bufio.NewReader(c.conn)

	var asm d2gs.TunnelAssembler

	for {
		plain, err := d2gs.ReadBlob(r)
		if err != nil {
			c.mu.Lock()
			c.err = err
			c.events = append(c.events, Disconnected{Err: err})
			c.mu.Unlock()
			c.cond.Broadcast()

			return
		}

		pkts, _ := d2gs.SplitServer(plain)
		for _, p := range pkts {
			c.dispatch(&asm, p)
		}
	}
}

func (c *Client) dispatch(asm *d2gs.TunnelAssembler, p []byte) {
	switch p[0] {
	case d2gs.S2CMetaAE:
		typ, data, done, err := asm.Add(p)
		if err != nil || !done {
			return
		}

		if msg, err := Decode(typ, data); err == nil {
			c.push(msg)
		}
	case d2gs.S2CChat:
		if m, err := d2gs.ParseChatMessage(p); err == nil {
			c.push(Chat{Type: m.Type, UnitID: m.UnitID, Name: m.Name, Text: m.Text})
		}
	case d2gs.S2CPlayerInGame:
		if m, err := d2gs.ParseAssignPlayer(p); err == nil {
			c.push(m)
		}
	case d2gs.S2CPlayerLeave:
		if m, err := d2gs.ParsePlayerLeave(p); err == nil {
			c.push(m)
		}
	case d2gs.S2CGameFlags:
		if m, err := d2gs.ParseGameFlags(p); err == nil {
			c.push(m)
		}
	case d2gs.S2CLoadAct:
		if m, err := d2gs.ParseLoadAct(p); err == nil {
			c.push(m)
		}
	}
}

func (c *Client) write(pkts ...[]byte) error {
	b, err := d2gs.EncodeStream(pkts...)
	if err != nil {
		return err
	}

	c.wmu.Lock()
	defer c.wmu.Unlock()

	_, err = c.conn.Write(b)

	return err
}

// Send sends one realm message.
func (c *Client) Send(msg interface{}) error {
	typ, body := Encode(msg)

	return c.write(d2gs.Tunnel(d2gs.CtlTunnel, typ, body)...)
}

// Say sends a chat line; to is a whisper recipient or empty for everybody in
// the lobby or game.
func (c *Client) Say(text, to string) error { return c.write(ClientChat(text, to)) }

// Leave leaves the current game (0x69).
func (c *Client) Leave() error { return c.write(d2gs.LeaveGame()) }

// Wait removes and returns the first queued event for which match is true,
// waiting up to timeout for it. Events that do not match stay queued, in order.
func (c *Client) Wait(timeout time.Duration, match func(interface{}) bool) (interface{}, error) {
	deadline := time.Now().Add(timeout)
	timer := time.AfterFunc(timeout, c.cond.Broadcast)

	defer timer.Stop()

	c.mu.Lock()
	defer c.mu.Unlock()

	for {
		for i, ev := range c.events {
			if match(ev) {
				c.events = append(c.events[:i], c.events[i+1:]...)

				return ev, nil
			}
		}

		if c.err != nil {
			return nil, fmt.Errorf("%w: %v", ErrClosed, c.err)
		}

		if !time.Now().Before(deadline) {
			return nil, ErrTimeout
		}

		c.cond.Wait()
	}
}

// Pending returns a copy of the queued events (for assertions).
func (c *Client) Pending() []interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	return append([]interface{}(nil), c.events...)
}

const defaultWait = 5 * time.Second

// Hello logs in and returns the lobby roster.
func (c *Client) Hello(account string) (HelloAck, error) {
	if err := c.Send(Hello{Version: ProtocolVersion, Account: account}); err != nil {
		return HelloAck{}, err
	}

	ev, err := c.Wait(defaultWait, func(e interface{}) bool { _, ok := e.(HelloAck); return ok })
	if err != nil {
		return HelloAck{}, err
	}

	ack := ev.(HelloAck)
	if ack.Code != CodeOK {
		return ack, fmt.Errorf("hello: %s: %s", ack.Code, ack.Message)
	}

	return ack, nil
}

// request sends msg and waits for the Result of op, returning an error when
// its code is not OK.
func (c *Client) request(op byte, msg interface{}) error {
	if err := c.Send(msg); err != nil {
		return err
	}

	ev, err := c.Wait(defaultWait, func(e interface{}) bool { r, ok := e.(Result); return ok && r.Op == op })
	if err != nil {
		return err
	}

	if r := ev.(Result); r.Code != CodeOK {
		return &RequestError{Code: r.Code, Message: r.Message}
	}

	return nil
}

// RequestError is a refused request.
type RequestError struct {
	Code    Code
	Message string
}

func (e *RequestError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

// Upload validates and stores a character on the server and makes it active.
func (c *Client) Upload(d2s []byte) error { return c.request(MsgUploadChar, UploadChar{Data: d2s}) }

// Select activates a stored character and returns its file.
func (c *Client) Select(name string) ([]byte, error) {
	if err := c.request(MsgSelectChar, SelectChar{Name: name}); err != nil {
		return nil, err
	}

	ev, err := c.Wait(defaultWait, func(e interface{}) bool { _, ok := e.(CharData); return ok })
	if err != nil {
		return nil, err
	}

	return ev.(CharData).Data, nil
}

// Games fetches the game list.
func (c *Client) Games() ([]GameInfo, error) {
	if err := c.Send(ListGamesReq{}); err != nil {
		return nil, err
	}

	ev, err := c.Wait(defaultWait, func(e interface{}) bool { _, ok := e.(GameList); return ok })
	if err != nil {
		return nil, err
	}

	return ev.(GameList).Games, nil
}

// Chars lists the stored characters.
func (c *Client) Chars() ([]CharInfo, error) {
	if err := c.Send(ListCharsReq{}); err != nil {
		return nil, err
	}

	ev, err := c.Wait(defaultWait, func(e interface{}) bool { _, ok := e.(CharList); return ok })
	if err != nil {
		return nil, err
	}

	return ev.(CharList).Chars, nil
}

func (c *Client) enter(op byte, msg interface{}) (GameJoined, error) {
	if err := c.Send(msg); err != nil {
		return GameJoined{}, err
	}

	ev, err := c.Wait(defaultWait, func(e interface{}) bool {
		switch m := e.(type) {
		case GameJoined:
			return true
		case Result:
			return m.Op == op
		}

		return false
	})
	if err != nil {
		return GameJoined{}, err
	}

	if r, ok := ev.(Result); ok {
		return GameJoined{}, &RequestError{Code: r.Code, Message: r.Message}
	}

	return ev.(GameJoined), nil
}

// Create creates a game and joins it.
func (c *Client) Create(g CreateGame) (GameJoined, error) { return c.enter(MsgCreateGame, g) }

// Join joins a game.
func (c *Client) Join(name, password string) (GameJoined, error) {
	return c.enter(MsgJoinGame, JoinGame{Name: name, Password: password})
}
