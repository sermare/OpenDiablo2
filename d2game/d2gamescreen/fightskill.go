package d2gamescreen

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// The skills of a scripted fight (kill: steps of OD2_AUTOSCRIPT).
//
// A hero with a plain attack on the left button (the sample Sorceress, a new hero) fights as it always
// did: it walks up to the nearest monster and swings, and a Sorceress casts her right or best spell from a
// distance (attackSpell). A hero whose left button holds a real attack skill, as the generated class heroes
// do (d2core/d2hero/herogen), fights with it, like a player who clicks the left button on a monster:
//
//   - a melee skill (Zeal, Smite, Frenzy, Dragon Talon, Power Strike, ...; skills.txt range h2h) is cast
//     when the target is within reach, the hero walks up to it as for a plain swing;
//   - a ranged skill (Bone Spear, Lightning Fury, Firestorm, Fire Ball, ...) is cast from where the hero
//     stands as soon as the target is within killCastRange, even point blank, and the hero holds his ground
//     while casting;
//   - the skill on the right button, and the summon and buff skills on the hotkeys, are cast before the
//     attack and again when they have run out (Battle Orders, Bone Armor, Valkyrie, Oak Sage, a Shadow
//     Warrior, Fanaticism): auras only once, since a new aura replaces the old;
//   - a hero out of mana drinks a mana potion of the belt, like the life potion of drinkIfHurt;
//   - a skill the engine refuses again and again (no line of sight, no ammunition, an unsupported
//     weapon) is dropped for the rest of the fight and the hero swings instead: no god mode.

const (
	// castRetrySeconds is the wait after a refused cast before the next try.
	castRetrySeconds = 0.35
	// castFailLimit is how many refusals in a row (other than for mana and cooldown) drop the skill for the fight.
	castFailLimit = 12
	// standRange is the distance (tiles) up to which a hero with a ranged left skill holds his ground.
	standRange = 9.0
	// standSeconds is how long after a cast the hero keeps standing.
	standSeconds = 1.5
	// manaPotionSeconds is how long the hero waits after a mana potion.
	manaPotionSeconds = 6.0
	// manaPotionFraction is the mana fraction below which the hero drinks a mana potion.
	manaPotionFraction = 0.2
	// supportManaFraction is the mana fraction a support skill needs to be left over (the attack comes first).
	supportManaFraction = 0.35
)

// fightRole says what a skill is for in a fight.
type fightRole int

const (
	roleNone   fightRole = iota
	roleBuff             // a timed state on the hero (armors, Battle Orders, Fade)
	roleSummon           // a pet (skeletons, golems, Valkyrie, Oak Sage, spirits, shadows)
	roleAura             // an aura: only one is on, a new one replaces it
)

// supportSeconds is how long after a cast a support skill is due again.
var supportSeconds = map[fightRole]float64{roleBuff: 20, roleSummon: 45, roleAura: 600}

// fightAttackSkills are the attack skills a scripted fight casts at a monster (names of skills.txt): the
// projectile, area and strike skills of the seven classes that need nothing but a target. Skills that need
// a corpse, a form (the werewolf and werebear attacks) or a trap are left out.
var fightAttackSkills = map[string]bool{
	// Amazon
	"Magic Arrow": true, "Fire Arrow": true, "Cold Arrow": true, "Multiple Shot": true, "Poison Javelin": true,
	"Exploding Arrow": true, "Lightning Bolt": true, "Ice Arrow": true, "Guided Arrow": true, "Plague Javelin": true,
	"Strafe": true, "Immolation Arrow": true, "Freezing Arrow": true, "Lightning Fury": true, "Jab": true,
	"Power Strike": true, "Impale": true, "Charged Strike": true, "Fend": true, "Lightning Strike": true,
	// Sorceress
	"Fire Bolt": true, "Ice Bolt": true, "Charged Bolt": true, "Ice Blast": true, "Fire Ball": true,
	"Lightning": true, "Chain Lightning": true, "Glacial Spike": true, "Meteor": true, "Blizzard": true,
	"Frozen Orb": true, "Nova": true, "Inferno": true, "Fire Wall": true, "Hydra": true,
	// Necromancer
	"Teeth": true, "Poison Dagger": true, "Bone Spear": true, "Bone Spirit": true, "Poison Nova": true,
	// Paladin
	"Sacrifice": true, "Smite": true, "Holy Bolt": true, "Zeal": true, "Vengeance": true, "Blessed Hammer": true,
	"Fist of the Heavens": true,
	// Barbarian
	"Bash": true, "Double Swing": true, "Stun": true, "Concentrate": true, "Frenzy": true, "Berserk": true,
	"Double Throw": true,
	// Druid
	"Firestorm": true, "Molten Boulder": true, "Eruption": true, "Arctic Blast": true, "Twister": true,
	"Tornado": true, "Volcano": true,
	// Assassin
	"Psychic Hammer": true, "Tiger Strike": true, "Dragon Talon": true, "Fists of Fire": true, "Dragon Claw": true,
	"Cobra Strike": true, "Claws of Thunder": true, "Dragon Tail": true, "Mind Blast": true, "Blades of Ice": true,
	"Royal Strike": true,
}

