// Package tui implements the interactive terminal interface for passgen.
//
// It is deliberately line based rather than full screen: that keeps the whole
// package on the standard library, so it behaves identically on Windows, Linux
// and macOS and can be unit tested with ordinary readers and writers.
package tui

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/yourusername/passgen/internal/config"
	"github.com/yourusername/passgen/internal/generator"
)

// model holds the interactive session state. Generated secrets are printed
// immediately and never retained, so no password is kept in memory longer than
// the call that produced it.
type model struct {
	cfg     config.Config
	version string
	in      *bufio.Scanner
	out     io.Writer
	p       palette
	status  string // transient message shown above the prompt
	err     error  // first write error, checked by the main loop
}

// Run starts the interactive session, reading commands from in and rendering to
// out. It returns when the user quits or when in reaches EOF.
func Run(in io.Reader, out io.Writer, version string) error {
	return run(in, out, version, colorsEnabled(out))
}

func run(in io.Reader, out io.Writer, version string, color bool) error {
	m := &model{
		cfg:     config.Default(),
		version: version,
		in:      bufio.NewScanner(in),
		out:     out,
		p:       palette{enabled: color},
	}
	return m.loop()
}

func (m *model) loop() error {
	for {
		m.render()
		if m.err != nil {
			return m.err
		}

		command, ok := m.readLine("> ")
		if !ok {
			// EOF (Ctrl-D) is a normal way to leave the session.
			m.println("")
			return m.finish()
		}

		if quit := m.handle(command); quit {
			m.println(m.p.dim("bye"))
			return m.finish()
		}
		if m.err != nil {
			return m.err
		}
	}
}

// handle applies one command and reports whether the session should end.
func (m *model) handle(command string) bool {
	switch strings.ToLower(strings.TrimSpace(command)) {
	case "":
		m.status = ""
	case "1", "m", "mode":
		m.toggleMode()
	case "2":
		if m.cfg.Passphrase {
			m.editWords()
		} else {
			m.editLength()
		}
	case "3":
		m.editCount()
	case "4":
		if m.cfg.Passphrase {
			m.editSeparator()
		} else {
			m.editCategories()
		}
	case "5":
		if m.cfg.Passphrase {
			m.unknown("5")
		} else {
			m.toggleAmbiguous()
		}
	case "6":
		if m.cfg.Passphrase {
			m.unknown("6")
		} else {
			m.editExclusions()
		}
	case "g", "generate":
		m.generate()
	case "r", "reset":
		m.cfg = config.Default()
		m.info("settings reset to defaults")
	case "h", "help", "?":
		m.showHelp()
	case "q", "quit", "exit":
		return true
	default:
		m.unknown(command)
	}
	return false
}

// --- settings editors ---------------------------------------------------

func (m *model) toggleMode() {
	m.cfg.Passphrase = !m.cfg.Passphrase
	if m.cfg.Passphrase {
		m.info("switched to passphrase mode")
		return
	}
	m.info("switched to password mode")
}

func (m *model) editLength() {
	value, ok := m.promptInt(
		fmt.Sprintf("password length (%d-%d)", config.MinLength, config.MaxLength),
		m.cfg.Length,
		func(candidate config.Config, v int) config.Config {
			candidate.Length = v
			return candidate
		},
	)
	if ok {
		m.cfg.Length = value
		m.info(fmt.Sprintf("length set to %d", value))
	}
}

func (m *model) editCount() {
	value, ok := m.promptInt(
		fmt.Sprintf("how many to generate (%d-%d)", config.MinCount, config.MaxCount),
		m.cfg.Count,
		func(candidate config.Config, v int) config.Config {
			candidate.Count = v
			return candidate
		},
	)
	if ok {
		m.cfg.Count = value
		m.info(fmt.Sprintf("count set to %d", value))
	}
}

func (m *model) editWords() {
	value, ok := m.promptInt(
		fmt.Sprintf("number of words (%d-%d)", config.MinWords, config.MaxWords),
		m.cfg.Words,
		func(candidate config.Config, v int) config.Config {
			candidate.Words = v
			return candidate
		},
	)
	if ok {
		m.cfg.Words = value
		m.info(fmt.Sprintf("word count set to %d", value))
	}
}

func (m *model) editSeparator() {
	value, ok := m.readLine(fmt.Sprintf("separator (current %q, blank to keep): ", m.cfg.Separator))
	if !ok {
		return
	}
	if value == "" {
		m.info("separator unchanged")
		return
	}
	m.cfg.Separator = value
	m.info(fmt.Sprintf("separator set to %q", value))
}

