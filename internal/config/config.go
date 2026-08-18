// Package config defines the generation settings shared by the CLI and the
// generator, together with the validation rules applied to them.
package config

import "fmt"

// Limits accepted by the generator. They are exported so the CLI (and tests)
// can reference the exact same bounds used during validation.
const (
	MinLength = 4
	MaxLength = 256

	MinCount = 1
	MaxCount = 100

	MinWords = 3
	MaxWords = 12

	DefaultLength    = 16
	DefaultCount     = 1
	DefaultWords     = 4
	DefaultSeparator = "-"
)

// Config describes a single generation request.
type Config struct {
	Length           int
	Count            int
	UseLowercase     bool
	UseUppercase     bool
	UseNumbers       bool
	UseSymbols       bool
	ExcludeAmbiguous bool
	Exclude          string
	Passphrase       bool
	Words            int
	Separator        string
	ShowStrength     bool
	JSON             bool
	Quiet            bool
}

// Default returns a Config populated with the documented default behaviour:
// one 16 character password using every character category.
func Default() Config {
	return Config{
		Length:       DefaultLength,
		Count:        DefaultCount,
		UseLowercase: true,
		UseUppercase: true,
		UseNumbers:   true,
		UseSymbols:   true,
		Words:        DefaultWords,
		Separator:    DefaultSeparator,
	}
}

// EnabledCategories reports how many character categories are enabled.
func (c Config) EnabledCategories() int {
	n := 0
	for _, enabled := range []bool{c.UseLowercase, c.UseUppercase, c.UseNumbers, c.UseSymbols} {
		if enabled {
			n++
		}
	}
	return n
}

// Validate checks the configuration and returns a descriptive error for the
// first problem found. Password and passphrase modes are validated separately
// because they use different fields.
func (c Config) Validate() error {
	if c.Count < MinCount || c.Count > MaxCount {
		return fmt.Errorf("password count must be between %d and %d (got %d)", MinCount, MaxCount, c.Count)
	}

	if c.Passphrase {
		return c.validatePassphrase()
	}
	return c.validatePassword()
}

func (c Config) validatePassphrase() error {
	if c.Words < MinWords || c.Words > MaxWords {
		return fmt.Errorf("passphrase word count must be between %d and %d (got %d)", MinWords, MaxWords, c.Words)
	}
	return nil
}

func (c Config) validatePassword() error {
	if c.Length < MinLength || c.Length > MaxLength {
		return fmt.Errorf("password length must be between %d and %d (got %d)", MinLength, MaxLength, c.Length)
	}

	categories := c.EnabledCategories()
	if categories == 0 {
		return fmt.Errorf("at least one character category must be enabled (--lowercase, --uppercase, --numbers, --symbols)")
	}
	if c.Length < categories {
		return fmt.Errorf("password length %d is too short for %d enabled character categories", c.Length, categories)
	}
	return nil
}