// fightSupportSkills are the buffs, summons and auras a scripted fight casts for the hero.
var fightSupportSkills = map[string]fightRole{
	// buffs
	"Battle Orders": roleBuff, "Shout": roleBuff, "Battle Command": roleBuff, "Bone Armor": roleBuff,
	"Frozen Armor": roleBuff, "Shiver Armor": roleBuff, "Chilling Armor": roleBuff, "Energy Shield": roleBuff,
	"Cyclone Armor": roleBuff, "Fade": roleBuff, "Quickness": roleBuff, "Venom": roleBuff, "Holy Shield": roleBuff,
	// summons
	"Raise Skeleton": roleSummon, "Raise Skeletal Mage": roleSummon, "Clay Golem": roleSummon,
	"BloodGolem": roleSummon, "IronGolem": roleSummon, "FireGolem": roleSummon, "Valkyrie": roleSummon,
	"Shadow Warrior": roleSummon, "Shadow Master": roleSummon, "Oak Sage": roleSummon,
	"Heart of Wolverine": roleSummon, "Spirit of Barbs": roleSummon, "Summon Spirit Wolf": roleSummon,
	"Summon Fenris": roleSummon, "Summon Grizzly": roleSummon, "Raven": roleSummon,
	// auras
	"Might": roleAura, "Prayer": roleAura, "Resist Fire": roleAura, "Resist Cold": roleAura,
	"Resist Lightning": roleAura, "Thorns": roleAura, "Defiance": roleAura, "Cleansing": roleAura,
	"Concentration": roleAura, "Holy Fire": roleAura, "Vigor": roleAura, "Holy Freeze": roleAura,
	"Holy Shock": roleAura, "Sanctuary": roleAura, "Meditation": roleAura, "Fanaticism": roleAura,
	"Conviction": roleAura, "Redemption": roleAura, "Salvation": roleAura,
}

// skillView is what the choice of a fight skill needs to know of one of the hero's skills.
type skillView struct {
	ID        int
	Name      string
	Points    int
	Range     string // skills.txt range: h2h (melee), rng, both or none
	Supported bool   // the skill engine can cast it
}

// fightPick is the attack a fight casts.
type fightPick struct {
	ID    int
	Melee bool // cast only when the target is within reach (skills.txt range h2h)
	Left  bool // the hero's left skill: also cast at point blank, and the hero holds his ground
}

// fightSupport is a skill cast before the attack, and again after Every seconds.
type fightSupport struct {
	ID    int
	Name  string
	Every float64
}

// leftFightPick turns the hero's left skill into a fight attack, when it is one he can use: it has
// points, is an attack skill of the table, the engine casts it and it was not dropped in this fight.
func leftFightPick(s skillView, dropped map[int]bool) (fightPick, bool) {
	if s.Points < 1 || !s.Supported || dropped[s.ID] || !fightAttackSkills[s.Name] {
		return fightPick{}, false
	}

	return fightPick{ID: s.ID, Melee: s.Range == "h2h", Left: true}, true
}

// supportRoleOf is the role of a skill in the fight (roleNone for an attack, a passive, a curse...).
func supportRoleOf(name string) fightRole { return fightSupportSkills[name] }

// planSupports lists the support skills of a fight: the skill on the right button, then the summons and
// buffs of the hotkeys. An aura is only taken from the right button, since each aura replaces the one before.
func planSupports(right skillView, hotkeys []skillView) []fightSupport {
	var out []fightSupport

	seen := map[int]bool{}

	add := func(s skillView, fromRight bool) {
		role := supportRoleOf(s.Name)
		if role == roleNone || seen[s.ID] || s.Points < 1 || !s.Supported || (role == roleAura && !fromRight) {
			return
		}

		seen[s.ID] = true
		out = append(out, fightSupport{ID: s.ID, Name: s.Name, Every: supportSeconds[role]})
	}

	add(right, true)

	for _, h := range hotkeys {
		add(h, false)
	}

	return out
}

