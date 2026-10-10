//go:build ignore

// d2s-make-hero writes a level 94 sample hero (.d2s) generated from the game
// tables, for the long playthrough scenarios (OD2_HERO=<class>). Nothing it writes
// may be committed: pass a scratch path.
//
//	go run scripts/d2s-make-hero.go [-class barbarian] [-name NokkaBarb] [-template sample.d2s] out.d2s $D2_TABLES
//
// -class is amazon, sorc (sorceress), necro (necromancer), paladin, barb
// (barbarian), druid or assassin; -name defaults to Nokka<Class>.
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
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero/herogen"
)

func main() {
	class := flag.String("class", "barbarian", "hero preset: "+strings.Join(herogen.PresetNames(), ", "))
	name := flag.String("name", "", "character name (default Nokka<Class>)")
	template := flag.String("template", "", "a .d2s whose quests, waypoints, difficulty, map seed and mercenary are copied")
	flag.Parse()

	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: d2s-make-hero [-class barbarian] [-name N] [-template sample.d2s] out.d2s tables-dir")
		os.Exit(2)
	}

	out, dir := flag.Arg(0), flag.Arg(1)

	cls, ok := herogen.ClassOfPreset(*class)
	if !ok {
		fmt.Fprintf(os.Stderr, "d2s-make-hero: unknown class preset %q (use %s)\n", *class, strings.Join(herogen.PresetNames(), ", "))
		os.Exit(2)
	}

	if *name == "" {
		*name = herogen.DefaultName(cls)
	}

	spec, err := herogen.Preset(cls, *name)
	if err != nil {
		fatal(err)
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

	hero, err := t.Generate(spec, tmpl)
	if err != nil {
		fatal(err)
	}

	if err := os.WriteFile(out, hero.Data, 0o600); err != nil {
		fatal(err)
	}

	tot := hero.Totals
	fmt.Printf("hero %s %v level %d: life %d mana %d stamina %d defense %d attack rating %d damage %d-%d, left skill %s, %d items, %d bytes\n",
		hero.Character.Header.Name, hero.Character.Header.Class, hero.Character.Header.Level, tot.MaxLife, tot.MaxMana,
		tot.MaxStamina, tot.Defense, tot.AttackRating, tot.DamageMin, tot.DamageMax,
		herogen.SkillName(cls, spec.Left), len(hero.Character.Items), len(hero.Data))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "d2s-make-hero:", err)
	os.Exit(1)
}
