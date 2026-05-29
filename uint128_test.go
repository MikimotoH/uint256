package uint256

import (
	"math/big"
	"testing"
)

func TestNew128(t *testing.T) {
	u := New128(42)
	if u.String() != "42" {
		t.Fatalf("got %s, want 42", u.String())
	}
}

func TestNew128FromString(t *testing.T) {
	u, err := New128FromString("340282366920938463463374607431768211455")
	if err != nil {
		t.Fatal(err)
	}
	if !u.Equal(Max128()) {
		t.Fatal("expected max value")
	}

	if _, err := New128FromString("-1"); err == nil {
		t.Fatal("expected error for negative")
	}
	if _, err := New128FromString("340282366920938463463374607431768211456"); err == nil {
		t.Fatal("expected error for overflow")
	}
	if _, err := New128FromString("1.5"); err == nil {
		t.Fatal("expected error for fractional")
	}
}

func TestNew128FromBigInt(t *testing.T) {
	b := new(big.Int).SetUint64(1000)
	u, err := New128FromBigInt(b)
	if err != nil {
		t.Fatal(err)
	}
	if u.String() != "1000" {
		t.Fatalf("got %s", u.String())
	}
}

func TestNew128FromWords(t *testing.T) {
	word := func(shift uint) *big.Int {
		return new(big.Int).Lsh(big.NewInt(1), shift)
	}

	cases := []struct {
		a, b uint64
		want *big.Int
	}{
		{0, 1, big.NewInt(1)},
		{1, 0, word(64)},
		{0, 0, big.NewInt(0)},
		{^uint64(0), ^uint64(0), maxBigInt128},
		{1, 2, new(big.Int).Or(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(2))},
	}

	for _, tc := range cases {
		u := New128FromWords(tc.a, tc.b)
		if u.BigInt().Cmp(tc.want) != 0 {
			t.Errorf("New128FromWords(%d,%d) = %s, want %s",
				tc.a, tc.b, u, tc.want)
		}
	}

	u1 := New128FromWords(0xdeadbeef, 0xcafebabe)
	u2, _ := New128FromBigInt(u1.BigInt())
	if u1 != u2 {
		t.Fatal("round-trip BigInt mismatch")
	}
}

func TestUint128Arithmetic(t *testing.T) {
	a := New128(100)
	b := New128(42)

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

	if _, err := a.Div(Zero128()); err == nil {
		t.Fatal("Div by zero: expected error")
	}
}

func TestUint128Overflow(t *testing.T) {
	m := Max128()
	if _, err := m.Add(New128(1)); err == nil {
		t.Fatal("expected overflow error")
	}
	if _, err := m.Mul(New128(2)); err == nil {
		t.Fatal("expected overflow error")
	}
}

func TestBytes16(t *testing.T) {
	u := New128(256)
	b := u.Bytes16()
	if b[14] != 1 || b[15] != 0 {
		t.Fatalf("unexpected bytes: %v", b)
	}
}

func TestUint128MapKey(t *testing.T) {
	m := map[Uint128]string{}
	a, b := New128(1), New128(2)
	m[a] = "one"
	m[b] = "two"
	if m[a] != "one" || m[b] != "two" {
		t.Fatal("map key lookup failed")
	}
	a2 := New128(1)
	if a != a2 {
		t.Fatal("struct equality failed")
	}
}

func TestUint128Cmp(t *testing.T) {
	a, b := New128(10), New128(20)
	if a.Cmp(b) != -1 || b.Cmp(a) != 1 || a.Cmp(a) != 0 {
		t.Fatal("Cmp wrong")
	}
	if !a.Lt(b) || !b.Gt(a) || !a.Lte(a) || !a.Gte(a) {
		t.Fatal("comparison methods wrong")
	}
}
