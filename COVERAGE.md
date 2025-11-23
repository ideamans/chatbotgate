# Test Coverage Improvement Plan

**Current Coverage**: 50.1%
**Target Coverage**: 80%+
**Phase 1 Target**: 65%

**Last Updated**: 2025-11-23

---

## 📊 Current Status

| Package | Coverage | Status | Priority |
|---------|----------|--------|----------|
| **pkg/middleware/assets** | 100.0% | ✅ Excellent | - |
| **pkg/shared/config** | 95.7% | ✅ Excellent | - |
| **pkg/middleware/authz** | 95.8% | ✅ Excellent | - |
| **pkg/middleware/rules** | 91.6% | ✅ Good | - |
| **pkg/middleware/forwarding** | 85.3% | ✅ Good | - |
| **pkg/middleware/password** | 84.8% | ✅ Good | - |
| **pkg/middleware/ratelimit** | 82.0% | ✅ Good | - |
| **pkg/shared/filewatcher** | 77.8% | 🟡 Improve | Medium |
| **pkg/middleware/session** | 76.6% | 🟡 Improve | Medium |
| **pkg/shared/logging** | 74.0% | 🟡 Improve | Medium |
| **pkg/shared/i18n** | 70.3% | 🟡 Improve | Medium |
| **pkg/shared/kvs** | 64.9% | 🟠 Needs Work | High |
| **pkg/middleware/oauth2** | 63.6% | 🟠 Needs Work | High |
| **pkg/middleware/config** | 62.7% | 🟠 Needs Work | High |
| **pkg/middleware/factory** | 55.6% | 🔴 Critical | High |
| **pkg/middleware/email** | 39.4% | 🔴 **CRITICAL** | **P0** |
| **pkg/middleware/core** | 29.6% | 🔴 **CRITICAL** | **P0** |
| **cmd/chatbotgate/server** | 13.3% | 🔴 **CRITICAL** | **P0** |

---

## 🔴 Critical Issues (P0 - Immediate Action Required)

### 1. pkg/middleware/core (29.6% → Target: 60%)

**Security-Critical Functions (0% coverage)**:
- ❌ `isValidRedirectURL()` - Prevents open redirect attacks
- ❌ `isValidEmail()` - Prevents SMTP injection attacks
- ❌ `sanitizeHeaderValue()` - Prevents header injection (87.5% - incomplete)

**Core Handlers (0% coverage)**:
- ❌ `ServeHTTP()` - Main router (12.5% only)
- ❌ `requireAuth()` - Authentication check (47.1% only)
- ❌ `redirectToLogin()` - Login redirect logic (0%)
- ❌ `handleOAuth2Start()` - OAuth2 flow start (0%)
- ❌ `handleOAuth2Callback()` - OAuth2 callback handler (0%)
- ❌ `handleEmailSend()` - Email auth send (0%)
- ❌ `handleEmailVerify()` - Email auth verify (0%)
- ❌ `handlePasswordLogin()` - Password login (0%)
- ❌ `handle404()`, `handle500()`, `handleForbidden()` - Error pages (0%)

**Estimated Effort**: 15-20 hours

**Impact**: 🔴 CRITICAL - Core security and functionality untested

---

### 2. pkg/middleware/auth/email (39.4% → Target: 65%)

**Sender Implementations (0% coverage)**:
- ❌ `SMTPSender.Send()` / `SendHTML()` - SMTP email sending
- ❌ `SendGridSender.Send()` / `SendHTML()` - SendGrid integration
- ❌ `SendmailSender.Send()` / `SendHTML()` - Sendmail integration

**Token Management (0% coverage)**:
- ❌ `VerifyOTP()` - OTP verification flow
- ❌ `normalizeOTP()` - OTP formatting
- ❌ `CleanupExpired()` - Token cleanup

**Estimated Effort**: 3-4 days

**Impact**: 🔴 CRITICAL - Email authentication completely untested

---

### 3. cmd/chatbotgate/server (13.3% → Target: 50%)

**Untested Functionality**:
- ❌ Server startup/shutdown
- ❌ Signal handling (SIGTERM, SIGINT)
- ❌ Configuration reload
- ❌ Health check integration
- ❌ Graceful shutdown

**Estimated Effort**: 2-3 days

**Impact**: 🔴 CRITICAL - Main entry point untested

---

## 🟠 High Priority Issues (P1)

### 4. pkg/middleware/auth/oauth2 (63.6% → Target: 75%)