// dueSupport returns the first support skill that is due at the fight clock (never cast, or cast Every
// seconds ago).
func dueSupport(plan []fightSupport, last map[int]float64, now float64) (fightSupport, bool) {
	for _, s := range plan {
		if at, ok := last[s.ID]; !ok || now-at >= s.Every {
			return s, true
		}
	}

	return fightSupport{}, false
}

// refusalCounts says whether a refused cast counts against the skill: a cast refused for mana or because
// its delay has not run out is no sign that the skill cannot be used.
func refusalCounts(reason string) bool {
	return reason != d2skill.ReasonMana && reason != d2skill.ReasonCooldown
}

// fightPlan is what a fight casts, worked out at its first tick.
type fightPlan struct {
	left     fightPick
	hasLeft  bool
	supports []fightSupport
}

// viewOf describes one of the hero's skills for the plan.
func (v *Game) viewOf(s *d2hero.HeroSkill) skillView {
	if s == nil || s.SkillRecord == nil {
		return skillView{ID: -1}
	}

	sv := skillView{ID: s.ID, Name: s.SkillRecord.Skill, Points: s.SkillPoints, Range: s.SkillRecord.Range}

	if eng := v.skillEngine(); eng != nil {
		sv.Supported = eng.Supported(s.ID)
	}

	return sv
}

// planFight works out the attack and the supports of the hero's skill bar.
func (v *Game) planFight() *fightPlan {
	p := v.localPlayer
	plan := &fightPlan{}

	if p.LeftSkill != nil && !isPlainAttack(p.LeftSkill.ID) {
		plan.left, plan.hasLeft = leftFightPick(v.viewOf(p.LeftSkill), nil)
	}

	// the supports only come with a left attack skill: a hero with the plain attack on the left button
	// (the sample Sorceress) fights exactly as before
	if !plan.hasLeft {
		return plan
	}

	var hot []skillView

	if p.SkillBar != nil {
		for _, h := range p.SkillBar.Hotkeys {
			if h.Skill >= 0 {
				if s := p.Skills[h.Skill]; s != nil {
					hot = append(hot, v.viewOf(s))
				}
			}
		}
	}

	plan.supports = planSupports(v.viewOf(p.RightSkill), hot)

	return plan
}

// isPlainAttack is the basic Attack (and the Assassin's Left Hand Swing) of skills.txt.
func isPlainAttack(id int) bool { return id == 0 || id == 5 }

// heroAmmoLeft and heroUseAmmo are the skill engine's view of the quiver and the thrown stack.
func (v *Game) heroAmmoLeft() int {
	if v.gameControls == nil {
		return 0
	}

	return v.gameControls.AmmoLeft()
}

func (v *Game) heroUseAmmo() bool {
	return v.gameControls != nil && v.gameControls.UseAmmo()
}

// dryOf says whether the hero is short of mana for a skill: below manaPotionFraction of the pool, or
// short of its cost.
func (v *Game) dryFor(id int) bool {
	st := v.localPlayer.Stats
	if st.MaxMana > 0 && float64(st.Mana) < manaPotionFraction*float64(st.MaxMana) {
		return true
	}

	eng := v.skillEngine()

	return eng != nil && id >= 0 && !eng.CanAfford(v.localPlayer, id)
}

// drinkMana drinks the front potion of a belt column that holds a mana potion, not more often than a
// potion takes to work.
func (v *Game) drinkMana(k *killState) {
	if v.levels.clock-k.lastMana < manaPotionSeconds || v.gameControls == nil {
		return
	}

	for col := 0; col < 4; col++ {
		code := v.gameControls.BeltFrontCode(col)
		if !strings.HasPrefix(code, "mp") {
			continue
		}

		if v.gameControls.UseBeltColumnNoSave(col) {
			k.manaPotions++
			k.lastMana = v.levels.clock
			st := v.localPlayer.Stats
			v.Infof("KILL drank the mana potion %s of column %d at %d/%d mana", code, col+1, st.Mana, st.MaxMana)

			return
		}
	}
}
