package main

import (
	"errors"
	"log"
	"net/http"

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

func userFromPayloadOld(ctx *gin.Context) (*user, int, string, error) {
	data, err := ctx.GetRawData()
	if err != nil {
		return nil, http.StatusBadRequest, "please provide a username and password", err
	}
	userProto := &protos.SignIn{}
	if err = proto.Unmarshal(data, userProto); err != nil {
		return nil, http.StatusBadRequest, "unable to parse provided credentials", err
	}
	userObj := newUser(userProto.Username, userProto.Password)
	if err != nil {
		return nil, http.StatusInternalServerError, "unable to create the user object", err
	}
	return userObj, http.StatusOK, "credentials parsed", nil
}

func signin(ctx *gin.Context) error {
	newUser, statusCode, status, err := userFromPayloadOld(ctx)
	if err != nil {
		cError := NewBaseError(statusCode, status)
		_ = ctx.Error(err).SetType(1 << 1).SetMeta(*cError)
		return err
	}
	if err := newUser.validate(map[string]interface{}{"passwd": nil}); err != nil {
		cError := NewBaseError(http.StatusBadRequest, "malformed credentials")
		_ = ctx.Error(err).SetType(1 << 1).SetMeta(*cError)
		return err
	}
	exists, err := newUser.exists(GetClient())
	if err != nil {
		cError := NewBaseError(http.StatusInternalServerError, "invalid credentials")
		_ = ctx.Error(err).SetType(1 << 1).SetMeta(*cError)
		return err
	}
	if !exists {
		err = errors.New("user does not exist")
		cError := NewBaseError(http.StatusNotFound, "invalid credentials")
		_ = ctx.Error(err).SetType(1 << 1).SetMeta(*cError)
		return err
	}
	if passwordOk, err := newUser.passOk(GetClient()); err != nil {
		cError := NewBaseError(http.StatusNotFound, "invalid credentials")
		_ = ctx.Error(err).SetType(1 << 1).SetMeta(*cError)
		return err
	} else if !passwordOk {
		err = errors.New("user password does not match")
		cError := NewBaseError(http.StatusNotFound, "invalid credentials")
		_ = ctx.Error(err).SetType(1 << 1).SetMeta(*cError)
		return err
	}
	jwt, err := newUser.jwt()
	if err != nil {
		cError := NewBaseError(http.StatusBadRequest, "invalid credentials")
		_ = ctx.Error(err).SetType(1 << 1).SetMeta(*cError)
		return err
	}
	ctx.SetCookie("auth-jwt", jwt, 3600, "", "", false, true)
	return nil
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
		users.POST("/signin", func(ctx *gin.Context) {
			if err := signin(ctx); err != nil {
				log.Printf("/api/users/signin: %v", err)
			} else {
				ctx.String(http.StatusOK, "signin successful")
			}
		})
		users.GET("/signout", func(ctx *gin.Context) {
			ctx.SetCookie("auth-jwt", "", -1, "", "", false, true)
			ctx.Status(http.StatusOK)
		})
		users.POST("/signup", func(ctx *gin.Context) {
			if err := signupUser(ctx, conf); err != nil {
				log.Printf("user signup failed: %v", err)
			} else {
				ctx.String(http.StatusCreated, "signup complete")
			}
		})
	}
}
