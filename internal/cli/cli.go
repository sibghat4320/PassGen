// Package cli implements command line parsing and output formatting for
// passgen. It is written so that tests can call Run with in-memory writers.
package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/yourusername/passgen/internal/config"
	"github.com/yourusername/passgen/internal/generator"
)

// Version is the released version of passgen. It can be overridden at build
// time with -ldflags "-X github.com/yourusername/passgen/internal/cli.Version=..."
var Version = "1.0.0"

// ErrHelpRequested is returned internally when the user asked for --help.
var errHelpRequested = errors.New("help requested")

// options mirrors the raw command line flags before they are turned into a
// config.Config.
type options struct {
	length           int
	count            int
	lowercase        bool
	uppercase        bool
	numbers          bool
	symbols          bool
	noSymbols        bool
	excludeAmbiguous bool
	exclude          string
	passphrase       bool
	words            int
	separator        string
	strength         bool
	jsonOut          bool
	quiet            bool
	version          bool
}

// Run parses args, generates the requested secrets and writes them to stdout.
// Any error is reported on stderr and also returned, so callers only need to
// decide the exit code.
func Run(args []string, stdout, stderr io.Writer) error {
	if err := run(args, stdout); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return err
	}
	return nil
}

func run(args []string, stdout io.Writer) error {
	opts, fs, err := parseArgs(args)
	if err != nil {
		if errors.Is(err, errHelpRequested) {
			printUsage(stdout)
			return nil
		}
		return err
	}

	if opts.version {
		return writeLine(stdout, fmt.Sprintf("passgen v%s", Version))
	}

	cfg, err := buildConfig(opts, fs)
	if err != nil {
		return err
	}

	secrets, err := generator.Generate(cfg)
	if err != nil {
		return err
	}
	return writeOutput(stdout, cfg, secrets)
}

func parseArgs(args []string) (options, *flag.FlagSet, error) {
	var opts options

	fs := flag.NewFlagSet("passgen", flag.ContinueOnError)
	// Parsing problems are reported through the returned error instead of the
	// flag package's own output, so the CLI has a single error format.
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	fs.IntVar(&opts.length, "length", config.DefaultLength, "password length")
	fs.IntVar(&opts.length, "l", config.DefaultLength, "password length (shorthand)")
	fs.IntVar(&opts.count, "count", config.DefaultCount, "number of secrets to generate")
	fs.IntVar(&opts.count, "c", config.DefaultCount, "number of secrets to generate (shorthand)")
	fs.BoolVar(&opts.lowercase, "lowercase", false, "include lowercase letters")
	fs.BoolVar(&opts.uppercase, "uppercase", false, "include uppercase letters")
	fs.BoolVar(&opts.numbers, "numbers", false, "include numbers")
	fs.BoolVar(&opts.symbols, "symbols", false, "include symbols")
	fs.BoolVar(&opts.noSymbols, "no-symbols", false, "exclude symbols (lowercase, uppercase and numbers only)")
	fs.BoolVar(&opts.excludeAmbiguous, "exclude-ambiguous", false, "exclude easily confused characters (0Oo1lI)")
	fs.StringVar(&opts.exclude, "exclude", "", "characters to exclude from every character set")
	fs.BoolVar(&opts.passphrase, "passphrase", false, "generate a word based passphrase instead of a password")
	fs.IntVar(&opts.words, "words", config.DefaultWords, "number of words in a passphrase")
	fs.StringVar(&opts.separator, "separator", config.DefaultSeparator, "separator between passphrase words")
	fs.BoolVar(&opts.strength, "strength", false, "show an entropy estimate and strength label")
	fs.BoolVar(&opts.jsonOut, "json", false, "print results as JSON")
	fs.BoolVar(&opts.quiet, "quiet", false, "print only generated values")
	fs.BoolVar(&opts.quiet, "q", false, "print only generated values (shorthand)")
	fs.BoolVar(&opts.version, "version", false, "print the passgen version")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return opts, fs, errHelpRequested
		}
		return opts, fs, fmt.Errorf("%w (run 'passgen --help' for usage)", err)
	}
	if fs.NArg() > 0 {
		return opts, fs, fmt.Errorf("unexpected argument %q (passgen only accepts flags, see --help)", fs.Arg(0))
	}
	return opts, fs, nil
}

