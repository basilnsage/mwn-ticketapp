package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/basilnsage/mwn-ticketapp/common/protos"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"google.golang.org/protobuf/proto"
)

type creds struct {
	Email  string `validate:"required,email"`
	Passwd string `validate:"required,passwd"`
}

var v = validator.New()

func signupUser(c *gin.Context, v *validator.Validate) error {
	// parse raw binary data from request
	// this should be a protobuf message
	data, err := c.GetRawData()
	if err != nil {
		cError := NewBaseError(http.StatusBadRequest, "please provide a username and password")
		_ = c.Error(errors.New(err.Error())).SetType(1 << 1).SetMeta(*cError)
		return err
	}
	// unmarshal payload into a proto def
	signinCreds := &protos.SignIn{}
	if err := proto.Unmarshal(data, signinCreds); err != nil {
		cError := NewBaseError(http.StatusBadRequest, "unable to parse provided credentials")
		_ = c.Error(errors.New(err.Error())).SetType(1 << 1).SetMeta(*cError)
		return err
	}
	// convert to struct for native validation
	signinStruct := creds{
		Email:  signinCreds.Username,
		Passwd: signinCreds.Password,
	}
	err = v.Struct(signinStruct)
	if err != nil {
		invalidFields := make([]string, 0)
		for _, err := range err.(validator.ValidationErrors) {
			invalidFields = append(invalidFields, err.Field())
		}
		if len(invalidFields) > 0 {
			invalidFieldString := fmt.Sprintf("unable to validate these fields: %v", strings.Join(invalidFields, ","))
			cError := NewBaseError(http.StatusBadRequest, invalidFieldString)
			_ = c.Error(errors.New("failed to validate signup data")).SetType(1 << 1).SetMeta(*cError)
			return errors.New(invalidFieldString)
		}
	}
	log.Printf("[DEBUG] - username: %v, password: %v", signinCreds.Username, signinCreds.Password)
	return nil
}

func UseUserRoutes(r *gin.Engine) {
	_ = v.RegisterValidation("passwd", func(f validator.FieldLevel) bool {
		return len(f.Field().String()) >= 8
	})
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
		users.POST("/signup", func(c *gin.Context) {
			if err := signupUser(c, v); err != nil {
				log.Printf("user signup failed: %v", err)
			} else {
				c.String(http.StatusCreated, "signup complete")
			}
		})
	}
}
