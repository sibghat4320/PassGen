package wordlist

import "testing"

func TestWordsAreLoaded(t *testing.T) {
	words := Words()
	if len(words) < 300 {
		t.Fatalf("expected several hundred embedded words, got %d", len(words))
	}
	if Len() != len(words) {
		t.Fatalf("Len() = %d, expected %d", Len(), len(words))
	}
	for _, word := range words {
		if word == "" {
			t.Fatal("word list must not contain empty entries")
		}
	}
}

func TestWordsIsCached(t *testing.T) {
	first := Words()
	second := Words()
	if len(first) != len(second) {
		t.Fatalf("inconsistent word list length: %d != %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("word list changed between calls at index %d", i)
		}
	}
}
