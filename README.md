# go-autumn-web

A comprehensive Go HTTP middleware library providing essential components for building robust web applications and services. This library offers a collection of middleware for authentication, logging, metrics, security, validation, and more.

[![Go Version](https://img.shields.io/badge/go-1.23%2B-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Tests](https://github.com/Roshick/go-autumn-web/workflows/Tests/badge.svg)](https://github.com/Roshick/go-autumn-web/actions)

## Features

- 🔐 **Authentication & Authorization** - JWT, Basic Auth, and permission-based middleware
- 📊 **Metrics & Monitoring** - OpenTelemetry integration for request metrics
- 🔒 **Security** - CORS, security headers, and input validation
- 📝 **Logging** - Structured logging with context propagation
- 🔄 **Resiliency** - Panic recovery and circuit breakers
- 🔍 **Tracing** - Distributed tracing with OpenTelemetry
- ✅ **Validation** - Request body and header validation
- 🧪 **Testing** - Test utilities and mock transports

## Installation

```bash
go get github.com/Roshick/go-autumn-web
```

## Quick Start

```go
package main

import (
    "net/http"
    "time"
    
    "github.com/go-chi/chi/v5"
    "github.com/Roshick/go-autumn-web/security"
    "github.com/Roshick/go-autumn-web/logging"
    "github.com/Roshick/go-autumn-web/metrics"
    "github.com/Roshick/go-autumn-web/resiliency"
	"github.com/Roshick/go-autumn-web/tracing"
)

func main() {
    r := chi.NewRouter()
    
    // Add middleware stack
	r.Use(resiliency.NewPanicRecoveryMiddleware(nil))
	r.Use(security.NewCORSMiddleware(nil))
	r.Use(tracing.NewRequestIDHeaderMiddleware(nil))
	r.Use(logging.NewContextLoggerMiddleware(nil))
	r.Use(tracing.NewRequestIDLoggerMiddleware(nil))
	r.Use(metrics.NewRequestMetricsMiddleware(nil))
	r.Use(logging.NewRequestLoggerMiddleware(nil))
    
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })
    
    http.ListenAndServe(":8080", r)
}
```

## Middleware Components

### 🔐 Authentication (`auth`)

Provides JWT and Basic Authentication middleware with flexible authorization functions.

```go
import "github.com/Roshick/go-autumn-web/auth"

// Basic Authentication
basicAuth := auth.AllowBasicAuthUser(auth.AllowBasicAuthUserOptions{
    Username: "admin",
    Password: "secret",
})

// JWT Authentication
jwtAuth := auth.AllowBearerTokenUser(auth.AllowBearerTokenUserOptions{
    ParseOptions: []jwt.ParseOption{jwt.WithKey(jwa.HS256, []byte("secret"))},
})

// Authorization Middleware
r.Use(auth.NewAuthorizationMiddleware(&auth.AuthorizationMiddlewareOptions{
    AuthorizationFns: []auth.AuthorizationFn{basicAuth, jwtAuth},
}))

// Permission Middleware
r.Use(auth.NewPermissionMiddleware(&auth.PermissionMiddlewareOptions{
    PermissionFns: []auth.PermissionFn{
        func(req *http.Request) bool {
            // Custom permission logic
            return true
        },
    },
}))
```

### 🔒 Security (`security`)

CORS middleware with secure defaults and comprehensive security headers.

```go
import "github.com/Roshick/go-autumn-web/security"

// CORS with custom configuration
r.Use(security.NewCORSMiddleware(&security.CORSMiddlewareOptions{
    AllowOrigin:      "https://yourdomain.com",
    AllowCredentials: true,
    MaxAge:           3600,
    AdditionalAllowHeaders: []string{"X-Custom-Header"},
}))

// Security defaults (wildcard origin, no credentials)
r.Use(security.NewCORSMiddleware(nil))
```

**Security Features:**
- ✅ Prevents wildcard origin with credentials (security vulnerability)
- ✅ Configurable preflight caching
- ✅ Proper HTTP status codes for OPTIONS requests

### 📝 Logging (`logging`)

Structured logging with context propagation and request/response logging.

```go
import "github.com/Roshick/go-autumn-web/logging"

// Context logger (adds logger to request context)
r.Use(logging.NewContextLoggerMiddleware(nil))

// Request logger (logs all HTTP requests)
r.Use(logging.NewRequestLoggerMiddleware(nil))

// Context cancellation logger
r.Use(logging.NewContextCancellationLoggerMiddleware(&logging.ContextCancellationLoggerMiddlewareOptions{
    Description: "api-server",
}))
```

### 📊 Metrics (`metrics`)

OpenTelemetry metrics integration for monitoring HTTP requests.

```go
import "github.com/Roshick/go-autumn-web/metrics"

// Request metrics (duration, status codes, etc.)
r.Use(metrics.NewRequestMetricsMiddleware(nil))
```

**Metrics Collected:**
- Request duration (histogram)
- HTTP status codes
- Request methods
- URL patterns (from chi router)

### 🔄 Resiliency (`resiliency`)

Timeout handling and panic recovery for robust applications.

```go
import "github.com/Roshick/go-autumn-web/resiliency"

// Panic recovery
r.Use(resiliency.NewPanicRecoveryMiddleware(nil))

```

**Features:**
- ✅ Graceful panic recovery with stack traces

### 🔍 Tracing (`tracing`)

Distributed tracing and request ID propagation.

```go
import "github.com/Roshick/go-autumn-web/tracing"

// Request ID generation and propagation
r.Use(tracing.NewRequestIDHeaderMiddleware(nil))

// Tracing logger integration
r.Use(tracing.NewTracingLoggerMiddleware(nil))

// Request ID in logs
r.Use(tracing.NewRequestIDLoggerMiddleware(nil))
```

### ✅ Validation (`validation`)

Request body and header validation middleware.

```go
import "github.com/Roshick/go-autumn-web/validation"

type UserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Request body validation
r.Use(validation.NewContextRequestBodyMiddleware[UserRequest](nil))

// Required header validation
r.Use(validation.NewRequiredHeaderMiddleware("Authorization", nil))

// In your handler
func handleUser(w http.ResponseWriter, r *http.Request) {
    user := validation.RequestBodyFromContext[UserRequest](r.Context())
    // Use validated user data
}
```

### 🧪 Testing (`testutils`)

Mock transports and HTTP testing utilities.

```go
import "github.com/Roshick/go-autumn-web/testutils"

func TestAPI(t *testing.T) {
    mockTransport := testutils.NewMockInteractionRoundTripper(t, nil)
    
    // Setup expected interactions
    mockTransport.ExpectRequest(testutils.TestRequest{
        Method: "GET",
        URL:    "https://api.example.com/users",
    }).WillReturnResponse(&testutils.TestResponse{
        Status: 200,
        Body:   `{"users": []}`,
    })
    
    client := &http.Client{Transport: mockTransport}
    // Test your code with the mock client
}
```

## Error Handling

All middleware components provide customizable error responses:

```go
import "github.com/Roshick/go-autumn-web/errors"

// Custom error responses
authMiddleware := auth.NewAuthorizationMiddleware(&auth.AuthorizationMiddlewareOptions{
    ErrorResponse: errors.NewAuthenticationRequiredResponse(),
})
```

## Configuration

### Recommended Middleware Stack

```go
func SetupMiddleware(r chi.Router) {
    // 1. Panic recovery (outermost)
    r.Use(resiliency.NewPanicRecoveryMiddleware(nil))
    
    // 2. Security headers
    r.Use(security.NewCORSMiddleware(nil))
    
    // 3. Request ID generation
    r.Use(tracing.NewRequestIDHeaderMiddleware(nil))
    
    // 4. Logging setup
    r.Use(logging.NewContextLoggerMiddleware(nil))
    r.Use(tracing.NewRequestIDLoggerMiddleware(nil))
    
    // 5. Metrics collection
    r.Use(metrics.NewRequestMetricsMiddleware(nil))
    
    // 6. Request logging
    r.Use(logging.NewRequestLoggerMiddleware(nil))
	
    
    // 8. Authentication/Authorization (route-specific)
    // Add these to specific route groups as needed
}
```

## Requirements

- Go 1.23 or later
- Compatible with `net/http` and `chi` router
- OpenTelemetry for metrics and tracing

## Dependencies

- `github.com/go-chi/chi/v5` - HTTP router
- `github.com/go-chi/render` - Response rendering
- `go.opentelemetry.io/otel` - Observability
- `github.com/lestrrat-go/jwx/v3` - JWT handling
- `github.com/StephanHCB/go-autumn-logging` - Logging framework

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
