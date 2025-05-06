//go:build integration

package integration

import (
	"context"
	"testing"

	deckDump "github.com/kong/go-database-reconciler/pkg/dump"
	"github.com/stretchr/testify/require"
)

func Test_Apply_Custom_Entities(t *testing.T) {
	runWhen(t, "enterprise", ">=3.0.0")
	setup(t)
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}
	ctx := context.Background()
	tests := []struct {
		name                   string
		initialStateFile       string
		targetPartialStateFile string
	}{
		{
			name:                   "certificate",
			initialStateFile:       "testdata/apply/001-custom-entities/initial-state.yaml",
			targetPartialStateFile: "testdata/apply/001-custom-entities/partial-update.yaml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mustResetKongState(ctx, t, client, deckDump.Config{})
			err := sync(tc.initialStateFile)
			require.NoError(t, err)

			err = apply(tc.targetPartialStateFile)
			require.NoError(t, err)
		})
	}
}
