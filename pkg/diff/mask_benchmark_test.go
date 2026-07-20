package diff

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// assertMasked runs the masking function once (outside the timed loop) and
// fails the benchmark if it produced no [masked] output. This guards against
// silently benchmarking broken masking logic.
func assertMasked(b *testing.B, out string) {
	b.Helper()
	if !strings.Contains(out, maskedValue) {
		b.Fatalf("masking produced no %q output; benchmark data is not exercising the masker", maskedValue)
	}
}

// Benchmark_maskingOldApproach_SmallConfig benchmarks the old approach
// (recompiling regex per-call) on a small config (5K events).
func Benchmark_maskingOldApproach_SmallConfig(b *testing.B) {
	setupMaskingEnvVars(b)

	diffData := buildRealisticDiffForBenchmark(5000)
	assertMasked(b, maskEnvVarValueNoCache(diffData))

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = maskEnvVarValueNoCache(diffData)
	}
}

// Benchmark_maskingNewApproach_SmallConfig benchmarks the new approach
// (precomputed cache) on a small config (5K events).
func Benchmark_maskingNewApproach_SmallConfig(b *testing.B) {
	setupMaskingEnvVars(b)

	diffData := buildRealisticDiffForBenchmark(5000)
	cache := NewEnvVarCache()
	assertMasked(b, maskEnvVarValueWithCache(diffData, cache))

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = maskEnvVarValueWithCache(diffData, cache)
	}
}

// Benchmark_maskingOldApproach_LargeConfig benchmarks the old approach
// on a large config (50K events).
func Benchmark_maskingOldApproach_LargeConfig(b *testing.B) {
	setupMaskingEnvVars(b)

	diffData := buildRealisticDiffForBenchmark(50000)
	assertMasked(b, maskEnvVarValueNoCache(diffData))

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = maskEnvVarValueNoCache(diffData)
	}
}

// Benchmark_maskingNewApproach_LargeConfig benchmarks the new approach
// on a large config (50K events).
func Benchmark_maskingNewApproach_LargeConfig(b *testing.B) {
	setupMaskingEnvVars(b)

	diffData := buildRealisticDiffForBenchmark(50000)
	cache := NewEnvVarCache()
	assertMasked(b, maskEnvVarValueWithCache(diffData, cache))

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = maskEnvVarValueWithCache(diffData, cache)
	}
}

// Benchmark_maskingOldApproach_WithPEM benchmarks the old approach
// when PEM certificates are present (expensive regex compilation).
func Benchmark_maskingOldApproach_WithPEM(b *testing.B) {
	setupMaskingEnvVarsWithPEM(b)

	diffData := buildRealisticDiffForBenchmarkWithPEM(10000)
	assertMasked(b, maskEnvVarValueNoCache(diffData))

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = maskEnvVarValueNoCache(diffData)
	}
}

// Benchmark_maskingNewApproach_WithPEM benchmarks the new approach
// when PEM certificates are present.
func Benchmark_maskingNewApproach_WithPEM(b *testing.B) {
	setupMaskingEnvVarsWithPEM(b)

	diffData := buildRealisticDiffForBenchmarkWithPEM(10000)
	cache := NewEnvVarCache()
	assertMasked(b, maskEnvVarValueWithCache(diffData, cache))

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = maskEnvVarValueWithCache(diffData, cache)
	}
}

// Benchmark_maskingOldApproach_WithJWK benchmarks the old approach
// when JWK keys are present (another complex regex case).
func Benchmark_maskingOldApproach_WithJWK(b *testing.B) {
	setupMaskingEnvVarsWithJWK(b)

	diffData := buildRealisticDiffForBenchmarkWithJWK(10000)
	assertMasked(b, maskEnvVarValueNoCache(diffData))

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = maskEnvVarValueNoCache(diffData)
	}
}

// Benchmark_maskingNewApproach_WithJWK benchmarks the new approach
// when JWK keys are present.
func Benchmark_maskingNewApproach_WithJWK(b *testing.B) {
	setupMaskingEnvVarsWithJWK(b)

	diffData := buildRealisticDiffForBenchmarkWithJWK(10000)
	cache := NewEnvVarCache()
	assertMasked(b, maskEnvVarValueWithCache(diffData, cache))

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = maskEnvVarValueWithCache(diffData, cache)
	}
}

// === Helper functions ===

