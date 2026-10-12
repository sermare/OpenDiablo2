package d2gamescreen

import (
	"errors"
	"strconv"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
)

// commandGoldMove is the scripted side of the gold dialogs (scenario w48-gold-dialogs):
//
//	goldmove drop|deposit|withdraw <amount>   what OK of the gold dialog does
//	goldmove stash <amount>                   sets the stash gold
//	goldmove msg <code>                       shows the vendor reply dialog of a trade reply code
func (v *Game) commandGoldMove(args []string) error {
	if len(args) != 2 || v.localPlayer == nil || v.gameControls == nil {
		return errors.New("usage: goldmove <drop|deposit|withdraw|stash|msg> <n> (in a game)")
	}

	n, err := strconv.Atoi(args[1])
	if err != nil || n < 0 {
		return errors.New("goldmove needs a non-negative number")
	}

	gc := v.gameControls

	switch args[0] {
	case "drop":
		v.logGold("drop", gc.GoldMove(d2inventory.GoldDialogDrop, n), n)
	case "deposit":
		v.logGold("deposit", gc.GoldMove(d2inventory.GoldDialogDeposit, n), n)
	case "withdraw":
		v.logGold("withdraw", gc.GoldMove(d2inventory.GoldDialogWithdraw, n), n)
	case "stash":
		v.localPlayer.StashGold = d2inventory.SanitizeLoadedGold(n, d2inventory.StashGoldLimit)
		v.Infof("GOLDMOVE stash set=%d", v.localPlayer.StashGold)
	case "msg":
		gc.Trade.ShowReply(n)
	default:
		return errors.New("goldmove: unknown kind " + args[0])
	}

	return nil
}

func (v *Game) logGold(kind string, ok bool, n int) {
	v.Infof("GOLDMOVE %s amount=%d ok=%v gold=%d stash=%d", kind, n, ok, v.localPlayer.Gold, v.localPlayer.StashGold)
}
