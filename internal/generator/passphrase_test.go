package generator

import (
	"strings"
	"testing"

	"github.com/yourusername/passgen/internal/config"
	"github.com/yourusername/passgen/internal/wordlist"
)

func TestGeneratePassphraseWordCount(t *testing.T) {
	for words := config.MinWords; words <= config.MaxWords; words++ {
		phrase, err := GeneratePassphrase(words, "-")
		if err != nil {
			t.Fatalf("%d words: unexpected error: %v", words, err)
		}
		got := strings.Split(phrase, "-")
		if len(got) != words {
			t.Fatalf("expected %d words, got %d (%q)", words, len(got), phrase)
		}
	}
}

func TestGeneratePassphraseSeparator(t *testing.T) {
	for _, sep := range []string{"-", "_", ".", " ", "::"} {
		phrase, err := GeneratePassphrase(4, sep)
		if err != nil {
			t.Fatalf("separator %q: unexpected error: %v", sep, err)
		}
		if strings.Count(phrase, sep) < 3 {
			t.Fatalf("expected 3 separators %q in %q", sep, phrase)
		}
	}
}

func TestGeneratePassphraseUsesWordList(t *testing.T) {
	known := make(map[string]bool)
	for _, w := range wordlist.Words() {
		known[w] = true
	}

	phrase, err := GeneratePassphrase(6, "-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, word := range strings.Split(phrase, "-") {
		if !known[word] {
			t.Fatalf("word %q is not part of the embedded word list", word)
		}
	}
}

func TestGeneratePassphraseInvalidWordCount(t *testing.T) {
	for _, words := range []int{-1, 0, 2, 13, 100} {
		if _, err := GeneratePassphrase(words, "-"); err == nil {
			t.Fatalf("expected an error for %d words", words)
		}
	}
}

func TestGeneratePassphrasesCount(t *testing.T) {
	cfg := config.Default()
	cfg.Passphrase = true
	cfg.Count = 5
	cfg.Words = 3
	cfg.Separator = "_"

	phrases, err := GeneratePassphrases(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(phrases) != 5 {
		t.Fatalf("expected 5 passphrases, got %d", len(phrases))
	}
	for _, phrase := range phrases {
		if len(strings.Split(phrase, "_")) != 3 {
			t.Fatalf("expected 3 words in %q", phrase)
		}
	}
}

func TestWordListQuality(t *testing.T) {
	words := wordlist.Words()
	if len(words) < 300 {
		t.Fatalf("expected several hundred words, got %d", len(words))
	}

	seen := make(map[string]bool, len(words))
	for _, word := range words {
		if seen[word] {
			t.Fatalf("duplicate word %q in the word list", word)
		}
		seen[word] = true
		if word != strings.ToLower(word) {
			t.Fatalf("word %q should be lowercase", word)
		}
		if strings.ContainsAny(word, " \t-_") {
			t.Fatalf("word %q should not contain separators", word)
		}
	}
}

func TestWordsReturnsCopy(t *testing.T) {
	words := wordlist.Words()
	original := words[0]
	words[0] = "mutated"

	if wordlist.Words()[0] != original {
		t.Fatal("Words() must return a copy of the shared word list")
	}
}
