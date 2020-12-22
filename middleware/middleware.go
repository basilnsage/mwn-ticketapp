package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func UserValidator(v *JWTValidator, header string) func(c *gin.Context) {
	return func(c *gin.Context) {
		jwtHeader := c.GetHeader(header)
		if jwtHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"errors": []string{"User is not signed in"},
			})
			return
		}
		_, err := v.Parse(jwtHeader)
		if err != nil {
			_ = c.Error(err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"errors": []string{"Unauthorized"},
			})
			return
		}
	}
}
