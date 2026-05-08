package file

import (
	"errors"
	"fmt"

	"github.com/kong/go-database-reconciler/pkg/cprint"
	"github.com/kong/go-database-reconciler/pkg/utils"
)

func (b *stateBuilder) emitDiagnostic(code utils.DiagnosticCode, msg string) error {
	if b.diagnosticPolicy.ResolveSeverity(code) == utils.SeverityWarning {
		cprint.UpdatePrintlnStdErr(msg)
		return nil
	}

	if utils.DefaultSeverity(code) == utils.SeverityWarning {
		return fmt.Errorf("warning (%s): %s", code, msg)
	}

	return errors.New(msg)
}
