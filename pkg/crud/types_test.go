package crud

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpString(t *testing.T) {
	assert := assert.New(t)
	op := Op{"foo"}
	var op2 Op
	assert.Equal("foo", op.String())
	assert.Equal("", op2.String())
}

func TestActionError(t *testing.T) {
	err := fmt.Errorf("something wrong")
	actionErr := &ActionError{
		OperationType: Create,
		Kind:          Kind("service"),
		Name:          "service-test",
		Err:           err,
	}
	assert.Equal(t, "Create service service-test failed: something wrong", actionErr.Error())
}