**Missing Tests**:
- ❌ `Manager.GetAuthURLWithRedirect()` - Dynamic redirect URL generation (0%)
- ❌ `Manager.ExchangeWithRedirect()` - Token exchange with custom redirect (0%)
- ❌ `GoogleProvider.GetUserInfo()` - Google authentication flow (0%)
- ❌ CSRF protection verification
- ❌ State generation entropy validation

**Estimated Effort**: 2-3 days

---

### 5. pkg/shared/kvs (64.9% → Target: 75%)

**Missing Tests**:
- ❌ `New()` - Invalid store type error handling
- ❌ Context cancellation behavior
- ❌ Unicode/special character keys
- ❌ Negative TTL handling
- ❌ Redis connection failure scenarios

**Estimated Effort**: 1-2 days

---

## 🟡 Medium Priority Issues (P2)

### 6. pkg/shared/i18n (70.3% → Target: 80%)

**Missing Tests**:
- ❌ `DetectTheme()` - Theme detection (0%)
- ❌ `normalizeTheme()` - Theme normalization (0%)
- ❌ Accept-Language quality score handling
- ❌ Malformed header processing

**Estimated Effort**: 4-6 hours

---

### 7. pkg/shared/logging (74.0% → Target: 80%)

**Missing Tests**:
- ❌ `TestLogger` all methods (0% - test infrastructure itself untested!)
- ❌ Log rotation verification
- ❌ Compression functionality

**Estimated Effort**: 4-6 hours

---

### 8. pkg/middleware/session (76.6% → Target: 80%)

**Missing Tests**:
- ❌ KVS backend error handling (64.3%)
- ❌ JSON unmarshal errors
- ❌ Context timeout behavior
- ❌ Concurrent access testing

**Estimated Effort**: 4-6 hours

---

## 📅 Phase 1: Target 65% Coverage (Current Sprint)

### Week 1: Security-Critical Tests

**Day 1-2: Helper Security Functions** (pkg/middleware/core)
```go
// Tests to implement:
func TestIsValidRedirectURL(t *testing.T) {
    // Test cases:
    // - Valid relative URLs: "/path", "/path?query=value"
    // - Invalid protocol-relative: "//evil.com"
    // - Invalid absolute: "http://evil.com", "https://evil.com"
    // - Edge cases: "\t", "\n", control characters
}

func TestIsValidEmail(t *testing.T) {
    // Test cases:
    // - Valid emails: "user@example.com", "user+tag@example.com"
    // - Invalid: no @, double @
    // - SMTP injection: CR/LF characters
    // - Control characters, null bytes
}

func TestSanitizeHeaderValue(t *testing.T) {
    // Test cases:
    // - Clean values (passthrough)
    // - CR/LF injection removal
    // - Null byte removal
    // - Length limiting (> 8192 bytes)
}
```

**Expected Impact**: core 29.6% → 40%

---

**Day 3-5: Authentication Flow Integration Tests** (pkg/middleware/core)
```go
// Tests to implement:
func TestMiddleware_OAuth2Flow(t *testing.T) {
    // 1. handleOAuth2Start: state cookie + redirect
    // 2. handleOAuth2Callback: state verification + token exchange
    // 3. Session creation + redirect to original URL
}

func TestMiddleware_EmailFlow(t *testing.T) {
    // 1. handleEmailSend: rate limiting + token generation
    // 2. handleEmailVerify: token validation + session creation
    // 3. handleEmailVerifyOTP: OTP validation + session
}

func TestMiddleware_PasswordFlow(t *testing.T) {
    // 1. handlePasswordLogin: credential check + session
}

func TestMiddleware_ServeHTTP_Routing(t *testing.T) {
    // Test all path routing logic
    // Test rules evaluation (allow/auth/deny)
}
```

**Expected Impact**: core 40% → 55%

---

### Week 2: Email Authentication Tests

**Day 1-3: SMTP/SendGrid Sender Tests** (pkg/middleware/auth/email)
```go
// Tests to implement:
func TestSMTPSender_Send(t *testing.T) {
    // Use mock SMTP server (net.Listen + smtp parsing)
    // Test with/without auth
    // Test TLS vs STARTTLS
}

func TestSMTPSender_SendHTML(t *testing.T) {
    // Test multipart HTML/text email
    // Verify MIME structure
}

func TestSendGridSender_SendHTML(t *testing.T) {
    // Mock SendGrid API with httptest.Server
    // Verify API request format
    // Test error handling
}

func TestSendmailSender_Send(t *testing.T) {
    // Mock exec.Command
    // Verify command-line args
}
```

**Expected Impact**: email 39.4% → 60%

---

