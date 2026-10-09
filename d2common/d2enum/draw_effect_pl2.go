package d2enum

// PL2Table names the PL2 table the real game applies for a draw mode
// (Game.exe GFX_GetDrawModeTable 0x50fea0, verified: 0 -> T2, 1 -> T1,
// 2 -> T0, 3 -> additive, 4 -> multiplicative, 6 -> table G, 5 and others
// none; mode 7 forces the palette brightened to 170%).
type PL2Table int

// PL2 tables used by draw modes.
const (
	PL2TableNone           PL2Table = iota
	PL2TableAlpha0                  // T0, 25% weight on the first index (d2pl2.PL2.AlphaBlend[0])
	PL2TableAlpha1                  // T1, 50% (AlphaBlend[1])
	PL2TableAlpha2                  // T2, 75% (AlphaBlend[2])
	PL2TableAdditive                // table D (AdditiveBlend)
	PL2TableMultiplicative          // table E (MultiplicativeBlend)
	PL2TableG                       // table G (MaxComponentBlend), semantics unresolved
	PL2TableBright170               // computed at load: palette x170%
)

// Mod2XBrightness is the brightening of draw mode 7 (the hovered unit): the
// palette is brightened to 170% (verified in the notes).
const Mod2XBrightness = 1.7

// PL2Table returns the PL2 table of the draw mode.
func (d DrawEffect) PL2Table() PL2Table {
	switch d {
	case DrawEffectPctTransparency25:
		return PL2TableAlpha2
	case DrawEffectPctTransparency50:
		return PL2TableAlpha1
	case DrawEffectPctTransparency75:
		return PL2TableAlpha0
	case DrawEffectModulate:
		return PL2TableAdditive
	case DrawEffectBurn:
		return PL2TableMultiplicative
	case DrawEffectMod2XTrans:
		return PL2TableG
	case DrawEffectMod2X:
		return PL2TableBright170
	default:
		return PL2TableNone
	}
}
