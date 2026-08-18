package generator

import (
	"math"

	"github.com/yourusername/passgen/internal/config"
	"github.com/yourusername/passgen/internal/wordlist"
)

// Strength labels reported alongside an entropy estimate.
const (
	StrengthWeak       = "Weak"
	StrengthModerate   = "Moderate"
	StrengthStrong     = "Strong"
	StrengthVeryStrong = "Very Strong"
)

// Entropy thresholds (in bits) used to map an estimate onto a label.
const (
	moderateThreshold   = 36
	strongThreshold     = 60
	veryStrongThreshold = 80
)

// Entropy estimates the entropy of the secrets described by cfg, in bits.
//
// For passwords the estimate is length x log2(pool size); for passphrases it is
// word count x log2(available words). This is an estimate of the generator's
// output space, not a guarantee about real-world resistance to cracking.
func Entropy(cfg config.Config) (float64, error) {
	if cfg.Passphrase {
		return float64(cfg.Words) * math.Log2(float64(wordlist.Len())), nil
	}
	poolSize, err := PoolSize(cfg)
	if err != nil {
		return 0, err
	}
	return float64(cfg.Length) * math.Log2(float64(poolSize)), nil
}

// StrengthLabel maps an entropy estimate onto a human readable label.
func StrengthLabel(entropyBits float64) string {
	switch {
	case entropyBits < moderateThreshold:
		return StrengthWeak
	case entropyBits < strongThreshold:
		return StrengthModerate
	case entropyBits < veryStrongThreshold:
		return StrengthStrong
	default:
		return StrengthVeryStrong
	}
}
