package diablo2item

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func flagProp(code string, on bool) *Property {
	return &Property{record: &d2records.PropertyRecord{Code: code}, computedBool: on}
}

func TestApplyFlagProperties(t *testing.T) {
	for _, tc := range []struct {
		name     string
		start    itemAttributes
		props    []*Property
		eth, ind bool
	}{
		{"ethereal only is not indestructible", itemAttributes{}, []*Property{flagProp("ethereal", true)}, true, false},
		{"indestruct only", itemAttributes{}, []*Property{flagProp("indestruct", true)}, false, true},
		{"already ethereal, indestruct off", itemAttributes{ethereal: true}, []*Property{flagProp("indestruct", false)}, true, false},
		{"already indestructible kept", itemAttributes{indestructable: true}, []*Property{flagProp("ethereal", true)}, true, true},
	} {
		attrs := tc.start
		applyFlagProperties(&attrs, tc.props)

		if attrs.ethereal != tc.eth || attrs.indestructable != tc.ind {
			t.Errorf("%s: got eth=%v ind=%v", tc.name, attrs.ethereal, attrs.indestructable)
		}
	}
}
