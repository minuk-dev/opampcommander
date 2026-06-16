package security_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
)

func newHasher(t *testing.T, pepper string) *security.PasswordHasher {
	t.Helper()

	//exhaustruct:ignore
	cfg := &security.Config{
		BasicAuthSettings: security.BasicAuthSettings{Pepper: pepper},
	}

	return security.NewPasswordHasher(cfg)
}

func TestPasswordHasher_HashAndVerify(t *testing.T) {
	t.Parallel()

	hasher := newHasher(t, "super-secret-pepper")
	require.True(t, hasher.Enabled())

	hash, err := hasher.Hash("correct horse battery staple")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotContains(t, hash, "correct horse battery staple", "hash must not contain the plaintext")

	require.NoError(t, hasher.Verify("correct horse battery staple", hash))

	err = hasher.Verify("wrong password", hash)
	require.Error(t, err, "wrong password must not verify")
}

func TestPasswordHasher_DistinctSaltsPerHash(t *testing.T) {
	t.Parallel()

	hasher := newHasher(t, "pepper")

	first, err := hasher.Hash("same-password")
	require.NoError(t, err)

	second, err := hasher.Hash("same-password")
	require.NoError(t, err)

	assert.NotEqual(t, first, second, "per-user salt must make identical passwords hash differently")
	require.NoError(t, hasher.Verify("same-password", first))
	require.NoError(t, hasher.Verify("same-password", second))
}

func TestPasswordHasher_PepperMatters(t *testing.T) {
	t.Parallel()

	hasherA := newHasher(t, "pepper-a")
	hasherB := newHasher(t, "pepper-b")

	hash, err := hasherA.Hash("password")
	require.NoError(t, err)

	require.NoError(t, hasherA.Verify("password", hash))

	err = hasherB.Verify("password", hash)
	require.Error(t, err, "a different pepper must fail verification")
}

func TestPasswordHasher_DisabledWhenNoPepper(t *testing.T) {
	t.Parallel()

	hasher := newHasher(t, "")
	assert.False(t, hasher.Enabled())

	_, err := hasher.Hash("password")
	require.ErrorIs(t, err, security.ErrBasicAuthDisabled)

	err = hasher.Verify("password", "irrelevant")
	require.ErrorIs(t, err, security.ErrBasicAuthDisabled)
}
