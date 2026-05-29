package uint256

import (
	"math/big"
	"testing"
)

func TestNew384(t *testing.T) {
	u := New384(42)
	if u.String() != "42" {
		t.Fatalf("got %s, want 42", u.String())
	}
}

func TestNew384FromString(t *testing.T) {
	u, err := New384FromString(maxBigInt384.String())
	if err != nil {
		t.Fatal(err)
	}
	if !u.Equal(Max384()) {
		t.Fatal("expected max value")
	}

	if _, err := New384FromString("-1"); err == nil {
		t.Fatal("expected error for negative")
	}

	overflow := new(big.Int).Add(maxBigInt384, big.NewInt(1)).String()
	if _, err := New384FromString(overflow); err == nil {
		t.Fatal("expected error for overflow")
	}
	if _, err := New384FromString("1.5"); err == nil {
		t.Fatal("expected error for fractional")
	}
}

func TestNew384FromBigInt(t *testing.T) {
	b := new(big.Int).SetUint64(1000)
	u, err := New384FromBigInt(b)
	if err != nil {
		t.Fatal(err)
	}
	if u.String() != "1000" {
		t.Fatalf("got %s", u.String())
	}
}

func TestNew384FromWords(t *testing.T) {
	word := func(shift uint) *big.Int {
		return new(big.Int).Lsh(big.NewInt(1), shift)
	}

	cases := []struct {
		words [6]uint64
		want  *big.Int
	}{
		{[6]uint64{0, 0, 0, 0, 0, 1}, big.NewInt(1)},
		{[6]uint64{0, 0, 0, 0, 1, 0}, word(64)},
		{[6]uint64{0, 0, 0, 1, 0, 0}, word(128)},
		{[6]uint64{0, 0, 1, 0, 0, 0}, word(192)},
		{[6]uint64{0, 1, 0, 0, 0, 0}, word(256)},
		{[6]uint64{1, 0, 0, 0, 0, 0}, word(320)},
		{[6]uint64{}, big.NewInt(0)},
		{[6]uint64{
			^uint64(0), ^uint64(0), ^uint64(0),
			^uint64(0), ^uint64(0), ^uint64(0),
		}, maxBigInt384},
	}

	for _, tc := range cases {
		u := New384FromWords(
			tc.words[0], tc.words[1], tc.words[2],
			tc.words[3], tc.words[4], tc.words[5],
		)
		if u.BigInt().Cmp(tc.want) != 0 {
			t.Errorf("New384FromWords(%v) = %s, want %s", tc.words, u, tc.want)
		}
	}

	u1 := New384FromWords(0xdeadbeef, 0xcafebabe, 0x12345678, 0xabcdef01, 1, 2)
	u2, _ := New384FromBigInt(u1.BigInt())
	if u1 != u2 {
		t.Fatal("round-trip BigInt mismatch")
	}
}

func TestUint384Arithmetic(t *testing.T) {
	a := New384(100)
	b := New384(42)

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

	if _, err := a.Div(Zero384()); err == nil {
		t.Fatal("Div by zero: expected error")
	}
}

func TestUint384Overflow(t *testing.T) {
	m := Max384()
	if _, err := m.Add(New384(1)); err == nil {
		t.Fatal("expected overflow error")
	}
	if _, err := m.Mul(New384(2)); err == nil {
		t.Fatal("expected overflow error")
	}
}

func TestBytes48(t *testing.T) {
	u := New384(256)
	b := u.Bytes48()
	if b[46] != 1 || b[47] != 0 {
		t.Fatalf("unexpected bytes: %v", b)
	}
}

func TestUint384MapKey(t *testing.T) {
	m := map[Uint384]string{}
	a, b := New384(1), New384(2)
	m[a] = "one"
	m[b] = "two"
	if m[a] != "one" || m[b] != "two" {
		t.Fatal("map key lookup failed")
	}
	a2 := New384(1)
	if a != a2 {
		t.Fatal("struct equality failed")
	}
}

func TestUint384Cmp(t *testing.T) {
	a, b := New384(10), New384(20)
	if a.Cmp(b) != -1 || b.Cmp(a) != 1 || a.Cmp(a) != 0 {
		t.Fatal("Cmp wrong")
	}
	if !a.Lt(b) || !b.Gt(a) || !a.Lte(a) || !a.Gte(a) {
		t.Fatal("comparison methods wrong")
	}
}
