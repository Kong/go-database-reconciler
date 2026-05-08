package file

import (
	"testing"

	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateBuilderEmitException(t *testing.T) {
	t.Run("returns error when warning is configured as error", func(t *testing.T) {
		b := &stateBuilder{
			diagnosticPolicy: utils.NewDiagnosticPolicy([]utils.DiagnosticCode{utils.DiagnosticCodeRouteRegexPathFormat}, nil),
		}

		err := b.emitDiagnostic(utils.DiagnosticCodeRouteRegexPathFormat, "warning message")
		require.Error(t, err)
		assert.ErrorContains(t, err, "warning (route-regex-path-format)")
	})

	t.Run("prints warning when warning is not configured as error", func(t *testing.T) {
		b := &stateBuilder{}
		err := b.emitDiagnostic(utils.DiagnosticCodeRouteRegexPathFormat, "warning message")
		require.NoError(t, err)
	})
	t.Run("returns error by default", func(t *testing.T) {
		b := &stateBuilder{}
		err := b.emitDiagnostic(utils.DiagnosticCodeOIDCMissingConfig, "validation message")
		require.Error(t, err)
		assert.EqualError(t, err, "validation message")
	})

	t.Run("downgrades to warning when configured", func(t *testing.T) {
		b := &stateBuilder{
			diagnosticPolicy: utils.NewDiagnosticPolicy(nil, []utils.DiagnosticCode{utils.DiagnosticCodeOIDCMissingConfig}),
		}
		err := b.emitDiagnostic(utils.DiagnosticCodeOIDCMissingConfig, "validation message")
		require.NoError(t, err)
	})
}
