package errors

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	defaultStatus  = "unable to process request"
	defaultCode    = http.StatusBadRequest
)

// TODO: instead of defining a custom interface, what about implementing the `error` interface?
// could pass errors directly into ctx.Error(...) instead of meta
// not sure how to get both error code and error message out of that though, just extend `error` interface?
// ClientError defines methods for shared error reporting
type ClientError interface {
	Msg() string
	RespCode() int
}

// BaseError implements ClientError for the most basic errors
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

func (e BaseError) Error() error {
	return errors.New(e.status)
}

func NewBaseError(code int, status string) *BaseError {
	return &BaseError{
		code:   code,
		status: status,
	}
}

func HandleErrors() gin.HandlerFunc {
	// go through errors and determine if they should be logged and/or sent to the client
	// if there are multiple errors to send to the client, only send the first error
	return func(c *gin.Context) {
		var (
			respStatus string
			respCode   int
			eCode      int
			eStatus    string
			e          error
			eType      gin.ErrorType
		)
		c.Next()
		if len(c.Errors) > 0 {
			respDefined := false
			respStatus = defaultStatus
			respCode = defaultCode
			for _, err := range c.Errors {
				e = err.Err
				eType = err.Type
				eCode = defaultCode
				switch m := err.Meta.(type) {
				case ClientError:
					eCode = m.RespCode()
					eStatus = m.Msg()
				}
				// check if ErrorType public and if a status has already been set
				if !respDefined && eType == 1<<1 {
					respCode = eCode
					respStatus = eStatus
					respDefined = true
				}
				log.Printf("[ERROR] - at middleware, code: %v, err: %v", eCode, e)
			}
			c.JSON(respCode, gin.H{"error": respStatus})
		}
	}
}
