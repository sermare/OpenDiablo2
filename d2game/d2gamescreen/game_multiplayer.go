package d2gamescreen

import (
	"errors"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
)

// commandPlayers logs the players this client knows with their tile
// positions (the multiplayer autotest reads the line).
func (v *Game) commandPlayers(_ []string) error {
	v.Infof("%s", v.gameClient.PlayersSummary())

	return nil
}

// commandChat sends a chat line to every player ("_" stands for a space, the
// terminal splits its arguments on spaces).
func (v *Game) commandChat(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: chat <text_with_underscores>")
	}

	text := strings.ReplaceAll(args[0], "_", " ")

	pkt, err := d2netpacket.CreateChatPacket(v.gameClient.PlayerID, "", text)
	if err != nil {
		return err
	}

	return v.gameClient.SendPacketToServer(pkt)
}

// leaveNetworkGame tells the server this hero leaves (the other players see it
// go); single-player games have nothing to tell.
func (v *Game) leaveNetworkGame() {
	if v.gameClient == nil || v.gameClient.IsSinglePlayer() {
		return
	}

	if err := v.gameClient.Close(); err != nil {
		v.Errorf("leaving the game: %v", err)
	}
}

// announceCast tells the other players about a cast the skill pipeline ran
// locally (with real missiles). The server relays it to everybody; this client
// skips the echo of its own cast, because it already played it.
func (v *Game) announceCast(skillID int, targetX, targetY float64) {
	if v.gameClient.IsSinglePlayer() {
		return
	}

	cp, err := d2netpacket.CreateCastPacket(v.gameClient.PlayerID, skillID, targetX, targetY)
	if err != nil {
		v.Errorf("CastPacket: %v", err)
		return
	}

	v.gameClient.SkipOwnCastEcho()

	if err := v.gameClient.SendPacketToServer(cp); err != nil {
		v.Errorf(castErrStr, v.gameClient.PlayerID, skillID, targetX, targetY)
	}
}
