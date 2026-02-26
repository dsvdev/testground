package faker_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/dsvdev/testground/faker"
)

// uuidPattern matches the canonical UUID v4 format and validates version/variant bits.
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestRandomInt_InRange(t *testing.T) {
	for range 1000 {
		v := faker.RandomInt(10, 20)
		if v < 10 || v > 20 {
			t.Fatalf("RandomInt(10, 20) = %d, want [10, 20]", v)
		}
	}
}

func TestRandomInt_Equal(t *testing.T) {
	v := faker.RandomInt(7, 7)
	if v != 7 {
		t.Fatalf("RandomInt(7, 7) = %d, want 7", v)
	}
}

func TestRandomInt_PanicOnInvalidRange(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("RandomInt(5, 1) should panic")
		}
	}()
	faker.RandomInt(5, 1)
}

func TestRandomInt64_InRange(t *testing.T) {
	for range 1000 {
		v := faker.RandomInt64(100, 200)
		if v < 100 || v > 200 {
			t.Fatalf("RandomInt64(100, 200) = %d, want [100, 200]", v)
		}
	}
}

func TestRandomInt64_Equal(t *testing.T) {
	v := faker.RandomInt64(42, 42)
	if v != 42 {
		t.Fatalf("RandomInt64(42, 42) = %d, want 42", v)
	}
}

func TestRandomInt64_PanicOnInvalidRange(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("RandomInt64(10, 5) should panic")
		}
	}()
	faker.RandomInt64(10, 5)
}

func TestRandomString_Length(t *testing.T) {
	for _, length := range []int{0, 1, 5, 16, 100} {
		s := faker.RandomString(length)
		if len(s) != length {
			t.Fatalf("RandomString(%d): got len %d", length, len(s))
		}
	}
}

func TestRandomString_Charset(t *testing.T) {
	s := faker.RandomString(500)
	for _, ch := range s {
		if ch < 'a' || ch > 'z' {
			t.Fatalf("RandomString contains invalid character %q", ch)
		}
	}
}

func TestRandomString_EmptyOnZero(t *testing.T) {
	s := faker.RandomString(0)
	if s != "" {
		t.Fatalf("RandomString(0) = %q, want empty string", s)
	}
}

func TestRandomUUID_Format(t *testing.T) {
	for range 100 {
		u := faker.RandomUUID()
		if !uuidPattern.MatchString(u) {
			t.Fatalf("RandomUUID() = %q is not a valid UUID v4", u)
		}
	}
}

func TestRandomUUID_Version4(t *testing.T) {
	u := faker.RandomUUID()
	parts := strings.Split(u, "-")
	if len(parts) != 5 {
		t.Fatalf("RandomUUID() has wrong number of parts: %d", len(parts))
	}
	if parts[2][0] != '4' {
		t.Fatalf("RandomUUID() version nibble = %c, want 4", parts[2][0])
	}
}

func TestRandomUUID_Variant10xx(t *testing.T) {
	for range 1000 {
		u := faker.RandomUUID()
		parts := strings.Split(u, "-")
		variantChar := parts[3][0]
		if variantChar != '8' && variantChar != '9' && variantChar != 'a' && variantChar != 'b' {
			t.Fatalf("RandomUUID() variant nibble = %c, want one of [89ab]", variantChar)
		}
	}
}

func TestRandomInt_Uniqueness(t *testing.T) {
	a := faker.RandomInt(0, 1<<30)
	b := faker.RandomInt(0, 1<<30)
	if a == b {
		t.Log("RandomInt returned equal values (rare but possible); retrying once")
		b = faker.RandomInt(0, 1<<30)
		if a == b {
			t.Fatal("RandomInt returned the same value three times; generator may be broken")
		}
	}
}

func TestRandomString_Uniqueness(t *testing.T) {
	a := faker.RandomString(20)
	b := faker.RandomString(20)
	if a == b {
		t.Log("RandomString returned equal values (very rare); retrying once")
		b = faker.RandomString(20)
		if a == b {
			t.Fatal("RandomString returned the same value three times; generator may be broken")
		}
	}
}

func TestRandomUUID_Uniqueness(t *testing.T) {
	a := faker.RandomUUID()
	b := faker.RandomUUID()
	if a == b {
		t.Fatal("RandomUUID returned duplicate UUIDs")
	}
}
