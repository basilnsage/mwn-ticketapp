package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/basilnsage/mwn-ticketapp/auth/users"
	"github.com/basilnsage/mwn-ticketapp/common/protos"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
)

func userFromPayload(ctx *gin.Context) (*users.User, int, string, error) {
	data, err := ctx.GetRawData()
	if err != nil {
		return nil, http.StatusBadRequest, "please provide a username and password", err
	}
	userProto := &protos.SignIn{}
	if err = proto.Unmarshal(data, userProto); err != nil {
		return nil, http.StatusBadRequest, "unable to parse provided credentials", err
	}
	userObj, err := users.NewUser(userProto.Username, userProto.Password)
	if err != nil {
		return nil, http.StatusInternalServerError, "unable to create the user object", err
	}
	return userObj, http.StatusOK, "credentials parsed", nil
}

func UseUserRoutes(r *gin.Engine, conf config) {
	// init user validator
	if err := initValidator(); err != nil {
		log.Fatalf("UseUserRoutes.initValidtor: %v", err)
	}
	users := r.Group("/api/users")
	// TODO: break out payload validation into middleware?
	// keep following along with class first and see what they do about /signout and /signup
	// how they implement these routes will affect how to organize/apply the middlewear
	{
		users.GET("/whoami", func(ctx *gin.Context) {
			Whoami(ctx, conf.authValidator)
		})
		users.POST("/signup", func(ginCtx *gin.Context) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			SignupUser(ctx, ginCtx, conf.collection, conf.authValidator)
		})
		users.POST("/signin", func(ginCtx *gin.Context) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			Signin(ctx, ginCtx, conf.collection, conf.authValidator)
		})
		users.GET("/signout", func(ctx *gin.Context) {
			ctx.SetCookie("auth-jwt", "", -1, "", "", false, true)
			ctx.Status(http.StatusOK)
		})
	}
}
