package diff

import (
	"os"
	"strings"
)

// MaskingStrategy defines which masking approach to use
type MaskingStrategy string

const (
	// StrategyFieldBased masks by field name (per-entity secret tracking)
	StrategyFieldBased MaskingStrategy = "field-based"

	// StrategyValueBased masks by value matching (regex-based)
	StrategyValueBased MaskingStrategy = "value-based"
)

// currentStrategy holds the active masking strategy
var currentStrategy MaskingStrategy = StrategyFieldBased

// SetMaskingStrategy sets the masking strategy to use
func SetMaskingStrategy(strategy MaskingStrategy) {
	if strategy == StrategyFieldBased || strategy == StrategyValueBased {
		currentStrategy = strategy
	}
}

// GetMaskingStrategy returns the current masking strategy
func GetMaskingStrategy() MaskingStrategy {
	return currentStrategy
}

// GetMaskingStrategyFromEnv reads the masking strategy from environment variable
// DECK_MASKING_STRATEGY=field-based or value-based
// Defaults to field-based if not set or invalid
func GetMaskingStrategyFromEnv() MaskingStrategy {
	strategy := os.Getenv("DECK_MASKING_STRATEGY")
	strategy = strings.TrimSpace(strings.ToLower(strategy))

	switch MaskingStrategy(strategy) {
	case StrategyFieldBased, StrategyValueBased:
		return MaskingStrategy(strategy)
	default:
		return StrategyFieldBased
	}
}

// InitMaskingStrategyFromEnv initializes the strategy from environment
func InitMaskingStrategyFromEnv() {
	SetMaskingStrategy(GetMaskingStrategyFromEnv())
}
