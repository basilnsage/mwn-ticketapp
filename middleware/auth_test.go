package middleware

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dgrijalva/jwt-go"
)

func TestNewJWTValidator(t *testing.T) {
	v, err := NewJWTValidator([]byte("password"), "HS256")
	if err != nil {
		t.Errorf("unexpected error creating new JWT validator: %v", err)
	}

	if got, want := v.key, []byte("password"); !bytes.Equal(got, want) {
		t.Errorf("new JWT validator does not have expected key: %v, want %v", string(got), string(want))
	}
	if got, want := v.signer.Alg(), "HS256"; got != want {
		t.Errorf("new JWT validator does not have expected signing alg: %v, want %v", got, want)
	}

	_, err = NewJWTValidator([]byte("password"), "DNE")
	if err != nil {
		if got, want := err.Error(), "unsupported signing method: DNE"; got != want {
			t.Errorf("new JWT validator did not return expected error: %v, want %v", got, want)
		}
	} else {
		t.Errorf("bad NewJWTValidtor call should result in non-nil error")
	}
}

func TestSign(t *testing.T) {
	v, _ := NewJWTValidator([]byte("password"), "HS256")
	claims := UserClaims{"foo@bar.com", "0"}
	token, err := claims.Tokenize(v)
	if err != nil {
		t.Errorf("unexpected error while creating UserClaims JWT: %v", err)
	}

	parts := strings.Split(token, ".")
	// alg: HS256, typ: JWT
	if got, want := parts[0], "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"; got != want {
		t.Errorf("unexpected JWT header: %v, want %v", got, want)
	}
	// email: foo@bar.com, id: 0
	if got, want := parts[1], "eyJlbWFpbCI6ImZvb0BiYXIuY29tIiwiaWQiOiIwIn0"; got != want {
		t.Errorf("unexpected JWT payload: %v, want %v", got, want)
	}
}

func TestParse(t *testing.T) {
	v, _ := NewJWTValidator([]byte("password"), "HS256")
	claims := UserClaims{"foo@bar.com", "0"}
	tokenStr, _ := claims.Tokenize(v)

	// alg: HS256, typ: JWT, no body
	token, err := v.Parse(tokenStr)
	if err != nil {
		t.Errorf("unexpected error while parsing token string: %v", err)
	}

	if err = token.Claims.Valid(); err != nil {
		t.Errorf("parsed token claims invalid: %v", err)
	}
	if got, want := token.Claims.(jwt.MapClaims)["email"], "foo@bar.com"; got != want {
		t.Errorf("JWT does not contain correct email: %v, want %v", got, want)
	}
	if got, want := token.Claims.(jwt.MapClaims)["id"], "0"; got != want {
		t.Errorf("JWT does not contain correct id: %v, want %v", got, want)
	}
}

func TestNewFromToken(t *testing.T) {
	v, _ := NewJWTValidator([]byte("password"), "HS256")
	// gen with jwt.io
	tokenParts := []string{
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		"eyJlbWFpbCI6ImZvb0BiYXIuY29tIiwiaWQiOiIwIn0",
		"itigCGh9yImD1NEW-hv5MS__uMdhLcz4qcdrvA4c3A4",
	}

	var claims UserClaims
	if err := claims.NewFromToken(v, strings.Join(tokenParts, ".")); err != nil {
		t.Errorf("unexpected error while parsing token: %v", err)
	}
	if got, want := claims.Email, "foo@bar.com"; got != want {
		t.Errorf("incorrect parsed Email: %v, want %v", got, want)
	}
	if got, want := claims.Id, "0"; got != want {
		t.Errorf("incorrect parsed Id: %v, want %v", got, want)
	}
}
