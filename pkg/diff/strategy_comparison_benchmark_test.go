package diff

import (
	"fmt"
	"testing"

	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/file"
	"github.com/kong/go-kong/kong"
)

// Strategy Comparison Benchmarks: FieldBased vs ValueBased
// Tests with mixed entity types (Services, Routes, Plugins) at realistic scale
// 14,000 total entities, ~20% change rate, 5K/10K/30K secrets
// =================================================================

const (
	totalEntitiesForStrategyBench = 14000
	changePercentForStrategyBench = 20
	strategyBenchServiceCount     = 4667
	strategyBenchRouteCount       = 4666
	strategyBenchPluginCount      = 4667
)

// strategySecretAssignments spreads secrets evenly across entities
func strategySecretAssignments(total, secretCount int) ([]int, map[int]bool) {
	var indices []int
	membership := make(map[int]bool, secretCount)
	if secretCount == 0 || total == 0 {
		return indices, membership
	}

	step := float64(total) / float64(secretCount)
	next := step / 2
	for i := 0; i < secretCount && int(next) < total; i++ {
		idx := int(next)
		indices = append(indices, idx)
		membership[idx] = true
		next += step
	}
	return indices, membership
}

// strategyChangedIndices returns 20% of secret-bearing indices
func strategyChangedIndices(secretCount int) []int {
	ordered, _ := strategySecretAssignments(totalEntitiesForStrategyBench, secretCount)
	changedCount := len(ordered) * changePercentForStrategyBench / 100
	return ordered[:changedCount]
}

// buildStrategyBenchMixedContent creates content with mixed entity types for strategy comparison
func buildStrategyBenchMixedContent(secretCount int) *file.Content {
	_, membership := strategySecretAssignments(totalEntitiesForStrategyBench, secretCount)

	content := &file.Content{}
	idx := 0

	// Services - test with simple string config to avoid Kong struct issues
	for i := 0; i < strategyBenchServiceCount; i++ {
		serviceID := fmt.Sprintf("service-%d", i)
		if membership[idx] {
			content.Services = append(content.Services, file.FService{
				Service: kong.Service{
					Name: kong.String(serviceID),
					ID:   kong.String(serviceID),
					Host: kong.String(fmt.Sprintf("DECK_SERVICE_USER_%d", i)),
				},
			})
		} else {
			content.Services = append(content.Services, file.FService{
				Service: kong.Service{
					Name: kong.String(serviceID),
					ID:   kong.String(serviceID),
				},
			})
		}
		idx++
	}

	// Routes
	for i := 0; i < strategyBenchRouteCount; i++ {
		routeID := fmt.Sprintf("route-%d", i)
		if membership[idx] {
			content.Routes = append(content.Routes, file.FRoute{
				Route: kong.Route{
					Name:  kong.String(routeID),
					ID:    kong.String(routeID),
					Paths: []*string{kong.String(fmt.Sprintf("DECK_ROUTE_KEY_%d", i))},
				},
			})
		} else {
			content.Routes = append(content.Routes, file.FRoute{
				Route: kong.Route{
					Name: kong.String(routeID),
					ID:   kong.String(routeID),
				},
			})
		}
		idx++
	}

	// Plugins
	for i := 0; i < strategyBenchPluginCount; i++ {
		pluginID := fmt.Sprintf("plugin-%d", i)
		config := kong.Configuration{}
		if membership[idx] {
			config["minute"] = fmt.Sprintf("DECK_RATE_LIMIT_%d", i)
			config["hour"] = fmt.Sprintf("DECK_RATE_LIMIT_HOUR_%d", i)
		} else {
			config["minute"] = (i % 60) + 1
		}
		content.Plugins = append(content.Plugins, file.FPlugin{
			Plugin: kong.Plugin{
				Name:   kong.String("rate-limiting"),
				ID:     kong.String(pluginID),
				Config: config,
			},
		})
		idx++
	}

	return content
}

