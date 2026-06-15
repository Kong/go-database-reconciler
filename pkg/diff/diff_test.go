package diff

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSolve_InvalidParallelism_Zero(t *testing.T) {
	parallelism := 0
	sc, _ := NewSyncer(SyncerOpts{})
	_, errs, _ := sc.Solve(context.Background(), parallelism, true, false)
	assert.EqualError(t, errs[0], "parallelism can not be less than 1")
	assert.Equal(t, 1, len(errs), "Solve should return exactly one error for parallelism=%d", parallelism)
}

func TestSolve_InvalidParallelism_Negative(t *testing.T) {
	parallelism := -1
	sc, _ := NewSyncer(SyncerOpts{})
	_, errs, _ := sc.Solve(context.Background(), parallelism, true, false)
	assert.EqualError(t, errs[0], "parallelism can not be less than 1")
	assert.Equal(t, 1, len(errs), "Solve should return exactly one error for parallelism=%d", parallelism)
}
