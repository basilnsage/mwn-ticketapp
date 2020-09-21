package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type SignVerify interface {
	Sign() error
	Verify(string, interface{}) error
}

type JWT struct {
	key string
	signer jose.Signer
}

func NewJWT(key string) (*JWT, error) {
	signKey := jose.SigningKey{
		Algorithm: jose.HS512,
		Key: key,
	}
	signer, err := jose.NewSigner(signKey, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return nil, fmt.Errorf("jose.NewSigner: %v", err)
	}
	return &JWT{key, signer}, nil
}

func (j *JWT) Verify(token string, receiver interface{}) error {
	jws, err := jose.ParseSigned(token)
	if err != nil {
		return fmt.Errorf("jose.ParseSigned: %v", err)
	}

	data, err := jws.Verify(key)
	if err != nil {
		return fmt.Errorf("jose.JSONWebSignature.Verify: %v", err)
	}

	var token = new(jwt.Claims)
	if err = json.Unmarshal(data, token); err != nil {
		return fmt.Errorf("json.Unmarshal: %v", err)
	}

	jwt
	return nil
}

func Sign() {
	fmt.Println(signer)
}
