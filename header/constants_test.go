package header

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeaderConstants(t *testing.T) {
	assert.Equal(t, "Accept", Accept)
	assert.Equal(t, "Access-Control-Allow-Origin", AccessControlAllowOrigin)
	assert.Equal(t, "Access-Control-Allow-Methods", AccessControlAllowMethods)
	assert.Equal(t, "Access-Control-Allow-Headers", AccessControlAllowHeaders)
	assert.Equal(t, "Access-Control-Allow-Credentials", AccessControlAllowCredentials)
	assert.Equal(t, "Access-Control-Expose-Headers", AccessControlExposeHeaders)
	assert.Equal(t, "Authorization", Authorization)
	assert.Equal(t, "Cache-Control", CacheControl)
	assert.Equal(t, "Content-Type", ContentType)
	assert.Equal(t, "Content-Security-Policy", ContentSecurityPolicy)
	assert.Equal(t, "ETag", ETag)
	assert.Equal(t, "If-Match", IfMatch)
	assert.Equal(t, "Location", Location)
	assert.Equal(t, "X-Request-ID", XRequestID)
}
