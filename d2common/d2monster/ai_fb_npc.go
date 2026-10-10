package d2monster

// Faithful ports of the town and quest NPC thinkers: Navi 0x5e6d10, Npc
// 0x5e6070, NpcBarb 0x5ecd70, NpcOutOfTown 0x5e67a0, NpcStationary 0x5e62f0,
// Towner 0x5e6460, TownRogue 0x5e6c50, Vendor 0x5e8eb0, Wussie 0x5ed4d0.
// The decision flow is the exe's; everything that belongs to the quest engine
// or the NPC interaction server is an optional host interface (FBN prefix)
// whose fallback is "quest condition false / nobody interacting", so a host
// without them gets plain town NPCs.

// Monster class ids the exe tests inside the NPC thinkers (VERIFIED values).
const (
	fbnClassCainStart = 0x109 // Act 1 Search for Cain walker
	fbnClassAct5Door  = 0x200 // Act 5 Betrayal of Harrogath NPC
	fbnClassAct3Quest = 0xff  // Act 3 quest walker
	fbnClassArcane    = 0xc9  // Act 2 Arcane Sanctuary NPC
	fbnClassAct2Flag  = 0xfe  // Act 2 quest flag NPC
	fbnClassTyrael2   = 0xfb  // NpcStationary Act 2 (Seven Tombs)
	fbnClassIzual     = 0x196 // NpcStationary Act 4 (Izual)
	fbnClassPrison    = 0x20f // Act 5 Prison of Ice NPC (NpcOutOfTown)
)

// FBNNpcHost is the NPC interaction server and player scan an NPC reads.
type FBNNpcHost interface {
	// HasInteractEntry is NPCSRV_HasAnyInteractEntry: some player is in the
	// NPC's interaction list (talking/trading).
	HasInteractEntry(b *Brain) bool
	// InteractListHasPlayer is NPCSRV_InteractListContainsPlayer.
	InteractListHasPlayer(b *Brain) bool
	// InteractBusy is PLAYER_IsInteractionBusy for the given player.
	InteractBusy(b *Brain, p Target) bool
	// NearestPlayerNearby is MONAI_FindNearestPlayerNearby: the nearest player
	// in the room set; near is its "closer than 4" flag. ok is false when
	// there is none (the exe then returns the NPC itself).
	NearestPlayerNearby(b *Brain) (p Target, near bool, ok bool)
	// FacePlayer is UNIT_SetOffsetAndRegisterInRoom(player): the NPC turns
	// the player's way / registers it (UNVERIFIED exact effect).
	FacePlayer(b *Brain, p Target)
	// PlayGestures is MONAI_Npc_PlayQueuedGestures (the AI list node of
	// queued walks and animations); true when it acted.
	PlayGestures(b *Brain) bool
	// RandomIdle is MONAI_Npc_RandomIdleAction (66 percent of the time one
	// entry of the class's idle table); true when it acted.
	RandomIdle(b *Brain) bool
}

// FBNQuestHost holds the quest callbacks of Npc/NpcOutOfTown/NpcStationary.
// All are keyed on the NPC's class id (the fbnClass constants).
type FBNQuestHost interface {
	// QuestWalkTarget is the quest's "go there" point for the NPC, ok false
	// when the quest does not move it.
	QuestWalkTarget(b *Brain, class int) (Point, bool)
	// QuestArrived runs the quest's arrival action (set done flag, clear
	// flag 6 and animate, notify).
	QuestArrived(b *Brain, class int)
	// QuestFlag reports the class's quest flag (Act 2 flag 0 for 0xfe, the
	// proximity checks for 0xfb/0x196, passive for 0xc9).
	QuestFlag(b *Brain, class int) bool
	// QuestClear clears that flag.
	QuestClear(b *Brain, class int)
	// QuestThink is the unconditional per-think refresh (quest log refresh,
	// Place NPC in Act 5 ...).
	QuestThink(b *Brain, class int)
}

