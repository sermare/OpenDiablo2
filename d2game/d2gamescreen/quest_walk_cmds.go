package d2gamescreen

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Debug commands of the quest walkthroughs (scripts/verify.d/9j-*, 9k-*): they only reach what a player does
// with the mouse (use an item from the inventory, read the quest log page) so that a scripted run can check it.

// commandUseItem is "useitem <code>": right click on the inventory item with that base code.
func (v *Game) commandUseItem(args []string) error {
	if len(args) != 1 || v.gameControls == nil {
		return errors.New("usage: useitem <code>")
	}

	ok := v.gameControls.UseInventoryItem(args[0])
	v.Infof("USEITEM code=%s took_effect=%v", args[0], ok)

	return nil
}

// commandClearInv is "clearinv": empties the inventory grid so that quest items fit.
func (v *Game) commandClearInv(_ []string) error {
	if v.gameControls == nil {
		return errors.New("not in a game")
	}

	v.Infof("CLEARINV removed %d inventory item(s)", v.gameControls.ClearInventoryGrid())

	return nil
}

// commandQuestPanel is "questpanel <act> <quest>": opens the quest log on that quest and logs the title and the
// description page the panel shows.
func (v *Game) commandQuestPanel(args []string) error {
	act, err1 := strconv.Atoi(args[0])
	idx, err2 := strconv.Atoi(args[1])

	if err1 != nil || err2 != nil || act < 1 || act > 5 || idx < 1 || idx > 6 || v.gameControls == nil {
		return fmt.Errorf("usage: questpanel <act 1-5> <quest 1-6>")
	}

	if r := v.quests(); r != nil && r.dirty {
		v.syncQuestLog()
	}

	ql := v.gameControls.QuestLog()
	ql.Select(act, idx)
	title, descr := ql.Title(act, idx), ql.DescriptionText(act, idx)

	v.Infof("QUESTPANEL act=%d quest=%d status=%d title=%q text=%q", act, idx, ql.Status(act, idx), title,
		shorten(strings.TrimSpace(descr), 90))
	ql.Close()

	return nil
}
