package d2combat

// MulDiv is the game's fixed point helper at 0x47f2c0 (a*b/c, truncating
// toward zero, not rounding like the Win32 MulDiv). Verified against the
// emulated exe. For huge operands the exe switches algorithm to avoid 32-bit
// overflow; those branches are reproduced (a > 0x100000 or b > 0x10000).
// c == 0 gives 0.
func MulDiv(a, b, c int32) int32 {
	if c == 0 {
		return 0
	}

	switch {
	case a > 0x100000:
		if c > a>>4 {
			return int32(int64(a) * int64(b) / int64(c))
		}

		return (a / c) * b
	case b > 0x10000:
		if c > b>>4 {
			return int32(int64(a) * int64(b) / int64(c))
		}

		return (b / c) * a
	}

	return (a * b) / c // 32-bit wraparound in the product is the exe's behaviour
}

// AbsorbCap is the maximum absorb percent (0x28), verified in 0x579c20.
const AbsorbCap = 40

// ResolveComponent applies flat reduction, resist percent and absorb to one
// damage component in 8.8 fixed point, exactly as COMBAT_ApplyResistToDamageType
// (0x579c90) does after the effective resist was computed. Returns the new
// component and the life healed by absorb. Verified against the emulated exe
// (golden dm_res).
//
//	dmg < 1                       -> 0 (no heal)
//	ignore (struct flag vs class) -> res = min(res,0)... positive resist -> 0, flat and absorb skipped
//	else                          -> dmg -= flat (NOT floored: may go negative)
//	dmg > 0 && res != 0           -> res = min(res,100); dmg = MulDiv(dmg, 100-res, 100)
//	absorb (type has an absorb stat, not ignore):
//	  pct > 0 -> pct = min(pct,40); x = MulDiv(dmg,pct,100); heal += x; dmg -= x
//	  flat*256 > 0 -> f = min(flat*256, dmg); heal += f; dmg -= f   (dmg may be negative)
func ResolveComponent(dmg, flat int32, res int, ignore, hasAbsorb bool, absorbPct, absorbFlat int32) (out, heal int32) {
	if dmg < 1 {
		return 0, 0
	}

	if ignore {
		if res > 0 {
			res = 0
		}
	} else {
		dmg -= flat
	}

	if dmg > 0 && res != 0 {
		if res >= 100 {
			res = 100
		}

		dmg = MulDiv(dmg, int32(100-res), 100)
	}

	if ignore || !hasAbsorb {
		return dmg, 0
	}

	if absorbPct > 0 {
		if absorbPct > AbsorbCap {
			absorbPct = AbsorbCap
		}

		x := MulDiv(dmg, absorbPct, 100)
		heal += x
		dmg -= x
	}

	if f := absorbFlat << 8; f > 0 {
		if f > dmg {
			f = dmg
		}

		heal += f
		dmg -= f
	}

	return dmg, heal
}
