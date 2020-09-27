package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	e "github.com/basilnsage/mwn-ticketapp/auth/errors"
	"github.com/basilnsage/mwn-ticketapp/auth/users"
	"github.com/gin-gonic/gin"
)

func Signin(ctx context.Context, ginCtx *gin.Context, crud users.CRUD, signer users.Signer) {
	if err := signin(ctx, ginCtx, crud, signer); err != nil {
		cError := e.NewBaseError(http.StatusBadRequest, "invalid credentials")
		_ = ginCtx.Error(err).SetType(1 << 1).SetMeta(*cError)
	} else {
		ginCtx.Status(http.StatusOK)
	}
}

func signin(ctx context.Context, ginCtx *gin.Context, crud users.CRUD, signer users.Signer) error {
	newUser, _, _, err := userFromPayload(ginCtx)
	if err != nil {
		return fmt.Errorf("unable to parse user from payload: %v", err)
	}

	exists, err := newUser.Exists(ctx, crud)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("user does not exist")
	}
	if passwordOk, err := newUser.DoesPassMatch(ctx, crud); err != nil {
		return err
	} else if !passwordOk {
		return errors.New("user password does not match")
	}

	userJWT, err := newUser.CreateSessionToken(signer)
	if err != nil {
		return err
	}
	ginCtx.SetCookie("auth-jwt", userJWT, 3600, "", "", false, true)

	return nil
}
