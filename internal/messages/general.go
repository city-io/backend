package messages

type InternalError struct{}

func (e *InternalError) Error() string {
	return "Internal error"
}
