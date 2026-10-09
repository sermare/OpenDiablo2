package d2s

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Errors returned by Write.
var (
	ErrNoBody        = errors.New("d2s: character has a body but Body is nil")
	ErrBadName       = errors.New("d2s: character name is too long or contains a NUL")
	ErrValueRange    = errors.New("d2s: value does not fit its field")
	ErrSocketCount   = errors.New("d2s: SocketCount does not match the number of children")
	ErrPropertyOrder = errors.New("d2s: property group is incomplete")
)

const maxNameLen = nameLength - 1

// Write serialises c into a .d2s file. The size field and checksum are
// recomputed. Fields the parser keeps verbatim (Header.Raw, Body.QuestsRaw,
// Body.WaypointsRaw, Body.NPC, Character.CorpseHeader, Character.Trailing and
// the raw item flags) are written back unchanged, so Write(Parse(x)) == x for
// well-formed 1.14 saves. The structured fields take precedence over the raw
// ones, so editing them edits the file. Body.Stats is the source of truth for
// the stats section (see Body.SetStat); Attributes is not consulted.
//
// tables is required for any character with a body.
func Write(c *Character, tables *ItemTables) ([]byte, error) {
	if c == nil || c.Header == nil {
		return nil, errors.New("d2s: nil character or header")
	}

	out, err := c.Header.marshal()
	if err != nil {
		return nil, err
	}

	if !c.Header.IsNewCharacter() {
		if c.Body == nil {
			return nil, ErrNoBody
		}

		if tables == nil {
			return nil, ErrNoTables
		}

		if out, err = c.appendBody(out, tables); err != nil {
			return nil, err
		}
	}

	le := binary.LittleEndian
	le.PutUint32(out[8:], uint32(len(out)))
	le.PutUint32(out[checksumOffset:], 0)
	le.PutUint32(out[checksumOffset:], Checksum(out))

	return out, nil
}

// marshal returns the 0x14F header bytes (size and checksum not yet final).
func (h *Header) marshal() ([]byte, error) {
	if h.Class > Assassin {
		return nil, ErrInvalidClass
	}

	if h.Version < MinVersion || h.Version > MaxVersion {
		return nil, fmt.Errorf("%w: 0x%X", ErrBadVersion, h.Version)
	}

	if len(h.Name) > maxNameLen {
		return nil, ErrBadName
	}

	for i := 0; i < len(h.Name); i++ {
		if h.Name[i] == 0 {
			return nil, ErrBadName
		}
	}

	out := make([]byte, HeaderSize)
	copy(out, h.Raw[:])

	le := binary.LittleEndian
	le.PutUint32(out[0:], Magic)
	le.PutUint32(out[4:], h.Version)
	le.PutUint32(out[statusOffset:], h.Status)
	out[classOffset] = byte(h.Class)
	out[skillCountPos] = h.SkillCount
	out[levelOffset] = h.Level

	// keep stale bytes after the terminator unless the name changed
	if old, err := readName(out[nameOffset : nameOffset+nameLength]); err != nil || old != h.Name {
		region := out[nameOffset : nameOffset+nameLength]
		for i := range region {
			region[i] = 0
		}

		copy(region, h.Name)
	}

	m := h.Mercenary
	if (le.Uint16(out[mercDeadOffset:]) != 0) != m.Dead {
		var dead uint16
		if m.Dead {
			dead = 1
		}

		le.PutUint16(out[mercDeadOffset:], dead)
	}

	le.PutUint32(out[mercIDOffset:], m.ID)
	le.PutUint16(out[mercNameOffset:], m.NameID)
	le.PutUint16(out[mercTypeOffset:], m.Type)
	le.PutUint32(out[mercExpOffset:], m.Experience)

	copy(out[appearanceStart:], h.Appearance[:])
	copy(out[colorsStart:], h.Colors[:])

	return out, nil
}

