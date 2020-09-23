package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/basilnsage/mwn-ticketapp/auth/token"
	"github.com/basilnsage/mwn-ticketapp/auth/users"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
)

var (
	key = []byte("password")
	// { alg: HS256, typ: JWT }
	header = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	// { email: foo@example.com, uid: 5f47ec2c86ed3ef991cdfd94 }
	payload = "eyJlbWFpbCI6ImZvb0BleGFtcGxlLmNvbSIsInVpZCI6IjVmNDdlYzJjODZlZDNlZjk5MWNkZmQ5NCJ9"
	// HS256 signature with key "password"
	sig = "jrgWQhw5YFXm01UVbZ-ZWEpJgmM_iNXwwgPG4pJ6bcQ"
	sampleClaims = users.Claims{
		Email: "foo@example.com",
		UID: "5f47ec2c86ed3ef991cdfd94",
	}
	jwtString = fmt.Sprintf("%s.%s.%s", header, payload, sig)
	cookie = http.Cookie{
		Name:  "auth-jwt",
		Value: jwtString,
	}
)

func setup(t *testing.T) (*token.JWTValidator, *gin.Engine) {
	// create the JWT validator
	jwtValidator, err:= token.NewJWTValidator(key, "HS256")
	if err != nil {
		t.Fatalf("token.NewJWTValidator: %v", err)
	}

	eng := gin.Default()
	eng.Use(func (ctx *gin.Context) {
		ctx.Next()
		if len(ctx.Errors) > 0 {
			t.Errorf("ctx error: %v", ctx.Errors[0])
		}
	})
	return jwtValidator, eng
}

func parseBody(receiver *users.Claims, resp *http.Response) error {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ioutil.ReadAll: %v", err)
	}
	if err = json.Unmarshal(body, receiver); err != nil {
		return fmt.Errorf("json.Unmarshal: %v", err)
	}
	return nil
}

func TestWhoami(t *testing.T) {
	gin.SetMode(gin.TestMode)
	v, eng := setup(t)

	// stand up a test router and create a test route to invoke Whoami
	w := httptest.NewRecorder()
	eng.GET("/test", func(ctx *gin.Context) {
		Whoami(ctx, v)
	})

	// create an HTTP request and send it to the test route
	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	req.AddCookie(&cookie)
	eng.ServeHTTP(w, req)

	// check status
	resp := w.Result()
	if w.Code != 200 {
		t.Errorf("Whoami wrong status code: %v, want %v", w.Code, 200)
	}

	// check response body
	bodyJson := users.Claims{}
	if err := parseBody(&bodyJson, resp); err != nil {
		t.Errorf("Whoami could not parse resp body: %v", err)
	}
	if diff := cmp.Diff(sampleClaims, bodyJson); diff != "" {
		t.Errorf("Whoami: (-want, +got):\n%s", diff)
	}
}