// setupMaskingEnvVars sets up realistic env vars for benchmarking.
// b.Setenv isolates the environment per-benchmark and auto-restores it on
// cleanup, so results are not skewed by DECK_* vars present in the CI shell.
func setupMaskingEnvVars(b *testing.B) {
	b.Setenv("DECK_REDIS_HOST", "redis.internal.example.com")
	b.Setenv("DECK_REDIS_PASSWORD", "super_secret_redis_pass_12345")
	b.Setenv("DECK_API_KEY", "sk-1234567890abcdefghijklmnop")
	b.Setenv("DECK_OAUTH_SECRET", "oauth_client_secret_xyz_789")
	b.Setenv("DECK_DATABASE_URL", "postgres://user:pass@db.internal:5432/mydb")
	b.Setenv("DECK_AUTH_TOKEN", "ghp_1234567890abcdefghijklmnopqrstuvwxyz")
}

// setupMaskingEnvVarsWithPEM adds PEM certificates (expensive regex compilation).
func setupMaskingEnvVarsWithPEM(b *testing.B) {
	setupMaskingEnvVars(b)
	b.Setenv("DECK_CLIENT_CERT", `"-----BEGIN CERTIFICATE-----
MIIErDCCApSgAwIBAgIUaZRSadvXi4QaZssfTWp+gNNzU0kwDQYJKoZIhvcNAQEL
BQAwSTEUMBIGA1UEAwwLZXhhbXBsZS5jb20xCzAJBgNVBAYTAkdCMRAwDgYDVQQI
-----END CERTIFICATE-----"`)
	b.Setenv("DECK_CLIENT_KEY", `"-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDMW0C8u/cKBVjx
tiRTdd3BP5cefnksKlABZ2gvIrB+fEJTQRdDppaNxVN4dADFHBsaMPekOeRopiTa
-----END PRIVATE KEY-----"`)
}

// setupMaskingEnvVarsWithJWK adds JWK keys (another regex-intensive case).
func setupMaskingEnvVarsWithJWK(b *testing.B) {
	setupMaskingEnvVars(b)
	b.Setenv("DECK_JWK_KEY", `{"kty": "RSA", "use": "sig", "kid": "key-1", "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR", "e": "AQAB"}`)
}

// buildRealisticDiffForBenchmark creates a realistic diff output with
// the specified number of events, each with ~15-20 lines of JSON diff.
// This tests plain-text secret masking (redis host, password, API keys, etc.)
func buildRealisticDiffForBenchmark(numEvents int) string {
	var sb strings.Builder
	for e := 0; e < numEvents; e++ {
		fmt.Fprintf(&sb, `updating service service-%d
 {
   "id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeee%04d",
   "name": "service-%d",
-  "host": "redis.internal.example.com",
+  "host": "redis.production.example.com",
   "port": 6379,
-  "password": "super_secret_redis_pass_12345",
+  "password": "new_secret_pass_54321",
   "api_key": "sk-1234567890abcdefghijklmnop",
   "config": {
     "timeout": 5000,
-    "oauth_secret": "oauth_client_secret_xyz_789",
+    "oauth_secret": "oauth_new_secret_abc_123",
-    "db_url": "postgres://user:pass@db.internal:5432/mydb",
+    "db_url": "postgres://user:newpass@db.internal:5432/mydb",
     "retries": 3
   }
 }
`, e, e, e)
	}
	return sb.String()
}

