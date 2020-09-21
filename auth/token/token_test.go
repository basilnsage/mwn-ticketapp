package token

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/go-cmp/cmp"
	"testing"
)

var (
	key = []byte("password")
	// created with JWT.io
	headerMap = map[string]interface{}{
		"alg": "HS256",
		"typ": "JWT",
	}
	// encoded headerMap = jwt header
	header = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	// jwt claims
	claims = jwt.MapClaims{"name": "testing"}
	// encoded claims == jwt payload
	payload = "eyJuYW1lIjoidGVzdGluZyJ9"
	// HS256 signature with key "password"
	sig = "XoafP1e4AoyZGV7Pc6x9UiDXjU30fhbXXve8n6Ke9A4"
)

func TestNewJWTValidator(t *testing.T) {
	_, err := NewJWTValidator(key, "not implemented")
	if got, want := err.Error(), "unsupported signing method: not implemented"; got != want {
		t.Errorf("NewJWTValidator() wrong error: %v, want %v", got, want)
	}

	got, err := NewJWTValidator([]byte("password"), "HS256")
	if err != nil {
		t.Errorf("NewJWTValidator() unexpected error: %v", err)
	}
	want := &JWTValidator{key, jwt.SigningMethodHS256}
	if diff := cmp.Diff(got, want, cmp.AllowUnexported(JWTValidator{})); diff != "" {
		t.Errorf("NewJWTValidator() mismatch (-want +got):\n%s", diff)
	}
}

func TestParse(t *testing.T) {
	// modified header: { alg: none }
	// this tests the attack where alg: none is set and the sig is dropped
	noneAlgHeader := "eyJhbGciOiJub25lIn0K"
	tokenString := fmt.Sprintf("%s.%s.%s", noneAlgHeader, payload, sig)

	v, err := NewJWTValidator(key, "HS256")
	if err != nil {
		t.Fatalf("NewJWTValidator() error not nil: %v", err)
	}

	_, err = v.Parse(tokenString)
	if got, want := err.Error(), "jwt.Parse: unexpected alg detected: none"; got != want {
		t.Errorf("JWTValidator.Parse unexpected error: %v, want %v", got, want)
	}

	token, err := v.Parse(fmt.Sprintf("%s.%s.%s", header, payload, sig))
	if err != nil {
		t.Fatalf("JWTValidator.Parse() error not nil: %v", err)
	}

	if !token.Valid {
		t.Error("JWTValidator.Parse() parsed token invalid")
	}
	if diff := cmp.Diff(token.Header, headerMap); diff != "" {
		t.Errorf("JWTValidator.Parse() Header mismatch (-want, +got):\n%s", diff)
	}
	if diff := cmp.Diff(token.Claims, claims); diff != "" {
		t.Errorf("JWTValidator.Parse() Claims mismatch (-want, +got):\n%s", diff)
	}
}

func TestSign(t *testing.T) {
	v, err := NewJWTValidator(key, "HS256")
	if err != nil {
		t.Fatalf("NewJWTValidator() error not nil: %v", err)
	}

	tokenString, err := v.Sign(claims)
	if err != nil {
		t.Fatalf("JWTValidator.Sign() error not nil: %v", err)
	}
	if got, want := tokenString, fmt.Sprintf("%s.%s.%s", header, payload, sig); got != want {
		t.Errorf("JWTValidator.Sign() token mismatch: %v, want %v", got, want)
	}
}