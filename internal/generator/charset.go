package generator

import (
	"fmt"

	"github.com/yourusername/passgen/internal/config"
)

// Built-in character sets. They are intentionally restricted to ASCII.
const (
	Lowercase = "abcdefghijklmnopqrstuvwxyz"
	Uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	Numbers   = "0123456789"
	Symbols   = "!@#$%^&*()-_=+[]{}<>?/|~."

	// AmbiguousCharacters are easily confused when read by a human.
	AmbiguousCharacters = "0Oo1lI"
)

// category couples a human readable name with its raw character set.
type category struct {
	name  string
	chars string
}

// charsets holds the effective, exclusion-filtered character pools for a
// generation request.
type charsets struct {
	categories [][]rune // one entry per enabled category
	pool       []rune   // union of all enabled categories
}

// buildCharsets applies the enabled categories and the exclusion rules from cfg.
// An error is returned when an enabled category ends up empty, or when nothing
// is left to generate from.
func buildCharsets(cfg config.Config) (charsets, error) {
	excluded := excludedRunes(cfg)

	enabled := []category{}
	if cfg.UseLowercase {
		enabled = append(enabled, category{"lowercase", Lowercase})
	}
	if cfg.UseUppercase {
		enabled = append(enabled, category{"uppercase", Uppercase})
	}
	if cfg.UseNumbers {
		enabled = append(enabled, category{"number", Numbers})
	}
	if cfg.UseSymbols {
		enabled = append(enabled, category{"symbol", Symbols})
	}
	if len(enabled) == 0 {
		return charsets{}, fmt.Errorf("at least one character category must be enabled")
	}

	var sets charsets
	for _, c := range enabled {
		filtered := filterRunes(c.chars, excluded)
		if len(filtered) == 0 {
			return charsets{}, fmt.Errorf("enabled %s character set is empty after applying exclusions", c.name)
		}
		sets.categories = append(sets.categories, filtered)
		sets.pool = append(sets.pool, filtered...)
	}
	return sets, nil
}

// excludedRunes collects every rune that must not appear in a password.
// Duplicates and characters outside the built-in sets are harmless.
func excludedRunes(cfg config.Config) map[rune]struct{} {
	excluded := make(map[rune]struct{})
	if cfg.ExcludeAmbiguous {
		for _, r := range AmbiguousCharacters {
			excluded[r] = struct{}{}
		}
	}
	for _, r := range cfg.Exclude {
		excluded[r] = struct{}{}
	}
	return excluded
}

func filterRunes(set string, excluded map[rune]struct{}) []rune {
	filtered := make([]rune, 0, len(set))
	for _, r := range set {
		if _, skip := excluded[r]; skip {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

// PoolSize returns the number of distinct characters available for cfg. It is
// used for entropy estimation and returns an error for impossible settings.
func PoolSize(cfg config.Config) (int, error) {
	sets, err := buildCharsets(cfg)
	if err != nil {
		return 0, err
	}
	return len(sets.pool), nil
}
