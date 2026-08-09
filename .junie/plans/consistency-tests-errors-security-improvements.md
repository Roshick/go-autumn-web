---
sessionId: session-260809-162601-1b96
---

# Findings

### Overview & Goals
Analysis of `go-autumn-web` (Go 1.25, chi/render/jwx/otel-based middleware & transport library) across four axes: naming consistency, test coverage, error handling, and security/bugs. The plan introduces a consistent **functional-options API generation** next to the existing constructors (old ones marked `// Deprecated:`), fixes defects in place, raises coverage to ~85% per package, and aligns the README with reality. Breaking ideas are collected in the **V2** tab.

### 1. Naming & Parameter-Passing Inconsistencies

 Issue | Location |
---|---|
 Positional params instead of options: `NewBasicAuthTransport(rt, username, password, opts)`, `NewRequestMetricsTransport(base, clientName, opts)`, `NewRequiredHeaderMiddleware(headerName, opts)` | `auth/transport.go`, `metrics/transport.go`, `validation/middleware.go` |
 Base round tripper named `base` in metrics vs `rt` everywhere else | `metrics/transport.go:33` |
 `DefaultRequestIDLoggerMiddlewareOptions()` returns a **value** (all others return pointers) and `NewRequestIDLoggerMiddleware` duplicates the defaults inline instead of calling it | `tracing/middleware.go:98–109` |
 Param naming `options` vs `opts` (`AllowBasicAuthUser(options ...)` vs `AllowBearerTokenUser(opts ...)`) | `auth/middleware.go:24,55` |
 Return type `func(next http.Handler) http.Handler` vs `func(http.Handler) http.Handler` | `logging/middleware.go` vs rest |
 Empty options structs kept and stored (`BasicAuthTransportOptions{}`, `RequestMetricsTransportOptions{}`, `ContextLoggerMiddlewareOptions{}`) | multiple |
 String literals instead of `header` constants: `"Authorization"` (`auth/transport.go:26`), `"Access-Control-Max-Age"` (`security/middleware.go:72`); missing constants `Origin`, `Vary`, `AccessControlMaxAge` | `auth`, `security`, `header` |
 `validation` uses its own `requestBodyContextKey[B]` instead of `contextutils` | `validation/context.go` |
 Leftover dev comments: `// SECURITY FIX:`, `// FIX:`, `// Add the missing opts field`, `// Direct string conversion instead of ...` | `security/middleware.go`, `testutils/transport.go:104`, `tracing/context.go:14` |

**README drift:**
- `auth.NewPermissionMiddleware` / `PermissionFn` (README:89–97) **does not exist** → remove (per decision: fix README only).
- `testutils.NewMockInteractionRoundTripper` (README:225) vs actual `NewMockInteractionTransport` — the new API will adopt the README's `RoundTripper` name.
- "security headers" (README:13,102) and "Timeout handling" (README:162) are not implemented → reword.
- Circuit breaker, `RequestLoggerTransport`, `RequestMetricsTransport`, `BasicAuthTransport`, `RequestIDHeaderTransport` exist but are **undocumented**.
- Go badge says 1.23+, `go.mod` says 1.25.

### 2. Test Coverage (current)

 Package | Coverage | Gaps |
---|---|---|
 `errors` | **0%** | no tests at all |
 `auth` | **41.9%** | `ContextJWTMiddleware`, `AllowBearerTokenUser`, `key_provider.go`, `context.go` all 0% |
 `logging` | **58.7%** | `NewRequestLoggerMiddleware` 0%, cancellation branches 56% |
 `testutils` | **59.1%** | entire `http.go` (helpers, `MustParseResponse`, `PerformHTTPRequest`) 0% |
 `tracing` | 83.6% | fallback/edge branches |
 `validation` | 86.2% | render-error paths |
 `header` | n/a | constants only |
 others | 95–100% | fine |