// buildStrategyBenchChangedPairs generates changed entity pairs across all types
func buildStrategyBenchChangedPairs(secretCount int) (
	oldServices, newServices []file.FService,
	oldRoutes, newRoutes []file.FRoute,
	oldPlugins, newPlugins []file.FPlugin,
) {
	changedIndices := strategyChangedIndices(secretCount)

	for _, globalIdx := range changedIndices {
		if globalIdx < strategyBenchServiceCount {
			serviceID := fmt.Sprintf("service-%d", globalIdx)
			oldServices = append(oldServices, file.FService{
				Service: kong.Service{
					Name: kong.String(serviceID),
					ID:   kong.String(serviceID),
					Host: kong.String(fmt.Sprintf("old_user_%d", globalIdx)),
				},
			})
			newServices = append(newServices, file.FService{
				Service: kong.Service{
					Name: kong.String(serviceID),
					ID:   kong.String(serviceID),
					Host: kong.String(fmt.Sprintf("new_user_%d", globalIdx)),
				},
			})
		} else if globalIdx < strategyBenchServiceCount+strategyBenchRouteCount {
			localIdx := globalIdx - strategyBenchServiceCount
			routeID := fmt.Sprintf("route-%d", localIdx)
			oldRoutes = append(oldRoutes, file.FRoute{
				Route: kong.Route{
					Name:  kong.String(routeID),
					ID:    kong.String(routeID),
					Paths: []*string{kong.String(fmt.Sprintf("old_key_%d", localIdx))},
				},
			})
			newRoutes = append(newRoutes, file.FRoute{
				Route: kong.Route{
					Name:  kong.String(routeID),
					ID:    kong.String(routeID),
					Paths: []*string{kong.String(fmt.Sprintf("new_key_%d", localIdx))},
				},
			})
		} else {
			localIdx := globalIdx - strategyBenchServiceCount - strategyBenchRouteCount
			pluginID := fmt.Sprintf("plugin-%d", localIdx)
			oldPlugins = append(oldPlugins, file.FPlugin{
				Plugin: kong.Plugin{
					Name: kong.String("rate-limiting"),
					ID:   kong.String(pluginID),
					Config: kong.Configuration{
						"minute": fmt.Sprintf("old_limit_%d", localIdx),
					},
				},
			})
			newPlugins = append(newPlugins, file.FPlugin{
				Plugin: kong.Plugin{
					Name: kong.String("rate-limiting"),
					ID:   kong.String(pluginID),
					Config: kong.Configuration{
						"minute": fmt.Sprintf("new_limit_%d", localIdx),
					},
				},
			})
		}
	}
	return
}

// setupStrategyBenchEnvVars sets up environment variables for value-based masking
func setupStrategyBenchEnvVars(b *testing.B, secretCount int) {
	b.Helper()
	_, membership := strategySecretAssignments(totalEntitiesForStrategyBench, secretCount)

	idx := 0
	for i := 0; i < strategyBenchServiceCount; i++ {
		if membership[idx] {
			b.Setenv(fmt.Sprintf("DECK_SERVICE_USER_%d", i), fmt.Sprintf("old_user_%d", i))
		}
		idx++
	}
	for i := 0; i < strategyBenchRouteCount; i++ {
		if membership[idx] {
			b.Setenv(fmt.Sprintf("DECK_ROUTE_KEY_%d", i), fmt.Sprintf("old_key_%d", i))
		}
		idx++
	}
	for i := 0; i < strategyBenchPluginCount; i++ {
		if membership[idx] {
			b.Setenv(fmt.Sprintf("DECK_RATE_LIMIT_%d", i), fmt.Sprintf("old_limit_%d", i))
		}
		idx++
	}
}

// ============================================================================
// FieldBased Strategy Benchmarks
// ============================================================================

func BenchmarkFieldBased_Build_Mixed_5K(b *testing.B) {
	mockContent := buildStrategyBenchMixedContent(5000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = file.BuildSecretMap(mockContent)
	}
}

func BenchmarkFieldBased_Build_Mixed_10K(b *testing.B) {
	mockContent := buildStrategyBenchMixedContent(10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = file.BuildSecretMap(mockContent)
	}
}

func BenchmarkFieldBased_Build_Mixed_30K(b *testing.B) {
	mockContent := buildStrategyBenchMixedContent(30000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = file.BuildSecretMap(mockContent)
	}
}

