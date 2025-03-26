package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewState(t *testing.T) {
	state, err := NewKongState()
	assert := assert.New(t)
	require.NoError(t, err)
	assert.NotNil(state)
}

func state() *KongState {
	s, err := NewKongState()
	if err != nil {
		panic(err)
	}
	return s
}
