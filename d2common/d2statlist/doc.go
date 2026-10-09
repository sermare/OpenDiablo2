// Package d2statlist is a pure model of Diablo II's stat lists: the sums of
// item properties that equipment, charms, sockets and set bonuses add to a
// hero, and the derivations the game does with them (vitality -> life,
// energy -> mana, dexterity -> defense and attack rating, resistances with
// their caps, block chance, speed breakpoints inputs, skill bonuses).
//
// It has no engine dependencies and touches no game files: callers feed it
// numbers read from ItemStatCost.txt, charstats.txt, armor/weapons.txt and
// the item properties of a save. See the notes in the d2-re-notes repository
// (itemgen.md section 2, skills-combat.md section c) for the binary facts;
// statements that are not verified in the binary are marked "UNVERIFIED".
//
// Units: values are whole numbers as printed on an item ("+40 to Life"). The
// game keeps life, mana and stamina in 8.8 fixed point (value << ValShift 8);
// here only the internal derivations use 8.8 where fractions matter.
package d2statlist
