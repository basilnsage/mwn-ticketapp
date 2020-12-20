package main

import (
	"errors"
	"net/http"

	tJwt "github.com/basilnsage/mwn-ticketapp/tik-jwt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

func userAuthMiddleware(jwtKey string) (func(c *gin.Context), error) {
	v, err := tJwt.NewJWTValidator([]byte(jwtKey), "HS256")
	if err != nil {
		return nil, err
	}

	return func(c *gin.Context) {
		jwtHeader := c.GetHeader("auth-jwt")
		if jwtHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"errors": []string{"User is not signed in"},
			})
			return
		}
		isValid, err := isJWTValid(v, jwtHeader)
		if !isValid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"errors": []string{"Unauthorized"},
			})
			return
		}
		if err != nil {
			WarningLogger.Printf("unable to validate user JWT: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"errors": []string{"Unauthorized"},
			})
			return
		}
	}, nil

}

func isJWTValid(v *tJwt.JWTValidator, token string) (bool, error) {
	_, err := v.Parse(token)
	if err != nil {
		return false, err
	}
	return true, nil
}

func uidFromJWT(v *tJwt.JWTValidator, token string) (string, error) {
	jwtToken, err := v.Parse(token)
	if err != nil {
		return "", err
	}

	uid, ok := jwtToken.Claims.(jwt.MapClaims)["id"]
	if !ok {
		return "", errors.New("no field 'id' found in JWT")
	}
	return uid.(string), nil
}