func (c *Character) appendBody(out []byte, tables *ItemTables) ([]byte, error) {
	out = c.Body.appendSections(out, tables)

	var err error

	if out, err = appendItemList(out, c.Items, tables); err != nil {
		return nil, fmt.Errorf("items: %w", err)
	}

	if out, err = c.appendCorpse(out, tables); err != nil {
		return nil, fmt.Errorf("corpse: %w", err)
	}

	if c.Header.IsExpansion() {
		out = append(out, mercTag...)

		if c.Header.Mercenary.ID != 0 {
			if out, err = appendItemList(out, c.MercItems, tables); err != nil {
				return nil, fmt.Errorf("mercenary: %w", err)
			}
		}

		if c.Header.Class == Necromancer {
			if out, err = c.appendGolem(out, tables); err != nil {
				return nil, fmt.Errorf("golem: %w", err)
			}
		}
	}

	return append(out, c.Trailing...), nil
}

func (c *Character) appendCorpse(out []byte, tables *ItemTables) ([]byte, error) {
	out = append(out, itemTag...)

	if !c.HasCorpse && len(c.Corpse) == 0 {
		return append(out, 0, 0), nil
	}

	out = append(out, 1, 0)
	out = append(out, c.CorpseHeader[:]...)

	return appendItemList(out, c.Corpse, tables)
}

func (c *Character) appendGolem(out []byte, tables *ItemTables) ([]byte, error) {
	out = append(out, golemTag...)

	if c.Golem == nil {
		return append(out, 0), nil
	}

	out = append(out, 1)

	b, err := marshalItem(c.Golem, tables)
	if err != nil {
		return nil, err
	}

	return append(out, b...), nil
}

// SetStat sets (or appends) the stat with the given id, keeping Stats and
// Attributes consistent. value is the stored value (hit points, mana and
// stamina carry eight fractional bits).
func (b *Body) SetStat(id int, value uint64) {
	for i := range b.Stats {
		if b.Stats[i].ID == id {
			b.Stats[i].Value = value
			b.Attributes.set(id, value)

			return
		}
	}

	b.Stats = append(b.Stats, Stat{ID: id, Value: value})
	b.Attributes.set(id, value)
}

func (b *Body) appendSections(out []byte, tables *ItemTables) []byte {
	// quests
	q := b.QuestsRaw
	copy(q[:], questsTag[:])

	for d := 0; d < numDifficulties; d++ {
		copy(q[questsHeaderLen+d*questsPerDiff:], b.Quests[d][:])
	}

	out = append(out, q[:]...)

	// waypoints
	w := b.WaypointsRaw
	copy(w[:], waypointsTag[:])

	for d := 0; d < numDifficulties; d++ {
		start := waypointsHeaderLen + d*waypointRecordLen + 2
		for i := 0; i < waypointFlagsLen; i++ {
			w[start+i] = byte(b.Waypoints[d] >> (8 * uint(i)))
		}
	}

	out = append(out, w[:]...)

	// NPC
	out = append(out, npcTag[:]...)
	out = append(out, b.NPC[:]...)

	// stats
	out = append(out, statsTag[:]...)
	out = append(out, b.marshalStats(tables)...)

	// skills
	out = append(out, skillsTag[:]...)

	return append(out, b.SkillPoints[:]...)
}

func (b *Body) marshalStats(tables *ItemTables) []byte {
	w := &bitSink{}

	for _, s := range b.Stats {
		bits, paramBits, ok := tables.CharStatInfo(s.ID)
		if !ok || bits == 0 {
			// fall back to the built-in widths
			if info, found := DefaultStatStorage(s.ID); found {
				bits, paramBits = info.Bits, info.ParamBits
			}
		}

		w.write(uint64(s.ID), statIDBits)

		if paramBits > 0 {
			w.write(uint64(s.Param), paramBits)
		}

		w.write(s.Value, bits)
	}

	w.write(statEndID, statIDBits)
	w.align()

	return w.buf
}

