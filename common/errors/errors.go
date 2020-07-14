package errors

type ClientError interface {
	Msg() string
	RespCode() int
}

type BaseError struct {
	code int
	status string
}

func (e BaseError) Msg() string {
	return e.status
}

func (e BaseError) RespCode() int {
	return e.code
}

func NewBaseError(code int, status string) *BaseError {
	return &BaseError{
		code: code,
		status: status,
	}
}