func (m *model) toggleAmbiguous() {
	m.cfg.ExcludeAmbiguous = !m.cfg.ExcludeAmbiguous
	if err := m.validationError(m.cfg); err != nil {
		m.cfg.ExcludeAmbiguous = !m.cfg.ExcludeAmbiguous
		m.warn(err.Error())
		return
	}
	if m.cfg.ExcludeAmbiguous {
		m.info("ambiguous characters (" + generator.AmbiguousCharacters + ") excluded")
		return
	}
	m.info("ambiguous characters allowed")
}

func (m *model) editExclusions() {
	value, ok := m.readLine("characters to exclude (blank clears): ")
	if !ok {
		return
	}

	candidate := m.cfg
	candidate.Exclude = value
	if err := m.validationError(candidate); err != nil {
		m.warn(err.Error())
		return
	}

	m.cfg.Exclude = value
	if value == "" {
		m.info("custom exclusions cleared")
		return
	}
	m.info(fmt.Sprintf("excluding %q", value))
}

// editCategories runs a small sub-menu that toggles character categories. At
// least one category must stay enabled and the resulting set must be usable.
func (m *model) editCategories() {
	for {
		m.println("")
		m.println(m.p.bold("  character categories"))
		m.println(m.categoryLine("1", "lowercase", m.cfg.UseLowercase))
		m.println(m.categoryLine("2", "uppercase", m.cfg.UseUppercase))
		m.println(m.categoryLine("3", "numbers", m.cfg.UseNumbers))
		m.println(m.categoryLine("4", "symbols", m.cfg.UseSymbols))
		m.println(m.p.dim("  [1-4] toggle   [d] done"))

		choice, ok := m.readLine("categories> ")
		if !ok {
			return
		}

		targets := map[string]*bool{
			"1": &m.cfg.UseLowercase,
			"2": &m.cfg.UseUppercase,
			"3": &m.cfg.UseNumbers,
			"4": &m.cfg.UseSymbols,
		}
		choice = strings.ToLower(strings.TrimSpace(choice))
		switch choice {
		case "d", "done", "q", "":
			m.info("categories updated")
			return
		default:
			target, known := targets[choice]
			if !known {
				m.printWarn(fmt.Sprintf("unknown option %q", choice))
				continue
			}
			*target = !*target
			if err := m.validationError(m.cfg); err != nil {
				message := err.Error()
				if m.cfg.EnabledCategories() == 0 {
					message = "at least one character category must stay enabled"
				}
				*target = !*target // roll back an unusable selection
				m.printWarn(message)
			}
		}
	}
}

func (m *model) categoryLine(key, name string, enabled bool) string {
	mark := m.p.dim("off")
	if enabled {
		mark = m.p.green("on")
	}
	return fmt.Sprintf("    [%s] %-10s %s", key, name, mark)
}

// --- generation ---------------------------------------------------------

func (m *model) generate() {
	secrets, err := generator.Generate(m.cfg)
	if err != nil {
		m.warn(err.Error())
		return
	}

	m.println("")
	m.println("  " + m.p.dim("generated:"))
	for _, secret := range secrets {
		m.println("    " + m.p.bold(secret))
	}
	m.status = ""
}

// --- rendering ----------------------------------------------------------

func (m *model) render() {
	m.println("")
	m.println(m.p.cyan(fmt.Sprintf("passgen v%s", m.version)) + m.p.dim(" - interactive mode"))
	m.println("")

	if m.cfg.Passphrase {
		m.renderPassphraseSettings()
	} else {
		m.renderPasswordSettings()
	}

	m.println("  " + m.p.dim(fmt.Sprintf("%-12s", "entropy")) + m.entropySummary())
	m.println("")

	if m.cfg.Passphrase {
		m.println(m.p.dim("  [1] mode  [2] words  [3] count  [4] separator"))
	} else {
		m.println(m.p.dim("  [1] mode  [2] length  [3] count  [4] categories  [5] ambiguous  [6] exclude"))
	}
	m.println(m.p.dim("  [g] generate  [r] reset  [h] help  [q] quit"))

	if m.status != "" {
		m.println("")
		m.println("  " + m.status)
	}
	m.println("")
}

func (m *model) renderPasswordSettings() {
	m.setting("mode", "password")
	m.setting("length", strconv.Itoa(m.cfg.Length))
	m.setting("count", strconv.Itoa(m.cfg.Count))
	m.setting("categories", m.categorySummary())
	m.setting("ambiguous", boolLabel(m.cfg.ExcludeAmbiguous, "excluded", "allowed"))
	m.setting("exclude", valueOrNone(m.cfg.Exclude))
}

