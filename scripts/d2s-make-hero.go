//go:build ignore

// d2s-make-hero writes a level 94 sample hero (.d2s) generated from the game
// tables, for the long playthrough scenarios (OD2_HERO=barb). Nothing it writes
// may be committed: pass a scratch path.
//
//	go run scripts/d2s-make-hero.go [-class barbarian] [-name NokkaBarb] [-template sample.d2s] out.d2s $D2_TABLES
//
// -template is the sample Sorceress (D2S_SAMPLE_BODY): the hero then shares its
// quests, waypoints, NPC flags, active difficulty, map seed and mercenary, so a
// scenario plays the same game with a different hero. The hero is alive and
// not hardcore (the sample is a dead hardcore character, see d2s-revive.go).
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero/herogen"
)

func main() {
	class := flag.String("class", "barbarian", "hero preset (barbarian)")
	name := flag.String("name", "NokkaBarb", "character name")
	template := flag.String("template", "", "a .d2s whose quests, waypoints, difficulty, map seed and mercenary are copied")
	flag.Parse()

	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: d2s-make-hero [-class barbarian] [-name N] [-template sample.d2s] out.d2s tables-dir")
		os.Exit(2)
	}

	out, dir := flag.Arg(0), flag.Arg(1)

	if *class != "barbarian" {
		fmt.Fprintf(os.Stderr, "d2s-make-hero: unknown class preset %q\n", *class)
		os.Exit(2)
	}

	t, err := herogen.LoadTables(dir)
	if err != nil {
		fatal(err)
	}

	var tmpl *herogen.Template

	if *template != "" {
		data, err := os.ReadFile(*template)
		if err != nil {
			fatal(err)
		}

		c, err := d2s.Parse(data, t.Save)
		if err != nil {
			fatal(fmt.Errorf("template: %w", err))
		}

		if tmpl, err = herogen.TemplateFrom(c); err != nil {
			fatal(err)
		}
	}

	hero, err := t.Generate(herogen.Barbarian(*name), tmpl)
	if err != nil {
		fatal(err)
	}

	if err := os.WriteFile(out, hero.Data, 0o600); err != nil {
		fatal(err)
	}

	tot := hero.Totals
	fmt.Printf("hero %s %v level %d: life %d mana %d stamina %d defense %d attack rating %d damage %d-%d, %d items, %d bytes\n",
		hero.Character.Header.Name, hero.Character.Header.Class, hero.Character.Header.Level, tot.MaxLife, tot.MaxMana,
		tot.MaxStamina, tot.Defense, tot.AttackRating, tot.DamageMin, tot.DamageMax, len(hero.Character.Items),
		len(hero.Data))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "d2s-make-hero:", err)
	os.Exit(1)
}