func init() {
	fbnReg("Vendor", thinkVendorB)
	fbnReg("TownRogue", thinkTownRogueB)
	fbnReg("Towner", thinkTownerB)
	fbnReg("Navi", thinkNaviB)
	fbnReg("Npc", thinkNpcB)
	fbnReg("NpcStationary", thinkNpcStationaryB)
	fbnReg("NpcOutOfTown", thinkNpcOutOfTownB)
	fbnReg("NpcBarb", thinkNpcBarbB)
	fbnReg("Wussie", thinkWussieB)
}

func fbnReg(name string, t Think) {
	m, ok := AITargetMode(name)
	if !ok {
		m = TargetStandard
	}

	register(name, m, t)
}

func fbnHost(c *Ctx) (FBNNpcHost, bool) {
	h, ok := c.W.(FBNNpcHost)

	return h, ok
}

func fbnQuest(c *Ctx) FBNQuestHost {
	q, _ := c.W.(FBNQuestHost)

	return q
}

// fbnTiles is UNIT_IsWithinTileDistance(x, y): the distance in tiles
// (subtiles / 5, UNVERIFIED rounding).
func fbnTiles(b *Brain, x, y int) int { return Distance(b.X-x, b.Y-y) / 5 }

// fbnWalkPoint is MONAI_WalkToPoint: a walk to a map point.
func fbnWalkPoint(c *Ctx, p Point) bool { return c.move(p, nil, 0, false) }

// fbnStartAnchorWalk is MONAI_Npc_StartAnchorWalk 0x5e6070-family (VERIFIED):
// the first time, record the home position in the anchor command (type 10)
// and wait 20 frames.
func fbnStartAnchorWalk(c *Ctx) bool {
	b := c.B

	n := b.FindCommand(CmdAnchor)
	if n == nil {
		b.AppendCommand(Command{Type: CmdAnchor})
		n = b.FindCommand(CmdAnchor)
	}

	if n.X == 0 && n.Y == 0 {
		n.X, n.Y = b.X, b.Y
		c.Sleep(0x14)

		return true
	}

	return false
}

// fbnReturnToAnchor is MONAI_Npc_ReturnToAnchor (VERIFIED flow): farther
// than maxTiles from home, queue a walk command (type 4, count 12, delay 10)
// and wait 10.
func fbnReturnToAnchor(c *Ctx, maxTiles int) bool {
	b := c.B

	n := b.FindCommand(CmdAnchor)
	if n == nil {
		return fbnStartAnchorWalk(c)
	}

	if fbnTiles(b, n.X, n.Y) > maxTiles {
		if b.FindCommand(4) == nil {
			b.AppendCommand(Command{Type: 4, X: n.X, Y: n.Y, Count: 12, Delay: 10})
		}

		c.Sleep(10)

		return true
	}

	return false
}

// fbnReact is MONAI_Npc_ReactToNearestPlayer (VERIFIED flow). Scratch[0] is
// the chase counter, Scratch[1] the greeting timer (and the walk target x),
// Scratch[2] the walk target y of the busy branch.
func fbnReact(c *Ctx) bool {
	h, ok := fbnHost(c)
	if !ok {
		return false
	}

	b := c.B
	p, _, found := h.NearestPlayerNearby(b)

	dist := 0
	if found {
		dist = EdgeDistance(b.X-p.X, b.Y-p.Y, b.Size)
	}

	busy := h.InteractListHasPlayer(b)
	if found && p.IsPlayer && h.InteractBusy(b, p) {
		busy = true
	}

	anyEntry := h.HasInteractEntry(b)

	if !anyEntry && !busy && b.Scratch[0] < 1 {
		if !found {
			return false
		}

		if uint(dist-3) > 0x14 {
			if b.Scratch[1] == 0 {
				b.Scratch[1] = 0x3c

				if p.IsPlayer {
					h.FacePlayer(b, p)
					c.Sleep(0x14)

					return true
				}
			} else {
				if b.Scratch[1] < 0 {
					b.Scratch[1] = 0
				}

				if b.Scratch[1] > 0 {
					b.Scratch[1]--
				}
			}

			c.Sleep(0x14)

			return true
		}

		if !fbnReturnToAnchor(c, 0x10) {
			c.SetSpeed(0)

			if dist > 4 {
				c.WalkToRange(p, 3, 2)
			} else {
				c.WalkToRange(p, dist-2, 2)
			}
		}

		return true
	}

	if fbnTiles(b, b.Scratch[1], b.Scratch[2]) < 3 || b.Scratch[0] < 0x25 {
		if b.Scratch[0] > 0 {
			c.Sleep(8)
		}
	} else {
		fbnWalkPoint(c, Point{b.Scratch[1], b.Scratch[2]})
	}

	if b.Scratch[0] < 0 {
		b.Scratch[0] = 0

		return true
	}

	b.Scratch[0]--

	return true
}