// appendItemList writes a 'JM' list: the tag, the item count, then every item
// followed by its socketed children.
func appendItemList(out []byte, items []Item, tables *ItemTables) ([]byte, error) {
	if len(items) > 0xFFFF {
		return nil, ErrValueRange
	}

	out = append(out, itemTag...)
	out = append(out, byte(len(items)), byte(len(items)>>8))

	for i := range items {
		it := &items[i]

		if !it.Simple && int(it.SocketCount) != len(it.Children) {
			return nil, fmt.Errorf("item %d: %w", i, ErrSocketCount)
		}

		b, err := marshalItem(it, tables)
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", i, err)
		}

		out = append(out, b...)

		for c := range it.Children {
			cb, err := marshalItem(&it.Children[c], tables)
			if err != nil {
				return nil, fmt.Errorf("item %d socketed item %d: %w", i, c, err)
			}

			out = append(out, cb...)
		}
	}

	return out, nil
}

// flagBits returns Flags with the boolean fields applied to their bits.
func (it *Item) flagBits() uint32 {
	f := it.Flags

	set := func(bit uint, on bool) {
		if on {
			f |= 1 << bit
		} else {
			f &^= 1 << bit
		}
	}

	set(4, it.Identified)
	set(11, it.Socketed)
	set(13, it.New)
	set(16, it.Ear)
	set(17, it.Starter)
	set(21, it.Simple)
	set(22, it.Ethereal)
	set(24, it.Personalized)
	set(26, it.Runeword)

	return f
}

// marshalItem encodes one item without its socketed children.
func marshalItem(it *Item, t *ItemTables) ([]byte, error) {
	w := &bitSink{}

	w.write('J'|'M'<<8, 16)
	w.write(uint64(it.flagBits()), 32)
	w.write(uint64(it.Version), 10)
	w.write(uint64(it.Location), 3)
	w.write(uint64(it.Equipped), 4)
	w.write(uint64(it.X), 4)
	w.write(uint64(it.Y), 4)
	w.write(uint64(it.Page), 3)

	if it.Ear {
		if it.EarInfo == nil {
			return nil, errors.New("d2s: ear item without EarInfo")
		}

		w.write(uint64(it.EarInfo.Class), 3)
		w.write(uint64(it.EarInfo.Level), 7)

		if err := writeCString7(w, it.EarInfo.Name); err != nil {
			return nil, err
		}

		w.align()

		return w.buf, nil
	}

	if len(it.Code) > 4 {
		return nil, fmt.Errorf("%w: %q", ErrUnknownItem, it.Code)
	}

	for i := 0; i < 4; i++ {
		c := byte(' ')
		if i < len(it.Code) {
			c = it.Code[i]
		}

		w.write(uint64(c), 8)
	}

	w.write(uint64(it.SocketCount), 3)

	if !it.Simple {
		if err := it.writeExtended(w, t); err != nil {
			return nil, err
		}
	}

	w.align()

	return w.buf, nil
}

func writeCString7(w *bitSink, s string) error {
	for i := 0; i < len(s); i++ {
		if s[i] == 0 || s[i] >= 0x80 {
			return fmt.Errorf("%w: character %q", ErrValueRange, s[i])
		}

		w.write(uint64(s[i]), 7)
	}

	w.write(0, 7)

	return nil
}

func bit(b bool) uint64 {
	if b {
		return 1
	}

	return 0
}