// buildConfig turns parsed flags into a validated config.Config. Which flags
// were explicitly supplied matters, so the flag set is inspected as well.
func buildConfig(opts options, fs *flag.FlagSet) (config.Config, error) {
	set := explicitFlags(fs)

	if set["symbols"] && set["no-symbols"] {
		return config.Config{}, errors.New("--symbols and --no-symbols conflict, use only one of them")
	}

	cfg := config.Config{
		Length:           opts.length,
		Count:            opts.count,
		ExcludeAmbiguous: opts.excludeAmbiguous,
		Exclude:          opts.exclude,
		Passphrase:       opts.passphrase,
		Words:            opts.words,
		Separator:        opts.separator,
		ShowStrength:     opts.strength,
		JSON:             opts.jsonOut,
		Quiet:            opts.quiet,
	}

	categorySelected := set["lowercase"] || set["uppercase"] || set["numbers"] || set["symbols"]
	switch {
	case categorySelected:
		// Only the categories the user asked for are used.
		cfg.UseLowercase = opts.lowercase
		cfg.UseUppercase = opts.uppercase
		cfg.UseNumbers = opts.numbers
		cfg.UseSymbols = opts.symbols && !opts.noSymbols
	case set["no-symbols"]:
		cfg.UseLowercase, cfg.UseUppercase, cfg.UseNumbers = true, true, true
	default:
		cfg.UseLowercase, cfg.UseUppercase, cfg.UseNumbers, cfg.UseSymbols = true, true, true, true
	}

	if err := cfg.Validate(); err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

func explicitFlags(fs *flag.FlagSet) map[string]bool {
	set := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	return set
}

// jsonPlain is the JSON shape used when strength reporting is disabled.
type jsonPlain struct {
	Passwords []string `json:"passwords"`
}

// jsonSecret carries a value together with its entropy estimate.
type jsonSecret struct {
	Value       string  `json:"value"`
	EntropyBits float64 `json:"entropy_bits"`
	Strength    string  `json:"strength"`
}

type jsonDetailed struct {
	Passwords []jsonSecret `json:"passwords"`
}

func writeOutput(stdout io.Writer, cfg config.Config, secrets []string) error {
	switch {
	case cfg.JSON:
		return writeJSON(stdout, cfg, secrets)
	case cfg.Quiet || !cfg.ShowStrength:
		return writePlain(stdout, secrets)
	default:
		return writeDetailed(stdout, cfg, secrets)
	}
}

func writePlain(stdout io.Writer, secrets []string) error {
	for _, secret := range secrets {
		if err := writeLine(stdout, secret); err != nil {
			return err
		}
	}
	return nil
}

func writeDetailed(stdout io.Writer, cfg config.Config, secrets []string) error {
	entropy, err := generator.Entropy(cfg)
	if err != nil {
		return err
	}
	label := "Password"
	if cfg.Passphrase {
		label = "Passphrase"
	}

	for i, secret := range secrets {
		if i > 0 {
			if err := writeLine(stdout, ""); err != nil {
				return err
			}
		}
		lines := []string{
			fmt.Sprintf("%s: %s", label, secret),
			fmt.Sprintf("Entropy: %.1f bits", entropy),
			fmt.Sprintf("Strength: %s", generator.StrengthLabel(entropy)),
		}
		for _, line := range lines {
			if err := writeLine(stdout, line); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeJSON(stdout io.Writer, cfg config.Config, secrets []string) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	// Passwords contain characters such as < > &; escaping them as \u003c would
	// still be valid JSON but is needlessly hard to read.
	encoder.SetEscapeHTML(false)

	if !cfg.ShowStrength {
		if err := encoder.Encode(jsonPlain{Passwords: secrets}); err != nil {
			return fmt.Errorf("encode JSON output: %w", err)
		}
		return nil
	}

	entropy, err := generator.Entropy(cfg)
	if err != nil {
		return err
	}
	payload := jsonDetailed{Passwords: make([]jsonSecret, 0, len(secrets))}
	for _, secret := range secrets {
		payload.Passwords = append(payload.Passwords, jsonSecret{
			Value:       secret,
			EntropyBits: math.Round(entropy*10) / 10,
			Strength:    generator.StrengthLabel(entropy),
		})
	}
	if err := encoder.Encode(payload); err != nil {
		return fmt.Errorf("encode JSON output: %w", err)
	}
	return nil
}

func writeLine(w io.Writer, line string) error {
	if _, err := fmt.Fprintln(w, line); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

const usageText = `passgen - a secure password and passphrase generator.

Passwords are generated locally using Go's crypto/rand. Nothing is stored,
logged or sent over the network.

USAGE
  passgen [flags]

FLAGS
  -l, --length int        password length (default 16, min 4, max 256)
  -c, --count int         number of secrets to generate (default 1, min 1, max 100)
      --lowercase         include lowercase letters
      --uppercase         include uppercase letters
      --numbers           include numbers
      --symbols           include symbols
      --no-symbols        shortcut for lowercase + uppercase + numbers
      --exclude-ambiguous exclude easily confused characters (0Oo1lI)
      --exclude string    characters to remove from every character set
      --passphrase        generate a word based passphrase instead
      --words int         number of passphrase words (default 4, min 3, max 12)
      --separator string  passphrase separator (default "-")
      --strength          show an entropy estimate and strength label
      --json              print results as JSON
  -q, --quiet             print only generated values
      --version           print the passgen version
  -h, --help              show this help

CHARACTER CATEGORIES
  When no category flag is given, all four categories are enabled. When one or
  more category flags are given, only those categories are used. Every enabled
  category is guaranteed to appear at least once in each password.

EXAMPLES
  passgen
  passgen --length 32
  passgen --count 5
  passgen --lowercase --numbers
  passgen --exclude-ambiguous
  passgen --exclude "@#$"
  passgen --no-symbols --length 20
  passgen --passphrase
  passgen --passphrase --words 6 --separator "_"
  passgen --length 20 --strength
  passgen --count 2 --json

NOTE
  Entropy values are estimates based on the size of the generation space, not a
  guarantee of real-world cracking resistance.`

func printUsage(w io.Writer) {
	fmt.Fprintln(w, strings.TrimSpace(usageText))
}
