package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yourusername/passgen/internal/generator"
)

// session runs the TUI with scripted input and returns the plain text output.
// Colors are disabled so assertions can match on readable text.
func session(t *testing.T, input string) string {
	t.Helper()
	var out bytes.Buffer
	if err := run(strings.NewReader(input), &out, "1.0.0", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return out.String()
}

// generated extracts the values printed under a "generated:" heading. Parsing
// the labelled block keeps the assertions independent of the password content,
// which may itself start with characters such as "!" or "[".
func generated(out string) []string {
	var values []string
	inBlock := false
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "generated:" {
			inBlock = true
			continue
		}
		if !inBlock {
			continue
		}
		if !strings.HasPrefix(line, "    ") {
			inBlock = false
			continue
		}
		values = append(values, strings.TrimSpace(line))
	}
	return values
}

func TestInitialScreen(t *testing.T) {
	out := session(t, "q\n")

	for _, want := range []string{
		"passgen v1.0.0",
		"interactive mode",
		"mode",
		"length",
		"count",
		"categories",
		"entropy",
		"[g] generate",
		"[q] quit",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("initial screen is missing %q\n%s", want, out)
		}
	}
}

func TestGenerateUsesDefaults(t *testing.T) {
	values := generated(session(t, "g\nq\n"))
	if len(values) != 1 {
		t.Fatalf("expected one password, got %d: %q", len(values), values)
	}
	if len([]rune(values[0])) != 16 {
		t.Fatalf("expected a 16 character password, got %q", values[0])
	}
}

func TestChangeLengthThenGenerate(t *testing.T) {
	values := generated(session(t, "2\n24\ng\nq\n"))
	if len(values) != 1 {
		t.Fatalf("expected one password, got %q", values)
	}
	if len([]rune(values[0])) != 24 {
		t.Fatalf("expected a 24 character password, got %q", values[0])
	}
}

func TestChangeCountThenGenerate(t *testing.T) {
	values := generated(session(t, "3\n3\ng\nq\n"))
	if len(values) != 3 {
		t.Fatalf("expected 3 passwords, got %q", values)
	}
}

func TestInvalidLengthRepromptsWithoutCrashing(t *testing.T) {
	out := session(t, "2\n999\n16\ng\nq\n")
	if !strings.Contains(out, "password length must be between 4 and 256") {
		t.Fatalf("expected a validation message, got:\n%s", out)
	}
	values := generated(out)
	if len(values) != 1 || len([]rune(values[0])) != 16 {
		t.Fatalf("expected one 16 character password after the re-prompt, got %q", values)
	}
}

func TestNonNumericLengthReprompts(t *testing.T) {
	out := session(t, "2\nabc\n20\ng\nq\n")
	if !strings.Contains(out, "is not a whole number") {
		t.Fatalf("expected a numeric validation message, got:\n%s", out)
	}
	values := generated(out)
	if len(values) != 1 || len([]rune(values[0])) != 20 {
		t.Fatalf("expected one 20 character password, got %q", values)
	}
}

func TestBlankInputKeepsCurrentValue(t *testing.T) {
	out := session(t, "2\n\ng\nq\n")
	if !strings.Contains(out, "unchanged") {
		t.Fatalf("expected an 'unchanged' notice, got:\n%s", out)
	}
	values := generated(out)
	if len(values) != 1 || len([]rune(values[0])) != 16 {
		t.Fatalf("expected the default length to be kept, got %q", values)
	}
}

func TestPassphraseModeToggle(t *testing.T) {
	out := session(t, "1\ng\nq\n")
	if !strings.Contains(out, "switched to passphrase mode") {
		t.Fatalf("expected a mode switch notice, got:\n%s", out)
	}
	if !strings.Contains(out, "separator") {
		t.Fatalf("expected passphrase settings, got:\n%s", out)
	}

	values := generated(out)
	if len(values) != 1 {
		t.Fatalf("expected one passphrase, got %q", values)
	}
	if words := strings.Split(values[0], "-"); len(words) != 4 {
		t.Fatalf("expected 4 words by default, got %q", values[0])
	}
}

func TestPassphraseWordsAndSeparator(t *testing.T) {
	// 1 switches mode, 2 sets words, 4 sets the separator.
	values := generated(session(t, "1\n2\n6\n4\n_\ng\nq\n"))
	if len(values) != 1 {
		t.Fatalf("expected one passphrase, got %q", values)
	}
	if words := strings.Split(values[0], "_"); len(words) != 6 {
		t.Fatalf("expected 6 underscore separated words, got %q", values[0])
	}
}

func TestPassphraseInvalidWordCountReprompts(t *testing.T) {
	out := session(t, "1\n2\n99\n5\ng\nq\n")
	if !strings.Contains(out, "passphrase word count must be between 3 and 12") {
		t.Fatalf("expected a word count validation message, got:\n%s", out)
	}
	values := generated(out)
	if len(strings.Split(values[0], "-")) != 5 {
		t.Fatalf("expected 5 words after the re-prompt, got %q", values[0])
	}
}