func fbnGesturesOrIdle(c *Ctx) bool {
	h, ok := fbnHost(c)
	if !ok {
		return false
	}

	if h.PlayGestures(c.B) || h.RandomIdle(c.B) {
		c.busy()

		return true
	}

	return false
}

// thinkVendorB is MONAI_Think_Vendor 0x5e8eb0 (VERIFIED): 20 percent to play
// skill mode 1 (the vendor's wave), else wait 30.
func thinkVendorB(c *Ctx) {
	if c.B.Roll(100) < 0x14 {
		c.Attack(ModeSkill1, fbnSelf(c.B))

		return
	}

	c.Sleep(0x1e)
}

func fbnSelf(b *Brain) Target { return Target{ID: b.ID, X: b.X, Y: b.Y, Size: b.Size} }

// thinkTownRogueB is MONAI_Think_TownRogue 0x5e6c50 (VERIFIED): the Rogue
// Scout shoots its A1 at an attack target within 24, else waits 50.
func thinkTownRogueB(c *Ctx) {
	if t, d, ok := c.W.AttackTarget(c.B); ok && d < 0x19 {
		c.Attack(ModeAttack1, t)

		return
	}

	c.Sleep(0x32)
}

// thinkTownerB is MONAI_Think_Towner 0x5e6460 (VERIFIED).
func thinkTownerB(c *Ctx) {
	if fbnStartAnchorWalk(c) || fbnGesturesOrIdle(c) {
		return
	}

	c.Sleep(0xc)
}

// thinkNaviB is MONAI_Think_Navi 0x5e6d10 (VERIFIED; the hidden roll bound
// is 3, read from the disassembly). Scratch[1] is the greeting timer.
func thinkNaviB(c *Ctx) {
	b := c.B

	h, hasHost := fbnHost(c)
	if hasHost && h.HasInteractEntry(b) {
		c.Sleep(10)

		return
	}

	if hasHost {
		if p, near, ok := h.NearestPlayerNearby(b); ok && p.IsPlayer && !h.InteractBusy(b, p) &&
			b.Roll(3) != 0 && near {
			if b.Scratch[1] == 0 {
				b.Scratch[1] = 0x3c
				h.FacePlayer(b, p)
			} else {
				if b.Scratch[1] < 0 {
					b.Scratch[1] = 0
				}

				if b.Scratch[1] > 0 {
					b.Scratch[1]--
				}
			}

			c.Sleep(0x14)

			return
		}
	}

	if t, d, ok := c.W.AttackTarget(b); ok && d < 0x19 {
		c.Attack(ModeAttack1, t)

		return
	}

	c.Sleep(0x32)
}

// thinkNpcB is MONAI_Think_Npc 0x5e6070 (VERIFIED flow; quest bodies are
// FBNQuestHost hooks).
func thinkNpcB(c *Ctx) {
	b := c.B

	if fbnStartAnchorWalk(c) {
		return
	}

	q := fbnQuest(c)
	cls := b.Class

	if q != nil {
		switch {
		case cls == fbnClassCainStart:
			if p, ok := q.QuestWalkTarget(b, cls); ok {
				if fbnTiles(b, p.X, p.Y) > 2 {
					fbnWalkPoint(c, p)

					return
				}

				q.QuestArrived(b, cls)
			}
		case cls == fbnClassAct5Door:
			q.QuestThink(b, cls)
		case cls == fbnClassAct3Quest:
			if p, ok := q.QuestWalkTarget(b, cls); ok {
				if fbnTiles(b, p.X, p.Y) > 3 {
					fbnWalkPoint(c, p)

					return
				}

				c.Attack(ModeSkill1, fbnSelf(b))
				q.QuestArrived(b, cls)

				return
			}
		case cls == fbnClassArcane:
			if !q.QuestFlag(b, cls) {
				c.Sleep(0x28)

				return
			}

			q.QuestThink(b, cls)
		case cls == fbnClassAct2Flag:
			if q.QuestFlag(b, cls) {
				c.Attack(ModeSkill1, fbnSelf(b))
				q.QuestClear(b, cls)

				return
			}
		}
	}

	if fbnReact(c) || fbnGesturesOrIdle(c) {
		return
	}

	c.Sleep(8)
}