**Day 4-5: Token & OTP Tests** (pkg/middleware/auth/email)
```go
// Tests to implement:
func TestTokenStore_VerifyOTP(t *testing.T) {
    // Generate token → extract OTP
    // Verify OTP successfully
    // Test one-time use enforcement
    // Test expired OTP
    // Test invalid format
}

func TestTokenStore_CleanupExpired(t *testing.T) {
    // Mixed expired/valid tokens
    // Verify only expired removed
}

func TestHandler_TemplateErrors(t *testing.T) {
    // Invalid template data
    // Missing required fields
    // Language fallback
}
```

**Expected Impact**: email 60% → 70%

---

### Week 3: OAuth2 & High Priority

**Day 1-2: OAuth2 Manager Tests** (pkg/middleware/auth/oauth2)
```go
// Tests to implement:
func TestManager_GetAuthURLWithRedirect(t *testing.T) {
    // Test dynamic host detection
    // Test HTTPS scheme for non-localhost
    // Test custom auth prefix
}

func TestManager_ExchangeWithRedirect(t *testing.T) {
    // Test custom redirect URL matching
}

func TestGoogleProvider_GetUserInfo(t *testing.T) {
    // Mock Google API responses
    // Test all fields (_email, _username, _avatar_url)
    // Test error scenarios
}
```

**Expected Impact**: oauth2 63.6% → 75%

---

**Day 3-5: KVS & Shared Library Tests**

```go
// pkg/shared/kvs
func TestNew_InvalidType(t *testing.T)
func TestStore_ContextCancellation(t *testing.T)
func TestStore_UnicodeKeys(t *testing.T)
func TestStore_NegativeTTL(t *testing.T)

// pkg/shared/i18n
func TestDetectTheme(t *testing.T)
func TestNormalizeTheme(t *testing.T)

// pkg/shared/logging
func TestTestLogger(t *testing.T)
func TestTestLoggerVerbose(t *testing.T)
```

**Expected Impact**:
- kvs 64.9% → 75%
- i18n 70.3% → 80%
- logging 74.0% → 80%

---

## 📊 Expected Progress (Phase 1)

| Milestone | Overall Coverage | Key Packages |
|-----------|-----------------|--------------|
| **Start** | 50.1% | core: 29.6%, email: 39.4% |
| **Week 1** | 55% | core: 55%, email: 39.4% |
| **Week 2** | 60% | core: 55%, email: 70% |
| **Week 3** | **65%** ✅ | core: 55%, email: 70%, oauth2: 75% |

---

## 🎯 Success Criteria (Phase 1 - 65% Target)

### Must Have (P0):
- ✅ All security-critical functions tested (isValidRedirectURL, isValidEmail, sanitizeHeaderValue)
- ✅ OAuth2 flow integration tests (start → callback → session)
- ✅ Email flow integration tests (send → verify → session)
- ✅ SMTP/SendGrid sender implementation tests
- ✅ OTP verification tests

### Should Have (P1):
- ✅ OAuth2 manager redirect URL tests
- ✅ Google provider implementation tests
- ✅ KVS error handling tests
- ✅ i18n theme detection tests

### Nice to Have (P2):
- Session concurrent access tests
- Logging TestLogger tests
- Filewatcher error path tests

---

## 📈 Future Phases

### Phase 2: Target 75% Coverage (Next Sprint)
- Server command tests (cmd/chatbotgate/server)
- Complete ServeHTTP routing tests
- Middleware factory tests
- Config package improvements

### Phase 3: Target 80%+ Coverage (Future)
- Stress/load tests
- Fuzzing tests
- Benchmark tests
- E2E integration tests

---

## 🏆 Test Quality Best Practices

### Excellent Examples to Follow:
1. **pkg/shared/config** (95.7%) - 47 test cases, comprehensive edge cases
2. **pkg/middleware/forwarding** (85.3%) - 577 lines of non-existent path testing
3. **pkg/middleware/authz** (95.8%) - Complete whitelist logic coverage
4. **pkg/middleware/assets** (100%) - Perfect table-driven tests

### Patterns to Apply:
- ✅ Table-driven test design
- ✅ Clear test naming: `Test{Function}_{Scenario}`
- ✅ Comprehensive edge case coverage
- ✅ Security-focused testing
- ✅ Mock-based isolation
- ✅ Error path testing

---

## 📝 Notes

- All security-critical code must have 100% coverage
- Authentication flows must have integration tests
- Helper functions must test all edge cases
- Use `go test -race` for concurrency testing
- Run `go test -coverprofile=coverage.out ./...` to verify progress

---

**Maintained by**: Claude Code
**Review Date**: Every Sprint
**Next Review**: After Phase 1 completion
