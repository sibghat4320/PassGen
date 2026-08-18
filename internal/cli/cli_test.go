package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// runCLI executes Run with in-memory writers and returns stdout, stderr and the
// resulting error.
func runCLI(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	return runCLIWithInput(t, "", args...)
}

// runCLIWithInput is runCLI with scripted stdin, used for interactive mode.
func runCLIWithInput(t *testing.T, stdin string, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := Run(args, strings.NewReader(stdin), &stdout, &stderr)
	return stdout.String(), stderr.String(), err
}

func lines(out string) []string {
	trimmed := strings.TrimRight(out, "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func TestRunDefault(t *testing.T) {
	stdout, stderr, err := runCLI(t)
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr: %s)", err, stderr)
	}
	got := lines(stdout)
	if len(got) != 1 {
		t.Fatalf("expected one password, got %d lines", len(got))
	}
	if len([]rune(got[0])) != 16 {
		t.Fatalf("expected a 16 character password, got %q", got[0])
	}
}

func TestRunLengthAndCount(t *testing.T) {
	stdout, _, err := runCLI(t, "--length", "32", "--count", "3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := lines(stdout)
	if len(got) != 3 {
		t.Fatalf("expected 3 passwords, got %d", len(got))
	}
	for _, p := range got {
		if len([]rune(p)) != 32 {
			t.Fatalf("expected 32 characters, got %q", p)
		}
	}
}

func TestRunShorthandFlags(t *testing.T) {
	stdout, _, err := runCLI(t, "-l", "24", "-c", "2", "-q")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := lines(stdout)
	if len(got) != 2 {
		t.Fatalf("expected 2 passwords, got %d", len(got))
	}
	for _, p := range got {
		if len([]rune(p)) != 24 {
			t.Fatalf("expected 24 characters, got %q", p)
		}
	}
}

func TestRunCategorySelection(t *testing.T) {
	stdout, _, err := runCLI(t, "--numbers", "--length", "12")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	password := lines(stdout)[0]
	if strings.Trim(password, "0123456789") != "" {
		t.Fatalf("expected digits only, got %q", password)
	}
}

func TestRunNoSymbols(t *testing.T) {
	stdout, _, err := runCLI(t, "--no-symbols", "--length", "40")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	password := lines(stdout)[0]
	if strings.ContainsAny(password, "!@#$%^&*()-_=+[]{}<>?/|~.") {
		t.Fatalf("expected no symbols, got %q", password)
	}
	if !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") ||
		!strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") ||
		!strings.ContainsAny(password, "0123456789") {
		t.Fatalf("expected all three remaining categories in %q", password)
	}
}

func TestRunSymbolsAndNoSymbolsConflict(t *testing.T) {
	stdout, stderr, err := runCLI(t, "--symbols", "--no-symbols")
	if err == nil {
		t.Fatal("expected an error")
	}
	if stdout != "" {
		t.Fatalf("expected no stdout output, got %q", stdout)
	}
	if !strings.Contains(stderr, "conflict") {
		t.Fatalf("expected a conflict message on stderr, got %q", stderr)
	}
}

func TestRunExcludeAmbiguous(t *testing.T) {
	stdout, _, err := runCLI(t, "--exclude-ambiguous", "--length", "64", "--count", "5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, password := range lines(stdout) {
		if strings.ContainsAny(password, "0Oo1lI") {
			t.Fatalf("password %q contains an ambiguous character", password)
		}
	}
}

func TestRunCustomExclude(t *testing.T) {
	stdout, _, err := runCLI(t, "--exclude", "@#$%", "--length", "64", "--count", "5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, password := range lines(stdout) {
		if strings.ContainsAny(password, "@#$%") {
			t.Fatalf("password %q contains an excluded character", password)
		}
	}
}

func TestRunImpossibleExclusion(t *testing.T) {
	_, stderr, err := runCLI(t, "--numbers", "--exclude", "0123456789")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(stderr, "number character set is empty") {
		t.Fatalf("expected a descriptive error, got %q", stderr)
	}
}

func TestRunPassphrase(t *testing.T) {
	stdout, _, err := runCLI(t, "--passphrase")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	words := strings.Split(lines(stdout)[0], "-")
	if len(words) != 4 {
		t.Fatalf("expected 4 words by default, got %d", len(words))
	}
}

func TestRunPassphraseWordsAndSeparator(t *testing.T) {
	stdout, _, err := runCLI(t, "--passphrase", "--words", "6", "--separator", "_")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	phrase := lines(stdout)[0]
	if strings.Contains(phrase, "-") && !strings.Contains(phrase, "_") {
		t.Fatalf("expected underscore separators in %q", phrase)
	}
	if len(strings.Split(phrase, "_")) != 6 {
		t.Fatalf("expected 6 words, got %q", phrase)
	}
}

func TestRunStrengthOutput(t *testing.T) {
	stdout, _, err := runCLI(t, "--length", "20", "--strength")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := lines(stdout)
	if len(got) != 3 {
		t.Fatalf("expected 3 output lines, got %d: %q", len(got), stdout)
	}
	if !strings.HasPrefix(got[0], "Password: ") {
		t.Fatalf("unexpected first line %q", got[0])
	}
	if !strings.HasPrefix(got[1], "Entropy: ") || !strings.HasSuffix(got[1], " bits") {
		t.Fatalf("unexpected entropy line %q", got[1])
	}
	if !strings.HasPrefix(got[2], "Strength: ") {
		t.Fatalf("unexpected strength line %q", got[2])
	}
}

func TestRunStrengthPassphraseLabel(t *testing.T) {
	stdout, _, err := runCLI(t, "--passphrase", "--strength")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(stdout, "Passphrase: ") {
		t.Fatalf("expected a passphrase label, got %q", stdout)
	}
}

func TestRunQuietSuppressesStrengthLabels(t *testing.T) {
	stdout, _, err := runCLI(t, "--strength", "--quiet", "--count", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := lines(stdout)
	if len(got) != 2 {
		t.Fatalf("expected only 2 values, got %q", stdout)
	}
	for _, line := range got {
		if strings.Contains(line, ":") {
			t.Fatalf("quiet mode must print values only, got %q", line)
		}
	}
}

func TestRunJSONPlain(t *testing.T) {
	stdout, _, err := runCLI(t, "--count", "2", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload struct {
		Passwords []string `json:"passwords"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v (%q)", err, stdout)
	}
	if len(payload.Passwords) != 2 {
		t.Fatalf("expected 2 passwords, got %d", len(payload.Passwords))
	}
	for _, p := range payload.Passwords {
		if len([]rune(p)) != 16 {
			t.Fatalf("expected 16 character passwords, got %q", p)
		}
	}
}

func TestRunJSONWithStrength(t *testing.T) {
	stdout, _, err := runCLI(t, "--count", "2", "--json", "--strength")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload struct {
		Passwords []struct {
			Value       string  `json:"value"`
			EntropyBits float64 `json:"entropy_bits"`
			Strength    string  `json:"strength"`
		} `json:"passwords"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v (%q)", err, stdout)
	}
	if len(payload.Passwords) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(payload.Passwords))
	}
	for _, entry := range payload.Passwords {
		if entry.Value == "" || entry.EntropyBits <= 0 || entry.Strength == "" {
			t.Fatalf("incomplete JSON entry: %+v", entry)
		}
	}
}

func TestRunVersion(t *testing.T) {
	stdout, _, err := runCLI(t, "--version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(stdout) != "passgen v"+Version {
		t.Fatalf("unexpected version output %q", stdout)
	}
}

func TestRunHelp(t *testing.T) {
	stdout, stderr, err := runCLI(t, "--help")
	if err != nil {
		t.Fatalf("--help must not fail: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected help on stdout only, stderr was %q", stderr)
	}
	for _, want := range []string{"USAGE", "FLAGS", "EXAMPLES", "--passphrase", "--json", "--interactive"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("help output is missing %q", want)
		}
	}
}

