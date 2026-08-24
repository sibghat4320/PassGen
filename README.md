# passgen — Secure password & passphrase generator

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**passgen** is a fast, lightweight command-line tool for generating cryptographically secure passwords and passphrases. Built in Go with zero third-party dependencies, it delivers a single static binary that works everywhere.

## Features

- 🔐 **Cryptographically secure** — uses `crypto/rand` for all randomness
- ⚡ **Zero dependencies** — single static binary, no runtime requirements  
- 🎯 **Flexible generation** — passwords (4–256 chars) and passphrases (3–12 words)
- 📊 **Entropy estimation** — includes strength labels for generated secrets
- 🖥️ **Multiple output modes** — interactive TUI, JSON for scripting, quiet mode
- 🌍 **Cross-platform** — macOS, Linux, Windows

## Installation

### Using `go install`

```bash
go install github.com/sibghat4320/PassGen/cmd/passgen@latest
```

### Build from source

```bash
git clone https://github.com/sibghat4320/PassGen.git
cd PassGen
go build -o passgen ./cmd/passgen
```

The binary works standalone—copy it anywhere or add to your `$PATH`.

## Usage

### Interactive mode (default)

Simply run `passgen` to open the interactive terminal UI where you can customize all options.

```bash
passgen
```

### Generate passwords

```bash
# One 16-character password (default)
passgen --length 16

# Five 24-character passwords
passgen --count 5 --length 24

# Passwords with specific character sets
passgen --length 20 --no-uppercase  # lowercase + digits + symbols only
```

### Generate passphrases

```bash
# One 4-word passphrase
passgen --passphrase

# Six words joined with underscores
passgen --passphrase --words 6 --separator _

# Three words with hyphens
passgen --passphrase --words 3 --separator -
```

### Machine-readable output

```bash
# JSON output for scripting
passgen --count 2 --json

# Quiet mode (output only)
passgen --quiet
```

### Full options

```bash
passgen --help
```

## Why passgen?

- **No online tools** — keep your secrets on your machine
- **Audit-friendly** — open source, small codebase (~500 LOC)
- **Development-ready** — JSON output for integration with scripts and tools
- **Fast** — generates batches instantly, suitable for automation

## Development

Clone and test locally:

```bash
git clone https://github.com/sibghat4320/PassGen.git
cd PassGen
go test ./...
go build -o passgen ./cmd/passgen
```

Contributions welcome! Please:
- Keep changes focused and well-documented
- Add tests for new functionality
- Open an issue or discussion before major refactors

## License

Released under the MIT License. See [LICENSE](LICENSE) for details.
