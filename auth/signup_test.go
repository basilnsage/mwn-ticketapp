package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/basilnsage/mwn-ticketapp/common/protos"
	"github.com/golang/protobuf/proto"
	"github.com/basilnsage/mwn-ticketapp/auth/users"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
)

type mockSigner struct {
	mock.Mock
}

type mockCRUD struct {
	mock.Mock
}

func (m *mockSigner) Sign(claims map[string]interface{}) (string, error) {
	args := m.Called(claims)
	return args.String(0), args.Error(1)
}

func (m *mockCRUD) Read(ctx context.Context, user users.User) ([]users.User, error) {
	args := m.Called(ctx, user)
	return args.Get(0).([]users.User), args.Error(1)
}

func (m *mockCRUD) Write(ctx context.Context, user users.User) (interface{}, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(interface{}), args.Error(1)
}

func TestSignupFlow(t *testing.T) {
	signer := new(mockSigner)
	crud := new(mockCRUD)
	ctx := context.Background()
	user, err := users.NewUser(email, "password")
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

	signer.On("Sign", map[string]interface{}{
		"email": email,
		"id": interface{}(nil),
	}).Return(jwtString, nil)
	crud.On("Read", ctx, *user).Return(make([]users.User, 0), nil)
	crud.On("Write", ctx, *user).Return(uid, nil)

	gin.SetMode(gin.TestMode)
	eng := gin.Default()
	eng.Use(func (ctx *gin.Context) {
		ctx.Next()
		if len(ctx.Errors) > 0 {
			t.Errorf("ctx error: %v", ctx.Errors[0])
		}
	})
	eng.POST("/test", func(ginCtx *gin.Context) {
		SignupUser(ctx, ginCtx, crud, signer)
	})


	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(userBytes))
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
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


