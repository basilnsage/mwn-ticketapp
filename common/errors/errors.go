package errors

type ClientError struct {
	code int
	status string
}

func NewClientError(code int, status string) *ClientError {
	return &ClientError{
		code: code,
		status: status,
	}
}
