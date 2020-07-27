package main

import (
	"github.com/gin-gonic/gin"
	"log"
)

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

func handleErrors() gin.HandlerFunc {
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
