package d2equip

import (
	"bytes"
	"encoding/csv"
	"errors"
	"strconv"
	"strings"
)

// Type is one row of ItemTypes.txt as far as equipping needs it.
type Type struct {
	Code           string
	Equiv1, Equiv2 string
	Body           bool // the "Body" column: the type can be worn
	BodyLoc1       Loc
	BodyLoc2       Loc
	Shoots         string // quiver type a weapon of this type needs
	Quiver         string // weapon type a quiver of this type belongs to
	Class          string // "Class" column (ama, sor, ...) for class specific types
}

// Types is ItemTypes.txt keyed by code.
type Types struct {
	byCode map[string]*Type
}

// ParseTypes reads ItemTypes.txt.
func ParseTypes(data []byte) (*Types, error) {
	rows, col, err := readTSV(data)
	if err != nil {
		return nil, err
	}

	t := &Types{byCode: map[string]*Type{}}

	for _, r := range rows {
		code := cell(r, col, "code")
		if code == "" {
			continue
		}

		t.byCode[code] = &Type{
			Code: code, Equiv1: cell(r, col, "equiv1"), Equiv2: cell(r, col, "equiv2"),
			Body:     cell(r, col, "body") == "1",
			BodyLoc1: LocFromCode(cell(r, col, "bodyloc1")), BodyLoc2: LocFromCode(cell(r, col, "bodyloc2")),
			Shoots: cell(r, col, "shoots"), Quiver: cell(r, col, "quiver"), Class: cell(r, col, "class"),
		}
	}

	if len(t.byCode) == 0 {
		return nil, errors.New("d2equip: ItemTypes.txt has no rows")
	}

	return t, nil
}

// Get returns the type with a code, or nil.
func (t *Types) Get(code string) *Type {
	if t == nil {
		return nil
	}

	return t.byCode[code]
}

const maxEquivDepth = 16

// IsA reports whether type code is, or inherits (Equiv1/Equiv2, transitively)
// from, the ancestor type.
func (t *Types) IsA(code, ancestor string) bool {
	return t.isA(code, ancestor, 0)
}

func (t *Types) isA(code, ancestor string, depth int) bool {
	if code == "" || depth > maxEquivDepth {
		return false
	}

	if code == ancestor {
		return true
	}

	ty := t.Get(code)
	if ty == nil {
		return false
	}

	return t.isA(ty.Equiv1, ancestor, depth+1) || t.isA(ty.Equiv2, ancestor, depth+1)
}

// find returns the first value of f over the type and its ancestors that is non-empty.
func (t *Types) find(code string, f func(*Type) string, depth int) string {
	ty := t.Get(code)
	if ty == nil || depth > maxEquivDepth {
		return ""
	}

	if v := f(ty); v != "" {
		return v
	}

	if v := t.find(ty.Equiv1, f, depth+1); v != "" {
		return v
	}

	return t.find(ty.Equiv2, f, depth+1)
}

// ClassOf returns the class code (ama, sor, nec, pal, bar, dru, ass) an item
// type is restricted to, looking at the type and its ancestors; "" for none.
func (t *Types) ClassOf(code string) string {
	return t.find(code, func(ty *Type) string { return ty.Class }, 0)
}

// Locs returns the body locations the type can be worn in (nil when it cannot
// be worn). A type that does not say so itself inherits them from its ancestors
// (a "Circlet" row has its own, a class helm inherits "helm").
func (t *Types) Locs(code string) []Loc {
	return t.locs(code, 0)
}

func (t *Types) locs(code string, depth int) []Loc {
	ty := t.Get(code)
	if ty == nil || depth > maxEquivDepth {
		return nil
	}

	if ty.Body && ty.BodyLoc1 != LocNone {
		out := []Loc{ty.BodyLoc1}
		if ty.BodyLoc2 != LocNone && ty.BodyLoc2 != ty.BodyLoc1 {
			out = append(out, ty.BodyLoc2)
		}

		return out
	}

	if l := t.locs(ty.Equiv1, depth+1); l != nil {
		return l
	}

	return t.locs(ty.Equiv2, depth+1)
}

// Base is what armor.txt / weapons.txt say about a base item.
type Base struct {
	Code          string
	Type, Type2   string
	Weapon        bool
	TwoHanded     bool // the 2handed column
	OneOrTwo      bool // the 1or2handed column: a Barbarian may wield it in one hand
	ReqStr        int
	ReqDex        int
	ReqLevel      int // levelreq
	Durability    int
	NoDurability  bool
	TwoHandedKind string // 2handedwclass, informational
}

// Bases maps item codes to Base rows.
type Bases map[string]Base

// ParseBases reads armor.txt and weapons.txt (either may be nil).
func ParseBases(armor, weapons []byte) (Bases, error) {
	out := Bases{}

	for i, data := range [][]byte{armor, weapons} {
		if len(data) == 0 {
			continue
		}

		rows, col, err := readTSV(data)
		if err != nil {
			return nil, err
		}

		for _, r := range rows {
			code := cell(r, col, "code")
			if code == "" {
				continue
			}

			out[code] = Base{
				Code: code, Type: cell(r, col, "type"), Type2: cell(r, col, "type2"), Weapon: i == 1,
				TwoHanded: cell(r, col, "2handed") == "1", OneOrTwo: cell(r, col, "1or2handed") == "1",
				ReqStr: num(r, col, "reqstr"), ReqDex: num(r, col, "reqdex"), ReqLevel: num(r, col, "levelreq"),
				Durability: num(r, col, "durability"), NoDurability: cell(r, col, "nodurability") == "1",
				TwoHandedKind: cell(r, col, "2handedwclass"),
			}
		}
	}

	return out, nil
}

func readTSV(data []byte) ([][]string, map[string]int, error) {
	rd := csv.NewReader(bytes.NewReader(data))
	rd.Comma = '\t'
	rd.LazyQuotes = true
	rd.FieldsPerRecord = -1

	recs, err := rd.ReadAll()
	if err != nil {
		return nil, nil, err
	}

	if len(recs) < 2 {
		return nil, nil, errors.New("d2equip: empty table")
	}

	col := map[string]int{}

	for i, h := range recs[0] {
		h = strings.ToLower(strings.TrimSpace(h))
		if _, dup := col[h]; !dup {
			col[h] = i
		}
	}

	return recs[1:], col, nil
}

func cell(r []string, col map[string]int, name string) string {
	i, ok := col[name]
	if !ok || i >= len(r) {
		return ""
	}

	return strings.TrimSpace(r[i])
}

func num(r []string, col map[string]int, name string) int {
	n, _ := strconv.Atoi(cell(r, col, name))

	return n
}
