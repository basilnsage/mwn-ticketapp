package main

import (
	"context"
	"errors"
	"github.com/basilnsage/mwn-ticketapp/auth/users"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func SignupUser(ctx context.Context, ginCtx *gin.Context, crud users.CRUD, signer users.Signer) {
	if err, cError := signupUserFlow(ctx, ginCtx, crud, signer); err != nil {
		_ = ginCtx.Error(err).SetType(1 << 1).SetMeta(cError)
	} else {
		ginCtx.String(http.StatusCreated, "signup complete")
	}
}

func signupUserFlow(ctx context.Context, ginCtx *gin.Context, crud users.CRUD, signer users.Signer) (error, *BaseError) {
	// parse raw binary data from request
	// this should be a protobuf message
	newUser, statusCode, status, err := userFromPayload(ginCtx)
	if err != nil {
		return err, NewBaseError(statusCode, status)
	}

	// validate the user struct
	if err = newUser.Validate(nil); err != nil {
		return err, NewBaseError(http.StatusBadRequest, "signup failed")
	}

	// check for existng user
	userExists, err := newUser.Exists(ctx, crud)
	if err != nil {
		return err, NewBaseError(http.StatusInternalServerError, "signup failed")
	}
	if userExists {
		return errors.New("user already exists"), NewBaseError(http.StatusBadRequest, "signup failed")
	}

	// no errors fetching user and user does not exist --> lets make that user
	if uid, err := newUser.Write(ctx, crud); err != nil {
		return err, NewBaseError(http.StatusBadRequest, "signup failed")
	} else {
		log.Printf("user created with id: %v", uid)
	}

	// now create a JWT for the user and return this to the client
	if jwt, err := newUser.CreateSessionToken(signer); err != nil {
		return err, NewBaseError(http.StatusBadRequest, "signup failed")
	} else {
		ginCtx.SetCookie("auth-jwt", jwt, 3600, "", "", false, true)
	}
	return nil, nil
}
