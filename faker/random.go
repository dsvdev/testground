// Package faker provides utilities for generating random test data
// using a cryptographically secure random source (crypto/rand).
// All functions are safe for concurrent use.
package faker

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"
)

const letters = "abcdefghijklmnopqrstuvwxyz"

// RandomInt returns a random integer in the range [min, max].
// Panics if min > max.
func RandomInt(min, max int) int {
	if min > max {
		panic(fmt.Sprintf("faker.RandomInt: min (%d) > max (%d)", min, max))
	}
	n := randomBigInt(int64(max - min + 1))
	return min + int(n)
}

// RandomInt64 returns a random int64 in the range [min, max].
// Panics if min > max.
func RandomInt64(min, max int64) int64 {
	if min > max {
		panic(fmt.Sprintf("faker.RandomInt64: min (%d) > max (%d)", min, max))
	}
	n := randomBigInt(max - min + 1)
	return min + n
}

// RandomString returns a random string of the given length consisting of
// lowercase ASCII letters (a-z). Returns an empty string when length is 0.
func RandomString(length int) string {
	if length == 0 {
		return ""
	}
	b := make([]byte, length)
	for i := range b {
		idx := randomBigInt(int64(len(letters)))
		b[i] = letters[idx]
	}
	return string(b)
}

// RandomUUID returns a randomly generated UUID v4 string in the canonical
// format xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx, where version bits are set to
// 4 and variant bits are set to 10xx (RFC 4122).
func RandomUUID() string {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		panic(fmt.Sprintf("faker.RandomUUID: crypto/rand failed: %v", err))
	}
	// Set version 4 (bits 12-15 of byte 6).
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	// Set variant 10xx (bits 6-7 of byte 8).
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:16],
	)
}

// randomBigInt returns a cryptographically random int64 in [0, n).
func randomBigInt(n int64) int64 {
	if n <= 0 {
		panic("faker: randomBigInt called with n <= 0")
	}
	// Fast path for powers of two.
	if n&(n-1) == 0 {
		var v uint64
		if err := binary.Read(rand.Reader, binary.BigEndian, &v); err != nil {
			panic(fmt.Sprintf("faker: crypto/rand failed: %v", err))
		}
		return int64(v) & (n - 1)
	}
	val, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		panic(fmt.Sprintf("faker: crypto/rand failed: %v", err))
	}
	return val.Int64()
}