### 3. Error-Handling Defects
- `panic(err)` when `render.Render` fails — in `auth/middleware.go:97,132`, `validation/middleware.go:34,66`, and worst: **inside the panic-recovery deferred func** (`resiliency/middleware.go:38`), where a render failure re-panics and aborts the connection the middleware was supposed to protect.
- `metrics/transport.go:57–72`: instrument-creation errors silently discarded (`_`), inconsistent with `metrics/middleware.go` which logs and degrades to no-op.
- `testutils/transport.go:170`: `RoundTrip` returns `nil, nil` — violates the `http.RoundTripper` contract and panics `http.Client` callers.
- `testutils/http.go:64`: `MustParseResponse` logs the wrong (always-nil) `err` instead of `innerErr`.
- `testutils/http.go:92`: `PerformHTTPRequest` marshals a `nil` body into a literal `"null"` payload; also hardcodes `http.DefaultClient` so the mock transport can't be injected.
- `logging/middleware.go:62–68`: `ContextCancellationLoggerMiddleware` returns without writing a response on an already-cancelled context → implicit `200 OK`.
- `logging/transport.go:59`: `time.Now().Sub(startTime)` instead of `time.Since`; stray `break` in `testutils/transport.go:156`.

### 4. Security Issues & Bugs
- **Panic without chi** (bug): `metrics/middleware.go:47–48` dereferences `chi.RouteContext(...)` which is `nil` when the middleware is used outside a chi router.
- **Unverified JWT in context** (security): `auth/middleware.go:129` parses with `jwt.WithVerify(false)` and stores the token in the request context with no prominent warning and no way to opt into verification.
- **CORS** (`security/middleware.go`): mutates the caller's options struct (`opts.AllowCredentials = false`); treats **every** `OPTIONS` request as a preflight (even without `Origin`/`Access-Control-Request-Method`); never sets `Vary: Origin` (cache-poisoning risk when `AllowOrigin` is origin-specific).
- **Request-ID injection**: `tracing/middleware.go:79–83` echoes the inbound `X-Request-ID` into response header, context, and logs without any length/charset validation (log-injection / header-abuse vector).
- **Unbounded request body** (DoS): `validation/middleware.go:32` decodes `req.Body` with no `http.MaxBytesReader` limit.
- `auth/key_provider.go`: `FetchKeys` has a value receiver, so `p.fetcher = ...` (line 33) never persists; constructor stacks two `jwk.WithFetchWhitelist` options (line 12 + 16) — behavior depends on jwx's last-wins semantics and is untested.

# Technical Design

### Key Decisions (validated with user)
1. **Functional options** for the new API generation; old constructors stay and are marked `// Deprecated:` pointing to their replacement.
2. **New constructor names drop the `New` prefix.** Middlewares: `auth.AuthorizationMiddleware(opts ...)`. For transports the old exported struct names (`BasicAuthTransport`, …) must remain for compatibility, so the new transport constructors use the **`RoundTripper` suffix** (matching stdlib vocabulary and the name the README already uses): `auth.BasicAuthRoundTripper(...)`, `testutils.MockInteractionRoundTripper(...)`. Their concrete types are unexported; they return `http.RoundTripper`.
3. **Per-component option types** (e.g. `metrics.RequestMetricsRoundTripperOption`) for compile-time safety; `With*` functions are prefixed per component where names would clash within a package (e.g. `auth.WithAuthorizationErrorResponse` vs `auth.WithJWTErrorResponse`).
4. **Behavior-changing bugfixes land in place** (contract violations and panics are defects, not API guarantees).

