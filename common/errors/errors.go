package errors

type ClientError struct {
	Code int
	Status string
}

func NewClientError(code int, status string) *ClientError {
	return &ClientError{
		Code: code,
		Status: status,
	}
}
