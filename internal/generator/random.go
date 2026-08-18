package generator

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// secureRandomIndex returns a cryptographically secure random integer in the
// half-open interval [0, max). crypto/rand.Int is used so the result is free of
// modulo bias.
func secureRandomIndex(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("random index bound must be positive (got %d)", max)
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, fmt.Errorf("read secure random number: %w", err)
	}
	return int(n.Int64()), nil
}

// secureShuffle performs an in-place Fisher-Yates shuffle driven entirely by
// cryptographically secure randomness.
func secureShuffle(runes []rune) error {
	for i := len(runes) - 1; i > 0; i-- {
		j, err := secureRandomIndex(i + 1)
		if err != nil {
			return fmt.Errorf("shuffle characters: %w", err)
		}
		runes[i], runes[j] = runes[j], runes[i]
	}
	return nil
}

// randomRune picks one rune from pool using secure randomness.
func randomRune(pool []rune) (rune, error) {
	idx, err := secureRandomIndex(len(pool))
	if err != nil {
		return 0, err
	}
	return pool[idx], nil
}
