package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDiagnosticCodes(t *testing.T) {
	t.Run("parses comma separated list", func(t *testing.T) {
		codes, err := ParseDiagnosticCodes("route-regex-path-format")
		require.NoError(t, err)
		assert.Equal(t, []DiagnosticCode{DiagnosticCodeRouteRegexPathFormat}, codes)
	})

	t.Run("parses all supported warning codes", func(t *testing.T) {
		codes, err := ParseDiagnosticCodes(
			"route-regex-path-format,rla-consumer-groups-deprecated,oidc-missing-required-config",
		)
		require.NoError(t, err)
		assert.Equal(t, []DiagnosticCode{
			DiagnosticCodeRouteRegexPathFormat,
			DiagnosticCodeRLAConsumerGroups,
			DiagnosticCodeOIDCMissingConfig,
		}, codes)
	})

	t.Run("deduplicates values", func(t *testing.T) {
		codes, err := ParseDiagnosticCodes(" route-regex-path-format,route-regex-path-format ")
		require.NoError(t, err)
		assert.Equal(t, []DiagnosticCode{DiagnosticCodeRouteRegexPathFormat}, codes)
	})

	t.Run("rejects unknown warning code", func(t *testing.T) {
		_, err := ParseDiagnosticCodes("unknown-warning")
		require.Error(t, err)
		assert.ErrorContains(t, err, "unknown diagnostic code")
	})
}

func TestDiagnosticPolicyIsAlwaysError(t *testing.T) {
	policy := NewDiagnosticPolicy([]DiagnosticCode{DiagnosticCodeRouteRegexPathFormat}, nil)
	assert.True(t, policy.IsAlwaysError(DiagnosticCodeRouteRegexPathFormat))
	assert.False(t, policy.IsAlwaysError("different-warning"))
}

func TestDiagnosticPolicyResolveSeverity(t *testing.T) {
	t.Run("uses default severity", func(t *testing.T) {
		policy := NewDiagnosticPolicy(nil, nil)
		assert.Equal(t, SeverityWarning, policy.ResolveSeverity(DiagnosticCodeRouteRegexPathFormat))
		assert.Equal(t, SeverityError, policy.ResolveSeverity(DiagnosticCodeRLAConsumerGroups))
		assert.Equal(t, SeverityError, policy.ResolveSeverity(DiagnosticCodeOIDCMissingConfig))
	})

	t.Run("always warning can downgrade default error", func(t *testing.T) {
		policy := NewDiagnosticPolicy(nil, []DiagnosticCode{DiagnosticCodeOIDCMissingConfig})
		assert.Equal(t, SeverityWarning, policy.ResolveSeverity(DiagnosticCodeOIDCMissingConfig))
	})

	t.Run("always error wins conflict", func(t *testing.T) {
		policy := NewDiagnosticPolicy(
			[]DiagnosticCode{DiagnosticCodeOIDCMissingConfig},
			[]DiagnosticCode{DiagnosticCodeOIDCMissingConfig},
		)
		assert.Equal(t, SeverityError, policy.ResolveSeverity(DiagnosticCodeOIDCMissingConfig))
	})
}

func TestValidDiagnosticCodesString(t *testing.T) {
	assert.Equal(
		t,
		"oidc-missing-required-config,rla-consumer-groups-deprecated,route-regex-path-format",
		ValidDiagnosticCodesString(),
	)
}
