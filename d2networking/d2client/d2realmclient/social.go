package d2realmclient

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2party"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2mp"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2realm"
)

// UnitOfPeer is the inverse of PeerID: the realm unit id behind an engine
// player id of a remote hero.
func UnitOfPeer(id string) (uint32, bool) {
	if !strings.HasPrefix(id, "u") {
		return 0, false
	}

	n, err := strconv.ParseUint(id[1:], 10, 32)

	return uint32(n), err == nil
}

// RealmCommand translates an engine party command into the realm's command.
// ok=false means the realm has no such command (decline, hostility: the
// simulation models only invite, accept and leave); the reason says why.
func RealmCommand(op, target string) (cmd d2mp.Command, ok bool, reason string) {
	switch op {
	case d2netpacket.PartyInvite:
		unit, isPeer := UnitOfPeer(target)
		if !isPeer {
			return cmd, false, fmt.Sprintf("%q is not a player of this game", target)
		}

		return d2mp.Command{Type: d2mp.CmdPartyInvite, Target: unit}, true, ""
	case d2netpacket.PartyAccept:
		return d2mp.Command{Type: d2mp.CmdPartyAccept}, true, ""
	case d2netpacket.PartyLeave:
		return d2mp.Command{Type: d2mp.CmdPartyLeave}, true, ""
	case d2netpacket.PartyDecline:
		return cmd, false, "the realm keeps an invitation until it is replaced; there is nothing to decline"
	}

	return cmd, false, fmt.Sprintf("the realm does not model %q", op)
}

// LevelMessage is the realm's LevelChange for an engine level id. The realm
// numbers acts from 0.
func LevelMessage(levelID int) (d2realm.LevelChange, error) {
	if levelID < 1 {
		return d2realm.LevelChange{}, fmt.Errorf("level %d is not a level", levelID)
	}

	act := d2level.ActOfLevel(levelID) - 1
	if act < 0 {
		return d2realm.LevelChange{}, fmt.Errorf("level %d belongs to no act", levelID)
	}

	return d2realm.LevelChange{Act: byte(act), Level: uint16(levelID)}, nil
}

// ChangeLevel reports to the realm that the local hero entered a level
// (engine ChangeLevel packet), so the other members get a PlayerLevel.
func (b *Bridge) ChangeLevel(levelID int) error {
	msg, err := LevelMessage(levelID)
	if err != nil {
		return err
	}

	b.mu.Lock()
	b.area = msg.Level
	b.mu.Unlock()

	return b.client.Send(msg)
}

