package uint256

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/shopspring/decimal"
)

var (
	maxBigInt128 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
	decMax128    = decimal.NewFromBigInt(maxBigInt128, 0)
)

// Uint128 is an unsigned 128-bit integer stored as two big-endian uint64 words.
// v[0] holds bits 64-127 (most significant); v[1] holds bits 0-63.
// The struct is comparable and can be used as a map key.
type Uint128 struct {
	v [2]uint64
}

// --- internal helpers ---

func arr128ToBigInt(v [2]uint64) *big.Int {
	var buf [16]byte
	binary.BigEndian.PutUint64(buf[0:8], v[0])
	binary.BigEndian.PutUint64(buf[8:16], v[1])
	return new(big.Int).SetBytes(buf[:])
}

func bigIntToArr128(b *big.Int) [2]uint64 {
	raw := b.Bytes()
	var buf [16]byte
	copy(buf[16-len(raw):], raw)
	return [2]uint64{
		binary.BigEndian.Uint64(buf[0:8]),
		binary.BigEndian.Uint64(buf[8:16]),
	}
}

func (u Uint128) toDecimal() decimal.Decimal {
	return decimal.NewFromBigInt(arr128ToBigInt(u.v), 0)
}

func validate128(d decimal.Decimal) error {
	if d.Sign() < 0 {
		return fmt.Errorf("uint128: negative value %s", d)
	}
	if d.GreaterThan(decMax128) {
		return fmt.Errorf("uint128: overflow %s", d)
	}
	if !d.Equal(d.Floor()) {
		return fmt.Errorf("uint128: fractional value %s", d)
	}
	return nil
}

func fromDecimal128(d decimal.Decimal) (Uint128, error) {
	if err := validate128(d); err != nil {
		return Uint128{}, err
	}
	b, _ := new(big.Int).SetString(d.String(), 10)
	return Uint128{v: bigIntToArr128(b)}, nil
}

// --- constructors ---

// New128 creates a Uint128 from a uint64.
func New128(val uint64) Uint128 {
	return Uint128{v: [2]uint64{0, val}}
}go generate ./...

// New128FromString parses a decimal integer string.
func New128FromString(s string) (Uint128, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Uint128{}, err
	}
	return fromDecimal128(d)
}

// New128FromBigInt creates a Uint128 from a *big.Int.
func New128FromBigInt(b *big.Int) (Uint128, error) {
	if b.Sign() < 0 {
		return Uint128{}, fmt.Errorf("uint128: negative value")
	}
	if b.Cmp(maxBigInt128) > 0 {
		return Uint128{}, fmt.Errorf("uint128: overflow")
	}
	return Uint128{v: bigIntToArr128(b)}, nil
}

// New128FromWords creates a Uint128 from two 64-bit words interpreted as
// (a<<64) | b, where a is the most significant word.
func New128FromWords(a, b uint64) Uint128 {
	return Uint128{v: [2]uint64{a, b}}
}

// Zero128 returns the zero value.
func Zero128() Uint128 { return Uint128{} }

// Max128 returns the maximum uint128 value (2^128 - 1).
func Max128() Uint128 {
	return Uint128{v: [2]uint64{^uint64(0), ^uint64(0)}}
}

// --- arithmetic (via shopspring/decimal) ---

// Add returns u + v, or an error on overflow.
func (u Uint128) Add(v Uint128) (Uint128, error) {
	return fromDecimal128(u.toDecimal().Add(v.toDecimal()))
}

// Sub returns u - v, or an error on underflow.
func (u Uint128) Sub(v Uint128) (Uint128, error) {
	if u.Cmp(v) < 0 {
		return Uint128{}, fmt.Errorf("uint128: underflow")
	}
	b, _ := new(big.Int).SetString(u.toDecimal().Sub(v.toDecimal()).String(), 10)
	return Uint128{v: bigIntToArr128(b)}, nil
}

// Mul returns u * v, or an error on overflow.
func (u Uint128) Mul(v Uint128) (Uint128, error) {
	return fromDecimal128(u.toDecimal().Mul(v.toDecimal()))
}

// Div returns floor(u / v), or an error on division by zero.
func (u Uint128) Div(v Uint128) (Uint128, error) {
	if v.IsZero() {
		return Uint128{}, fmt.Errorf("uint128: division by zero")
	}
	return fromDecimal128(u.toDecimal().Div(v.toDecimal()).Floor())
}

// Mod returns u mod v, or an error on division by zero.
func (u Uint128) Mod(v Uint128) (Uint128, error) {
	if v.IsZero() {
		return Uint128{}, fmt.Errorf("uint128: division by zero")
	}
	return fromDecimal128(u.toDecimal().Mod(v.toDecimal()))
}

// --- comparisons (direct on words, no decimal allocation) ---

// Cmp returns -1, 0, or 1 if u < v, u == v, or u > v.
func (u Uint128) Cmp(w Uint128) int {
	for i := 0; i < 2; i++ {
		if u.v[i] < w.v[i] {
			return -1
		}
		if u.v[i] > w.v[i] {
			return 1
		}
	}
	return 0
}

func (u Uint128) Equal(w Uint128) bool { return u == w }
func (u Uint128) Lt(w Uint128) bool    { return u.Cmp(w) < 0 }
func (u Uint128) Gt(w Uint128) bool    { return u.Cmp(w) > 0 }
func (u Uint128) Lte(w Uint128) bool   { return u.Cmp(w) <= 0 }
func (u Uint128) Gte(w Uint128) bool   { return u.Cmp(w) >= 0 }
func (u Uint128) IsZero() bool         { return u.v == [2]uint64{} }

// --- conversions ---

// String returns the decimal string representation.
func (u Uint128) String() string { return arr128ToBigInt(u.v).String() }

// BigInt returns the value as a *big.Int.
func (u Uint128) BigInt() *big.Int { return arr128ToBigInt(u.v) }

// Bytes16 returns the 128-bit big-endian byte representation.
func (u Uint128) Bytes16() [16]byte {
	var buf [16]byte
	binary.BigEndian.PutUint64(buf[0:8], u.v[0])
	binary.BigEndian.PutUint64(buf[8:16], u.v[1])
	return buf
}
