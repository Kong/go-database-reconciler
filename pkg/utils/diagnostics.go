package utils

import (
	"fmt"
	"slices"
	"strings"
)

type DiagnosticCode string

const (
	DiagnosticCodeRouteRegexPathFormat DiagnosticCode = "route-regex-path-format"
	DiagnosticCodeRLAConsumerGroups    DiagnosticCode = "rla-consumer-groups-deprecated"
	DiagnosticCodeOIDCMissingConfig    DiagnosticCode = "oidc-missing-required-config"
)

var validDiagnosticCodes = map[DiagnosticCode]struct{}{
	DiagnosticCodeRouteRegexPathFormat: {},
	DiagnosticCodeRLAConsumerGroups:    {},
	DiagnosticCodeOIDCMissingConfig:    {},
}

type Severity string

const (
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

var defaultSeverityByDiagnosticCode = map[DiagnosticCode]Severity{
	DiagnosticCodeRouteRegexPathFormat: SeverityWarning,
	DiagnosticCodeRLAConsumerGroups:    SeverityError,
	DiagnosticCodeOIDCMissingConfig:    SeverityError,
}

type DiagnosticPolicy struct {
	AlwaysError   []DiagnosticCode
	AlwaysWarning []DiagnosticCode
}

func NewDiagnosticPolicy(alwaysError, alwaysWarning []DiagnosticCode) DiagnosticPolicy {
	return DiagnosticPolicy{
		AlwaysError:   deduplicateDiagnosticCodes(alwaysError),
		AlwaysWarning: deduplicateDiagnosticCodes(alwaysWarning),
	}
}

func ParseDiagnosticCodes(value string) ([]DiagnosticCode, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	parts := strings.Split(value, ",")
	codes := make([]DiagnosticCode, 0, len(parts))
	seen := map[DiagnosticCode]struct{}{}

	for _, part := range parts {
		normalized := DiagnosticCode(strings.TrimSpace(part))
		if normalized == "" {
			continue
		}
		if _, ok := validDiagnosticCodes[normalized]; !ok {
			return nil, fmt.Errorf("unknown diagnostic code: %s", normalized)
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		codes = append(codes, normalized)
	}

	return codes, nil
}

func ValidDiagnosticCodes() []DiagnosticCode {
	codes := make([]DiagnosticCode, 0, len(validDiagnosticCodes))
	for code := range validDiagnosticCodes {
		codes = append(codes, code)
	}
	slices.Sort(codes)
	return codes
}

func ValidDiagnosticCodesString() string {
	codes := ValidDiagnosticCodes()
	if len(codes) == 0 {
		return ""
	}

	stringCodes := make([]string, 0, len(codes))
	for _, code := range codes {
		stringCodes = append(stringCodes, string(code))
	}

	return strings.Join(stringCodes, ",")
}

func (p DiagnosticPolicy) IsAlwaysError(code DiagnosticCode) bool {
	return slices.Contains(p.AlwaysError, code)
}

func (p DiagnosticPolicy) IsAlwaysWarning(code DiagnosticCode) bool {
	return slices.Contains(p.AlwaysWarning, code)
}

func (p DiagnosticPolicy) ResolveSeverity(code DiagnosticCode) Severity {
	if p.IsAlwaysError(code) {
		return SeverityError
	}
	if p.IsAlwaysWarning(code) {
		return SeverityWarning
	}
	if severity, ok := defaultSeverityByDiagnosticCode[code]; ok {
		return severity
	}
	return SeverityWarning
}

func DefaultSeverity(code DiagnosticCode) Severity {
	if severity, ok := defaultSeverityByDiagnosticCode[code]; ok {
		return severity
	}
	return SeverityWarning
}

func deduplicateDiagnosticCodes(codes []DiagnosticCode) []DiagnosticCode {
	if len(codes) == 0 {
		return nil
	}
	seen := map[DiagnosticCode]struct{}{}
	result := make([]DiagnosticCode, 0, len(codes))
	for _, code := range codes {
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		result = append(result, code)
	}
	return result
}
