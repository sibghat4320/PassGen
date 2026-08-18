// Package wordlist provides the embedded list of words used for passphrase
// generation. The list ships with the binary, so nothing is downloaded at
// runtime.
package wordlist

import (
	_ "embed"
	"strings"
	"sync"
)

//go:embed words.txt
var raw string

var (
	once  sync.Once
	words []string
)

// Words returns the embedded passphrase word list. The returned slice is a copy,
// so callers cannot mutate the shared list.
func Words() []string {
	once.Do(func() {
		for _, line := range strings.Split(raw, "\n") {
			word := strings.TrimSpace(line)
			if word == "" {
				continue
			}
			words = append(words, word)
		}
	})
	out := make([]string, len(words))
	copy(out, words)
	return out
}

// Len returns the number of available words.
func Len() int {
	return len(Words())
}
