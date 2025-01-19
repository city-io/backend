package messages

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
