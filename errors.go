package webcam

import "fmt"

// Timeout error
type Timeout struct{}

func (e *Timeout) Error() string {
	return "Timeout occured"
}

type EventTypeError struct {
	Expected uint32
	Actual   uint32
}

func (e *EventTypeError) Error() string {
	return fmt.Sprintf("event is not of type %d but instead of type %d", e.Expected, e.Actual)
}