// thinkNpcStationaryB is MONAI_Think_NpcStationary 0x5e62f0 (VERIFIED flow).
// Scratch[1] is the greeting timer.
func thinkNpcStationaryB(c *Ctx) {
	b := c.B
	h, hasHost := fbnHost(c)
	q := fbnQuest(c)

	var (
		p     Target
		found bool
	)

	if hasHost {
		p, _, found = h.NearestPlayerNearby(b)
	}

	if !found {
		if q != nil && (b.Class == fbnClassTyrael2 || b.Class == fbnClassIzual) && q.QuestFlag(b, b.Class) &&
			!(hasHost && h.HasInteractEntry(b)) {
			c.Attack(ModeDying, fbnSelf(b))
			q.QuestArrived(b, b.Class)

			return
		}

		c.Sleep(0x14)

		return
	}

	dist := EdgeDistance(b.X-p.X, b.Y-p.Y, b.Size)

	if (p.IsPlayer && h.InteractBusy(b, p)) || h.HasInteractEntry(b) {
		c.Sleep(10)

		return
	}

	if dist < 0x18 {
		if b.Scratch[1] == 0 {
			b.Scratch[1] = 0x3c

			if p.IsPlayer {
				h.FacePlayer(b, p)
			}
		} else {
			b.Scratch[1]--
		}
	} else {
		if b.Scratch[1] < 1 {
			b.Scratch[1] = 0x3c
		}

		b.Scratch[1]--
	}

	c.Sleep(0x14)
}

// thinkNpcOutOfTownB is MONAI_Think_NpcOutOfTown 0x5e67a0 (flow VERIFIED, the
// quest spawn bodies are hooks). Scratch is not used: the exe keeps its state
// in the walk command node (type 3): X/Y target, Count phase, Delay retry.
func thinkNpcOutOfTownB(c *Ctx) {
	b := c.B
	q := fbnQuest(c)
	prison := b.Class == fbnClassPrison

	startSlot3 := func() bool {
		n := b.FindCommand(3)
		if n == nil {
			b.AppendCommand(Command{Type: 3})
			n = b.FindCommand(3)
		}

		if n.X == 0 && n.Y == 0 {
			n.X, n.Y = b.X+3, b.Y+3
			n.Count, n.Delay = 1, 0
			c.Sleep(1)

			return true
		}

		return false
	}

	if startSlot3() {
		return
	}

	if prison && q != nil {
		q.QuestThink(b, b.Class)
	}

	if h, ok := fbnHost(c); ok && h.HasInteractEntry(b) {
		c.Sleep(0x28)

		return
	}

	n := b.FindCommand(3)
	if b.Mode == ModeDead {
		return
	}

	fbnReact(c)

	if n == nil {
		c.Sleep(0x28)

		return
	}

	if n.Count < 2 {
		if fbnTiles(b, n.X, n.Y) > 1 && n.Delay < 6 {
			n.Delay++
			fbnWalkPoint(c, Point{n.X, n.Y})

			return
		}

		if n.Count == 1 {
			n.Count = 2
		}
	} else {
		n.Count++

		if q != nil {
			if p, ok := q.QuestWalkTarget(b, b.Class); ok {
				if n.Count < 8 && fbnTiles(b, p.X, p.Y) != 0 {
					fbnWalkPoint(c, p)

					return
				}

				q.QuestArrived(b, b.Class)
				c.Attack(ModeDead, fbnSelf(b))

				return
			}
		}
	}

	c.Sleep(0x14)
}

