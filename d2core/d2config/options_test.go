package d2config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOptionDefaultsAndRoundTrip(t *testing.T) {
	c := DefaultConfig()
	c.SetPath(filepath.Join(t.TempDir(), "config.json"))

	for _, d := range OptionDefs() {
		if d.Default < 0 || d.Default >= len(d.Values) {
			t.Errorf("%s: default %d outside the value list", d.Key, d.Default)
		}
	}

	if got := c.OptionValue(OptSound); got != "100%" {
		t.Errorf("default sound = %s", got)
	}

	if got := c.OptionValue(OptMusic); got != "30%" {
		t.Errorf("default music = %s", got)
	}

	for key, idx := range map[string]int{OptSound: 4, OptMusic: 7, OptGamma: 8, OptAutomapSize: 1, OptNpcSpeech: 2, OptAutomapNames: 1} {
		if err := c.SetOption(key, idx); err != nil {
			t.Fatal(err)
		}
	}

	if err := c.Save(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(c.Path())
	if err != nil {
		t.Fatal(err)
	}

	back := &Configuration{}
	if err := json.Unmarshal(data, back); err != nil {
		t.Fatal(err)
	}

	for key, want := range map[string]string{OptSound: "40%", OptMusic: "70%", OptGamma: "80%", OptAutomapSize: "MINI",
		OptNpcSpeech: "TEXT ONLY", OptAutomapNames: "NO", OptContrast: "50%"} {
		if got := back.OptionValue(key); got != want {
			t.Errorf("%s after reload = %s, want %s", key, got, want)
		}
	}
}

func TestOptionErrors(t *testing.T) {
	c := &Configuration{}

	for _, tc := range []struct {
		key string
		idx int
	}{{"nope", 0}, {OptGamma, -1}, {OptGamma, OptionLevels}, {OptAutomapFade, 2}} {
		if err := c.SetOption(tc.key, tc.idx); err == nil {
			t.Errorf("SetOption(%s,%d) accepted", tc.key, tc.idx)
		}
	}

	// a stale out-of-range stored index falls back to the default
	c.Options = map[string]int{OptGamma: 99}
	if got := c.OptionValue(OptGamma); got != "50%" {
		t.Errorf("stale gamma = %s", got)
	}
}