func TestRunInteractive(t *testing.T) {
	stdout, _, err := runCLIWithInput(t, "g\nq\n", "--interactive")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "interactive mode") {
		t.Fatalf("expected the interactive screen, got %q", stdout)
	}
}

func TestRunInteractiveShorthand(t *testing.T) {
	stdout, _, err := runCLIWithInput(t, "q\n", "-i")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "interactive mode") {
		t.Fatalf("expected the interactive screen, got %q", stdout)
	}
}

func TestRunInteractiveConflicts(t *testing.T) {
	for _, args := range [][]string{
		{"--interactive", "--json"},
		{"--interactive", "--quiet"},
		{"--interactive", "-q"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, stderr, err := runCLIWithInput(t, "q\n", args...)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(stderr, "cannot be combined") {
				t.Fatalf("expected a conflict message, got %q", stderr)
			}
		})
	}
}

func TestRunInvalidValues(t *testing.T) {
	tests := [][]string{
		{"--length", "0"},
		{"--length", "-1"},
		{"--length", "3"},
		{"--length", "500"},
		{"--count", "0"},
		{"--count", "-1"},
		{"--count", "500"},
		{"--lowercase", "--uppercase", "--numbers", "--symbols", "--length", "3"},
		{"--passphrase", "--words", "2"},
		{"--passphrase", "--words", "13"},
		{"--length", "not-a-number"},
		{"--unknown-flag"},
		{"extra-argument"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			stdout, stderr, err := runCLI(t, args...)
			if err == nil {
				t.Fatal("expected an error")
			}
			if stdout != "" {
				t.Fatalf("expected no stdout output, got %q", stdout)
			}
			if !strings.HasPrefix(stderr, "error: ") {
				t.Fatalf("expected an error message on stderr, got %q", stderr)
			}
		})
	}
}