func (m *model) renderPassphraseSettings() {
	m.setting("mode", "passphrase")
	m.setting("words", strconv.Itoa(m.cfg.Words))
	m.setting("count", strconv.Itoa(m.cfg.Count))
	m.setting("separator", fmt.Sprintf("%q", m.cfg.Separator))
}

func (m *model) setting(name, value string) {
	m.println("  " + m.p.dim(fmt.Sprintf("%-12s", name)) + value)
}

func (m *model) categorySummary() string {
	names := []string{}
	for _, c := range []struct {
		enabled bool
		name    string
	}{
		{m.cfg.UseLowercase, "lowercase"},
		{m.cfg.UseUppercase, "uppercase"},
		{m.cfg.UseNumbers, "numbers"},
		{m.cfg.UseSymbols, "symbols"},
	} {
		if c.enabled {
			names = append(names, c.name)
		}
	}
	if len(names) == 0 {
		return m.p.red("none")
	}
	return strings.Join(names, ", ")
}

// entropySummary renders the entropy estimate, or the reason it cannot be
// computed for the current settings.
func (m *model) entropySummary() string {
	entropy, err := generator.Entropy(m.cfg)
	if err != nil {
		return m.p.red(err.Error())
	}
	label := generator.StrengthLabel(entropy)
	return fmt.Sprintf("%.1f bits (%s)", entropy, m.p.strengthColor(label))
}

func (m *model) showHelp() {
	m.status = strings.Join([]string{
		m.p.bold("help"),
		m.p.dim("  numbers pick a setting, [g] generates with the current settings."),
		m.p.dim("  entropy is an estimate of the generation space, not a guarantee."),
		m.p.dim("  nothing is stored, logged or sent anywhere."),
	}, "\n  ")
}

// --- helpers ------------------------------------------------------------

// promptInt asks for an integer until the user supplies a valid value, submits
// a blank line, or input ends. apply builds the candidate configuration used
// for validation, so the real limits and messages come from config.Validate.
func (m *model) promptInt(label string, current int, apply func(config.Config, int) config.Config) (int, bool) {
	for {
		raw, ok := m.readLine(fmt.Sprintf("%s [%d]: ", label, current))
		if !ok {
			return 0, false
		}
		if raw == "" {
			m.info("unchanged")
			return 0, false
		}

		value, err := strconv.Atoi(raw)
		if err != nil {
			m.printWarn(fmt.Sprintf("%q is not a whole number", raw))
			continue
		}
		if err := m.validationError(apply(m.cfg, value)); err != nil {
			m.printWarn(err.Error())
			continue
		}
		return value, true
	}
}

// validationError reports why cfg cannot be used, combining the declarative
// validation rules with the character set construction rules.
func (m *model) validationError(cfg config.Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if cfg.Passphrase {
		return nil
	}
	_, err := generator.PoolSize(cfg)
	return err
}

// readLine prints prompt and reads one line. The second result is false when
// the input stream is exhausted.
func (m *model) readLine(prompt string) (string, bool) {
	m.print(prompt)
	if m.err != nil {
		return "", false
	}
	if !m.in.Scan() {
		return "", false
	}
	return strings.TrimSpace(m.in.Text()), true
}

func (m *model) info(message string) {
	m.status = m.p.green("ok: ") + message
}

func (m *model) warn(message string) {
	m.status = m.p.yellow("!  ") + message
}

func (m *model) unknown(command string) {
	m.warn(fmt.Sprintf("unknown option %q - press h for help", command))
}

// printWarn reports a problem straight away, used while a sub-prompt is active.
func (m *model) printWarn(message string) {
	m.println("  " + m.p.yellow("!  ") + message)
}

func (m *model) print(text string) {
	if m.err != nil {
		return
	}
	if _, err := io.WriteString(m.out, text); err != nil {
		m.err = fmt.Errorf("write output: %w", err)
	}
}

func (m *model) println(text string) {
	m.print(text + "\n")
}

// finish returns the first write error, or any error reported by the scanner.
func (m *model) finish() error {
	if m.err != nil {
		return m.err
	}
	if err := m.in.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	return nil
}

func boolLabel(value bool, whenTrue, whenFalse string) string {
	if value {
		return whenTrue
	}
	return whenFalse
}

func valueOrNone(value string) string {
	if value == "" {
		return "(none)"
	}
	return value
}
