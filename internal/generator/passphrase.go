package generator

import (
	"fmt"
	"strings"

	"github.com/yourusername/passgen/internal/config"
	"github.com/yourusername/passgen/internal/wordlist"
)

// GeneratePassphrase returns a passphrase of wordCount words joined by
// separator, e.g. "river-orbit-lantern-maple". Words are picked independently
// with cryptographically secure randomness, so repeats are possible.
func GeneratePassphrase(wordCount int, separator string) (string, error) {
	if wordCount < config.MinWords || wordCount > config.MaxWords {
		return "", fmt.Errorf("passphrase word count must be between %d and %d (got %d)", config.MinWords, config.MaxWords, wordCount)
	}
	words := wordlist.Words()
	if len(words) == 0 {
		return "", fmt.Errorf("passphrase word list is empty")
	}

	selected := make([]string, 0, wordCount)
	for i := 0; i < wordCount; i++ {
		idx, err := secureRandomIndex(len(words))
		if err != nil {
			return "", fmt.Errorf("select passphrase word: %w", err)
		}
		selected = append(selected, words[idx])
	}
	return strings.Join(selected, separator), nil
}

// GeneratePassphrases returns cfg.Count passphrases described by cfg.
func GeneratePassphrases(cfg config.Config) ([]string, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	results := make([]string, 0, cfg.Count)
	for i := 0; i < cfg.Count; i++ {
		phrase, err := GeneratePassphrase(cfg.Words, cfg.Separator)
		if err != nil {
			return nil, fmt.Errorf("generate passphrase %d: %w", i+1, err)
		}
		results = append(results, phrase)
	}
	return results, nil
}
