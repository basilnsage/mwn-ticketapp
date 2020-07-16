package errors

import "net/http"

type XClientError interface {
	Msg() string
	RespCode() int
}

type ClientError struct {
	code string
}

func (ce ClientError) Msg() string {
	return "unable to process request"
}

func (ce ClientError) RespCode() int {
	return http.StatusBadRequest
}
