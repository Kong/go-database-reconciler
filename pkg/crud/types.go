package crud

import (
	"context"
	"fmt"
)

// Op represents the type of the operation.
type Op struct {
	name string
}

func (op *Op) String() string {
	return op.name
}

var (
	// Create is a constant representing create operations.
	Create = Op{"Create"}
	// Update is a constant representing update operations.
	Update = Op{"Update"}
	// Delete is a constant representing delete operations.
	Delete = Op{"Delete"}
)

// Arg is an argument to a callback function.
type Arg interface{}

// Actions is an interface for CRUD operations on any entity
type Actions interface {
	Create(context.Context, ...Arg) (Arg, error)
	Delete(context.Context, ...Arg) (Arg, error)
	Update(context.Context, ...Arg) (Arg, error)
}

// Event represents an event to perform
// an imperative operation
// that gets Kong closer to the target state.
type Event struct {
	Op     Op
	Kind   Kind
	Obj    interface{}
	OldObj interface{}
}

// EventFromArg converts arg into Event.
// It panics if the type of arg is not Event.
func EventFromArg(arg Arg) Event {
	event, ok := arg.(Event)
	if !ok {
		panic("unexpected type, expected diff.Event")
	}
	return event
}

// ActionError represents an error happens in performing CRUD action of an entity.
type ActionError struct {
	OperationType Op     `json:"operation"`
	Kind          Kind   `json:"kind"`
	Name          string `json:"name"`
	Err           error  `json:"error"`
}

func (e *ActionError) Error() string {
	return fmt.Sprintf("%s %s %s failed: %v", e.OperationType.String(), e.Kind, e.Name, e.Err)
}
