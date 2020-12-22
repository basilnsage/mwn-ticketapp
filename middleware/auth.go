// middleware for managing JWT auth tokens
package middleware

import (
	"errors"
	"fmt"

	"github.com/dgrijalva/jwt-go"
)

type JWTValidator struct {
	key    []byte
	signer jwt.SigningMethod
}

func NewJWTValidator(key []byte, method string) (*JWTValidator, error) {
	switch method {
	case "HS256":
		return &JWTValidator{key, jwt.SigningMethodHS256}, nil
	default:
		return nil, fmt.Errorf("unsupported signing method: %v", method)
	}
}

func (v *JWTValidator) Parse(tokenString string) (*jwt.Token, error) {
	// keyFunc: check JWT headers to make it specifies the correct signature alg
	checkHeaders := func(token *jwt.Token) (interface{}, error) {
		untrustedAlg, ok := token.Header["alg"]
		if !ok {
			return nil, errors.New("no alg specified")
		}
		if untrustedAlg != v.signer.Alg() {
			return nil, fmt.Errorf("unexpected alg detected: %s", untrustedAlg)
		}
		return v.key, nil
	}

	token, err := jwt.Parse(tokenString, checkHeaders)
	if err != nil {
		return nil, fmt.Errorf("could not parse token: %v", err)
	}
	if !token.Valid {
		return nil, errors.New("JWT is not valid")
	}
	return token, nil
}

type UserClaims struct {
	Email string
	Id    string
}

func NewUserClaims(email, id string) *UserClaims {
	return &UserClaims{email, id}
}

func (u *UserClaims) Tokenize(v *JWTValidator) (string, error) {
	claims := jwt.MapClaims{"email": u.Email, "id": u.Id}
	token := jwt.NewWithClaims(v.signer, claims)
	signedTokenString, err := token.SignedString(v.key)
	if err != nil {
		return "", fmt.Errorf("could not sign token: %v", err)
	}
	return signedTokenString, nil
}

// WARNING: this may overwrite any already set fields in the UserClaims struct
func (u *UserClaims) NewFromToken(v *JWTValidator, token string) error {
	jwtToken, err := v.Parse(token)
	if err != nil {
		return err
	}

	claims := jwtToken.Claims
	if email, ok := claims.(jwt.MapClaims)["email"]; ok {
		switch t := email.(type) {
		case string:
			u.Email = t
		}
	}
	if id, ok := claims.(jwt.MapClaims)["id"]; ok {
		switch t := id.(type) {
		case string:
			u.Id = t
		}
	}

	return nil
}
