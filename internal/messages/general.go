package messages

import "fmt"

type InitDatabaseMessage struct{}
type PeriodicOperationMessage struct{}

type InternalError struct{}

func (e *InternalError) Error() string {
	return "Internal error"
}

type InvalidResponseTypeError struct{}

func (e *InvalidResponseTypeError) Error() string {
	return "Invalid response type"
}

type UnknownError struct {
	Message string
}

func (e *UnknownError) Error() string {
	return fmt.Sprintf("Unknown error: %s", e.Message)
}