### New API Contracts (examples)
```go
// Middlewares: <Name>Middleware(opts ...<Name>MiddlewareOption) func(http.Handler) http.Handler
func AuthorizationMiddleware(opts ...AuthorizationMiddlewareOption) func(http.Handler) http.Handler
func WithAuthorizationFns(fns ...AuthorizationFn) AuthorizationMiddlewareOption
func WithAuthorizationErrorResponse(r render.Renderer) AuthorizationMiddlewareOption

// Positional params become options:
func RequiredHeaderMiddleware(opts ...RequiredHeaderMiddlewareOption) func(http.Handler) http.Handler
func WithRequiredHeader(name string) RequiredHeaderMiddlewareOption

// Transports: <Name>RoundTripper(rt http.RoundTripper, opts ...<Name>RoundTripperOption) http.RoundTripper
func BasicAuthRoundTripper(rt http.RoundTripper, opts ...BasicAuthRoundTripperOption) http.RoundTripper
func WithBasicAuthCredentials(username, password string) BasicAuthRoundTripperOption
func RequestMetricsRoundTripper(rt http.RoundTripper, opts ...RequestMetricsRoundTripperOption) http.RoundTripper
func WithMetricsClientName(name string) RequestMetricsRoundTripperOption
func MockInteractionRoundTripper(t *testing.T, opts ...MockInteractionRoundTripperOption) *MockInteractionTransport // reuses existing type
```
Every deprecated symbol gets `// Deprecated: use <replacement>.` so IDEs/staticcheck flag usages. Old constructors are re-implemented as thin wrappers over the new ones to avoid logic duplication.

### Error-Handling Changes (in place)
- Add `errors.MustRender(w, req, renderer)` (name TBD): calls `render.Render`; on failure logs via `aulogging` and falls back to `http.Error(w, ..., 500)` — **never panics**. All `panic(err)` render sites in `auth`, `validation`, `resiliency` switch to it (critical: `resiliency` no longer re-panics inside its own recovery handler).
- `metrics/transport.go`: log instrument-creation errors and degrade to no-op, mirroring `metrics/middleware.go`.
- `testutils.MockInteractionTransport.RoundTrip`: when no response is configured, return a synthetic `204`-style `*http.Response` with empty body instead of `nil, nil` (documented); never return `nil, nil`.
- `testutils/http.go`: fix wrong error variable in `MustParseResponse`; `PerformHTTPRequest` skips the body for `nil` and gains a client-injection variant (`PerformHTTPRequestWithClient`).
- `logging.ContextCancellationLoggerMiddleware`: write `errors.NewTimeoutResponse()` (408) when the context is already cancelled instead of an implicit 200.
- `logging/transport.go`: `time.Since`; remove stray `break` in `testutils`.

### Security & Bug Fixes (in place)
- **metrics middleware**: nil-check `chi.RouteContext`; fall back to `req.Pattern` (Go 1.22+ ServeMux) or `""` — no more panic outside chi.
- **CORS**: copy options instead of mutating the caller's struct; only short-circuit `OPTIONS` when it is an actual preflight (`Origin` + `Access-Control-Request-Method` present); always set `Vary: Origin` when `AllowOrigin != "*"`; use new `header.AccessControlMaxAge` constant.
- **Request ID**: validate inbound header in `RequestIDHeaderMiddleware` (max length ~128, charset `[A-Za-z0-9._-]`); regenerate on violation.
- **Request body**: new `validation.ContextRequestBodyMiddleware[B]` gains `WithMaxBodyBytes(n)` (default 1 MiB, `0` = unlimited) using `http.MaxBytesReader`; legacy constructor keeps unlimited behavior for compatibility.
- **JWT**: new `auth.ContextJWTMiddleware` gains `WithJWTParseOptions(...jwt.ParseOption)` so verification can be enabled; both old and new get a prominent doc comment stating the default parses **without signature verification**.
- **key_provider**: pointer receiver for `FetchKeys` (fetcher assignment persists), resolve the double-whitelist option stacking, add tests with a mock `jwk.Fetcher`.

