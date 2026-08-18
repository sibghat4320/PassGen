package tui

import (
	"io"
	"os"
	"strings"
)

// ANSI escape sequences. Only a small, widely supported subset is used.
const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiCyan   = "\033[36m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
)

// palette wraps text in ANSI sequences, or returns it untouched when colors are
// disabled. Keeping that switch in one place keeps the render code readable.
type palette struct {
	enabled bool
}

func (p palette) wrap(code, text string) string {
	if !p.enabled || text == "" {
		return text
	}
	return code + text + ansiReset
}

func (p palette) bold(text string) string   { return p.wrap(ansiBold, text) }
func (p palette) dim(text string) string    { return p.wrap(ansiDim, text) }
func (p palette) cyan(text string) string   { return p.wrap(ansiCyan, text) }
func (p palette) green(text string) string  { return p.wrap(ansiGreen, text) }
func (p palette) yellow(text string) string { return p.wrap(ansiYellow, text) }
func (p palette) red(text string) string    { return p.wrap(ansiRed, text) }

// strengthColor maps a strength label onto a palette color.
func (p palette) strengthColor(label string) string {
	switch label {
	case "Very Strong", "Strong":
		return p.green(label)
	case "Moderate":
		return p.yellow(label)
	default:
		return p.red(label)
	}
}

// colorsEnabled reports whether ANSI output is appropriate for w. Colors are
// used only when writing to a real terminal and when NO_COLOR is unset, so
// piped or redirected output stays plain and script friendly.
func colorsEnabled(w io.Writer) bool {
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// stripANSI removes escape sequences from s. It is used by the tests so
// assertions can be written against plain text.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++ // skip the terminating 'm'
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
