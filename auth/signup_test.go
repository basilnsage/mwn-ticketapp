package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/basilnsage/mwn-ticketapp/auth/users"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSignupFlow(t *testing.T) {
	signer := new(mockSigner)
	crud := new(mockCRUD)
	ctx := context.Background()
	//user, err := users.NewUser(email, pass, passHash)
	//if err != nil {
	//	t.Fatalf("unable to create new user: %v", err)
	//}
	//userBytes, err := proto.Marshal(&protos.SignIn{
	//	Username: "foo@example.com",
	//	Password: "password",
	//})
	//if err != nil {
	//	t.Fatalf("unable to marshal user proto: %v", err)
	//}

	crud.On("Read", ctx, mock.MatchedBy(checkTestUser)).Return(make([]users.User, 0), nil)
	crud.On("Write", ctx, mock.MatchedBy(checkTestUser)).Return(uid, nil)
	signer.On("Sign", map[string]interface{}{
		"email": email,
		"id":    uid,
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
		SignupUser(ctx, ginCtx, crud, signer)
	})

	w := httptest.NewRecorder()
	payload, err := json.Marshal(&userFormData{Username: email, Password: pass})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	//req, err := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(userBytes))
	req, err := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	eng.ServeHTTP(w, req)

	resp := w.Result()
	if got, want := resp.StatusCode, http.StatusCreated; got != want {
		t.Errorf("wrong response code: %v, want: %v", got, want)
	}
	respString, _ := ioutil.ReadAll(resp.Body)
	if got, want := string(respString), "signup complete"; got != want {
		t.Errorf("wrong response status: %v, want: %v", got, want)
	}
}
