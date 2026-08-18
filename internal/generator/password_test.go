package generator

import (
	"strings"
	"testing"

	"github.com/yourusername/passgen/internal/config"
)

// passwordConfig returns a valid password configuration with only the requested
// categories enabled.
func passwordConfig(length int, lower, upper, numbers, symbols bool) config.Config {
	cfg := config.Default()
	cfg.Length = length
	cfg.UseLowercase = lower
	cfg.UseUppercase = upper
	cfg.UseNumbers = numbers
	cfg.UseSymbols = symbols
	return cfg
}

func containsAny(s, set string) bool {
	return strings.ContainsAny(s, set)
}

func onlyFrom(s, allowed string) bool {
	for _, r := range s {
		if !strings.ContainsRune(allowed, r) {
			return false
		}
	}
	return true
}

func TestGeneratePasswordLength(t *testing.T) {
	for _, length := range []int{4, 8, 16, 32, 64, 256} {
		cfg := passwordConfig(length, true, true, true, true)
		password, err := GeneratePassword(cfg)
		if err != nil {
			t.Fatalf("length %d: unexpected error: %v", length, err)
		}
		if len([]rune(password)) != length {
			t.Fatalf("expected length %d, got %d (%q)", length, len([]rune(password)), password)
		}
	}
}

func TestGeneratePasswordsCount(t *testing.T) {
	cfg := passwordConfig(16, true, true, true, true)
	cfg.Count = 7

	passwords, err := GeneratePasswords(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(passwords) != 7 {
		t.Fatalf("expected 7 passwords, got %d", len(passwords))
	}
	for _, p := range passwords {
		if len([]rune(p)) != 16 {
			t.Fatalf("expected each password to be 16 characters, got %q", p)
		}
	}
}

func TestSingleCategoryGeneration(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		allowed string
	}{
		{"lowercase only", passwordConfig(20, true, false, false, false), Lowercase},
		{"uppercase only", passwordConfig(20, false, true, false, false), Uppercase},
		{"numbers only", passwordConfig(12, false, false, true, false), Numbers},
		{"symbols only", passwordConfig(20, false, false, false, true), Symbols},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			password, err := GeneratePassword(tc.cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !onlyFrom(password, tc.allowed) {
				t.Fatalf("password %q contains characters outside %q", password, tc.allowed)
			}
		})
	}
}

func TestMixedCategoriesAreGuaranteed(t *testing.T) {
	cfg := passwordConfig(12, true, true, true, true)

	// Repeat a few times: the guarantee must hold for every generated password.
	for i := 0; i < 50; i++ {
		password, err := GeneratePassword(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for name, set := range map[string]string{
			"lowercase": Lowercase,
			"uppercase": Uppercase,
			"number":    Numbers,
			"symbol":    Symbols,
		} {
			if !containsAny(password, set) {
				t.Fatalf("password %q is missing a %s character", password, name)
			}
		}
	}
}

func TestLowercaseAndNumbersOnly(t *testing.T) {
	cfg := passwordConfig(16, true, false, true, false)

	for i := 0; i < 25; i++ {
		password, err := GeneratePassword(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !containsAny(password, Lowercase) || !containsAny(password, Numbers) {
			t.Fatalf("password %q must contain lowercase and numbers", password)
		}
		if containsAny(password, Uppercase) || containsAny(password, Symbols) {
			t.Fatalf("password %q must not contain uppercase or symbols", password)
		}
	}
}

func TestExcludeAmbiguous(t *testing.T) {
	cfg := passwordConfig(32, true, true, true, true)
	cfg.ExcludeAmbiguous = true

	for i := 0; i < 100; i++ {
		password, err := GeneratePassword(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if containsAny(password, AmbiguousCharacters) {
			t.Fatalf("password %q contains an ambiguous character", password)
		}
	}
}

func TestCustomExclusion(t *testing.T) {
	const excluded = "@#$abc"
	cfg := passwordConfig(32, true, true, true, true)
	cfg.Exclude = excluded + excluded // duplicates must be handled gracefully

	for i := 0; i < 100; i++ {
		password, err := GeneratePassword(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if containsAny(password, excluded) {
			t.Fatalf("password %q contains an excluded character", password)
		}
	}
}

func TestUnicodeExclusionHasNoEffect(t *testing.T) {
	cfg := passwordConfig(16, true, true, true, true)
	cfg.Exclude = "😀ü漢"

	password, err := GeneratePassword(cfg)
	if err != nil {
		t.Fatalf("unicode exclusions outside the character sets must not fail: %v", err)
	}
	if len([]rune(password)) != 16 {
		t.Fatalf("expected 16 characters, got %q", password)
	}
}

func TestImpossibleExclusionReturnsError(t *testing.T) {
	cfg := passwordConfig(16, false, false, true, false)
	cfg.Exclude = Numbers

	_, err := GeneratePassword(cfg)
	if err == nil {
		t.Fatal("expected an error when every number is excluded")
	}
	if !strings.Contains(err.Error(), "number character set is empty") {
		t.Fatalf("expected a descriptive error, got %v", err)
	}
}

func TestExcludeAmbiguousStillGuaranteesNumbers(t *testing.T) {
	cfg := passwordConfig(8, false, false, true, false)
	cfg.ExcludeAmbiguous = true

	password, err := GeneratePassword(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !onlyFrom(password, "23456789") {
		t.Fatalf("password %q should only use unambiguous digits", password)
	}
}

func TestInvalidConfigurations(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
	}{
		{"length zero", passwordConfig(0, true, true, true, true)},
		{"negative length", passwordConfig(-1, true, true, true, true)},
		{"length too short", passwordConfig(3, true, true, true, true)},
		{"length too long", passwordConfig(257, true, true, true, true)},
		{"no categories", passwordConfig(16, false, false, false, false)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := GeneratePassword(tc.cfg); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestGenerateDispatchesToPassphrase(t *testing.T) {
	cfg := config.Default()
	cfg.Passphrase = true
	cfg.Count = 3

	results, err := Generate(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 passphrases, got %d", len(results))
	}
	for _, phrase := range results {
		if !strings.Contains(phrase, "-") {
			t.Fatalf("expected a passphrase, got %q", phrase)
		}
	}
}

func TestGenerateRejectsInvalidConfig(t *testing.T) {
	cfg := config.Default()
	cfg.Count = 0
	if _, err := Generate(cfg); err == nil {
		t.Fatal("expected an error for an invalid count")
	}

	cfg = config.Default()
	cfg.Passphrase = true
	cfg.Words = 99
	if _, err := Generate(cfg); err == nil {
		t.Fatal("expected an error for an invalid word count")
	}
}

func BenchmarkGeneratePassword(b *testing.B) {
	cfg := passwordConfig(16, true, true, true, true)
	for i := 0; i < b.N; i++ {
		if _, err := GeneratePassword(cfg); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkGeneratePassword32(b *testing.B) {
	cfg := passwordConfig(32, true, true, true, true)
	for i := 0; i < b.N; i++ {
		if _, err := GeneratePassword(cfg); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
