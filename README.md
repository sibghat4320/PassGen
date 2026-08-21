# passgen — Secure password & passphrase generator

passgen is a small, dependency-free command-line tool written in Go that
produces cryptographically secure passwords and passphrases. It uses
crypto/rand for all randomness, guarantees inclusion of enabled character
categories, and provides both machine-friendly JSON output and an interactive
terminal UI.

Highlights
- Cryptographically secure (crypto/rand)
- No third-party runtime dependencies — single static binary
- Passwords (4–256 chars) and passphrases (3–12 words)
- Entropy estimation and strength labels
- JSON output and quiet mode for scripting
- Cross-platform: macOS, Linux, Windows

Installation
Clone and build (replace username with your GitHub handle if needed):

```bash
git clone https://github.com/sibghat4320/password_generator.git
cd password_generator
go build -o passgen ./cmd/passgen
```

Or install with `go install`:

```bash
go install github.com/sibghat4320/password_generator/cmd/passgen@latest
```

Quick usage
```
# Generate one 16-character password (default)
passgen

# Generate five 24-character passwords
passgen --count 5 --length 24

# Generate a 6-word passphrase joined with _
passgen --passphrase --words 6 --separator _

# Machine readable output
passgen --count 2 --json
```

Contributing
- Clone the repo, run `go test ./...`, and open a PR for changes.
- Keep changes small and document reasoning in PR descriptions.

License
Released under the MIT License. See LICENSE for details.
