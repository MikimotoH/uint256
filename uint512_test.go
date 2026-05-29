package uint256

import (
	"math/big"
	"testing"
)

func TestNew512(t *testing.T) {
	u := New512(42)
	if u.String() != "42" {
		t.Fatalf("got %s, want 42", u.String())
	}
}

func TestNew512FromString(t *testing.T) {
	u, err := New512FromString(maxBigInt512.String())
	if err != nil {
		t.Fatal(err)
	}
	if !u.Equal(Max512()) {
		t.Fatal("expected max value")
	}

	if _, err := New512FromString("-1"); err == nil {
		t.Fatal("expected error for negative")
	}

	overflow := new(big.Int).Add(maxBigInt512, big.NewInt(1)).String()
	if _, err := New512FromString(overflow); err == nil {
		t.Fatal("expected error for overflow")
	}
	if _, err := New512FromString("1.5"); err == nil {
		t.Fatal("expected error for fractional")
	}
}

func TestNew512FromBigInt(t *testing.T) {
	b := new(big.Int).SetUint64(1000)
	u, err := New512FromBigInt(b)
	if err != nil {
		t.Fatal(err)
	}
	if u.String() != "1000" {
		t.Fatalf("got %s", u.String())
	}
}

func TestNew512FromWords(t *testing.T) {
	word := func(shift uint) *big.Int {
		return new(big.Int).Lsh(big.NewInt(1), shift)
	}

	cases := []struct {
		words [8]uint64
		want  *big.Int
	}{
		{[8]uint64{0, 0, 0, 0, 0, 0, 0, 1}, big.NewInt(1)},
		{[8]uint64{0, 0, 0, 0, 0, 0, 1, 0}, word(64)},
		{[8]uint64{0, 0, 0, 0, 0, 1, 0, 0}, word(128)},
		{[8]uint64{0, 0, 0, 0, 1, 0, 0, 0}, word(192)},
		{[8]uint64{0, 0, 0, 1, 0, 0, 0, 0}, word(256)},
		{[8]uint64{0, 0, 1, 0, 0, 0, 0, 0}, word(320)},
		{[8]uint64{0, 1, 0, 0, 0, 0, 0, 0}, word(384)},
		{[8]uint64{1, 0, 0, 0, 0, 0, 0, 0}, word(448)},
		{[8]uint64{}, big.NewInt(0)},
		{[8]uint64{
			^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0),
			^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0),
		}, maxBigInt512},
	}

	for _, tc := range cases {
		u := New512FromWords(
			tc.words[0], tc.words[1], tc.words[2], tc.words[3],
			tc.words[4], tc.words[5], tc.words[6], tc.words[7],
		)
		if u.BigInt().Cmp(tc.want) != 0 {
			t.Errorf("New512FromWords(%v) = %s, want %s", tc.words, u, tc.want)
		}
	}

	u1 := New512FromWords(0xdeadbeef, 0xcafebabe, 0x12345678, 0xabcdef01, 1, 2, 3, 4)
	u2, _ := New512FromBigInt(u1.BigInt())
	if u1 != u2 {
		t.Fatal("round-trip BigInt mismatch")
	}
}

func TestUint512Arithmetic(t *testing.T) {
	a := New512(100)
	b := New512(42)

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

	if _, err := a.Div(Zero512()); err == nil {
		t.Fatal("Div by zero: expected error")
	}
}

func TestUint512Overflow(t *testing.T) {
	m := Max512()
	if _, err := m.Add(New512(1)); err == nil {
		t.Fatal("expected overflow error")
	}
	if _, err := m.Mul(New512(2)); err == nil {
		t.Fatal("expected overflow error")
	}
}

func TestBytes64(t *testing.T) {
	u := New512(256)
	b := u.Bytes64()
	if b[62] != 1 || b[63] != 0 {
		t.Fatalf("unexpected bytes: %v", b)
	}
}

func TestUint512MapKey(t *testing.T) {
	m := map[Uint512]string{}
	a, b := New512(1), New512(2)
	m[a] = "one"
	m[b] = "two"
	if m[a] != "one" || m[b] != "two" {
		t.Fatal("map key lookup failed")
	}
	a2 := New512(1)
	if a != a2 {
		t.Fatal("struct equality failed")
	}
}

func TestUint512Cmp(t *testing.T) {
	a, b := New512(10), New512(20)
	if a.Cmp(b) != -1 || b.Cmp(a) != 1 || a.Cmp(a) != 0 {
		t.Fatal("Cmp wrong")
	}
	if !a.Lt(b) || !b.Gt(a) || !a.Lte(a) || !a.Gte(a) {
		t.Fatal("comparison methods wrong")
	}
}