### Naming/Docs Cleanup
- Add `header.Origin`, `header.Vary`, `header.AccessControlMaxAge`; replace string literals in `auth/transport.go` and `security/middleware.go`.
- `DefaultRequestIDLoggerMiddlewareOptions` gets a pointer-returning sibling; `NewRequestIDLoggerMiddleware` calls it; old value-returning func deprecated.
- Unify parameter naming (`rt`, `opts`) and return type `func(http.Handler) http.Handler` in new APIs; remove leftover `// FIX:`-style comments.
- README: remove `PermissionMiddleware` section, correct `MockInteractionRoundTripper` usage to the (new, now-real) API, reword feature bullets to match reality, document all transports/roundtrippers, add a Deprecation/Migration section, fix Go version badge.

### File Structure
- New API code lives **next to** old code in the same files (`auth/middleware.go`, `auth/transport.go`, `security/middleware.go`, `logging/middleware.go`, `logging/transport.go`, `metrics/middleware.go`, `metrics/transport.go`, `resiliency/middleware.go`, `resiliency/transport.go`, `tracing/middleware.go`, `tracing/transport.go`, `validation/middleware.go`, `testutils/transport.go`, `testutils/http.go`); option types may go into per-package `options.go` files if files grow too large.
- `errors/render.go` (new): the `MustRender`-style safe render helper.
- New tests in existing `*_test.go` files plus new `errors/responses_test.go`, `auth/key_provider_test.go`, `auth/context_test.go`, `testutils/http_test.go`.

### Risks
- Old wrappers delegating to new constructors must be bit-for-bit behavior-compatible (except the agreed in-place bugfixes) — covered by characterization tests written **before** refactoring.
- `With*` name collisions inside packages with several components — mitigated by per-component prefixes.
- Coverage of `otel` no-op error branches is hard; ~85% target allows skipping unreachable fallbacks (e.g. `crypto/rand` failure).

# V2 (Breaking)

### V2 — proposed breaking changes (documentation only in this plan; module path `github.com/Roshick/go-autumn-web/v2`)

**API surface**
- Remove all `New*`/`Default*Options` deprecated constructors and the exported legacy transport structs (`BasicAuthTransport`, `RequestMetricsTransport`, `RequestLoggerTransport`, `CircuitBreakerTransport`, `RequestIDHeaderTransport`, `MockInteractionTransport`) — only the functional-options generation remains.
- Drop empty options structs entirely.

**Context accessors**
- `contextutils.GetValue[B]` → `(B, bool)` instead of `*B`.
- `tracing.RequestIDFromContext` → `(string, bool)` instead of `*string`.
- `validation.RequestBodyFromContext[B]` → `(B, bool)` instead of silent zero value; migrate its bespoke context key onto `contextutils`.

**Behavior**
- `auth.ContextJWTMiddleware` verifies signatures **by default**; opting out becomes explicit (`WithoutVerification()`), likely wired to `RemoteKeySetProvider` out of the box.
- `validation` body middleware: 1 MiB limit and `DisallowUnknownFields` become the defaults.
- CORS middleware redesigned around an **origin allow-list** (echo matched `Origin`, not a single static value), correct preflight semantics, configurable allow-methods.
- `errors.ErrorResponse` becomes RFC 7807 `application/problem+json` (or at minimum renders a documented JSON body contract incl. request ID).
- `resiliency.CircuitBreakerTransportOptions` stops embedding `gobreaker.Settings` directly (decouples public API from a third-party type).

**Misc**
- `logging` field constants move to dot-notation (`request.method`) aligning with OTel semantic conventions.
- `testutils.PerformHTTPRequest` takes an `*http.Client` parameter; `TestResponse.Body` matching becomes content-type-aware by contract.

# Testing

### Validation Approach
- Characterization tests are written for existing behavior **before** the deprecating refactor, then the new functional-options constructors are asserted to behave identically (table-driven, both generations run through the same assertions via `httptest`).
- `go test ./... -cover` gates each stage; target **~85% per package**; `go vet` clean.

