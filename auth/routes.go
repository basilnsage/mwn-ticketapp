package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/basilnsage/mwn-ticketapp/auth/users"
	"github.com/basilnsage/mwn-ticketapp/common/protos"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/bcrypt"
)

type userFormData struct {
	Username string `json:"username" binding:required`
	Password string `json:"password" binding:required`
}

func userFromForm(ctx *gin.Context) (*users.User, int, string, error) {
	data := new(userFormData)
	err := ctx.Bind(data)
	if err != nil {
		return nil, http.StatusBadRequest,
		"please provide a username and password", fmt.Errorf("ctx.Bind: %v", err)
	}
	userHash, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, http.StatusInternalServerError, "unable to hash password", err
	}
	userObj, err := users.NewUser(data.Username, data.Password, userHash)
	if err != nil {
		return nil, http.StatusBadRequest, "malformed request", err
	}
	return userObj, http.StatusOK, "credentials parsed", nil
}

func userFromPayload(ctx *gin.Context) (*users.User, int, string, error) {
	data, err := ctx.GetRawData()
	if err != nil {
		return nil, http.StatusBadRequest, "please provide a username and password", err
	}
	userProto := &protos.SignIn{}
	if err = proto.Unmarshal(data, userProto); err != nil {
		return nil, http.StatusBadRequest, "unable to parse provided credentials", err
	}
	userEmail, userPass := userProto.Username, userProto.Password
	userHash, err := bcrypt.GenerateFromPassword([]byte(userPass), bcrypt.DefaultCost)
	if err != nil {
		return nil, http.StatusInternalServerError, "unable to hash password", err
	}
	userObj, err := users.NewUser(userEmail, userPass, userHash)
	if err != nil {
		return nil, http.StatusInternalServerError, "unable to create the user object", err
	}
	return userObj, http.StatusOK, "credentials parsed", nil
}

func UseUserRoutes(r *gin.Engine, conf config) {
	userRoutePrefix := r.Group("/api/users")
	// TODO: break out payload validation into middleware?
	// keep following along with class first and see what they do about /signout and /signup
	// how they implement these routes will affect how to organize/apply the middlewear
	{
		userRoutePrefix.GET("/whoami", func(ctx *gin.Context) {
			Whoami(ctx, conf.authValidator)
		})
		userRoutePrefix.POST("/signup", func(ginCtx *gin.Context) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			SignupUser(ctx, ginCtx, conf.collection, conf.authValidator)
		})
		userRoutePrefix.POST("/signin", func(ginCtx *gin.Context) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			Signin(ctx, ginCtx, conf.collection, conf.authValidator)
		})
		userRoutePrefix.GET("/signout", func(ctx *gin.Context) {
			ctx.SetCookie("auth-jwt", "", -1, "", "", false, true)
			ctx.Status(http.StatusOK)
		})
	}
}
