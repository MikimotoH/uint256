package uint256

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/shopspring/decimal"
)

var (
	maxBigInt = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	decMax    = decimal.NewFromBigInt(maxBigInt, 0)
)

// Uint256 is an unsigned 256-bit integer stored as four big-endian uint64 words.
// v[0] holds bits 192–255 (most significant); v[3] holds bits 0–63.
// The struct is comparable and can be used as a map key.
type Uint256 struct {
	v [4]uint64
}

// --- internal helpers ---

func arrToBigInt(v [4]uint64) *big.Int {
	var buf [32]byte
	binary.BigEndian.PutUint64(buf[0:8], v[0])
	binary.BigEndian.PutUint64(buf[8:16], v[1])
	binary.BigEndian.PutUint64(buf[16:24], v[2])
	binary.BigEndian.PutUint64(buf[24:32], v[3])
	return new(big.Int).SetBytes(buf[:])
}

func bigIntToArr(b *big.Int) [4]uint64 {
	raw := b.Bytes()
	var buf [32]byte
	copy(buf[32-len(raw):], raw)
	return [4]uint64{
		binary.BigEndian.Uint64(buf[0:8]),
		binary.BigEndian.Uint64(buf[8:16]),
		binary.BigEndian.Uint64(buf[16:24]),
		binary.BigEndian.Uint64(buf[24:32]),
	}
}

func (u Uint256) toDecimal() decimal.Decimal {
	return decimal.NewFromBigInt(arrToBigInt(u.v), 0)
}

func validate(d decimal.Decimal) error {
	if d.Sign() < 0 {
		return fmt.Errorf("uint256: negative value %s", d)
	}
	if d.GreaterThan(decMax) {
		return fmt.Errorf("uint256: overflow %s", d)
	}
	if !d.Equal(d.Floor()) {
		return fmt.Errorf("uint256: fractional value %s", d)
	}
	return nil
}

func fromDecimal(d decimal.Decimal) (Uint256, error) {
	if err := validate(d); err != nil {
		return Uint256{}, err
	}
	b, _ := new(big.Int).SetString(d.String(), 10)
	return Uint256{v: bigIntToArr(b)}, nil
}

// --- constructors ---

// New creates a Uint256 from a uint64.
func New(val uint64) Uint256 {
	return Uint256{v: [4]uint64{0, 0, 0, val}}
}

// NewFromString parses a decimal integer string.
func NewFromString(s string) (Uint256, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Uint256{}, err
	}
	return fromDecimal(d)
}

// NewFromBigInt creates a Uint256 from a *big.Int.
func NewFromBigInt(b *big.Int) (Uint256, error) {
	if b.Sign() < 0 {
		return Uint256{}, fmt.Errorf("uint256: negative value")
	}
	if b.Cmp(maxBigInt) > 0 {
		return Uint256{}, fmt.Errorf("uint256: overflow")
	}
	return Uint256{v: bigIntToArr(b)}, nil
}

// NewFromWords creates a Uint256 from four 64-bit words interpreted as
// (a<<192) | (b<<128) | (c<<64) | d, where a is the most significant word.
func NewFromWords(a, b, c, d uint64) Uint256 {
	return Uint256{v: [4]uint64{a, b, c, d}}
}

// Zero returns the zero value.
func Zero() Uint256 { return Uint256{} }

// Max returns the maximum uint256 value (2^256 - 1).
func Max() Uint256 {
	return Uint256{v: [4]uint64{^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0)}}
}

// --- arithmetic (via shopspring/decimal) ---

// Add returns u + v, or an error on overflow.
func (u Uint256) Add(v Uint256) (Uint256, error) {
	return fromDecimal(u.toDecimal().Add(v.toDecimal()))
}

// Sub returns u - v, or an error on underflow.
func (u Uint256) Sub(v Uint256) (Uint256, error) {
	if u.Cmp(v) < 0 {
		return Uint256{}, fmt.Errorf("uint256: underflow")
	}
	b, _ := new(big.Int).SetString(u.toDecimal().Sub(v.toDecimal()).String(), 10)
	return Uint256{v: bigIntToArr(b)}, nil
}

// Mul returns u * v, or an error on overflow.
func (u Uint256) Mul(v Uint256) (Uint256, error) {
	return fromDecimal(u.toDecimal().Mul(v.toDecimal()))
}

// Div returns floor(u / v), or an error on division by zero.
func (u Uint256) Div(v Uint256) (Uint256, error) {
	if v.IsZero() {
		return Uint256{}, fmt.Errorf("uint256: division by zero")
	}
	return fromDecimal(u.toDecimal().Div(v.toDecimal()).Floor())
}

// Mod returns u mod v, or an error on division by zero.
func (u Uint256) Mod(v Uint256) (Uint256, error) {
	if v.IsZero() {
		return Uint256{}, fmt.Errorf("uint256: division by zero")
	}
	return fromDecimal(u.toDecimal().Mod(v.toDecimal()))
}

// --- comparisons (direct on words, no decimal allocation) ---

// Cmp returns -1, 0, or 1 if u < v, u == v, or u > v.
func (u Uint256) Cmp(w Uint256) int {
	for i := 0; i < 4; i++ {
		if u.v[i] < w.v[i] {
			return -1
		}
		if u.v[i] > w.v[i] {
			return 1
		}
	}
	return 0
}

func (u Uint256) Equal(w Uint256) bool { return u == w }
func (u Uint256) Lt(w Uint256) bool    { return u.Cmp(w) < 0 }
func (u Uint256) Gt(w Uint256) bool    { return u.Cmp(w) > 0 }
func (u Uint256) Lte(w Uint256) bool   { return u.Cmp(w) <= 0 }
func (u Uint256) Gte(w Uint256) bool   { return u.Cmp(w) >= 0 }
func (u Uint256) IsZero() bool         { return u.v == [4]uint64{} }

// --- conversions ---

// String returns the decimal string representation.
func (u Uint256) String() string { return arrToBigInt(u.v).String() }

// BigInt returns the value as a *big.Int.
func (u Uint256) BigInt() *big.Int { return arrToBigInt(u.v) }

// Bytes32 returns the 256-bit big-endian byte representation.
func (u Uint256) Bytes32() [32]byte {
	var buf [32]byte
	binary.BigEndian.PutUint64(buf[0:8], u.v[0])
	binary.BigEndian.PutUint64(buf[8:16], u.v[1])
	binary.BigEndian.PutUint64(buf[16:24], u.v[2])
	binary.BigEndian.PutUint64(buf[24:32], u.v[3])
	return buf
}
