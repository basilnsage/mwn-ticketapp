package main

import (
	"net/http"

	e "github.com/basilnsage/mwn-ticketapp/auth/errors"
	"github.com/basilnsage/mwn-ticketapp/auth/token"
	"github.com/basilnsage/mwn-ticketapp/auth/users"
	"github.com/gin-gonic/gin"
)

type Claims struct {
	users.Claims
}

// Whoami identifies a user by the provided JWT cookie and returns a representation of said user
func Whoami(ctx *gin.Context, validator *token.JWTValidator) {
	resp, err := userFromRequest(ctx, validator)
	if err != nil {
		cError := e.NewBaseError(http.StatusUnauthorized, "unauthorized")
		_ = ctx.Error(err).SetType(1 << 1).SetMeta(*cError)
	} else {
		ctx.JSON(http.StatusOK, resp)
	}
}

func userFromRequest(ctx *gin.Context, validator *token.JWTValidator) (*gin.H, error) {
	defaultResp := &gin.H{"email": nil, "id": nil}
	// check for the "auth-jwt" cookie from the request
	cookie, err := ctx.Cookie("auth-jwt")
	if err == http.ErrNoCookie {
		return defaultResp, nil
	} else if err != nil {
		return defaultResp, err
	}

	parsedClaims := new(Claims)
	// attempt to parse the cookie as a JWT
	_, err = validator.ParseWithClaims(cookie, parsedClaims)
	if err != nil {
		return defaultResp, err
	}
	return &gin.H{"email": parsedClaims.Email, "id": parsedClaims.ID}, nil
}