func BenchmarkFieldBased_Apply_Mixed_5K(b *testing.B) {
	mockContent := buildStrategyBenchMixedContent(5000)
	oldServices, newServices, oldRoutes, newRoutes, oldPlugins, newPlugins := buildStrategyBenchChangedPairs(5000)
	secretMap := file.BuildSecretMap(mockContent)
	envCache := NewEnvVarCache()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for idx := range oldServices {
			event := crud.Event{OldObj: &oldServices[idx], Obj: &newServices[idx]}
			_, _ = generateDiffStringWithCache(event, false, false, envCache, secretMap)
		}
		for idx := range oldRoutes {
			event := crud.Event{OldObj: &oldRoutes[idx], Obj: &newRoutes[idx]}
			_, _ = generateDiffStringWithCache(event, false, false, envCache, secretMap)
		}
		for idx := range oldPlugins {
			event := crud.Event{OldObj: &oldPlugins[idx], Obj: &newPlugins[idx]}
			_, _ = generateDiffStringWithCache(event, false, false, envCache, secretMap)
		}
	}
}

func BenchmarkFieldBased_Apply_Mixed_10K(b *testing.B) {
	mockContent := buildStrategyBenchMixedContent(10000)
	oldServices, newServices, oldRoutes, newRoutes, oldPlugins, newPlugins := buildStrategyBenchChangedPairs(10000)

	envCache := NewEnvVarCache()

	b.ReportAllocs()
	b.ResetTimer()
	secretMap := file.BuildSecretMap(mockContent)
	for i := 0; i < b.N; i++ {
		for idx := range oldServices {
			event := crud.Event{OldObj: &oldServices[idx], Obj: &newServices[idx]}
			//fmt.Println(len(secretMap))
			_, _ = generateDiffStringWithCache(event, false, false, envCache, secretMap)
		}
		for idx := range oldRoutes {
			event := crud.Event{OldObj: &oldRoutes[idx], Obj: &newRoutes[idx]}
			//fmt.Println(len(secretMap))
			_, _ = generateDiffStringWithCache(event, false, false, envCache, secretMap)
		}
		for idx := range oldPlugins {
			event := crud.Event{OldObj: &oldPlugins[idx], Obj: &newPlugins[idx]}
			//fmt.Println(len(secretMap))
			_, _ = generateDiffStringWithCache(event, false, false, envCache, secretMap)
		}
	}
}

func BenchmarkFieldBased_Apply_Mixed_30K(b *testing.B) {
	mockContent := buildStrategyBenchMixedContent(30000)
	oldServices, newServices, oldRoutes, newRoutes, oldPlugins, newPlugins := buildStrategyBenchChangedPairs(30000)
	envCache := NewEnvVarCache()

	b.ReportAllocs()
	b.ResetTimer()
	secretMap := file.BuildSecretMap(mockContent)
	for i := 0; i < b.N; i++ {
		for idx := range oldServices {
			event := crud.Event{OldObj: &oldServices[idx], Obj: &newServices[idx]}
			_, _ = generateDiffStringWithCache(event, false, false, envCache, secretMap)
		}
		for idx := range oldRoutes {
			event := crud.Event{OldObj: &oldRoutes[idx], Obj: &newRoutes[idx]}
			_, _ = generateDiffStringWithCache(event, false, false, envCache, secretMap)
		}
		for idx := range oldPlugins {
			event := crud.Event{OldObj: &oldPlugins[idx], Obj: &newPlugins[idx]}
			_, _ = generateDiffStringWithCache(event, false, false, envCache, secretMap)
		}
	}
}

// ============================================================================
// ValueBased Strategy Benchmarks
// ============================================================================

func BenchmarkValueBased_Build_Mixed_5K(b *testing.B) {
	setupStrategyBenchEnvVars(b, 5000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewEnvVarCache()
	}
}

func BenchmarkValueBased_Build_Mixed_10K(b *testing.B) {
	setupStrategyBenchEnvVars(b, 10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewEnvVarCache()
	}
}