// PeerLevel returns the level a remote hero (engine id) is known to be in.
func (b *Bridge) PeerLevel(id string) (level int, ok bool) {
	unit, isPeer := UnitOfPeer(id)
	if !isPeer {
		return 0, false
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	l, ok := b.levels[unit]

	return int(l), ok
}

// SameLevel reports whether a remote hero is in the level of the local hero.
// A hero whose level is unknown counts as present (the realm sends a level
// only after the first change).
func (b *Bridge) SameLevel(id string) bool {
	unit, isPeer := UnitOfPeer(id)
	if !isPeer {
		return false
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	l, ok := b.levels[unit]
	if !ok || b.area == 0 {
		return true
	}

	return l == b.area
}

// playerLevel records where a hero is (b.mu held).
func (b *Bridge) playerLevel(m d2realm.PlayerLevel) {
	if b.levels == nil {
		b.levels = map[uint32]uint16{}
	}

	b.levels[m.UnitID] = m.Level
	b.cfg.Logf("realm: unit %d is in level %d (act %d)", m.UnitID, m.Level, m.Act+1)
	b.emitRoster("")
}

// Party sends an engine party command to the realm. It returns a reason when
// the realm cannot do it (the caller logs it for the player).
func (b *Bridge) Party(op, target string) (string, error) {
	cmd, ok, reason := RealmCommand(op, target)
	if !ok {
		return reason, nil
	}

	return "", b.client.GameCommand(cmd)
}

// RosterSnapshot is the roster the party panel shows, built from the heroes
// the bridge knows (b.mu held): the local hero (self) and every announced
// remote hero, with their party ids from the simulation and their levels
// (area) as far as known.
func (b *Bridge) rosterSnapshot() d2party.Snapshot {
	var snap d2party.Snapshot

	if b.rep == nil {
		return snap
	}

	area := func(unit uint32, lvl uint16) int {
		if unit == b.joined.UnitID {
			return int(b.area)
		}

		if lvl != 0 {
			return int(lvl)
		}

		return int(b.levels[unit])
	}

	if u := b.rep.Unit(b.joined.UnitID); u != nil {
		snap.Players = append(snap.Players, d2party.Info{ID: b.self, Name: u.Name, Class: heroOfClass[d2s.Class(u.Type)],
			Level: 1, Area: area(u.ID, 0), Party: int(u.Party)})
	}

	// every hero of the game, in whatever level: the roster is global
	for _, h := range b.rep.Heroes() {
		if h.ID == b.joined.UnitID {
			continue
		}

		snap.Players = append(snap.Players, d2party.Info{ID: PeerID(h.ID), Name: h.Name, Class: heroOfClass[d2s.Class(h.Type)],
			Level: 1, Area: area(h.ID, h.Level), Party: int(h.Party)})
	}

	return snap
}

// emitRoster hands the engine a RosterUpdate (b.mu held).
func (b *Bridge) emitRoster(notice string) {
	snap := b.rosterSnapshot()
	if len(snap.Players) == 0 {
		return
	}

	b.emit(d2netpacket.CreateRosterUpdatePacket(d2netpacket.RosterUpdatePacket{Roster: snap, Notice: notice, Player: b.self}))
}

func sortUnits(ids []uint32) {
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && ids[j] < ids[j-1]; j-- {
			ids[j], ids[j-1] = ids[j-1], ids[j]
		}
	}
}

// Explain turns a refused create or join into a sentence for the player.
func Explain(err error) string {
	if err == nil {
		return ""
	}

	var re *d2realm.RequestError
	if !errors.As(err, &re) {
		return fmt.Sprintf("could not reach the game: %v", err)
	}

	switch re.Code {
	case d2realm.CodeGameNotFound:
		return "no game named \"" + DefaultGame + "\" is running on that host (is the host in the game yet?)"
	case d2realm.CodeGameFull:
		return "the game is full"
	case d2realm.CodeBadPassword:
		return "the game is password protected"
	case d2realm.CodeDifficultyLocked:
		return "your hero has not unlocked this difficulty"
	case d2realm.CodeLevelTooLow:
		return "your hero's level is too low for this game"
	case d2realm.CodeLevelTooHigh:
		return "your hero's level is too high for this game"
	case d2realm.CodeModeMismatch:
		return "hardcore and expansion heroes can only join games of the same kind"
	case d2realm.CodeAlreadyInGame:
		return "this character is already in a game"
	case d2realm.CodeNameTaken:
		return "a hero with this name is already connected to that host"
	case d2realm.CodeCharInvalid:
		return "the host refused the hero's save (" + re.Message + ")"
	case d2realm.CodeServerFull:
		return "the host is full"
	}

	return re.Error()
}

// retryJoin calls try until it succeeds, fails with a refusal that waiting
// cannot fix (anything but "game not found": the host may still be creating
// the game), or the deadline passes. wait is the pause between attempts.
func retryJoin(try func() (d2realm.GameJoined, error), deadline time.Time, wait func(), now func() time.Time) (d2realm.GameJoined, error) {
	for {
		j, err := try()
		if err == nil {
			return j, nil
		}

		var re *d2realm.RequestError
		if errors.As(err, &re) && re.Code != d2realm.CodeGameNotFound {
			return j, err
		}

		if now().After(deadline) {
			return j, err
		}

		wait()
	}
}
