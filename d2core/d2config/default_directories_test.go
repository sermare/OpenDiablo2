package d2config

import (
	"strings"
	"testing"
)

func TestWindowsConfigBase(t *testing.T) {
	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}

	cases := []struct {
		env  map[string]string
		want string
	}{
		{map[string]string{"APPDATA": `C:\Users\me\AppData\Roaming`}, `C:\Users\me\AppData\Roaming`},
		{map[string]string{"APPDATA": `C:\Users\me\AppData\Roaming\`}, `C:\Users\me\AppData\Roaming`},
		{map[string]string{"USERPROFILE": `C:\Users\me\`}, `C:\Users\me\AppData\Roaming`},
		{map[string]string{}, ``},
	}

	for _, c := range cases {
		if got := windowsConfigBase(env(c.env)); got != c.want {
			t.Errorf("windowsConfigBase(%v) = %q, want %q", c.env, got, c.want)
		}
	}
}

func TestConfigDirEnvOverride(t *testing.T) {
	t.Setenv(ConfigDirEnv, "/x/y")

	if ConfigDir() != "/x/y" {
		t.Fatal("OD2_CONFIG_DIR ignored")
	}
}

func TestDecodeConfigBOMAndCRLF(t *testing.T) {
	for _, in := range []string{
		"\xef\xbb\xbf{\r\n  \"MpqPath\": \"C:/Games/D2\"\r\n}\r\n",
		"{\"MpqPath\": \"C:/Games/D2\"}",
	} {
		cfg := &Configuration{}
		if err := DecodeConfig(strings.NewReader(in), cfg); err != nil || cfg.MpqPath != "C:/Games/D2" {
			t.Errorf("DecodeConfig(%q) = %v, MpqPath %q", in, err, cfg.MpqPath)
		}
	}

	if err := DecodeConfig(strings.NewReader("{bad"), &Configuration{}); err == nil {
		t.Error("bad JSON accepted")
	}
}
