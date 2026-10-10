package d2txt

import "testing"

func TestMissingHeaderReportedOnce(t *testing.T) {
	d := LoadDataDictionary([]byte("Name\tVal\nfoo\t7\n"))

	var got []string

	d.OnMissing = func(f string) { got = append(got, f) }

	if !d.Next() {
		t.Fatal("no row")
	}

	if d.Number("Val") != 7 || !d.Has("Val") || d.Has("val") {
		t.Error("existing/Has")
	}

	// legacy behaviour kept: a missing header reads column 0
	if d.String("Nope") != "foo" || d.String("Nope") != "foo" || d.Number("val") != 0 {
		t.Error("legacy column 0 read changed")
	}

	if len(got) != 2 || got[0] != "Nope" || got[1] != "val" {
		t.Errorf("OnMissing calls: %v", got)
	}

	if len(d.Missing()) != 2 {
		t.Errorf("Missing: %v", d.Missing())
	}
}
