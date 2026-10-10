// Command od2mpbot is a scripted headless multiplayer client for a d2realm
// server (cmd/od2server): it joins a game, forms a party, takes the exit portal,
// fights the monsters of the level with the other bots and prints the state of
// its world, so a script can compare what several processes saw. It opens no
// game window and needs no game files.
//
//	od2mpbot -addr 127.0.0.1:4000 -name alice -game duel -host -expect 2
//	od2mpbot -addr 127.0.0.1:4000 -name bob   -game duel       -expect 2
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2mp"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2realm"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:4000", "realm address")
	name := flag.String("name", "", "account and hero name (letters only)")
	game := flag.String("game", "bots", "game name")
	host := flag.Bool("host", false, "create the game (otherwise join it)")
	expect := flag.Int("expect", 2, "heroes to wait for before leaving town")
	limit := flag.Duration("limit", 90*time.Second, "give up after this long")
	flag.Parse()

	if *name == "" {
		fmt.Fprintln(os.Stderr, "need -name")
		os.Exit(2)
	}

	deadline := time.Now().Add(*limit)

	if err := run(*addr, *name, *game, *host, *expect, deadline); err != nil {
		fmt.Printf("BOT %s FAIL %v\n", *name, err)
		os.Exit(1)
	}
}

func run(addr, name, game string, host bool, expect int, deadline time.Time) error {
	var c *d2realm.Client

	for {
		var err error
		if c, err = d2realm.Dial(addr); err == nil {
			break
		}

		if time.Now().After(deadline) {
			return err
		}

		time.Sleep(200 * time.Millisecond)
	}

	defer c.Close()

	if _, err := c.Hello(name); err != nil {
		return err
	}

	d, err := d2s.NewCharacter(name, d2s.Sorceress, d2s.NewCharacterFlags{Expansion: true}, d2s.DefaultAppearance(d2s.Sorceress))
	if err != nil {
		return err
	}

	if err := c.Upload(d); err != nil {
		return err
	}

	var j d2realm.GameJoined

	if host {
		j, err = c.Create(d2realm.CreateGame{Name: game, MaxPlayers: 8})
	} else {
		for {
			if j, err = c.Join(game, ""); err == nil || time.Now().After(deadline) {
				break
			}

			time.Sleep(300 * time.Millisecond)
		}
	}

	if err != nil {
		return err
	}

	rules := d2mp.DefaultRules{}
	p := d2realm.NewPlayer(c, j, rules, nil)
	fmt.Printf("BOT %s joined game=%s unit=%d seed=%#x\n", name, game, j.UnitID, j.Seed)

	wait := func(what string, cond func(r *d2mp.Replica) bool) error {
		for time.Now().Before(deadline) {
			ok := false
			p.View(func(r *d2mp.Replica) { ok = cond(r) })

			if ok {
				return nil
			}

			time.Sleep(20 * time.Millisecond)
		}

		return fmt.Errorf("timeout waiting for %s", what)
	}

	players := func(r *d2mp.Replica) (n int) {
		for _, u := range r.Units() {
			if u.Kind == d2mp.KindPlayer {
				n++
			}
		}

		return n
	}

	if err := wait("all heroes in town", func(r *d2mp.Replica) bool { return r.Level == 1 && players(r) >= expect }); err != nil {
		return err
	}

	// party: the host invites everybody, the others accept
	if host {
		p.View(func(r *d2mp.Replica) {
			for _, u := range r.Units() {
				if u.Kind == d2mp.KindPlayer && u.ID != j.UnitID {
					_ = p.GameCommand(d2mp.Command{Type: d2mp.CmdPartyInvite, Target: u.ID})
				}
			}
		})
	} else {
		if err := wait("an invitation", func(r *d2mp.Replica) bool { return len(r.Msgs) > 0 }); err != nil {
			return err
		}

		_ = p.GameCommand(d2mp.Command{Type: d2mp.CmdPartyAccept})
	}

	if err := wait("party", func(r *d2mp.Replica) bool {
		u := r.Unit(j.UnitID)

		return u != nil && u.Party != 0
	}); err != nil {
		return err
	}

	// walk a bit (everybody sees it), then take the exit portal
	var x, y float64

	p.View(func(r *d2mp.Replica) { x, y, _ = r.Pos(j.UnitID) })
	_ = p.WalkTo(true, x+3, y)
	time.Sleep(800 * time.Millisecond)

	var exit uint32

	p.View(func(r *d2mp.Replica) {
		for _, u := range r.Units() {
			if u.Kind == d2mp.KindObject && u.Type == d2mp.ObjPortal {
				exit = u.ID
			}
		}
	})

	_ = p.Interact(d2mp.KindObject, exit)

	if err := wait("level 2", func(r *d2mp.Replica) bool { return r.Level == 2 && len(r.Units()) > 4 }); err != nil {
		return err
	}

	fmt.Printf("BOT %s in level 2\n", name)

	// fight: everybody goes for the live monster with the lowest id
	for time.Now().Before(deadline) {
		var target uint32

		p.View(func(r *d2mp.Replica) {
			for _, u := range r.Units() {
				if u.Kind == d2mp.KindMonster && !u.Dead {
					target = u.ID

					return
				}
			}
		})

		if target == 0 {
			break
		}

		_ = p.Interact(d2mp.KindMonster, target)

		time.Sleep(300 * time.Millisecond)
	}

	// let corpses vanish and the other bots finish, then report
	time.Sleep(4 * time.Second)

	var out string

	p.View(func(r *d2mp.Replica) {
		kills, items := 0, 0

		for k, v := range r.Stats {
			if k == d2mp.EvDeath {
				kills = v
			}
		}

		for _, u := range r.Units() {
			if u.Kind == d2mp.KindItem {
				items++
			}
		}

		u := r.Unit(j.UnitID)
		out = fmt.Sprintf("BOT %s RESULT level=%d digest=%016x deaths_seen=%d items_on_ground=%d xp=%d party=%d hp=%d heroes=%d",
			name, r.Level, r.Digest(), kills, items, r.XP, u.Party, u.HP, players(r))
	})

	fmt.Println(out)

	_ = p.Say("bot "+name+" done", "")

	// stay until every bot has reported (a leaver would change the others' state)
	time.Sleep(6 * time.Second)

	return p.Leave()
}
