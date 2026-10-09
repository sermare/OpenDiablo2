package diablo2item

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// CreationLine describes what the creator rolled for the item on one line, for
// the logs of the scenarios: code, quality, item level, name, base values
// (defense, durability, damage, sockets, ethereal), affixes and every property
// as stat id[/param]=value (set bonus tiers marked @pieces).
func (i *Item) CreationLine() string {
	var b strings.Builder

	cur, max := i.Durability()
	fmt.Fprintf(&b, "code=%s quality=%s ilvl=%d name=%q", i.CommonCode, i.QualityName(), i.itemLevel,
		strings.ReplaceAll(stripColors(i.Label()), "\n", " / "))

	if a := i.attributes; a != nil {
		fmt.Fprintf(&b, " identified=%t ethereal=%t sockets=%d defense=%d durability=%d/%d",
			a.identitified, a.ethereal, a.numSockets, a.defense, cur, max)

		if a.damageOneHand.max > 0 {
			fmt.Fprintf(&b, " dmg1=%d-%d", a.damageOneHand.min, a.damageOneHand.max)
		}

		if a.damageTwoHand.max > 0 {
			fmt.Fprintf(&b, " dmg2=%d-%d", a.damageTwoHand.min, a.damageTwoHand.max)
		}

		if a.currentStackSize > 0 {
			fmt.Fprintf(&b, " qty=%d", a.currentStackSize)
		}
	}

	if i.UniqueCode != "" {
		fmt.Fprintf(&b, " unique=%q", i.UniqueCode)
	}

	if i.SetItemCode != "" {
		fmt.Fprintf(&b, " set=%q setitem=%q", i.SetCode, i.SetItemCode)
	}

	if len(i.PrefixCodes)+len(i.SuffixCodes) > 0 {
		fmt.Fprintf(&b, " prefix=%v suffix=%v", i.PrefixCodes, i.SuffixCodes)
	}

	if i.rolled == nil {
		return b.String()
	}

	b.WriteString(" props=[")

	n := 0

	for _, w := range i.rolled.Writes {
		if w.Kind == 'S' {
			continue
		}

		if n > 0 {
			b.WriteString(" ")
		}

		n++

		b.WriteString(strconv.Itoa(w.Stat))

		if w.Param != 0 {
			fmt.Fprintf(&b, "/%d", w.Param)
		}

		fmt.Fprintf(&b, "=%d", w.Value>>uint(i.factory.valShift(i.factory.creator, w.Stat)))

		if w.List >= setListFirst && w.List <= setListLast {
			fmt.Fprintf(&b, "@%d", w.List-setListFirst+2)
		}
	}

	b.WriteString("]")

	return b.String()
}

var colorToken = regexp.MustCompile(`\[[a-z]+\]`)

func stripColors(s string) string {
	return colorToken.ReplaceAllString(s, "")
}
