package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func signupUser(ctx *gin.Context, conf config) error {
	if err, cError := signupUserFlow(ctx, conf); err != nil {
		_ = ctx.Error(err).SetType(1 << 1).SetMeta(cError)
		return err
	}
	return nil
}

func signupUserFlow(ctx *gin.Context, conf config) (error, *BaseError) {
	// parse raw binary data from request
	// this should be a protobuf message
	newUser, statusCode, status, err := userFromPayload(ctx)
	if err != nil {
		return err, NewBaseError(statusCode, status)
	}

	// validate the user struct
	if err = newUser.Validate(nil); err != nil {
		return err, NewBaseError(http.StatusBadRequest, "signup failed")
	}

	// check for existng user
	userExists, err := newUser.Exists(conf.collection)
	if err != nil {
		return err, NewBaseError(http.StatusInternalServerError, "signup failed")
	}
	if userExists {
		return errors.New("user already exists"), NewBaseError(http.StatusBadRequest, "signup failed")
	}

	// no errors fetching user and user does not exist --> lets make that user
	if uid, err := newUser.Write(conf.collection); err != nil {
		return err, NewBaseError(http.StatusBadRequest, "signup failed")
	} else {
		log.Printf("user created with id: %v", uid)
	}

	// now create a JWT for the user and return this to the client
	if jwt, err := newUser.CreateSessionToken(conf.authValidator); err != nil {
		return err, NewBaseError(http.StatusBadRequest, "signup failed")
	} else {
		ctx.SetCookie("auth-jwt", jwt, 3600, "", "", false, true)
	}
	return nil, nil
}