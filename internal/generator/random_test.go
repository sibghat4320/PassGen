package generator

import "testing"

func TestSecureRandomIndexBounds(t *testing.T) {
	const max = 10
	seen := make(map[int]bool)

	for i := 0; i < 500; i++ {
		got, err := secureRandomIndex(max)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got < 0 || got >= max {
			t.Fatalf("index %d is outside [0,%d)", got, max)
		}
		seen[got] = true
	}
	// With 500 draws over 10 values, hitting fewer than two distinct values
	// would indicate a broken generator rather than bad luck.
	if len(seen) < 2 {
		t.Fatalf("expected varied indices, only saw %d distinct values", len(seen))
	}
}

func TestSecureRandomIndexRejectsNonPositive(t *testing.T) {
	for _, max := range []int{0, -1, -100} {
		if _, err := secureRandomIndex(max); err == nil {
			t.Fatalf("expected an error for max %d", max)
		}
	}
}

func TestSecureShufflePreservesContents(t *testing.T) {
	original := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	shuffled := make([]rune, len(original))
	copy(shuffled, original)

	if err := secureShuffle(shuffled); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(shuffled) != len(original) {
		t.Fatalf("shuffle changed length: %d != %d", len(shuffled), len(original))
	}

	counts := make(map[rune]int)
	for _, r := range original {
		counts[r]++
	}
	for _, r := range shuffled {
		counts[r]--
	}
	for r, c := range counts {
		if c != 0 {
			t.Fatalf("character %q count changed by %d", r, -c)
		}
	}
}

func TestSecureShuffleHandlesSmallSlices(t *testing.T) {
	for _, input := range [][]rune{nil, {}, {'a'}} {
		if err := secureShuffle(input); err != nil {
			t.Fatalf("unexpected error for %q: %v", string(input), err)
		}
	}
}

func TestRandomRuneStaysInPool(t *testing.T) {
	pool := []rune("xyz")
	for i := 0; i < 50; i++ {
		got, err := randomRune(pool)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 'x' && got != 'y' && got != 'z' {
			t.Fatalf("rune %q is not in the pool", got)
		}
	}

	if _, err := randomRune(nil); err == nil {
		t.Fatal("expected an error for an empty pool")
	}
}
