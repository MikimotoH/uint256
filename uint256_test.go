package uint256

import (
	"math/big"
	"testing"
)

func TestNew(t *testing.T) {
	u := New(42)
	if u.String() != "42" {
		t.Fatalf("got %s, want 42", u.String())
	}
}

func TestNewFromString(t *testing.T) {
	u, err := NewFromString("115792089237316195423570985008687907853269984665640564039457584007913129639935")
	if err != nil {
		t.Fatal(err)
	}
	if !u.Equal(Max()) {
		t.Fatal("expected max value")
	}

	if _, err := NewFromString("-1"); err == nil {
		t.Fatal("expected error for negative")
	}
	if _, err := NewFromString("115792089237316195423570985008687907853269984665640564039457584007913129639936"); err == nil {
		t.Fatal("expected error for overflow")
	}
	if _, err := NewFromString("1.5"); err == nil {
		t.Fatal("expected error for fractional")
	}
}

func TestNewFromBigInt(t *testing.T) {
	b := new(big.Int).SetUint64(1000)
	u, err := NewFromBigInt(b)
	if err != nil {
		t.Fatal(err)
	}
	if u.String() != "1000" {
		t.Fatalf("got %s", u.String())
	}
}

func TestNewFromWords(t *testing.T) {
	// single word in each position maps to the right bit-shift
	word := func(shift uint) *big.Int {
		return new(big.Int).Lsh(big.NewInt(1), shift)
	}

	cases := []struct {
		a, b, c, d uint64
		want       *big.Int
	}{
		{0, 0, 0, 1, big.NewInt(1)},                          // d only
		{0, 0, 1, 0, word(64)},                               // c << 64
		{0, 1, 0, 0, word(128)},                              // b << 128
		{1, 0, 0, 0, word(192)},                              // a << 192
		{0, 0, 0, 0, big.NewInt(0)},                          // zero
		{^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0), maxBigInt}, // max
		{1, 2, 3, 4, new(big.Int).Or(new(big.Int).Or(new(big.Int).Or(
			new(big.Int).Lsh(big.NewInt(1), 192),
			new(big.Int).Lsh(big.NewInt(2), 128)),
			new(big.Int).Lsh(big.NewInt(3), 64)),
			big.NewInt(4))},
	}

	for _, tc := range cases {
		u := NewFromWords(tc.a, tc.b, tc.c, tc.d)
		if u.BigInt().Cmp(tc.want) != 0 {
			t.Errorf("NewFromWords(%d,%d,%d,%d) = %s, want %s",
				tc.a, tc.b, tc.c, tc.d, u, tc.want)
		}
	}

	// round-trip: BigInt → NewFromBigInt → Bytes32 must equal NewFromWords → Bytes32
	u1 := NewFromWords(0xdeadbeef, 0xcafebabe, 0x12345678, 0xabcdef01)
	u2, _ := NewFromBigInt(u1.BigInt())
	if u1 != u2 {
		t.Fatal("round-trip BigInt mismatch")
	}
}

func TestArithmetic(t *testing.T) {
	a := New(100)
	b := New(42)

	sum, err := a.Add(b)
	if err != nil || sum.String() != "142" {
		t.Fatalf("Add: got %v %v", sum, err)
	}

	diff, err := a.Sub(b)
	if err != nil || diff.String() != "58" {
		t.Fatalf("Sub: got %v %v", diff, err)
	}

	if _, err := b.Sub(a); err == nil {
		t.Fatal("Sub underflow: expected error")
	}

	prod, err := a.Mul(b)
	if err != nil || prod.String() != "4200" {
		t.Fatalf("Mul: got %v %v", prod, err)
	}

	quot, err := a.Div(b)
	if err != nil || quot.String() != "2" {
		t.Fatalf("Div: got %v %v", quot, err)
	}

	rem, err := a.Mod(b)
	if err != nil || rem.String() != "16" {
		t.Fatalf("Mod: got %v %v", rem, err)
	}

	if _, err := a.Div(Zero()); err == nil {
		t.Fatal("Div by zero: expected error")
	}
}

func TestOverflow(t *testing.T) {
	m := Max()
	if _, err := m.Add(New(1)); err == nil {
		t.Fatal("expected overflow error")
	}
	if _, err := m.Mul(New(2)); err == nil {
		t.Fatal("expected overflow error")
	}
}

func TestBytes32(t *testing.T) {
	u := New(256)
	b := u.Bytes32()
	if b[30] != 1 || b[31] != 0 {
		t.Fatalf("unexpected bytes: %v", b)
	}
}

func TestMapKey(t *testing.T) {
	m := map[Uint256]string{}
	a, b := New(1), New(2)
	m[a] = "one"
	m[b] = "two"
	if m[a] != "one" || m[b] != "two" {
		t.Fatal("map key lookup failed")
	}
	a2 := New(1)
	if a != a2 {
		t.Fatal("struct equality failed")
	}
}

func TestCmp(t *testing.T) {
	a, b := New(10), New(20)
	if a.Cmp(b) != -1 || b.Cmp(a) != 1 || a.Cmp(a) != 0 {
		t.Fatal("Cmp wrong")
	}
	if !a.Lt(b) || !b.Gt(a) || !a.Lte(a) || !a.Gte(a) {
		t.Fatal("comparison methods wrong")
	}
}
