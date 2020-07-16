package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

var (
	defaultStatus = "unable to process request"
	defaultCode   = http.StatusBadRequest
)

// this should go LAST in the chain of middlewares
func handleErrors() gin.HandlerFunc {
	return func(c *gin.Context) {
		// go through errors and determine if they should be logged and/or sent to the client
		// if there are multiple errors to send to the client, only send the first error
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

func main() {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"http://localhost:*"},
		AllowWildcard: true,
		AllowMethods:  []string{"GET", "POST"},
		AllowHeaders:  []string{"Origin", "Content-Type"},
		MaxAge:        12 * time.Hour,
	}))
	// use generic error-handling middleware
	router.Use(handleErrors())
	UseUserRoutes(router)
	if err := router.Run(":4000"); err != nil {
		log.Printf("unable to run auth service")
	}
}
