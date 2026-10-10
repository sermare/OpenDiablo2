// Command od2server runs a self-hosted realm/lobby for LAN and TCP/IP play.
// It contacts no Blizzard service. See docs/NETWORK.md.
//
//	od2server -listen :4000 -saves ./saves [-tables ~/git/d2-tables]
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2mp"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2realm"
)

func main() {
	listen := flag.String("listen", ":4000", "address to listen on")
	saves := flag.String("saves", "saves", "folder for the server-side character saves")
	tables := flag.String("tables", os.Getenv("D2_TABLES"), "folder with itemstatcost.bin (or ItemStatCost.txt), armor.txt, weapons.txt, misc.txt, ItemTypes.txt; enables item validation")
	name := flag.String("name", "OpenDiablo2 realm", "server name shown to clients")
	maxClients := flag.Int("max-clients", 256, "maximum number of connected players")
	maxGames := flag.Int("max-games", 128, "maximum number of games")
	idle := flag.Duration("idle", 10*time.Minute, "drop connections that are silent for this long (0 = never)")
	heroLife := flag.Int("hero-life", 0, "fixed life of every hero in the placeholder gameplay rules (0 = formula; scripted scenarios use a sturdy value)")
	flag.Parse()

	store, err := d2realm.NewDirStore(*saves)
	if err != nil {
		log.Fatal(err)
	}

	cfg := d2realm.Config{Store: store, ServerName: *name, MaxClients: *maxClients, MaxGames: *maxGames,
		IdleTimeout: *idle, Logf: log.Printf, Rules: d2mp.DefaultRules{HeroLife: int32(*heroLife)}}

	if *tables != "" {
		if cfg.Tables, err = loadTables(*tables); err != nil {
			log.Fatalf("tables: %v", err)
		}
	} else {
		log.Print("no -tables: items in uploaded characters are not checked and bodies use the default stat widths")
	}

	srv := d2realm.New(cfg)

	addr, err := srv.Listen(*listen)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("realm listening on %s, saves in %s", addr, *saves)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Print("shutting down")
	srv.Close()
}

func loadTables(dir string) (*d2s.ItemTables, error) {
	read := func(names ...string) ([]byte, error) {
		var err error

		for _, n := range names {
			var b []byte
			if b, err = os.ReadFile(filepath.Join(dir, n)); err == nil {
				return b, nil
			}
		}

		return nil, err
	}

	var files [5][]byte

	for i, names := range [][]string{{"itemstatcost.bin", "ItemStatCost.txt"}, {"armor.txt"}, {"weapons.txt"}, {"misc.txt"}, {"ItemTypes.txt"}} {
		b, err := read(names...)
		if err != nil {
			return nil, err
		}

		files[i] = b
	}

	return d2s.NewItemTables(files[0], files[1], files[2], files[3], files[4])
}
