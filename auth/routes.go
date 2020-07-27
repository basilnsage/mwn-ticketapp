package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func signupUser(c *gin.Context) error {
	// parse raw binary data from request
	// this should be a protobuf message
	data, err := c.GetRawData()
	if err != nil {
		cError := NewBaseError(http.StatusBadRequest, "please provide a username and password")
		_ = c.Error(err).SetType(1 << 1).SetMeta(*cError)
		return err
	}
	// unmarshal payload and create a new user struct
	newUser, err := userFromPayload(&data)
	if err != nil {
		cError := NewBaseError(http.StatusBadRequest, "unable to parse provided credentials")
		_ = c.Error(err).SetType(1 << 1).SetMeta(*cError)
		return err
	}
	// validate the user struct
	if err = newUser.validate(); err != nil {
		cError := NewBaseError(http.StatusBadRequest, "failed to validate signup data")
		_ = c.Error(err).SetType(1 << 1).SetMeta(*cError)
		return err
	}
	// check for existng user
	userExists, err := newUser.exists(GetClient())
	if err != nil {
		cError := NewBaseError(http.StatusInternalServerError, "unable to fetch user data")
		_ = c.Error(err).SetType(1 << 1).SetMeta(*cError)
		return err
	}
	if userExists {
		err = errors.New("user already exists")
		cError := NewBaseError(http.StatusBadRequest, err.Error())
		_ = c.Error(err).SetType(1 << 1).SetMeta(*cError)
		return err
	}
	// no errors fetching user and user does not exist --> lets make that user
	if res, err := newUser.write(GetClient()); err != nil {
		cError := NewBaseError(http.StatusBadRequest, "unable to create user")
		_ = c.Error(err).SetType(1 << 1).SetMeta(*cError)
		return err
	} else {
		log.Printf("user created with _id: %v", res.InsertedID)
	}
	return nil
}

func UseUserRoutes(r *gin.Engine) {
	users := r.Group("/api/users")
	{
		users.GET("/whoami", func(ctx *gin.Context) {
			ctx.String(http.StatusOK, "user whoami not implemented")
		})
		users.GET("/signin", func(ctx *gin.Context) {
			ctx.String(http.StatusOK, "user sign in not implemented")
		})
		users.GET("/signout", func(ctx *gin.Context) {
			ctx.String(http.StatusOK, "user sign out not implemented")
		})
		users.POST("/signup", func(ctx *gin.Context) {
			if err := signupUser(ctx); err != nil {
				log.Printf("user signup failed: %v", err)
			} else {
				ctx.String(http.StatusCreated, "signup complete")
			}
		})
	}
}
