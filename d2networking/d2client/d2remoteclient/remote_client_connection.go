package d2remoteclient

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2client/d2clientconnectiontype"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gs"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gsnet"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

const logPrefix = "Remote Client"

// RemoteClientConnection is the implementation of ClientConnection
// for a remote client.
type RemoteClientConnection struct {
	asset          *d2asset.AssetManager
	heroState      *d2hero.HeroStateFactory
	clientListener d2networking.ClientListener // The GameClient
	uniqueID       string                      // Unique ID generated on construction
	tcpConnection  *net.TCPConn                // UDP connection to the server
	active         bool                        // The connection is currently open

	// OD2_PROTO=d2gs: the game protocol instead of JSON (see d2gsnet)
	gs      *d2gsnet.ClientSide
	writeMu sync.Mutex

	*d2util.Logger
}

// Create constructs a new RemoteClientConnection
// and returns a pointer to it.
func Create(l d2util.LogLevel, asset *d2asset.AssetManager) (*RemoteClientConnection, error) {
	heroStateFactory, err := d2hero.NewHeroStateFactory(asset)
	if err != nil {
		return nil, err
	}

	result := &RemoteClientConnection{
		asset:     asset,
		heroState: heroStateFactory,
		uniqueID:  uuid.New().String(),
	}

	result.Logger = d2util.NewLogger()
	result.Logger.SetPrefix(logPrefix)
	result.Logger.SetLevel(l)

	return result, nil
}

// Open runs serverListener() in a goroutine to continuously read UDP packets.
// It also sends a PlayerConnectionRequestPacket packet to the server (see d2netpacket).
func (r *RemoteClientConnection) Open(connectionString, saveFilePath string) error {
	if !strings.Contains(connectionString, ":") {
		connectionString += ":" + defaultServerPort()
	}

	tcpAddress, err := net.ResolveTCPAddr("tcp", connectionString)

	if err != nil {
		return err
	}

	r.tcpConnection, err = dialWithRetry(tcpAddress)
	if err != nil {
		return err
	}

	r.active = true

	if d2gsnet.Enabled() {
		r.gs = &d2gsnet.ClientSide{IDs: d2gsnet.NewIDs(), OnInfo: r.onGameInfo}
		go r.serverListenerD2GS()
	} else {
		go r.serverListener()
	}

	r.Infof("Connected to server at %s", r.tcpConnection.RemoteAddr().String())

	gameState := r.heroState.LoadHeroState(saveFilePath)

	packet, err := d2netpacket.CreatePlayerConnectionRequestPacket(r.GetUniqueID(), gameState)
	if err != nil {
		r.Errorf("PlayerConnectionRequestPacket: %v", err)
	}

	err = r.SendPacketToServer(packet)

	if err != nil {
		r.Errorf("RemoteClientConnection: error sending PlayerConnectionRequestPacket to server.")
		return err
	}

	return nil
}

// Close informs the server that this client has disconnected and sets
// RemoteClientConnection.active to false.
func (r *RemoteClientConnection) Close() error {
	r.active = false

	pd, err := d2netpacket.CreatePlayerDisconnectRequestPacket(r.GetUniqueID())
	if err != nil {
		return fmt.Errorf("PlayerDisconnectRequestPacket: %v", err)
	}

	err = r.SendPacketToServer(pd)

	if err != nil {
		return err
	}

	if r.gs != nil { // the leave is on the wire; now close the socket
		return r.tcpConnection.Close()
	}

	return nil
}

// GetUniqueID returns RemoteClientConnection.uniqueID.
func (r *RemoteClientConnection) GetUniqueID() string {
	return r.uniqueID
}

// GetConnectionType returns an enum representing the connection type.
// See: d2clientconnectiontype
func (r *RemoteClientConnection) GetConnectionType() d2clientconnectiontype.ClientConnectionType {
	return d2clientconnectiontype.LANClient
}

// SetClientListener sets RemoteClientConnection.clientListener to the given value.
func (r *RemoteClientConnection) SetClientListener(listener d2networking.ClientListener) {
	r.clientListener = listener
}

// SendPacketToServer compresses the JSON encoding of a NetPacket and
// sends it to the server.
func (r *RemoteClientConnection) SendPacketToServer(packet d2netpacket.NetPacket) error {
	if r.gs != nil {
		return r.sendD2GS(packet)
	}

	encoder := json.NewEncoder(r.tcpConnection)

	err := encoder.Encode(packet)
	if err != nil {
		return err
	}

	return nil
}

// serverListener runs a while loop, reading from the GameServer's TCP
// connection.
func (r *RemoteClientConnection) serverListener() {
	decoder := json.NewDecoder(r.tcpConnection)

	for {
		var packet d2netpacket.NetPacket

		err := decoder.Decode(&packet)
		if err != nil {
			switch err {
			case io.EOF:
				break // the other side closed the connection
			default:
				r.Errorf("failed to decode the packet, err: %v\n", err)
			}

			return // allow the connection to close
		}

		p, err := r.decodeToPacket(packet.PacketType, string(packet.PacketData))
		if err != nil {
			r.Errorf("%v %v", packet.PacketType, err)
		}

		err = r.clientListener.OnPacketReceived(p)
		if err != nil {
			r.Errorf("%v %v", packet.PacketType, err)
		}
	}
}