// buildRealisticDiffForBenchmarkWithPEM creates diff output that includes
// PEM certificates. This tests the expensive PEM regex compilation and masking.
func buildRealisticDiffForBenchmarkWithPEM(numEvents int) string {
	var sb strings.Builder
	certContent := `-----BEGIN CERTIFICATE-----
MIIErDCCApSgAwIBAgIUaZRSadvXi4QaZssfTWp+gNNzU0kwDQYJKoZIhvcNAQEL
BQAwSTEUMBIGA1UEAwwLZXhhbXBsZS5jb20xCzAJBgNVBAYTAkdCMRAwDgYDVQQI
-----END CERTIFICATE-----`

	keyContent := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDMW0C8u/cKBVjx
tiRTdd3BP5cefnksKlABZ2gvIrB+fEJTQRdDppaNxVN4dADFHBsaMPekOeRopiTa
-----END PRIVATE KEY-----`

	for e := 0; e < numEvents; e++ {
		fmt.Fprintf(&sb, `updating certificate cert-%d
 {
   "id": "cert-%d",
-  "cert": "%s",
+  "cert": "NEW_CERTIFICATE_CONTENT_HERE",
-  "key": "%s",
+  "key": "NEW_PRIVATE_KEY_CONTENT_HERE",
   "tags": ["ssl"]
 }
`, e, e, certContent, keyContent)
	}
	return sb.String()
}

// buildRealisticDiffForBenchmarkWithJWK creates diff output that includes
// JWK (JSON Web Key) objects. This tests complex JSON object masking.
func buildRealisticDiffForBenchmarkWithJWK(numEvents int) string {
	var sb strings.Builder
	jwkContent := `{"kty": "RSA", "use": "sig", "kid": "key-1", "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR", "e": "AQAB"}`

	for e := 0; e < numEvents; e++ {
		fmt.Fprintf(&sb, `updating key key-%d
 {
   "id": "key-%d",
-  "jwk": %s,
+  "jwk": {"kty": "RSA", "use": "sig", "kid": "key-new", "n": "NEW_KEY_MATERIAL", "e": "AQAB"},
   "tags": ["signing"]
 }
`, e, e, jwkContent)
	}
	return sb.String()
}

// maskEnvVarValueNoCache simulates the OLD approach: it recompiles all regex
// patterns (secrets, PEM blocks, JWK) on every call. Used only for benchmarking
// to quantify the cost of not caching. It mirrors the full masking behaviour of
// maskEnvVarValueWithCache so the comparison is apples-to-apples.
func maskEnvVarValueNoCache(diffString string) string {
	envVars := parseDeckEnvVars()
	if len(envVars) == 0 {
		return diffString
	}

	// Old way: build the secret set and detect PEM/JWK types on every call.
	var secrets []string
	pemTypes := make(map[string]bool)
	hasJWK := false
	seen := make(map[string]bool, len(envVars))

	for _, ev := range envVars {
		if ev.Value != "" && !seen[ev.Value] {
			secrets = append(secrets, ev.Value)
			seen[ev.Value] = true
		}
		if t := pemTypeFromValue(ev.Value); t != "" {
			pemTypes[t] = true
		}
		if isJWK(ev.Value) {
			hasJWK = true
		}
	}

	// Phase 1: mask PEM blocks (regex recompiled on every call).
	for pemType := range pemTypes {
		pattern := regexp.MustCompile(
			`(?s)(.*?)-----BEGIN\s+` + regexp.QuoteMeta(pemType) +
				`\s*-----.*?-----END\s+` + regexp.QuoteMeta(pemType) +
				`\s*-----(["',]*)`)
		diffString = pattern.ReplaceAllString(diffString, `$1`+maskedValue+`$2`)
	}

	// Phase 2: mask JWK patterns (regex recompiled on every call).
	if hasJWK {
		jwk := regexp.MustCompile(
			`\{(?:[^"{}]|"(?:\\.|[^"\\])*"|(?:\{[^{}]*\}))*` +
				`"kty"\s*:\s*"[^"]*"` +
				`(?:[^"{}]|"(?:\\.|[^"\\])*"|(?:\{[^{}]*\}))*\}`,
		)
		diffString = jwk.ReplaceAllString(diffString, maskedValue)
	}

	if len(secrets) == 0 {
		return diffString
	}

	sort.Slice(secrets, func(i, j int) bool {
		return len(secrets[i]) > len(secrets[j])
	})

	// Recompile secret patterns on every call (the expensive part).
	secretPatterns := make([]*regexp.Regexp, len(secrets))
	for idx, secret := range secrets {
		secretPatterns[idx] = regexp.MustCompile(`\b` + regexp.QuoteMeta(secret) + `\b`)
	}

	maskFn := func(s string) string {
		for _, re := range secretPatterns {
			s = re.ReplaceAllString(s, maskedValue)
		}
		return s
	}

	isJSON := strings.Contains(diffString, `":`)
	lines := strings.Split(diffString, "\n")
	for i, line := range lines {
		result := kvPattern.ReplaceAllStringFunc(line, func(match string) string {
			sub := kvPattern.FindStringSubmatch(match)
			if sub == nil {
				return match
			}
			switch {
			case sub[1] != "":
				masked := maskFn(sub[1])
				if masked != sub[1] {
					return match[:len(match)-len(`"`+sub[1]+`"`)] + `"` + masked + `"`
				}
			case sub[2] != "":
				if seen[sub[2]] {
					if isJSON {
						return `: "` + maskedValue + `"`
					}
					return ": " + maskedValue
				}
			case sub[3] != "":
				masked := maskFn(sub[3])
				if masked != sub[3] {
					return ": " + masked
				}
			}
			return match
		})

		if result == line {
			result = arrayElemPattern.ReplaceAllStringFunc(line, func(match string) string {
				sub := arrayElemPattern.FindStringSubmatch(match)
				if sub == nil {
					return match
				}
				masked := maskFn(sub[2])
				if masked == sub[2] {
					return match
				}
				quoted := `"` + sub[2] + `"`
				suffix := match[len(sub[1])+len(quoted):]
				return sub[1] + `"` + masked + `"` + suffix
			})
		}

		lines[i] = result
	}
	return strings.Join(lines, "\n")
}
