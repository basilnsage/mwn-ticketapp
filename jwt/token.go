package token

import (
	"errors"
	"fmt"

	"github.com/dgrijalva/jwt-go"
)

//type SignVerify interface {
//	Sign(map[string]interface{}) (string, error)
//	Parse([]byte, string) (*jwt.Token, error)
//}

type JWTValidator struct {
	key []byte
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

func (j *JWTValidator) Parse(tokenString string) (*jwt.Token, error) {
	// keyFunc: check JWT headers to make it specifies the correct signature alg
	checkHeaders := func(token *jwt.Token) (interface{}, error) {
		untrustedAlg, ok := token.Header["alg"]
		if !ok {
			return nil, errors.New("no alg specified")
		}
		if untrustedAlg != j.signer.Alg() {
			return nil, fmt.Errorf("unexpected alg detected: %s", untrustedAlg)
		}
		return j.key, nil
	}

	token, err := jwt.Parse(tokenString, checkHeaders)
	if err != nil {
		return nil, fmt.Errorf("jwt.Parse: %v", err)
	}
	if !token.Valid {
		return nil, errors.New("could not validate JWT")
	}
	return token, nil
}

func (j *JWTValidator) ParseWithClaims(tokenString string, claims jwt.Claims) (*jwt.Token, error) {
	// keyFunc: check JWT headers to make it specifies the correct signature alg
	checkHeaders := func(token *jwt.Token) (interface{}, error) {
		untrustedAlg, ok := token.Header["alg"]
		if !ok {
			return nil, errors.New("no alg specified")
		}
		if untrustedAlg != j.signer.Alg() {
			return nil, fmt.Errorf("unexpected alg detected: %s", untrustedAlg)
		}
		return j.key, nil
	}

	token, err := jwt.ParseWithClaims(tokenString, claims, checkHeaders)
	if err != nil {
		return nil, fmt.Errorf("jwt.Parse: %v", err)
	}
	if !token.Valid {
		return nil, errors.New("could not validate JWT")
	}
	return token, nil
}

func (j *JWTValidator) Sign(claims map[string]interface{}) (string, error) {
	token := jwt.NewWithClaims(j.signer, jwt.MapClaims(claims))
	signedTokenString, err := token.SignedString(j.key)
	if err != nil {
		return "", fmt.Errorf("jwt.Token.SignedString: %v", err)
	}
	return signedTokenString, nil
}
