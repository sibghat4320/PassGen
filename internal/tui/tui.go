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
	value, ok := m.subPrompt(fmt.Sprintf("separator (current %q, blank to keep): ", m.cfg.Separator))
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

// editCategories runs a small sub-menu that toggles character categories. At
// least one category must stay enabled and the resulting set must be usable.
func (m *model) editCategories() {
	for {
		m.println("")
		m.println(indent + m.p.bold("character categories"))
		m.println(m.categoryLine("1", "lowercase", m.cfg.UseLowercase))
		m.println(m.categoryLine("2", "uppercase", m.cfg.UseUppercase))
		m.println(m.categoryLine("3", "numbers", m.cfg.UseNumbers))
		m.println(m.categoryLine("4", "symbols", m.cfg.UseSymbols))
		m.println("")
		m.println(indent + m.menuItems([][2]string{
			{"1-4", "toggle"}, {"d", "done"},
		}))
		m.println("")

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
	return fmt.Sprintf("    %s  %-12s%s", m.p.cyan(key), name, mark)
}

// --- generation ---------------------------------------------------------

func (m *model) generate() {
	secrets, err := generator.Generate(m.cfg)
	if err != nil {
		m.warn(err.Error())
		return
	}

	m.println("")
	m.println(indent + m.p.dim("generated:"))
	for _, secret := range secrets {
		m.println("    " + m.p.bold(secret))
	}
	m.status = ""
}

// --- rendering ----------------------------------------------------------

// Layout constants for the settings panel.
const (
	panelWidth = 58
	indent     = "  "
	labelWidth = 12
)

func (m *model) render() {
	m.println("")
	m.renderHeader()
	m.println("")

	if m.cfg.Passphrase {
		m.renderPassphraseSettings()
	} else {
		m.renderPasswordSettings()
	}

	m.println("")
	m.renderStrength()
	m.println("")
	m.println(indent + m.p.dim(rule('-', panelWidth)))
	m.renderMenu()

	if m.status != "" {
		m.println("")
		m.println(indent + m.status)
	}
	m.println("")
}

// renderHeader draws the title bar with the version on the left and the mode on
// the right, padded to the panel width.
func (m *model) renderHeader() {
	title := fmt.Sprintf("passgen v%s", m.version)
	mode := "interactive mode"

	gap := panelWidth - len(title) - len(mode)
	if gap < 1 {
		gap = 1
	}
	m.println(indent + m.p.dim(rule('=', panelWidth)))
	m.println(indent + m.p.bold(m.p.cyan(title)) + strings.Repeat(" ", gap) + m.p.dim(mode))
	m.println(indent + m.p.dim(rule('=', panelWidth)))
}

func (m *model) renderMenu() {
	if m.cfg.Passphrase {
		m.println(indent + m.menuItems([][2]string{
			{"1", "mode"}, {"2", "words"}, {"3", "count"}, {"4", "separator"},
		}))
	} else {
		m.println(indent + m.menuItems([][2]string{
			{"1", "mode"}, {"2", "length"}, {"3", "count"}, {"4", "categories"},
		}))
	}
	m.println(indent + m.menuItems([][2]string{
		{"g", "generate"}, {"r", "reset"}, {"h", "help"}, {"q", "quit"},
	}))
}

// menuItems renders "key label" pairs in evenly spaced columns.
func (m *model) menuItems(items [][2]string) string {
	var b strings.Builder
	for _, item := range items {
		entry := fmt.Sprintf("%s %s", m.p.cyan(item[0]), item[1])
		// Pad on the plain text so colored and plain output align identically.
		padding := 14 - (len(item[0]) + 1 + len(item[1]))
		if padding < 1 {
			padding = 1
		}
		b.WriteString(entry + strings.Repeat(" ", padding))
	}
	return strings.TrimRight(b.String(), " ")
}

func (m *model) renderPasswordSettings() {
	m.setting("mode", "password")
	m.setting("length", strconv.Itoa(m.cfg.Length))
	m.setting("count", strconv.Itoa(m.cfg.Count))
	m.setting("categories", m.categorySummary())
}

func (m *model) renderPassphraseSettings() {
	m.setting("mode", "passphrase")
	m.setting("words", strconv.Itoa(m.cfg.Words))
	m.setting("count", strconv.Itoa(m.cfg.Count))
	m.setting("separator", fmt.Sprintf("%q", m.cfg.Separator))
}

func (m *model) setting(name, value string) {
	m.println(indent + m.p.dim(fmt.Sprintf("%-*s", labelWidth, name)) + value)
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

// renderStrength shows the entropy estimate together with a proportional meter,
// or the reason the estimate cannot be produced.
func (m *model) renderStrength() {
	entropy, err := generator.Entropy(m.cfg)
	if err != nil {
		m.println(indent + m.p.dim(fmt.Sprintf("%-*s", labelWidth, "strength")) + m.p.red(err.Error()))
		return
	}

	label := generator.StrengthLabel(entropy)
	value := fmt.Sprintf("%.1f bits", entropy)
	line := fmt.Sprintf("%-11s %s  %s", value, m.meter(entropy, label), m.p.strengthColor(label))
	m.println(indent + m.p.dim(fmt.Sprintf("%-*s", labelWidth, "strength")) + line)
}

// meterCap is the entropy at which the strength meter is considered full.
const meterCap = 128

func (m *model) meter(entropy float64, label string) string {
	const cells = 16

	filled := int(entropy / meterCap * cells)
	if filled > cells {
		filled = cells
	}
	if filled < 1 {
		filled = 1
	}
	bar := strings.Repeat("#", filled) + strings.Repeat(".", cells-filled)
	return "[" + m.p.byStrength(label, bar) + "]"
}

// rule returns a horizontal rule of the requested width.
func rule(char rune, width int) string {
	return strings.Repeat(string(char), width)
}

func (m *model) showHelp() {
	m.status = strings.Join([]string{
		m.p.bold("help"),
		m.p.dim("numbers pick a setting, g generates with the current settings."),
		m.p.dim("strength is an estimate of the generation space, not a guarantee."),
		m.p.dim("nothing is stored, logged or sent anywhere."),
		m.p.dim("character exclusions are available as CLI flags:"),
		m.p.dim("passgen --exclude-ambiguous --exclude \"@#$\""),
	}, "\n"+indent+"  ")
}

// --- helpers ------------------------------------------------------------

// promptInt asks for an integer until the user supplies a valid value, submits
// a blank line, or input ends. apply builds the candidate configuration used
// for validation, so the real limits and messages come from config.Validate.
func (m *model) promptInt(label string, current int, apply func(config.Config, int) config.Config) (int, bool) {
	for {
		raw, ok := m.subPrompt(fmt.Sprintf("%s [%d]: ", label, current))
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
// the input stream is exhausted. Prompts are indented so they line up with the
// settings panel.
func (m *model) readLine(prompt string) (string, bool) {
	m.print(indent + prompt)
	if m.err != nil {
		return "", false
	}
	if !m.in.Scan() {
		return "", false
	}
	return strings.TrimSpace(m.in.Text()), true
}

// subPrompt asks a follow-up question on its own line, so it never runs into
// the main "> " prompt the user has just answered.
func (m *model) subPrompt(prompt string) (string, bool) {
	m.println("")
	return m.readLine(prompt)
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
	m.println(indent + m.p.yellow("!  ") + message)
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
