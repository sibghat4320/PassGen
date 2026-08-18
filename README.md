# Passgen

A secure command-line password and passphrase generator written in Go.

`passgen` generates passwords with Go's `crypto/rand`, so every character is
chosen with cryptographically secure randomness. It has **zero third-party
runtime dependencies** and runs on Windows, Linux and macOS.

```bash
$ passgen
mH7#qL2!xP9@vK4z
```

## Features

- Secure generation backed by `crypto/rand` (no `math/rand` anywhere)
- Custom password lengths (4–256, default 16)
- Selectable character categories: lowercase, uppercase, numbers, symbols
- Every enabled category is guaranteed to appear at least once
- Multiple passwords per invocation (1–100)
- Ambiguous character exclusion (`0 O o 1 l I`)
- Custom character exclusions
- Passphrase mode with an embedded word list and custom separators
- Entropy estimation with a readable strength label
- JSON output for scripting
- Quiet output for shell pipelines
- Cross-platform, dependency-free single binary

## Installation

Replace `yourusername` with your own GitHub account if you fork this project;
it is only a placeholder module path.

```bash
git clone https://github.com/yourusername/passgen.git
cd passgen
go build -o passgen ./cmd/passgen
```

Or install it straight into `$GOPATH/bin`:

```bash
go install github.com/yourusername/passgen/cmd/passgen@latest
```

With the repository checked out you can also run it without building:

```bash
go run ./cmd/passgen --length 24
```

## Usage

```text
passgen [flags]
```

| Flag | Short | Default | Description |
| --- | --- | --- | --- |
| `--length` | `-l` | `16` | Password length (min 4, max 256) |
| `--count` | `-c` | `1` | Number of secrets to generate (min 1, max 100) |
| `--lowercase` | | all categories | Include lowercase letters |
| `--uppercase` | | all categories | Include uppercase letters |
| `--numbers` | | all categories | Include numbers |
| `--symbols` | | all categories | Include symbols |
| `--no-symbols` | | `false` | Shortcut for lowercase + uppercase + numbers |
| `--exclude-ambiguous` | | `false` | Exclude `0 O o 1 l I` |
| `--exclude` | | `""` | Characters to remove from every character set |
| `--passphrase` | | `false` | Generate a word based passphrase |
| `--words` | | `4` | Passphrase word count (min 3, max 12) |
| `--separator` | | `-` | Passphrase separator |
| `--strength` | | `false` | Show entropy estimate and strength label |
| `--json` | | `false` | Print results as JSON |
| `--quiet` | `-q` | `false` | Print only generated values |
| `--version` | | | Print the passgen version |
| `--help` | `-h` | | Show the help screen |

### Character categories

- When no category flag is given, all four categories are enabled.
- When one or more category flags are given, only those categories are used.
- Each enabled category is guaranteed to appear at least once in every password.
- `--symbols` together with `--no-symbols` is rejected as a conflict.

### Character sets

```text
lowercase  abcdefghijklmnopqrstuvwxyz
uppercase  ABCDEFGHIJKLMNOPQRSTUVWXYZ
numbers    0123456789
symbols    !@#$%^&*()-_=+[]{}<>?/|~.
```

The built-in sets are ASCII. Unicode characters passed to `--exclude` that are
not part of any set simply have no effect.

## Examples

```bash
# One 16 character password using every category
passgen

# A long password
passgen --length 32

# Five passwords, one per line
passgen --count 5

# Digits only
passgen --numbers --length 12

# Lowercase letters and digits only
passgen --lowercase --numbers --length 16

# Avoid characters that are easy to misread
passgen --exclude-ambiguous

# Remove specific characters
passgen --exclude "@#$%"

# No symbols (useful for systems that reject them)
passgen --no-symbols --length 20

# Passphrases
passgen --passphrase
passgen --passphrase --words 6 --separator "_"

# Entropy estimate
passgen --length 20 --strength

# Machine readable output
passgen --count 2 --json
passgen --count 2 --json --strength
```

Example strength output:

```text
$ passgen --length 20 --strength
Password: mH7#qL2!xP9@vK4zT6$r
Entropy: 128.9 bits
Strength: Very Strong
```

Example JSON output:

```json
{
  "passwords": [
    "hL9@xK3!mP7#qR2$",
    "vT4!zN8@bQ2#yJ6%"
  ]
}
```

Strength labels are derived from the entropy estimate:

| Entropy | Label |
| --- | --- |
| < 36 bits | Weak |
| 36–59 bits | Moderate |
| 60–79 bits | Strong |
| ≥ 80 bits | Very Strong |

Entropy is estimated as `length × log2(pool size)` for passwords and
`word count × log2(available words)` for passphrases. **This is an estimate of
the generator's output space, not a guarantee about real-world resistance to
cracking.**

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | Invalid input or generation failure (message printed to stderr) |

```bash
$ passgen --length 3
error: password length must be between 4 and 256 (got 3)
$ echo $?
1
```

## Security

- All randomness comes from `crypto/rand`; `math/rand` is never used.
- Random indices are produced with `crypto/rand.Int`, which avoids modulo bias.
- The final shuffle is a Fisher-Yates shuffle driven by secure randomness.
- Passwords are generated entirely locally.
- Nothing is sent over the network — the tool makes no network calls at all.
- Generated passwords are never logged, cached or written to disk.
- The passphrase word list is embedded in the binary, so nothing is downloaded
  at runtime.

## Development

```bash
go build ./...
go test ./...
go vet ./...
go test -race ./...
go test -coverprofile=coverage.out ./...
gofmt -l .
```

Or via the Makefile:

```bash
make build    # build ./bin/passgen
make test     # run tests
make vet      # run go vet
make fmt      # format the source
make check    # format check + vet + tests
make run      # go run ./cmd/passgen
make cover    # coverage profile and summary
make clean    # remove build artifacts
```

### Project structure

```text
.
├── cmd/passgen/main.go          # entry point, keeps main() tiny
├── internal/cli/                # flag parsing and output formatting
├── internal/config/             # configuration model and validation
├── internal/generator/          # secure random, charsets, password/passphrase, strength
└── internal/wordlist/           # embedded passphrase word list
```

## Possible future enhancements

- Optional clipboard integration (intentionally omitted to stay dependency-free)
- Shell completion scripts
- Pronounceable password mode

## License

Released under the [MIT License](LICENSE).
