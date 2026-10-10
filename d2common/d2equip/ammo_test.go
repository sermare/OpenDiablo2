package d2equip

import "testing"

// miniAmmoTypes has the ammunition rows of ItemTypes.txt: a bow shoots its quiver, javelins, throwing
// knives and throwing axes are thrown, a spear is not.
const miniAmmoTypes = "ItemType\tCode\tEquiv1\tEquiv2\tBody\tBodyLoc1\tBodyLoc2\tShoots\tQuiver\tThrowable\tClass\n" +
	"Weapon\tweap\t\t\t0\t\t\t\t\t0\t\n" +
	"Melee\tmele\tweap\t\t0\t\t\t\t\t0\t\n" +
	"Spear\tspea\tmele\t\t0\t\t\t\t\t0\t\n" +
	"Missile Weapon\tmiss\tweap\t\t0\t\t\t\t\t0\t\n" +
	"Thrown Weapon\tthro\tweap\t\t0\t\t\t\t\t1\t\n" +
	"Combo Weapon\tcomb\tmele\tthro\t0\t\t\t\t\t1\t\n" +
	"Javelin\tjave\tcomb\tspea\t1\trarm\tlarm\t\t\t1\t\n" +
	"Amazon Javelin\tajav\tjave\t\t1\trarm\tlarm\t\t\t1\tama\n" +
	"Bow\tbow\tmiss\t\t1\trarm\t\tbowq\t\t0\t\n" +
	"Amazon Bow\tabow\tbow\t\t1\trarm\t\tbowq\t\t0\tama\n" +
	"Quiver\tbowq\tmiss\t\t1\tlarm\t\t\tbow\t0\t\n"

func TestShootsAndThrowable(t *testing.T) {
	ty, err := ParseTypes([]byte(miniAmmoTypes))
	if err != nil {
		t.Fatal(err)
	}

	r := Rules{Types: ty}

	for typ, want := range map[string]string{"bow": "bowq", "abow": "bowq", "jave": "", "spea": "", "bowq": ""} {
		if got := r.Shoots(typ); got != want {
			t.Errorf("Shoots(%q) = %q, want %q", typ, got, want)
		}
	}

	for typ, want := range map[string]bool{
		"jave": true, "ajav": true, "thro": true, "bow": false, "abow": false, "spea": false, "mele": false,
		"bowq": false, "": false, "nosuchtype": false,
	} {
		if got := r.IsThrowable(typ); got != want {
			t.Errorf("IsThrowable(%q) = %v, want %v", typ, got, want)
		}
	}
}
