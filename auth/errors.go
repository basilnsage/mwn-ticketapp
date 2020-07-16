package main

type ClientError interface {
	Msg() string
	RespCode() int
}

// should implement github.com/basilnsage/mwn-ticketapp/common/errors.ClientError
// seems like this ^ is not recommended, including the interface in the package until it needs to be shared
type BaseError struct {
	code   int
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
		code:   code,
		status: status,
	}
}
