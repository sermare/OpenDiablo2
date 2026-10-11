package d2monsters

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"

// zeroCalcEnv evaluates calc columns that hold plain numbers.
type zeroCalcEnv struct{ d2calc.ZeroEnv }