//nolint:gocyclo // inverse of the long fixed sequence in readExtended
func (it *Item) writeExtended(w *bitSink, t *ItemTables) error {
	kind := t.ItemKindOf(it.Code)
	if kind == 0 {
		return fmt.Errorf("%w: %q", ErrUnknownItem, it.Code)
	}

	w.write(uint64(it.ID), 32)
	w.write(uint64(it.Level), 7)
	w.write(uint64(it.Quality), 4)

	w.write(bit(it.HasPicture), 1)

	if it.HasPicture {
		w.write(uint64(it.Picture), 3)
	}

	w.write(bit(it.HasClassData), 1)

	if it.HasClassData {
		w.write(uint64(it.ClassData), 11)
	}

	switch it.Quality {
	case QualityLow:
		w.write(uint64(it.LowQualityID), 3)
	case QualityHigh:
		w.write(uint64(it.HighQuality), 3)
	case QualityMagic:
		w.write(uint64(it.MagicPrefix), 11)
		w.write(uint64(it.MagicSuffix), 11)
	case QualitySet:
		w.write(uint64(it.SetID), 12)
	case QualityUnique:
		w.write(uint64(it.UniqueID), 12)
	case QualityRare, QualityCrafted:
		w.write(uint64(it.RareName1), 8)
		w.write(uint64(it.RareName2), 8)

		for i := range it.RareAffixes {
			present := it.RareMask>>uint(i)&1 == 1
			w.write(bit(present), 1)

			if present {
				w.write(uint64(it.RareAffixes[i]), 11)
			}
		}
	}

	if it.Runeword {
		w.write(uint64(it.RunewordID), 12)
		w.write(uint64(it.RunewordUnknown), 4)
	}

	if it.Personalized {
		if err := writeCString7(w, it.PersonalName); err != nil {
			return err
		}
	}

	if t.IsTome(it.Code) {
		w.write(uint64(it.TomeBits), 5)
	}

	w.write(bit(it.Timestamp), 1)

	if err := it.writeBaseStats(w, t, kind); err != nil {
		return err
	}

	if it.Socketed {
		w.write(uint64(it.TotalSockets), 4)
	}

	if it.Quality == QualitySet {
		w.write(uint64(it.SetListMask), 5)
	}

	if err := writeProperties(w, t, it.Properties); err != nil {
		return err
	}

	for _, list := range it.SetProperties {
		if err := writeProperties(w, t, list); err != nil {
			return err
		}
	}

	if it.Runeword {
		return writeProperties(w, t, it.RunewordProperties)
	}

	return nil
}

// writeStored writes v+add into a field of the given width.
func writeStored(w *bitSink, v int64, add, bits int) error {
	s := v + int64(add)
	if s < 0 || (bits < 63 && s >= int64(1)<<uint(bits)) {
		return fmt.Errorf("%w: %d in %d bits", ErrValueRange, v, bits)
	}

	w.write(uint64(s), bits)

	return nil
}

func (it *Item) writeBaseStats(w *bitSink, t *ItemTables, kind ItemKind) error {
	if kind == KindArmor {
		b, _, add, _, ok := t.StatSaveInfo(statArmorClass)
		if !ok {
			return fmt.Errorf("%w: armorclass", ErrUnknownStat)
		}

		if err := writeStored(w, int64(it.Defense), add, b); err != nil {
			return err
		}
	}

	if kind == KindArmor || kind == KindWeapon {
		mb, _, _, _, ok := t.StatSaveInfo(statMaxDurability)
		if !ok {
			return fmt.Errorf("%w: maxdurability", ErrUnknownStat)
		}

		if err := writeStored(w, int64(it.MaxDurability), 0, mb); err != nil {
			return err
		}

		if it.MaxDurability > 0 {
			cb, _, _, _, ok := t.StatSaveInfo(statDurability)
			if !ok {
				return fmt.Errorf("%w: durability", ErrUnknownStat)
			}

			if err := writeStored(w, int64(it.Durability), 0, cb); err != nil {
				return err
			}
		}
	}

	if t.IsStackable(it.Code) {
		return writeStored(w, int64(it.Quantity), 0, quantityBits)
	}

	return nil
}

func writeProperties(w *bitSink, t *ItemTables, props []Property) error {
	for i := 0; i < len(props); i++ {
		p := props[i]

		w.write(uint64(p.ID), 9)

		if err := writeProperty(w, t, p, true); err != nil {
			return err
		}

		for _, f := range groupFollowers[p.ID] {
			i++

			if i >= len(props) || props[i].ID != f {
				return fmt.Errorf("%w: stat %d needs follower %d", ErrPropertyOrder, p.ID, f)
			}

			if err := writeProperty(w, t, props[i], false); err != nil {
				return err
			}
		}
	}

	w.write(propertyEndID, 9)

	return nil
}

func writeProperty(w *bitSink, t *ItemTables, p Property, withParam bool) error {
	valBits, paramBits, add, _, ok := t.StatSaveInfo(p.ID)
	if !ok {
		return fmt.Errorf("%w: %d", ErrUnknownStat, p.ID)
	}

	if withParam && paramBits > 0 {
		w.write(uint64(p.Param), paramBits)
	}

	return writeStored(w, p.Value, add, valBits)
}
