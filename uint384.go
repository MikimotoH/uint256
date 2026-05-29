package uint256

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/shopspring/decimal"
)

var (
	maxBigInt384 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 384), big.NewInt(1))
	decMax384    = decimal.NewFromBigInt(maxBigInt384, 0)
)

// Uint384 is an unsigned 384-bit integer stored as six big-endian uint64 words.
// v[0] holds bits 320-383 (most significant); v[5] holds bits 0-63.
// The struct is comparable and can be used as a map key.
type Uint384 struct {
	v [6]uint64
}

// --- internal helpers ---

func arr384ToBigInt(v [6]uint64) *big.Int {
	var buf [48]byte
	for i := 0; i < 6; i++ {
		binary.BigEndian.PutUint64(buf[i*8:(i+1)*8], v[i])
	}
	return new(big.Int).SetBytes(buf[:])
}

func bigIntToArr384(b *big.Int) [6]uint64 {
	raw := b.Bytes()
	var buf [48]byte
	copy(buf[48-len(raw):], raw)

	var v [6]uint64
	for i := 0; i < 6; i++ {
		v[i] = binary.BigEndian.Uint64(buf[i*8 : (i+1)*8])
	}
	return v
}

func (u Uint384) toDecimal() decimal.Decimal {
	return decimal.NewFromBigInt(arr384ToBigInt(u.v), 0)
}

func validate384(d decimal.Decimal) error {
	if d.Sign() < 0 {
		return fmt.Errorf("uint384: negative value %s", d)
	}
	if d.GreaterThan(decMax384) {
		return fmt.Errorf("uint384: overflow %s", d)
	}
	if !d.Equal(d.Floor()) {
		return fmt.Errorf("uint384: fractional value %s", d)
	}
	return nil
}

func fromDecimal384(d decimal.Decimal) (Uint384, error) {
	if err := validate384(d); err != nil {
		return Uint384{}, err
	}
	b, _ := new(big.Int).SetString(d.String(), 10)
	return Uint384{v: bigIntToArr384(b)}, nil
}

// --- constructors ---

// New384 creates a Uint384 from a uint64.
func New384(val uint64) Uint384 {
	return Uint384{v: [6]uint64{0, 0, 0, 0, 0, val}}
}

// New384FromString parses a decimal integer string.
func New384FromString(s string) (Uint384, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Uint384{}, err
	}
	return fromDecimal384(d)
}

// New384FromBigInt creates a Uint384 from a *big.Int.
func New384FromBigInt(b *big.Int) (Uint384, error) {
	if b.Sign() < 0 {
		return Uint384{}, fmt.Errorf("uint384: negative value")
	}
	if b.Cmp(maxBigInt384) > 0 {
		return Uint384{}, fmt.Errorf("uint384: overflow")
	}
	return Uint384{v: bigIntToArr384(b)}, nil
}

// New384FromWords creates a Uint384 from six 64-bit words interpreted as
// (a<<320) | (b<<256) | (c<<192) | (d<<128) | (e<<64) | f,
// where a is the most significant word.
func New384FromWords(a, b, c, d, e, f uint64) Uint384 {
	return Uint384{v: [6]uint64{a, b, c, d, e, f}}
}

// Zero384 returns the zero value.
func Zero384() Uint384 { return Uint384{} }

// Max384 returns the maximum uint384 value (2^384 - 1).
func Max384() Uint384 {
	return Uint384{v: [6]uint64{
		^uint64(0), ^uint64(0), ^uint64(0),
		^uint64(0), ^uint64(0), ^uint64(0),
	}}
}

// --- arithmetic (via shopspring/decimal) ---

// Add returns u + v, or an error on overflow.
func (u Uint384) Add(v Uint384) (Uint384, error) {
	return fromDecimal384(u.toDecimal().Add(v.toDecimal()))
}

// Sub returns u - v, or an error on underflow.
func (u Uint384) Sub(v Uint384) (Uint384, error) {
	if u.Cmp(v) < 0 {
		return Uint384{}, fmt.Errorf("uint384: underflow")
	}
	b, _ := new(big.Int).SetString(u.toDecimal().Sub(v.toDecimal()).String(), 10)
	return Uint384{v: bigIntToArr384(b)}, nil
}

// Mul returns u * v, or an error on overflow.
func (u Uint384) Mul(v Uint384) (Uint384, error) {
	return fromDecimal384(u.toDecimal().Mul(v.toDecimal()))
}

// Div returns floor(u / v), or an error on division by zero.
func (u Uint384) Div(v Uint384) (Uint384, error) {
	if v.IsZero() {
		return Uint384{}, fmt.Errorf("uint384: division by zero")
	}
	return fromDecimal384(u.toDecimal().Div(v.toDecimal()).Floor())
}

// Mod returns u mod v, or an error on division by zero.
func (u Uint384) Mod(v Uint384) (Uint384, error) {
	if v.IsZero() {
		return Uint384{}, fmt.Errorf("uint384: division by zero")
	}
	return fromDecimal384(u.toDecimal().Mod(v.toDecimal()))
}

// --- comparisons (direct on words, no decimal allocation) ---

// Cmp returns -1, 0, or 1 if u < v, u == v, or u > v.
func (u Uint384) Cmp(w Uint384) int {
	for i := 0; i < 6; i++ {
		if u.v[i] < w.v[i] {
			return -1
		}
		if u.v[i] > w.v[i] {
			return 1
		}
	}
	return 0
}

func (u Uint384) Equal(w Uint384) bool { return u == w }
func (u Uint384) Lt(w Uint384) bool    { return u.Cmp(w) < 0 }
func (u Uint384) Gt(w Uint384) bool    { return u.Cmp(w) > 0 }
func (u Uint384) Lte(w Uint384) bool   { return u.Cmp(w) <= 0 }
func (u Uint384) Gte(w Uint384) bool   { return u.Cmp(w) >= 0 }
func (u Uint384) IsZero() bool         { return u.v == [6]uint64{} }

// --- conversions ---

// String returns the decimal string representation.
func (u Uint384) String() string { return arr384ToBigInt(u.v).String() }

// BigInt returns the value as a *big.Int.
func (u Uint384) BigInt() *big.Int { return arr384ToBigInt(u.v) }

// Bytes48 returns the 384-bit big-endian byte representation.
func (u Uint384) Bytes48() [48]byte {
	var buf [48]byte
	for i := 0; i < 6; i++ {
		binary.BigEndian.PutUint64(buf[i*8:(i+1)*8], u.v[i])
	}
	return buf
}
