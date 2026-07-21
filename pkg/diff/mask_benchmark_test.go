package diff

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
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
	jwkKey := `{"kty": "RSA", "use": "sig", "kid": "key-1", ` +
		`"n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR", ` +
		`"e": "AQAB"}`
	b.Setenv("DECK_JWK_KEY", jwkKey)
}

// buildRealisticDiffForBenchmark creates a realistic diff output with
// the specified number of events, each with ~15-20 lines of JSON diff.
// This tests plain-text secret masking (redis host, password, API keys, etc.)
func buildRealisticDiffForBenchmark(numEvents int) string {
	var sb strings.Builder
	for e := range numEvents {
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

	for e := range numEvents {
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
	jwkContent := `{"kty": "RSA", "use": "sig", "kid": "key-1", ` +
		`"n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR", ` +
		`"e": "AQAB"}`

	for e := range numEvents {
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

// maskEnvVarValueMain simulates the MAIN branch implementation (no caching at all).
// This is the baseline to compare against the current optimized implementation.
func maskEnvVarValueMain(diffString string) string {
	envVars := parseDeckEnvVars()
	if len(envVars) == 0 {
		return diffString
	}

	// Build sorted list of values (longest first) for substring replacement
	var secrets []string
	seen := make(map[string]bool, len(envVars))
	for _, ev := range envVars {
		if ev.Value != "" && !seen[ev.Value] {
			secrets = append(secrets, ev.Value)
			seen[ev.Value] = true
		}
	}
	if len(secrets) == 0 {
		return diffString
	}
	sort.Slice(secrets, func(i, j int) bool {
		return len(secrets[i]) > len(secrets[j])
	})

	// Pre-compile word-boundary patterns once for all secrets (main branch does this per call)
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

	isJSON := jsonKeyPattern.MatchString(diffString)

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

// TestMainVsOptimized compares the main branch implementation (no caching) vs
// the current optimized version (with pre-computed cache). This shows the real-world
// performance improvement from adding the EnvVarCache optimization.
func TestMainVsOptimized(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T)
		buildFunc func(int) string
		numEvents int
	}{
		{
			name: "Main_vs_Optimized_SmallConfig_5K",
			setupFunc: func(t *testing.T) {
				t.Setenv("DECK_REDIS_HOST", "redis.internal.example.com")
				t.Setenv("DECK_REDIS_PASSWORD", "super_secret_redis_pass_12345")
				t.Setenv("DECK_API_KEY", "sk-1234567890abcdefghijklmnop")
				t.Setenv("DECK_OAUTH_SECRET", "oauth_client_secret_xyz_789")
				t.Setenv("DECK_DATABASE_URL", "postgres://user:pass@db.internal:5432/mydb")
				t.Setenv("DECK_AUTH_TOKEN", "ghp_1234567890abcdefghijklmnopqrstuvwxyz")
			},
			buildFunc: buildRealisticDiffForBenchmark,
			numEvents: 5000,
		},
		{
			name: "Main_vs_Optimized_LargeConfig_50K",
			setupFunc: func(t *testing.T) {
				t.Setenv("DECK_REDIS_HOST", "redis.internal.example.com")
				t.Setenv("DECK_REDIS_PASSWORD", "super_secret_redis_pass_12345")
				t.Setenv("DECK_API_KEY", "sk-1234567890abcdefghijklmnop")
				t.Setenv("DECK_OAUTH_SECRET", "oauth_client_secret_xyz_789")
				t.Setenv("DECK_DATABASE_URL", "postgres://user:pass@db.internal:5432/mydb")
				t.Setenv("DECK_AUTH_TOKEN", "ghp_1234567890abcdefghijklmnopqrstuvwxyz")
			},
			buildFunc: buildRealisticDiffForBenchmark,
			numEvents: 50000,
		},
		{
			name: "Main_vs_Optimized_WithPEM_10K",
			setupFunc: func(t *testing.T) {
				t.Setenv("DECK_REDIS_HOST", "redis.internal.example.com")
				t.Setenv("DECK_REDIS_PASSWORD", "super_secret_redis_pass_12345")
				t.Setenv("DECK_API_KEY", "sk-1234567890abcdefghijklmnop")
				t.Setenv("DECK_OAUTH_SECRET", "oauth_client_secret_xyz_789")
				t.Setenv("DECK_DATABASE_URL", "postgres://user:pass@db.internal:5432/mydb")
				t.Setenv("DECK_AUTH_TOKEN", "ghp_1234567890abcdefghijklmnopqrstuvwxyz")
				t.Setenv("DECK_CLIENT_CERT", `"-----BEGIN CERTIFICATE-----
MIIErDCCApSgAwIBAgIUaZRSadvXi4QaZssfTWp+gNNzU0kwDQYJKoZIhvcNAQEL
BQAwSTEUMBIGA1UEAwwLZXhhbXBsZS5jb20xCzAJBgNVBAYTAkdCMRAwDgYDVQQI
-----END CERTIFICATE-----"`)
				t.Setenv("DECK_CLIENT_KEY", `"-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDMW0C8u/cKBVjx
tiRTdd3BP5cefnksKlABZ2gvIrB+fEJTQRdDppaNxVN4dADFHBsaMPekOeRopiTa
-----END PRIVATE KEY-----"`)
			},
			buildFunc: buildRealisticDiffForBenchmarkWithPEM,
			numEvents: 10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc(t)

			diffData := tt.buildFunc(tt.numEvents)

			// Warm up
			mainWarmup := maskEnvVarValueMain(diffData)
			// Note: maskEnvVarValueMain doesn't support PEM/JWK, so only validate
			// for non-PEM scenarios
			if !strings.Contains(tt.name, "WithPEM") && !strings.Contains(mainWarmup, maskedValue) {
				t.Fatalf("main masking produced no %q output", maskedValue)
			}
			cache := NewEnvVarCache()
			_ = maskEnvVarValueWithCache(diffData, cache)

			// Measure MAIN branch (no caching)
			mainStart := time.Now()
			var mainResult string
			for range 2 {
				mainResult = maskEnvVarValueMain(diffData)
			}
			mainDuration := time.Since(mainStart)
			mainAverage := mainDuration / 2
			_ = mainResult // Verify result was computed

			// Measure CURRENT (with cache)
			cache = NewEnvVarCache()
			currentStart := time.Now()
			for range 2 {
				_ = maskEnvVarValueWithCache(diffData, cache)
			}
			currentDuration := time.Since(currentStart)
			currentAverage := currentDuration / 2

			// Calculate improvement
			speedup := float64(mainAverage) / float64(currentAverage)

			t.Logf("Performance Comparison for %s:", tt.name)
			t.Logf("  Scenario: %d events", tt.numEvents)
			t.Logf("  Main branch (no caching):  %v per run", mainAverage)
			t.Logf("  Current (with cache):      %v per run", currentAverage)
			t.Logf("  Improvement: %.2f×", speedup)
		})
	}
}
