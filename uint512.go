package uint256

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/shopspring/decimal"
)

var (
	maxBigInt512 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 512), big.NewInt(1))
	decMax512    = decimal.NewFromBigInt(maxBigInt512, 0)
)

// Uint512 is an unsigned 512-bit integer stored as eight big-endian uint64 words.
// v[0] holds bits 448-511 (most significant); v[7] holds bits 0-63.
// The struct is comparable and can be used as a map key.
type Uint512 struct {
	v [8]uint64
}

// --- internal helpers ---

func arr512ToBigInt(v [8]uint64) *big.Int {
	var buf [64]byte
	for i := 0; i < 8; i++ {
		binary.BigEndian.PutUint64(buf[i*8:(i+1)*8], v[i])
	}
	return new(big.Int).SetBytes(buf[:])
}

func bigIntToArr512(b *big.Int) [8]uint64 {
	raw := b.Bytes()
	var buf [64]byte
	copy(buf[64-len(raw):], raw)

	var v [8]uint64
	for i := 0; i < 8; i++ {
		v[i] = binary.BigEndian.Uint64(buf[i*8 : (i+1)*8])
	}
	return v
}

func (u Uint512) toDecimal() decimal.Decimal {
	return decimal.NewFromBigInt(arr512ToBigInt(u.v), 0)
}

func validate512(d decimal.Decimal) error {
	if d.Sign() < 0 {
		return fmt.Errorf("uint512: negative value %s", d)
	}
	if d.GreaterThan(decMax512) {
		return fmt.Errorf("uint512: overflow %s", d)
	}
	if !d.Equal(d.Floor()) {
		return fmt.Errorf("uint512: fractional value %s", d)
	}
	return nil
}

func fromDecimal512(d decimal.Decimal) (Uint512, error) {
	if err := validate512(d); err != nil {
		return Uint512{}, err
	}
	b, _ := new(big.Int).SetString(d.String(), 10)
	return Uint512{v: bigIntToArr512(b)}, nil
}

// --- constructors ---

// New512 creates a Uint512 from a uint64.
func New512(val uint64) Uint512 {
	return Uint512{v: [8]uint64{0, 0, 0, 0, 0, 0, 0, val}}
}

// New512FromString parses a decimal integer string.
func New512FromString(s string) (Uint512, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Uint512{}, err
	}
	return fromDecimal512(d)
}

// New512FromBigInt creates a Uint512 from a *big.Int.
func New512FromBigInt(b *big.Int) (Uint512, error) {
	if b.Sign() < 0 {
		return Uint512{}, fmt.Errorf("uint512: negative value")
	}
	if b.Cmp(maxBigInt512) > 0 {
		return Uint512{}, fmt.Errorf("uint512: overflow")
	}
	return Uint512{v: bigIntToArr512(b)}, nil
}

// New512FromWords creates a Uint512 from eight 64-bit words interpreted as
// (a<<448) | (b<<384) | (c<<320) | (d<<256) | (e<<192) | (f<<128) | (g<<64) | h,
// where a is the most significant word.
func New512FromWords(a, b, c, d, e, f, g, h uint64) Uint512 {
	return Uint512{v: [8]uint64{a, b, c, d, e, f, g, h}}
}

// Zero512 returns the zero value.
func Zero512() Uint512 { return Uint512{} }

// Max512 returns the maximum uint512 value (2^512 - 1).
func Max512() Uint512 {
	return Uint512{v: [8]uint64{
		^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0),
		^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0),
	}}
}

// --- arithmetic (via shopspring/decimal) ---

// Add returns u + v, or an error on overflow.
func (u Uint512) Add(v Uint512) (Uint512, error) {
	return fromDecimal512(u.toDecimal().Add(v.toDecimal()))
}

// Sub returns u - v, or an error on underflow.
func (u Uint512) Sub(v Uint512) (Uint512, error) {
	if u.Cmp(v) < 0 {
		return Uint512{}, fmt.Errorf("uint512: underflow")
	}
	b, _ := new(big.Int).SetString(u.toDecimal().Sub(v.toDecimal()).String(), 10)
	return Uint512{v: bigIntToArr512(b)}, nil
}

// Mul returns u * v, or an error on overflow.
func (u Uint512) Mul(v Uint512) (Uint512, error) {
	return fromDecimal512(u.toDecimal().Mul(v.toDecimal()))
}

// Div returns floor(u / v), or an error on division by zero.
func (u Uint512) Div(v Uint512) (Uint512, error) {
	if v.IsZero() {
		return Uint512{}, fmt.Errorf("uint512: division by zero")
	}
	return fromDecimal512(u.toDecimal().Div(v.toDecimal()).Floor())
}

// Mod returns u mod v, or an error on division by zero.
func (u Uint512) Mod(v Uint512) (Uint512, error) {
	if v.IsZero() {
		return Uint512{}, fmt.Errorf("uint512: division by zero")
	}
	return fromDecimal512(u.toDecimal().Mod(v.toDecimal()))
}

// --- comparisons (direct on words, no decimal allocation) ---

// Cmp returns -1, 0, or 1 if u < v, u == v, or u > v.
func (u Uint512) Cmp(w Uint512) int {
	for i := 0; i < 8; i++ {
		if u.v[i] < w.v[i] {
			return -1
		}
		if u.v[i] > w.v[i] {
			return 1
		}
	}
	return 0
}

func (u Uint512) Equal(w Uint512) bool { return u == w }
func (u Uint512) Lt(w Uint512) bool    { return u.Cmp(w) < 0 }
func (u Uint512) Gt(w Uint512) bool    { return u.Cmp(w) > 0 }
func (u Uint512) Lte(w Uint512) bool   { return u.Cmp(w) <= 0 }
func (u Uint512) Gte(w Uint512) bool   { return u.Cmp(w) >= 0 }
func (u Uint512) IsZero() bool         { return u.v == [8]uint64{} }

// --- conversions ---

// String returns the decimal string representation.
func (u Uint512) String() string { return arr512ToBigInt(u.v).String() }

// BigInt returns the value as a *big.Int.
func (u Uint512) BigInt() *big.Int { return arr512ToBigInt(u.v) }

// Bytes64 returns the 512-bit big-endian byte representation.
func (u Uint512) Bytes64() [64]byte {
	var buf [64]byte
	for i := 0; i < 8; i++ {
		binary.BigEndian.PutUint64(buf[i*8:(i+1)*8], u.v[i])
	}
	return buf
}
