package diff

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSolve_InvalidParallelism_Negative(t *testing.T) {
	parallelism := -1
	sc, _ := NewSyncer(SyncerOpts{})
	_, errs, _ := sc.Solve(context.Background(), parallelism, true, false)
	require.Len(t, errs, 1, "Solve should return exactly one error for parallelism=%d", parallelism)
	require.EqualError(t, errs[0], "parallelism can not be less than 1")
}
