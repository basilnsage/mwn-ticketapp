package main

import (
	"bytes"
	"context"
	"github.com/basilnsage/mwn-ticketapp/auth/users"
	"github.com/basilnsage/mwn-ticketapp/common/protos"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignin(t *testing.T) {
	signer := new(mockSigner)
	crud := new(mockCRUD)
	ctx := context.Background()
	user, err := users.NewUser(email, pass, passHash)
	if err != nil {
		t.Fatalf("unable to create new user: %v", err)
	}
	userBytes, err := proto.Marshal(&protos.SignIn{
		Username: "foo@example.com",
		Password: "password",
	})
	if err != nil {
		t.Fatalf("unable to marshal user proto: %v", err)
	}

	//crud.On("Read", ctx, *user).Return([]users.User{*user}, nil)
	crud.On("Read", ctx, mock.MatchedBy(checkTestUser)).Return([]users.User{*user}, nil)
	crud.On("Write", ctx, mock.MatchedBy(checkTestUser)).Return(uid, nil)
	signer.On("Sign", map[string]interface{}{
		"email": email,
		"id":    nil,
	}).Return(jwtString, nil)

	gin.SetMode(gin.TestMode)
	eng := gin.Default()
	eng.Use(func(ctx *gin.Context) {
		ctx.Next()
		if len(ctx.Errors) > 0 {
			t.Errorf("ctx error: %v", ctx.Errors[0])
		}
	})
	eng.POST("/test", func(ginCtx *gin.Context) {
		Signin(ctx, ginCtx, crud, signer)
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(userBytes))
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	eng.ServeHTTP(w, req)

	resp := w.Result()
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("wrong response code: %v, want: %v", got, want)
	}
	cookieFound := false
	for _, cookie := range resp.Cookies() {
		if name := cookie.Name; name == "auth-jwt" {
			cookieFound = true
			if got, want := cookie.Value, jwtString; got != want {
				t.Errorf("wrong auth-jwt cookie: %v, want: %v", got, want)
			} else {
				break
			}
		}
		if !cookieFound {
			t.Error("auth-jwt cookie not set")
		}
	}

}
