// Package generator implements secure password and passphrase generation.
// Every random decision is made with crypto/rand.
package generator

import "github.com/yourusername/passgen/internal/config"

// Generate produces cfg.Count secrets, using passphrase mode when requested.
func Generate(cfg config.Config) ([]string, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if cfg.Passphrase {
		return GeneratePassphrases(cfg)
	}
	return GeneratePasswords(cfg)
}
