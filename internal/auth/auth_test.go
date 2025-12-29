package auth

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateJWT_Success(t *testing.T) {
	os.Setenv( //nolint:errcheck // test fixture
	"JWT_SECRET", "test-secret-key-for-testing")
	defer os.Unsetenv( //nolint:errcheck // test cleanup
	"JWT_SECRET")

	token, err := GenerateJWT("user-123", "test@example.com")

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, len(token) > 50, "JWT should be reasonably long")
	assert.Equal(t, 3, len(strings.Split(token, ".")), "JWT should have 3 parts")
}

func TestGenerateJWT_MissingSecret(t *testing.T) {
	os.Unsetenv( //nolint:errcheck // test cleanup
	"JWT_SECRET")

	_, err := GenerateJWT("user-123", "test@example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET not set")
}

func TestValidateJWT_ValidToken(t *testing.T) {
	os.Setenv( //nolint:errcheck // test fixture
	"JWT_SECRET", "test-secret-key-for-testing")
	defer os.Unsetenv( //nolint:errcheck // test cleanup
	"JWT_SECRET")

	token, err := GenerateJWT("user-123", "test@example.com")
	require.NoError(t, err)

	claims, err := ValidateJWT(token)

	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	os.Setenv( //nolint:errcheck // test fixture
	"JWT_SECRET", "test-secret-key-for-testing")
	defer os.Unsetenv( //nolint:errcheck // test cleanup
	"JWT_SECRET")

	// create an expired token
	claims := Claims{
		UserID: "user-123",
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret-key-for-testing"))
	require.NoError(t, err)

	_, err = ValidateJWT(tokenString)

	assert.Error(t, err, "Expired token should be rejected")
}

func TestValidateJWT_TamperedToken(t *testing.T) {
	os.Setenv( //nolint:errcheck // test fixture
	"JWT_SECRET", "test-secret-key-for-testing")
	defer os.Unsetenv( //nolint:errcheck // test cleanup
	"JWT_SECRET")

	token, err := GenerateJWT("user-123", "test@example.com")
	require.NoError(t, err)

	// tamper with the token by changing a character
	tamperedToken := token[:len(token)-5] + "XXXXX"

	_, err = ValidateJWT(tamperedToken)
	assert.Error(t, err, "tampered token should be rejected")
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	os.Setenv( //nolint:errcheck // test fixture
	"JWT_SECRET", "test-secret-key-for-testing")
	token, err := GenerateJWT("user-123", "test@example.com")
	require.NoError(t, err)

	// change the secret
	os.Setenv( //nolint:errcheck // test fixture
	"JWT_SECRET", "different-secret-key")
	defer os.Unsetenv( //nolint:errcheck // test cleanup
	"JWT_SECRET")

	_, err = ValidateJWT(token)

	assert.Error(t, err, "token signed with different secret should be rejected")
}

func TestValidateJWT_AlgorithmConfusionAttack(t *testing.T) {
	os.Setenv( //nolint:errcheck // test fixture
	"JWT_SECRET", "test-secret-key-for-testing")
	defer os.Unsetenv( //nolint:errcheck // test cleanup
	"JWT_SECRET")

	claims := Claims{
		UserID: "attacker",
		Email:  "attacker@evil.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// attempt to use different signing method
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType) //nolint:errcheck // test code

	_, err := ValidateJWT(tokenString)
	assert.Error(t, err, "token with 'none' algorithm should be rejected")
}

func TestValidateJWT_MalformedToken(t *testing.T) {
	os.Setenv( //nolint:errcheck // test fixture
	"JWT_SECRET", "test-secret-key-for-testing")
	defer os.Unsetenv( //nolint:errcheck // test cleanup
	"JWT_SECRET")

	malformedTokens := []string{
		"",
		"not.a.jwt",
		"only.two",
		"too.many.parts.in.this.token",
		"<script>alert('xss')</script>",
	}

	for _, token := range malformedTokens {
		_, err := ValidateJWT(token)
		assert.Error(t, err, "malformed token '%s' should be rejected", token)
	}
}

func TestJWT_TokenExpiration(t *testing.T) {
	os.Setenv( //nolint:errcheck // test fixture
	"JWT_SECRET", "test-secret-key-for-testing")
	defer os.Unsetenv( //nolint:errcheck // test cleanup
	"JWT_SECRET")

	token, err := GenerateJWT("user-123", "test@example.com")
	require.NoError(t, err)

	claims, err := ValidateJWT(token)
	require.NoError(t, err)

	// verify expiration is set to 7 days
	expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
	actualExpiry := claims.ExpiresAt.Time
	timeDiff := actualExpiry.Sub(expectedExpiry).Abs()

	assert.Less(t, timeDiff, 5*time.Second, "expiration should be approximately 7 days from now")
}

func TestJWT_ClaimsIntegrity(t *testing.T) {
	os.Setenv( //nolint:errcheck // test fixture
	"JWT_SECRET", "test-secret-key-for-testing")
	defer os.Unsetenv( //nolint:errcheck // test cleanup
	"JWT_SECRET")

	testCases := []struct {
		userID string
		email  string
	}{
		{"user-123", "test@example.com"},
		{"user-456", "another@example.com"},
		{"user-789-with-special-chars", "user+tag@example.com"},
	}

	for _, tc := range testCases {
		token, err := GenerateJWT(tc.userID, tc.email)
		require.NoError(t, err)

		claims, err := ValidateJWT(token)
		require.NoError(t, err)

		assert.Equal(t, tc.userID, claims.UserID, "userID should match")
		assert.Equal(t, tc.email, claims.Email, "email should match")
	}
}