func TestCategoryToggles(t *testing.T) {
	// Enter the category menu and switch off uppercase, numbers and symbols.
	values := generated(session(t, "4\n2\n3\n4\nd\n2\n20\ng\nq\n"))
	if len(values) != 1 {
		t.Fatalf("expected one password, got %q", values)
	}
	if strings.Trim(values[0], generator.Lowercase) != "" {
		t.Fatalf("expected lowercase only, got %q", values[0])
	}
}

func TestCategoryMenuRejectsEmptySelection(t *testing.T) {
	// Turning every category off must be refused, and the last toggle rolled back.
	out := session(t, "4\n1\n2\n3\n4\nd\ng\nq\n")
	if !strings.Contains(out, "at least one character category") {
		t.Fatalf("expected a category validation message, got:\n%s", out)
	}
	values := generated(out)
	if len(values) != 1 || len([]rune(values[0])) != 16 {
		t.Fatalf("expected generation to still work, got %q", values)
	}
}

func TestCategoryMenuUnknownOption(t *testing.T) {
	out := session(t, "4\nz\nd\nq\n")
	if !strings.Contains(out, `unknown option "z"`) {
		t.Fatalf("expected an unknown option warning, got:\n%s", out)
	}
}

func TestToggleAmbiguous(t *testing.T) {
	out := session(t, "5\n2\n40\ng\nq\n")
	if !strings.Contains(out, "excluded") {
		t.Fatalf("expected an exclusion notice, got:\n%s", out)
	}
	values := generated(out)
	if strings.ContainsAny(values[0], generator.AmbiguousCharacters) {
		t.Fatalf("password %q contains an ambiguous character", values[0])
	}
}

func TestCustomExclusions(t *testing.T) {
	out := session(t, "6\n@#$%\n2\n40\ng\nq\n")
	values := generated(out)
	if len(values) != 1 {
		t.Fatalf("expected one password, got %q", values)
	}
	if strings.ContainsAny(values[0], "@#$%") {
		t.Fatalf("password %q contains an excluded character", values[0])
	}
}

func TestImpossibleExclusionIsRejected(t *testing.T) {
	out := session(t, "4\n1\n2\n4\nd\n6\n0123456789\ng\nq\n")
	if !strings.Contains(out, "number character set is empty") {
		t.Fatalf("expected an exclusion error, got:\n%s", out)
	}
	// The rejected exclusion must not be applied, so generation still works.
	if len(generated(out)) != 1 {
		t.Fatalf("expected generation to still succeed:\n%s", out)
	}
}

func TestResetRestoresDefaults(t *testing.T) {
	out := session(t, "2\n64\nr\ng\nq\n")
	if !strings.Contains(out, "settings reset to defaults") {
		t.Fatalf("expected a reset notice, got:\n%s", out)
	}
	values := generated(out)
	if len([]rune(values[0])) != 16 {
		t.Fatalf("expected the default length after reset, got %q", values[0])
	}
}

func TestHelpCommand(t *testing.T) {
	out := session(t, "h\nq\n")
	if !strings.Contains(out, "estimate") {
		t.Fatalf("expected the help text, got:\n%s", out)
	}
}

func TestUnknownCommand(t *testing.T) {
	out := session(t, "zzz\nq\n")
	if !strings.Contains(out, `unknown option "zzz"`) {
		t.Fatalf("expected an unknown option warning, got:\n%s", out)
	}
}

func TestBlankCommandIsIgnored(t *testing.T) {
	out := session(t, "\n\nq\n")
	if strings.Contains(out, "unknown option") {
		t.Fatalf("blank input must be ignored, got:\n%s", out)
	}
}

func TestQuitAndEOFExitCleanly(t *testing.T) {
	for name, input := range map[string]string{
		"quit letter": "q\n",
		"quit word":   "quit\n",
		"exit word":   "exit\n",
		"eof":         "",
		"eof mid":     "2\n",
	} {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			if err := run(strings.NewReader(input), &out, "1.0.0", false); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestWriteErrorIsReported(t *testing.T) {
	err := run(strings.NewReader("g\nq\n"), failingWriter{}, "1.0.0", false)
	if err == nil {
		t.Fatal("expected a write error to be reported")
	}
	if !strings.Contains(err.Error(), "write output") {
		t.Fatalf("expected a wrapped write error, got %v", err)
	}
}

// failingWriter always fails, so write error handling can be exercised.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWrite
}

var errWrite = writeError("disk full")

type writeError string

func (e writeError) Error() string { return string(e) }

func TestColorOutputContainsEscapeSequences(t *testing.T) {
	var out bytes.Buffer
	if err := run(strings.NewReader("g\nq\n"), &out, "1.0.0", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "\033[") {
		t.Fatal("expected ANSI sequences when colors are enabled")
	}
	if strings.Contains(stripANSI(out.String()), "\033[") {
		t.Fatal("stripANSI should remove every escape sequence")
	}
}

func TestPlainOutputHasNoEscapeSequences(t *testing.T) {
	if strings.Contains(session(t, "g\nq\n"), "\033[") {
		t.Fatal("expected plain output when colors are disabled")
	}
}

func TestColorsEnabledForNonTerminal(t *testing.T) {
	if colorsEnabled(&bytes.Buffer{}) {
		t.Fatal("colors must be disabled for non terminal writers")
	}
}

func TestColorsDisabledByNoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if colorsEnabled(&bytes.Buffer{}) {
		t.Fatal("NO_COLOR must disable colors")
	}
}
