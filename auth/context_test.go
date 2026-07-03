package auth

import (
	"testing"

	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTFromContext(t *testing.T) {
	t.Run("returns nil when no token in context", func(t *testing.T) {
		token := JWTFromContext(t.Context())
		assert.Nil(t, token)
	})

	t.Run("returns token from context", func(t *testing.T) {
		token := jwt.New()
		require.NoError(t, token.Set(jwt.SubjectKey, "test-subject"))

		ctx := ContextWithJWT(t.Context(), token)

		retrieved := JWTFromContext(ctx)
		require.NotNil(t, retrieved)

		subject, ok := retrieved.Subject()
		require.True(t, ok)
		assert.Equal(t, "test-subject", subject)
	})
}
