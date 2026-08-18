package config

import "testing"

func TestDefaultIsValid(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config should be valid, got %v", err)
	}
	if cfg.Length != 16 || cfg.Count != 1 {
		t.Fatalf("unexpected defaults: length=%d count=%d", cfg.Length, cfg.Count)
	}
	if cfg.EnabledCategories() != 4 {
		t.Fatalf("expected 4 enabled categories, got %d", cfg.EnabledCategories())
	}
}

func TestValidateLength(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{"zero", 0, true},
		{"negative", -1, true},
		{"below minimum", 3, true},
		{"minimum", 4, false},
		{"typical", 16, false},
		{"maximum", 256, false},
		{"above maximum", 257, true},
		{"far above maximum", 500, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Default()
			cfg.Length = tc.length
			err := cfg.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for length %d", tc.length)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for length %d: %v", tc.length, err)
			}
		})
	}
}

func TestValidateCount(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		wantErr bool
	}{
		{"zero", 0, true},
		{"negative", -1, true},
		{"minimum", 1, false},
		{"maximum", 100, false},
		{"above maximum", 101, true},
		{"far above maximum", 500, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Default()
			cfg.Count = tc.count
			err := cfg.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for count %d", tc.count)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for count %d: %v", tc.count, err)
			}
		})
	}
}

func TestValidateNoCategories(t *testing.T) {
	cfg := Default()
	cfg.UseLowercase, cfg.UseUppercase, cfg.UseNumbers, cfg.UseSymbols = false, false, false, false
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected an error when no character category is enabled")
	}
}

func TestValidateLengthShorterThanCategories(t *testing.T) {
	cfg := Default()
	cfg.Length = 3
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected an error for length 3 with 4 categories")
	}

	// Length 4 with 4 categories is the smallest valid combination.
	cfg.Length = 4
	if err := cfg.Validate(); err != nil {
		t.Fatalf("length 4 with 4 categories should be valid, got %v", err)
	}
}

func TestValidatePassphraseWords(t *testing.T) {
	tests := []struct {
		words   int
		wantErr bool
	}{
		{0, true},
		{2, true},
		{3, false},
		{4, false},
		{12, false},
		{13, true},
	}

	for _, tc := range tests {
		cfg := Default()
		cfg.Passphrase = true
		cfg.Words = tc.words
		err := cfg.Validate()
		if tc.wantErr && err == nil {
			t.Fatalf("expected error for %d words", tc.words)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("unexpected error for %d words: %v", tc.words, err)
		}
	}
}

func TestPassphraseIgnoresPasswordLength(t *testing.T) {
	cfg := Default()
	cfg.Passphrase = true
	cfg.Length = 1 // irrelevant in passphrase mode
	if err := cfg.Validate(); err != nil {
		t.Fatalf("passphrase mode should not validate password length: %v", err)
	}
}
