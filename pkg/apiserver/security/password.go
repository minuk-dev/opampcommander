package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/awnumar/memguard"
	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher hashes and verifies basic-auth passwords.
//
// Each password is first combined with a server-side pepper via HMAC-SHA256 and then hashed
// with bcrypt (which adds a per-user random salt). The pepper is held in a memguard Enclave so
// the secret is kept in locked, encrypted memory and only decrypted transiently during an
// operation. When no pepper is configured the hasher is disabled and every operation returns
// [ErrBasicAuthDisabled].
type PasswordHasher struct {
	// pepper is nil when basic auth is disabled (no pepper configured).
	pepper *memguard.Enclave
}

// NewPasswordHasher builds a PasswordHasher from the security config.
// When the configured pepper is empty the returned hasher is disabled: Hash and Verify
// return [ErrBasicAuthDisabled]. The plaintext pepper is sealed into a memguard Enclave and
// the caller-visible config string is the only remaining plaintext copy.
func NewPasswordHasher(cfg *Config) *PasswordHasher {
	pepper := cfg.BasicAuthSettings.Pepper
	if pepper == "" {
		return &PasswordHasher{pepper: nil}
	}

	return &PasswordHasher{pepper: memguard.NewEnclave([]byte(pepper))}
}

// Enabled reports whether basic-auth password operations are available (a pepper is configured).
func (h *PasswordHasher) Enabled() bool {
	return h.pepper != nil
}

// Hash returns a one-way hash of the password, peppered and salted, suitable for persistence.
// Returns [ErrBasicAuthDisabled] when no pepper is configured.
func (h *PasswordHasher) Hash(password string) (string, error) {
	peppered, err := h.pepperedPassword(password)
	if err != nil {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword(peppered, bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// Verify checks a plaintext password against a stored hash.
// It returns nil on a match, [ErrBasicAuthDisabled] when no pepper is configured, and a
// non-nil error otherwise (including a wrong password).
func (h *PasswordHasher) Verify(password, hash string) error {
	peppered, err := h.pepperedPassword(password)
	if err != nil {
		return err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), peppered)
	if err != nil {
		return fmt.Errorf("password verification failed: %w", err)
	}

	return nil
}

// pepperedPassword returns base64(HMAC-SHA256(password, pepper)).
// base64 encoding keeps the result free of NUL bytes (bcrypt truncates at the first NUL) and
// at a fixed 44-byte length, well within bcrypt's 72-byte input limit.
// The pepper is opened from the enclave only for the duration of the HMAC and then destroyed.
func (h *PasswordHasher) pepperedPassword(password string) ([]byte, error) {
	if h.pepper == nil {
		return nil, ErrBasicAuthDisabled
	}

	buf, err := h.pepper.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open pepper enclave: %w", err)
	}
	defer buf.Destroy()

	mac := hmac.New(sha256.New, buf.Bytes())
	_, _ = mac.Write([]byte(password)) // hash.Hash.Write never returns an error
	sum := mac.Sum(nil)

	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(sum)))
	base64.StdEncoding.Encode(encoded, sum)

	return encoded, nil
}