// thinkNpcBarbB is MONAI_Think_NpcBarb 0x5ecd70 (VERIFIED, roll order from the
// decompile). aip1 post-attack stall, aip2 charge %, aip3 charge distance.
// Without a fight it shuffles around its post with up to three walk tries,
// each drawing two LCG steps (three on the later tries).
func thinkNpcBarbB(c *Ctx) {
	b := c.B

	if c.Target != nil {
		t := *c.Target

		if c.InRange {
			c.Attack(ModeAttack1, t)
			c.Sleep(b.AIP(1))

			return
		}

		if c.Dist < b.AIP(3) && b.Roll(100) < b.AIP(2) {
			c.SetSpeed(100)

			if !c.RunTo(t, 0) {
				c.Sleep(10)
			}

			return
		}
	}

	x0, y0 := b.X, b.Y
	m := func(v uint32) int { return int(v % 20) }

	p := b.Seed.Step()
	q := b.Seed.Step()

	if c.move(Point{x0 + m(p) - 40, y0 + m(q) - 10}, nil, 0, false) {
		return
	}

	a := b.Seed.Step()
	bb := b.Seed.Step()
	cc := b.Seed.Step()

	sgn := -1
	if a&1 != 0 {
		sgn = 1
	}

	if c.move(Point{x0 - 10 + m(bb), y0 - 10 - 30*sgn + m(cc)}, nil, 0, false) {
		return
	}

	d := b.Seed.Step()
	e := b.Seed.Step()

	if c.move(Point{x0 - 10 + m(d), y0 - 10 + 30*sgn + m(e)}, nil, 0, false) {
		return
	}

	c.Sleep(0xf)
}

// FBNWussieHost is the Act 5 Rescue on Mount Arreat state the Wussie reads.
type FBNWussieHost interface {
	// SavedGroupUnit is QUEST_A5_..._GetGroupSavedUnit: saved is true when
	// the quest group has been saved; u is the unit to walk to (ok false if
	// none).
	SavedGroupUnit(b *Brain) (u Target, ok bool, saved bool)
	// GroupCounted is IsGroupMemberCounted.
	GroupCounted(b *Brain) bool
	// CorpseNear is IsClass1B2CorpseNearby (the barbarian corpses).
	CorpseNear(b *Brain) bool
	// NearGroup is HandleWussieNearGroup (+ target-list bookkeeping when the
	// unit is in mode 0xb).
	NearGroup(b *Brain)
	// Refresh is RefreshLogIfStarted.
	Refresh(b *Brain)
	// Shout sends the help packet to the player.
	Shout(b *Brain, p Target)
	// Rescued increments the group counter, clears the path, cancels the
	// think event and removes the unit from the level.
	Rescued(b *Brain)
}

// thinkWussieB is MONAI_Think_Wussie 0x5ed4d0 (VERIFIED flow; rolls: a 1000
// bounded draw below 100 for the shout, then 100 bounded draws 70 / 10).
func thinkWussieB(c *Ctx) {
	b := c.B
	w, hasQuest := c.W.(FBNWussieHost)

	if !hasQuest {
		c.Sleep(0x19)

		return
	}

	u, haveUnit, saved := w.SavedGroupUnit(b)

	if !saved {
		if h, ok := fbnHost(c); ok {
			if p, _, found := h.NearestPlayerNearby(b); found && p.IsPlayer {
				if !w.CorpseNear(b) {
					w.Refresh(b)

					if b.Roll(1000) < 100 {
						w.Shout(b, p)
					}
				} else {
					w.NearGroup(b)
				}
			}
		}

		c.Sleep(0x19)

		return
	}

	if haveUnit {
		if !w.GroupCounted(b) {
			if fbnTiles(b, u.X, u.Y) > 3 {
				if b.Roll(100) < 0x46 {
					fbnWalk(c, u)

					return
				}

				if b.Roll(100) < 10 {
					c.Wander(4)

					return
				}

				c.Sleep(10)

				return
			}
		}

		w.Rescued(b)
		fbnStop(c)

		return
	}

	c.Sleep(0x19)
}

func fbnWalk(c *Ctx, t Target) {
	if !c.WalkTo(t, 7) {
		c.Sleep(10)
	}
}

func fbnStop(c *Ctx) {
	c.acted = true
	c.B.Wake = waitForever
}