func BenchmarkValueBased_Build_Mixed_30K(b *testing.B) {
	setupStrategyBenchEnvVars(b, 30000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewEnvVarCache()
	}
}

func BenchmarkValueBased_Apply_Mixed_5K(b *testing.B) {
	setupStrategyBenchEnvVars(b, 5000)
	oldServices, newServices, oldRoutes, newRoutes, oldPlugins, newPlugins := buildStrategyBenchChangedPairs(5000)
	cache := NewEnvVarCache()
	envCacheForFieldBased := NewEnvVarCache() // For comparison with actual diff generation

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// ValueBased: Generate diffs and apply masking
		for idx := range oldServices {
			event := crud.Event{OldObj: &oldServices[idx], Obj: &newServices[idx]}
			diffText, _ := generateDiffStringWithCache(event, false, false, envCacheForFieldBased, nil)
			_ = maskEnvVarValueWithCache(diffText, cache)
		}
		for idx := range oldRoutes {
			event := crud.Event{OldObj: &oldRoutes[idx], Obj: &newRoutes[idx]}
			diffText, _ := generateDiffStringWithCache(event, false, false, envCacheForFieldBased, nil)
			_ = maskEnvVarValueWithCache(diffText, cache)
		}
		for idx := range oldPlugins {
			event := crud.Event{OldObj: &oldPlugins[idx], Obj: &newPlugins[idx]}
			diffText, _ := generateDiffStringWithCache(event, false, false, envCacheForFieldBased, nil)
			_ = maskEnvVarValueWithCache(diffText, cache)
		}
	}
}

func BenchmarkValueBased_Apply_Mixed_10K(b *testing.B) {
	setupStrategyBenchEnvVars(b, 10000)
	oldServices, newServices, oldRoutes, newRoutes, oldPlugins, newPlugins := buildStrategyBenchChangedPairs(10000)
	cache := NewEnvVarCache()
	envCacheForFieldBased := NewEnvVarCache()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for idx := range oldServices {
			event := crud.Event{OldObj: &oldServices[idx], Obj: &newServices[idx]}
			diffText, _ := generateDiffStringWithCache(event, false, false, envCacheForFieldBased, nil)
			_ = maskEnvVarValueWithCache(diffText, cache)
		}
		for idx := range oldRoutes {
			event := crud.Event{OldObj: &oldRoutes[idx], Obj: &newRoutes[idx]}
			diffText, _ := generateDiffStringWithCache(event, false, false, envCacheForFieldBased, nil)
			_ = maskEnvVarValueWithCache(diffText, cache)
		}
		for idx := range oldPlugins {
			event := crud.Event{OldObj: &oldPlugins[idx], Obj: &newPlugins[idx]}
			diffText, _ := generateDiffStringWithCache(event, false, false, envCacheForFieldBased, nil)
			_ = maskEnvVarValueWithCache(diffText, cache)
		}
	}
}

func BenchmarkValueBased_Apply_Mixed_30K(b *testing.B) {
	setupStrategyBenchEnvVars(b, 30000)
	oldServices, newServices, oldRoutes, newRoutes, oldPlugins, newPlugins := buildStrategyBenchChangedPairs(30000)
	cache := NewEnvVarCache()
	envCacheForFieldBased := NewEnvVarCache()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for idx := range oldServices {
			event := crud.Event{OldObj: &oldServices[idx], Obj: &newServices[idx]}
			diffText, _ := generateDiffStringWithCache(event, false, false, envCacheForFieldBased, nil)
			_ = maskEnvVarValueWithCache(diffText, cache)
		}
		for idx := range oldRoutes {
			event := crud.Event{OldObj: &oldRoutes[idx], Obj: &newRoutes[idx]}
			diffText, _ := generateDiffStringWithCache(event, false, false, envCacheForFieldBased, nil)
			_ = maskEnvVarValueWithCache(diffText, cache)
		}
		for idx := range oldPlugins {
			event := crud.Event{OldObj: &oldPlugins[idx], Obj: &newPlugins[idx]}
			diffText, _ := generateDiffStringWithCache(event, false, false, envCacheForFieldBased, nil)
			_ = maskEnvVarValueWithCache(diffText, cache)
		}
	}
}
