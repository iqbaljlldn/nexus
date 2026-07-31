package errors

type InfrastructureError struct {
	Message string
	Err     error
}

func (e *InfrastructureError) Error() string {
	return e.Message
}

func (e *InfrastructureError) Unwrap() error {
	return e.Err
}
