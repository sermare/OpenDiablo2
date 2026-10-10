package d2gamescreen

import (
	"errors"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
)

const subtilesPerTileRealm = 5

// realmMonsterKeys maps the simulation's placeholder monster types
// (d2mp.DefaultRules, in order) to monstats keys. UNVERIFIED stand-ins: an
// engine host with real rules would send the monstats id instead.
var realmMonsterKeys = []string{"fallen1", "fallen1", "zombie1", "skeleton1", "goatman1", "quillrat1", "brute1", "fallen1"}

// realmState is the game screen's side of a game played through the realm:
// the monsters of the authoritative simulation, drawn as mirror monsters.
type realmState struct {
	pending []d2netpacket.RealmUnitPacket
	mirrors map[uint32]*d2mapentity.Monster
	unitOf  map[string]uint32 // monster entity id -> simulation unit
	dead    map[uint32]string // simulation unit -> killer's name
	lastHit uint32
}

// hookRealm connects the screen to the realm connection, if the game has one.
func (v *Game) hookRealm() {
	if !v.gameClient.IsRealm() {
		return
	}

	v.realm = &realmState{mirrors: map[uint32]*d2mapentity.Monster{}, unitOf: map[string]uint32{}, dead: map[uint32]string{}}
	v.gameClient.OnRealmUnit = func(p d2netpacket.RealmUnitPacket) { v.realm.pending = append(v.realm.pending, p) }
}

// advanceRealm creates, hurts and kills the mirror monsters as the simulation
// reports, and keeps them where it has them.
func (v *Game) advanceRealm(_ float64) {
	r := v.realm
	if r == nil {
		return
	}

	d := v.monsterDirector()
	if d == nil {
		return
	}

	pending := r.pending
	r.pending = nil

	for _, p := range pending {
		m := r.mirrors[p.UnitID]

		switch p.Op {
		case d2netpacket.RealmUnitSpawn:
			if m == nil {
				v.realmSpawn(p)
			}
		case d2netpacket.RealmUnitHit:
			if m != nil {
				d.MirrorHP(m, int(p.HP), int(p.MaxHP))
				v.Infof("REALM HIT unit=%d hp=%d/%d", p.UnitID, p.HP, p.MaxHP)
			}
		case d2netpacket.RealmUnitDeath:
			if m != nil && d.MirrorKill(m) {
				by := v.playerName(p.Killer)
				r.dead[p.UnitID] = by
				v.Infof("REALM KILL unit=%d name=%q by=%q", p.UnitID, m.Label(), by)
			}
		}
	}

	for unit, m := range r.mirrors {
		if !m.Alive() {
			continue
		}

		x, y, ok := v.gameClient.RealmUnitPos(unit)
		if !ok {
			continue
		}

		sx, sy := int(x*subtilesPerTileRealm), int(y*subtilesPerTileRealm)
		if cx, cy := m.SubtilePos(); cx != sx || cy != sy {
			d.MirrorMove(m, sx, sy)
		}
	}
}

func (v *Game) playerName(id string) string {
	if p := v.gameClient.Players[id]; p != nil {
		return p.Name()
	}

	return ""
}

func (v *Game) realmSpawn(p d2netpacket.RealmUnitPacket) {
	d := v.monsters

	key := realmMonsterKeys[0]
	if int(p.Type) < len(realmMonsterKeys) {
		key = realmMonsterKeys[p.Type]
	}

	stat := d.FindStat(key)
	if stat == nil {
		stat = d.FindStat(realmMonsterKeys[0])
	}

	if stat == nil {
		v.Errorf("REALM: no monster record for type %d (%s)", p.Type, key)

		return
	}

	m, err := d.SpawnMirror(stat, int(p.X*subtilesPerTileRealm), int(p.Y*subtilesPerTileRealm))
	if err != nil {
		v.Errorf("REALM: spawning monster %d: %v", p.UnitID, err)

		return
	}

	v.realm.mirrors[p.UnitID] = m
	v.realm.unitOf[m.ID()] = p.UnitID
	v.Infof("REALM MONSTER unit=%d name=%q tile=(%.1f,%.1f)", p.UnitID, p.Name, p.X, p.Y)
}

// onMirrorHit is the director's hook: a blow of the local hero hit a mirror
// monster; the realm decides what it does.
func (v *Game) onMirrorHit(m *d2mapentity.Monster, src *d2mapentity.Player) {
	if v.realm == nil || src != v.localPlayer {
		return
	}

	unit, ok := v.realm.unitOf[m.ID()]
	if !ok {
		return
	}

	if unit != v.realm.lastHit {
		v.realm.lastHit = unit
		v.Infof("REALM ATTACK unit=%d", unit)
	}

	if err := v.gameClient.RealmAttack(unit); err != nil {
		v.Errorf("REALM: attack: %v", err)
	}
}

// nearestMirror is the nearest living mirror monster to the hero.
func (v *Game) nearestMirror() (*d2mapentity.Monster, uint32) {
	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())

	var (
		best     *d2mapentity.Monster
		bestUnit uint32
		bestDist int
	)

	units := make([]uint32, 0, len(v.realm.mirrors))
	for u := range v.realm.mirrors {
		units = append(units, u)
	}

	sort.Slice(units, func(i, j int) bool { return units[i] < units[j] })

	for _, u := range units {
		m := v.realm.mirrors[u]
		if !m.Alive() {
			continue
		}

		mx, my := m.SubtilePos()
		if d := d2monster.Distance(hx-mx, hy-my); best == nil || d < bestDist {
			best, bestUnit, bestDist = m, u, d
		}
	}

	return best, bestUnit
}

// commandMPKill is "mpkill": the hero walks up to the nearest monster of the
// realm and fights it (the realm resolves the fight; scenarios use it).
func (v *Game) commandMPKill(_ []string) error {
	if v.realm == nil || v.localPlayer == nil || v.monsters == nil {
		return errors.New("not in a realm game")
	}

	m, unit := v.nearestMirror()
	if m == nil {
		return errors.New("no living monster of the realm")
	}

	v.Infof("REALM KILLCMD unit=%d name=%q", unit, m.Label())
	v.OnPlayerAttack(m)

	return nil
}

// commandMPWorld is "mpworld": one line with the state of the simulation as
// this client sees it and the monsters as the engine draws them. Two clients
// of one game print the same digest once their events have settled.
func (v *Game) commandMPWorld(_ []string) error {
	if v.realm == nil {
		return errors.New("not in a realm game")
	}

	alive, dead := 0, 0

	for _, m := range v.realm.mirrors {
		if m.Alive() {
			alive++
		} else {
			dead++
		}
	}

	v.Infof("REALM WORLD %s engine_monsters_alive=%d engine_monsters_dead=%d", v.gameClient.RealmSummary(), alive, dead)

	return nil
}