### Key Scenarios
- Middlewares (old + new constructor): default options, custom options, error-response path, happy path — via `httptest.Server`/`httptest.NewRecorder` with chi router.
- RoundTrippers (old + new): header injection (`BasicAuth`, `RequestID`), logging/metrics recording, circuit-breaker open/close — via stubbed `http.RoundTripper`.
- `errors`: every `New*Response` renders the expected status; `Render` sets status via `render.Status`; new safe-render helper falls back to plain 500 on marshal failure without panicking.
- `auth`: `ContextJWTMiddleware` (valid/malformed/missing bearer), `AllowBearerTokenUser` with HS256 key, `key_provider.FetchKeys` with mock `jwk.Fetcher` (kid found/missing, fetch error, whitelist).
- `logging`: `RequestLoggerMiddleware` info vs warn threshold, missing context logger; cancellation middleware pre-/post-cancel writes 408.
- `testutils`: `MustParseResponse` (JSON + plain text + malformed JSON), `PerformHTTPRequest` nil-body, `RoundTrip` no-response returns a valid non-nil response.

### Edge Cases (regression tests for fixed bugs)
- Metrics middleware mounted on plain `http.ServeMux` (no chi) — must not panic.
- CORS: non-preflight `OPTIONS` reaches the handler; caller's options struct unmodified after use; `Vary: Origin` present for non-wildcard origin.
- Inbound `X-Request-ID` with newline/overlong value is replaced by a generated ID.
- Panic-recovery middleware with a failing `ErrorResponse` renderer — request ends in 500, no escaped panic.
- Body middleware with `WithMaxBodyBytes`: oversized body → 400, no unbounded read.

### Test Changes
- New: `errors/responses_test.go`, `errors/render_test.go`, `auth/key_provider_test.go`, `auth/context_test.go`, `testutils/http_test.go`.
- Extended: all existing `*_test.go` files gain new-API cases and regression cases listed above.
- Skipped intentionally: `crypto/rand` failure fallback in `DefaultRequestIDGenerator`, otel meter no-op internals.

# Delivery Steps

###   Step 1: Fix error-handling defects and contract violations in place
All panics-on-render, RoundTripper contract violations and helper bugs are fixed with regression tests; existing tests still pass.

- Add safe-render helper in `errors/render.go` (logs + plain-500 fallback, never panics) and replace all `panic(err)` render sites in `auth/middleware.go`, `validation/middleware.go`, `resiliency/middleware.go`.
- `metrics/transport.go`: log instrument-creation errors and degrade to no-op (mirror `metrics/middleware.go`).
- `testutils/transport.go`: `RoundTrip` returns a valid empty `*http.Response` instead of `nil, nil`; remove stray `break` and leftover comments.
- `testutils/http.go`: fix wrong error variable in `MustParseResponse`; skip marshaling `nil` bodies in `PerformHTTPRequest`; add `PerformHTTPRequestWithClient`.
- `logging`: cancellation middleware writes 408 (`errors.NewTimeoutResponse`) on already-cancelled context; `time.Since` in `logging/transport.go`.
- Add regression tests for each fix in the affected packages.

###   Step 2: Fix security issues and runtime bugs in place
The metrics middleware no longer panics outside chi, CORS behaves per spec without side effects, and request IDs are sanitized.

- `metrics/middleware.go`: nil-check `chi.RouteContext`, fall back to `req.Pattern`/empty route.
- `security/middleware.go`: copy options instead of mutating caller's struct; only short-circuit real preflights (`Origin` + `Access-Control-Request-Method`); set `Vary: Origin` for non-wildcard origins; remove `// SECURITY FIX`/`// FIX` comments.
- `tracing/middleware.go`: validate inbound `X-Request-ID` (length ≤ 128, charset `[A-Za-z0-9._-]`), regenerate on violation.
- `auth/key_provider.go`: pointer receiver for `FetchKeys`, resolve double `WithFetchWhitelist` stacking.
- Add `header.Origin`, `header.Vary`, `header.AccessControlMaxAge` constants and replace string literals in `auth/transport.go` and `security/middleware.go`.
- Regression tests: plain `ServeMux` metrics, non-preflight OPTIONS, options-struct immutability, `Vary` header, request-ID injection attempts.

