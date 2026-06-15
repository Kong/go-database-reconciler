package diff

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSyncer(t *testing.T) *Syncer {
	sc, err := NewSyncer(SyncerOpts{})
	require.NoError(t, err)
	return sc
}

func TestSolve_InvalidParallelism_Zero(t *testing.T) {
	parallelism := 0
	t.Run("", func(t *testing.T) {
		sc := newTestSyncer(t)
		go func() {
			_, errs, _ := sc.Solve(context.Background(), parallelism, true, false)
			assert.Equal(t, 1, len(errs), "Solve should return exactly one error for parallelism=%d", parallelism)
		}()
	})
}

func TestSolve_InvalidParallelism_Negative(t *testing.T) {
	parallelism := -1
	t.Run("", func(t *testing.T) {
		sc := newTestSyncer(t)
		go func() {
			_, errs, _ := sc.Solve(context.Background(), parallelism, true, false)
			assert.Equal(t, 1, len(errs), "Solve should return exactly one error for parallelism=%d", parallelism)
		}()
	})
}
