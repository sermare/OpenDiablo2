package d2setup

import (
	"reflect"
	"runtime"
	"testing"
)

func TestNormalizeInstallPath(t *testing.T) {
	cases := []struct{ in, want string }{
		{`C:\Program Files (x86)\Diablo II\`, `C:\Program Files (x86)\Diablo II`},
		{`"C:\Program Files (x86)\Diablo II"`, `C:\Program Files (x86)\Diablo II`},
		{"C:/Games/Diablo II/", `C:\Games\Diablo II`},
		{`C:\Games\Diablo II\Game.exe`, `C:\Games\Diablo II`},
		{`"D:\D2\DIABLO II.EXE"`, `D:\D2`},
		{"  C:\\D2\x00", `C:\D2`},
		{`C:\`, `C:\`},
		{``, ``},
	}

	for _, c := range cases {
		if got := NormalizeInstallPath(c.in); got != c.want {
			t.Errorf("NormalizeInstallPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestWinJoin(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{`C:\`, "Games", "Diablo II"}, `C:\Games\Diablo II`},
		{[]string{`C:\Program Files`, `\Diablo II\`}, `C:\Program Files\Diablo II`},
		{[]string{"C:/a", "b/c"}, `C:\a\b\c`},
		{[]string{"", "x"}, `x`},
	}

	for _, c := range cases {
		if got := winJoin(c.in...); got != c.want {
			t.Errorf("winJoin(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestWindowsGameDirsFor(t *testing.T) {
	got := windowsGameDirsFor([]string{"C:"}, []string{`C:\Program Files (x86)`, "", `c:\program files (x86)`})

	if got[0] != `C:\Program Files (x86)\Diablo II` {
		t.Fatalf("first candidate %q", got[0])
	}

	seen := map[string]bool{}

	for _, g := range got {
		if seen[g] {
			t.Errorf("duplicate candidate %q", g)
		}

		seen[g] = true
	}

	for _, want := range []string{
		`C:\Program Files\Diablo II`,
		`C:\Program Files (x86)\Steam\steamapps\common\Diablo II`,
		`C:\Games\Diablo II`,
		`C:\SteamLibrary\steamapps\common\Diablo II`,
	} {
		if !seen[want] {
			t.Errorf("missing candidate %q", want)
		}
	}
}

func TestWindowsInstallsFromRegistry(t *testing.T) {
	vals := map[registryInstallKey]string{
		registryInstallKeys[0]: `C:\Program Files (x86)\Diablo II\`,
		registryInstallKeys[2]: `"D:\D2\Game.exe"`,
		registryInstallKeys[3]: `c:/program files (x86)/steam`,
	}

	want := []string{
		`C:\Program Files (x86)\Diablo II`,
		`D:\D2`,
		`c:\program files (x86)\steam\steamapps\common\Diablo II`,
	}

	if got := windowsInstallsFromRegistry(vals); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}

	if got := windowsInstallsFromRegistry(nil); len(got) != 0 {
		t.Fatalf("empty registry gave %q", got)
	}
}

func TestWindowsSaveGlobs(t *testing.T) {
	got := windowsSaveGlobs(`C:\Users\me`, `C:\Users\me\OneDrive`)
	want := []string{
		`C:\Users\me\Saved Games\Diablo II*`, `C:\Users\me\Saved Games\Diablo II*\Save`,
		`C:\Users\me\OneDrive\Saved Games\Diablo II*`, `C:\Users\me\OneDrive\Saved Games\Diablo II*\Save`,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}

	if len(windowsSaveGlobs("", "")) != 0 {
		t.Fatal("empty roots should give nothing")
	}

	if !isResurrectedDir(`C:\Users\me\Saved Games\Diablo II Resurrected`) || isResurrectedDir(`C:\Users\me\Saved Games\Diablo II`) {
		t.Fatal("isResurrectedDir")
	}
}

func TestNonWindowsHasNoWindowsCandidates(t *testing.T) {
	if got := windowsGameDirs(); got != nil && runtime.GOOS != "windows" {
		t.Fatalf("expected none off Windows, got %q", got)
	}
}
