package routes

import (
	"net/http"

	"github.com/basilnsage/mwn-ticketapp/auth/errors"
	"github.com/basilnsage/mwn-ticketapp/auth/jwt"
	"github.com/basilnsage/mwn-ticketapp/auth/users"
	"github.com/gin-gonic/gin"
)

// Whoami identifies a user by the provided JWT cookie and returns a representation of said user
func Whoami(ctx *gin.Context) {
	resp, err := whoami(ctx)
	if err != nil {
		cError := errors.NewBaseError(http.StatusUnauthorized, "unauthorized")
		_ = ctx.Error(err).SetType(1 << 1).SetMeta(*cError)
	} else {
		ctx.JSON(http.StatusOK, resp)
	}
}

func whoami(ctx *gin.Context) (*users.PrivClaims, error) {
	token, err := ctx.Cookie("auth-jwt")
	if err != nil {
		return nil, err
	}
	userClaims := new(users.PrivClaims)
	err = jwt.Verify(token, userClaims)
	if err != nil {
		return nil, err
	}
	return userClaims, nil
}