###   Step 3: Introduce functional-options middleware constructors, deprecate old ones
Every middleware has a new `<Name>Middleware(opts ...<Name>MiddlewareOption)` constructor next to the deprecated `New<Name>Middleware(opts *Options)` one.

- Add new constructors + per-component option types with `With*` functions in `auth`, `security`, `logging`, `metrics`, `resiliency`, `tracing`, `validation` (e.g. `auth.AuthorizationMiddleware`, `validation.RequiredHeaderMiddleware` with `WithRequiredHeader(name)` replacing the positional param).
- New `validation.ContextRequestBodyMiddleware[B]` gains `WithMaxBodyBytes(n)` (default 1 MiB); new `auth.ContextJWTMiddleware` gains `WithJWTParseOptions(...)` plus prominent unverified-by-default doc comments on both generations.
- Re-implement old constructors as thin wrappers over the new ones; mark them and `Default*Options` funcs `// Deprecated: use <replacement>.`; fix `DefaultRequestIDLoggerMiddlewareOptions` value/pointer inconsistency.
- Characterization tests written first assert old and new constructors behave identically; new-API cases added to all middleware test files.

###   Step 4: Introduce functional-options RoundTripper constructors, deprecate old transports
Every transport has a new `<Name>RoundTripper(rt http.RoundTripper, opts ...Option) http.RoundTripper` constructor; old structs and `New*Transport` constructors are deprecated.

- Add `auth.BasicAuthRoundTripper` (`WithBasicAuthCredentials(username, password)`), `logging.RequestLoggerRoundTripper`, `metrics.RequestMetricsRoundTripper` (`WithMetricsClientName(name)`), `resiliency.CircuitBreakerRoundTripper`, `tracing.RequestIDHeaderRoundTripper`, `testutils.MockInteractionRoundTripper` — the last matching the name the README already documents.
- Unify parameter naming (`rt` everywhere, `base` → `rt` in metrics); use `header.Authorization` constant in `auth`.
- Mark old constructors/structs `// Deprecated:` and delegate their logic to the new implementations.
- Extend `*_test.go` in each package to cover both generations via stubbed round trippers.

###   Step 5: Raise test coverage to ~85% per package
`go test ./... -cover` reports ≥85% for every package with statements.

- `errors` (0% → 85%+): new `errors/responses_test.go` covering all `New*Response` constructors, `Render` status handling, and the safe-render helper fallback.
- `auth` (42% → 85%+): tests for `ContextJWTMiddleware` (valid/malformed/absent bearer), `AllowBearerTokenUser` (HS256 signed/expired token), `context.go` helpers, and `key_provider.go` with a mock `jwk.Fetcher` (kid found/missing, fetch error).
- `logging` (59% → 85%+): `RequestLoggerMiddleware` info/warn threshold and no-context-logger paths; remaining cancellation branches.
- `testutils` (59% → 85%+): new `testutils/http_test.go` for `MustParseResponse`, `MustReadResponseFromFile`, `Require*` helpers and `PerformHTTPRequest` against an `httptest.Server`.
- Top up `tracing` and `validation` edge branches; document intentionally skipped unreachable fallbacks.

###   Step 6: Align README, docs and naming cleanup
README accurately documents the actual and new APIs, including deprecations and a migration guide.

- Remove the non-existent `auth.NewPermissionMiddleware` section; reword "security headers", "timeout handling" and feature bullets to match reality; fix Go version badge (1.25).
- Update `testutils` example to the now-real `MockInteractionRoundTripper`; document all RoundTrippers (basic auth, logging, metrics, circuit breaker, request-ID) with examples.
- Add a "Deprecations & Migration" section mapping every old constructor to its functional-options replacement, and a "V2 Roadmap" section with the breaking changes from the V2 tab.
- Sweep remaining leftover dev comments and ensure every exported symbol has a godoc comment; final `go vet ./...` and full `go test ./... -cover` run.