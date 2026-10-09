package d2enum

import "testing"

func TestDrawEffectPL2Table(t *testing.T) {
	for effect, want := range map[DrawEffect]PL2Table{
		DrawEffectPctTransparency25: PL2TableAlpha2,
		DrawEffectPctTransparency50: PL2TableAlpha1,
		DrawEffectPctTransparency75: PL2TableAlpha0,
		DrawEffectModulate:          PL2TableAdditive,
		DrawEffectBurn:              PL2TableMultiplicative,
		DrawEffectNormal:            PL2TableNone,
		DrawEffectMod2XTrans:        PL2TableG,
		DrawEffectMod2X:             PL2TableBright170,
		DrawEffectNone:              PL2TableNone,
	} {
		if got := effect.PL2Table(); got != want {
			t.Errorf("%v: table %d want %d", effect, got, want)
		}
	}
}
