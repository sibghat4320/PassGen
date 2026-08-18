package generator

import (
	"math"
	"testing"

	"github.com/yourusername/passgen/internal/config"
	"github.com/yourusername/passgen/internal/wordlist"
)

func TestPoolSize(t *testing.T) {
	full := passwordConfig(16, true, true, true, true)
	got, err := PoolSize(full)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := len(Lowercase) + len(Uppercase) + len(Numbers) + len(Symbols)
	if got != want {
		t.Fatalf("expected pool size %d, got %d", want, got)
	}

	lowerOnly := passwordConfig(16, true, false, false, false)
	got, err = PoolSize(lowerOnly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != len(Lowercase) {
		t.Fatalf("expected pool size %d, got %d", len(Lowercase), got)
	}
}

func TestPoolSizeWithExclusions(t *testing.T) {
	cfg := passwordConfig(16, false, false, true, false)
	cfg.ExcludeAmbiguous = true // removes 0 and 1 from the digits

	got, err := PoolSize(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 8 {
		t.Fatalf("expected 8 remaining digits, got %d", got)
	}
}

func TestEntropyPassword(t *testing.T) {
	cfg := passwordConfig(20, false, false, true, false) // 10 digits
	entropy, err := Entropy(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := 20 * math.Log2(10)
	if math.Abs(entropy-want) > 1e-9 {
		t.Fatalf("expected %.4f bits, got %.4f", want, entropy)
	}

	lower := passwordConfig(16, true, false, false, false) // 26 letters
	entropy, err = Entropy(lower)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want = 16 * math.Log2(26)
	if math.Abs(entropy-want) > 1e-9 {
		t.Fatalf("expected %.4f bits, got %.4f", want, entropy)
	}
}

func TestEntropyPassphrase(t *testing.T) {
	cfg := config.Default()
	cfg.Passphrase = true
	cfg.Words = 5

	entropy, err := Entropy(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := 5 * math.Log2(float64(wordlist.Len()))
	if math.Abs(entropy-want) > 1e-9 {
		t.Fatalf("expected %.4f bits, got %.4f", want, entropy)
	}
}

func TestEntropyImpossibleConfig(t *testing.T) {
	cfg := passwordConfig(16, false, false, true, false)
	cfg.Exclude = Numbers

	if _, err := Entropy(cfg); err == nil {
		t.Fatal("expected an error when the pool is empty")
	}
}

func TestStrengthLabel(t *testing.T) {
	tests := []struct {
		entropy float64
		want    string
	}{
		{0, StrengthWeak},
		{35.9, StrengthWeak},
		{36, StrengthModerate},
		{59.9, StrengthModerate},
		{60, StrengthStrong},
		{79.9, StrengthStrong},
		{80, StrengthVeryStrong},
		{128, StrengthVeryStrong},
	}

	for _, tc := range tests {
		if got := StrengthLabel(tc.entropy); got != tc.want {
			t.Fatalf("entropy %.1f: expected %q, got %q", tc.entropy, tc.want, got)
		}
	}
}
