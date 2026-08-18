package generator

import (
	"fmt"

	"github.com/yourusername/passgen/internal/config"
)

// GeneratePassword returns a single password built according to cfg.
//
// The password is guaranteed to contain at least one character from every
// enabled category: one character is drawn per category first, the remainder is
// drawn from the combined pool, and the result is shuffled securely.
func GeneratePassword(cfg config.Config) (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", err
	}
	sets, err := buildCharsets(cfg)
	if err != nil {
		return "", err
	}
	return generatePassword(cfg.Length, sets)
}

// generatePassword builds one password from pre-computed character sets.
func generatePassword(length int, sets charsets) (string, error) {
	if length < len(sets.categories) {
		return "", fmt.Errorf("password length %d is too short for %d enabled character categories", length, len(sets.categories))
	}

	password := make([]rune, 0, length)
	for _, set := range sets.categories {
		r, err := randomRune(set)
		if err != nil {
			return "", fmt.Errorf("select guaranteed character: %w", err)
		}
		password = append(password, r)
	}
	for len(password) < length {
		r, err := randomRune(sets.pool)
		if err != nil {
			return "", fmt.Errorf("select password character: %w", err)
		}
		password = append(password, r)
	}
	if err := secureShuffle(password); err != nil {
		return "", err
	}
	return string(password), nil
}

// GeneratePasswords returns cfg.Count independently generated passwords.
func GeneratePasswords(cfg config.Config) ([]string, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	// The character sets are identical for every password, so they are built
	// once and any exclusion error surfaces before generation starts.
	sets, err := buildCharsets(cfg)
	if err != nil {
		return nil, err
	}

	results := make([]string, 0, cfg.Count)
	for i := 0; i < cfg.Count; i++ {
		password, err := generatePassword(cfg.Length, sets)
		if err != nil {
			return nil, fmt.Errorf("generate password %d: %w", i+1, err)
		}
		results = append(results, password)
	}
	return results, nil
}