// bytesToJSON reads the packet type, decompresses the packet and returns a JSON string.
// nolint:unused // WIP
func (r *RemoteClientConnection) bytesToJSON(buffer []byte) (string, d2netpackettype.NetPacketType, error) {
	packet, err := d2netpacket.UnmarshalNetPacket(buffer)
	if err != nil {
		return "", 0, err
	}

	return string(packet.PacketData), packet.PacketType, nil
}

// decodeToPacket unmarshals the JSON string into the correct struct
// and returns a NetPacket declaring that struct.
// nolint:gocyclo,funlen // switch statement on packet type makes sense, no need to change
func (r *RemoteClientConnection) decodeToPacket(
	t d2netpackettype.NetPacketType,
	data string) (d2netpacket.NetPacket, error) {
	var (
		np  = d2netpacket.NetPacket{}
		err error
		p   interface{}
	)

	switch t {
	case d2netpackettype.GenerateMap:
		p, err = d2netpacket.UnmarshalGenerateMap([]byte(data))
	case d2netpackettype.MovePlayer:
		p, err = d2netpacket.UnmarshalMovePlayer([]byte(data))
	case d2netpackettype.UpdateServerInfo:
		p, err = d2netpacket.UnmarshalUpdateServerInfo([]byte(data))
	case d2netpackettype.AddPlayer:
		p, err = d2netpacket.UnmarshalAddPlayer([]byte(data))
	case d2netpackettype.CastSkill:
		p, err = d2netpacket.UnmarshalCast([]byte(data))
	case d2netpackettype.Ping:
		p, err = d2netpacket.UnmarshalPing([]byte(data))
	case d2netpackettype.PlayerDisconnectionNotification:
		p, err = d2netpacket.UnmarshalPlayerDisconnectionRequest([]byte(data))
	case d2netpackettype.ServerClosed:
		p, err = d2netpacket.UnmarshalServerClosed([]byte(data))
	default:
		err = fmt.Errorf("RemoteClientConnection: unrecognized packet type: %v", t)
	}

	if err != nil {
		return np, err
	}

	mp, marshalErr := d2netpacket.MarshalPacket(p)
	if marshalErr != nil {
		r.Errorf("MarshalPacket: %v", marshalErr)
	}

	np = d2netpacket.NetPacket{PacketType: t, PacketData: mp}

	return np, nil
}

// defaultServerPort matches the server's listen port; OD2_PORT overrides it.
func defaultServerPort() string {
	if p := os.Getenv("OD2_PORT"); p != "" {
		return p
	}

	return "6669"
}

// dialWithRetry connects to the host. OD2_JOIN_RETRY=<seconds> keeps trying
// while the host is still starting (used by the two-process autotest).
func dialWithRetry(addr *net.TCPAddr) (*net.TCPConn, error) {
	deadline := time.Now()

	if v, err := strconv.Atoi(os.Getenv("OD2_JOIN_RETRY")); err == nil && v > 0 {
		deadline = deadline.Add(time.Duration(v) * time.Second)
	}

	for {
		c, err := net.DialTCP("tcp", nil, addr)
		if err == nil || time.Now().After(deadline) {
			return c, err
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// GameInfoListener is implemented by the game client: the host's map seed and
// difficulty arrive with the act load of the game protocol.
type GameInfoListener interface {
	SetGameInfo(mapSeed uint32, difficulty uint8)
}

func (r *RemoteClientConnection) onGameInfo(info d2gsnet.GameInfo) {
	if l, ok := r.clientListener.(GameInfoListener); ok {
		l.SetGameInfo(info.MapSeed, info.Difficulty)
	}
}

// sendD2GS translates a packet to the game protocol and writes it as blobs.
func (r *RemoteClientConnection) sendD2GS(packet d2netpacket.NetPacket) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()

	pkts, err := r.gs.Encode(packet)
	if err != nil {
		return err
	}

	blob, err := d2gs.EncodeStream(pkts...)
	if err != nil {
		return err
	}

	_, err = r.tcpConnection.Write(blob)

	return err
}

// serverListenerD2GS reads blobs from the server until it closes.
func (r *RemoteClientConnection) serverListenerD2GS() {
	reader := bufio.NewReader(r.tcpConnection)

	for {
		plain, err := d2gs.ReadBlob(reader)
		if err != nil {
			if !d2gsnet.IsClosed(err) && r.active {
				r.Errorf("d2gs read: %v", err)
			}

			return
		}

		packets, _, err := r.gs.Decode(plain)
		if err != nil {
			r.Errorf("d2gs decode: %v", err)
			return
		}

		for _, p := range packets {
			if err := r.clientListener.OnPacketReceived(p); err != nil {
				r.Errorf("%v %v", p.PacketType, err)
			}
		}
	}
}
