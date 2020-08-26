package jwt

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"gopkg.in/square/go-jose.v2"
)

var signer jose.Signer
var sharedPass []byte

func getPass() error {
	pass, exists := os.LookupEnv("JWT_SIGN_KEY")
	if !exists {
		return errors.New("getPass: no JWT_SIGN_KEY env var found")
	}
	sharedPass = []byte(pass)
	return nil
}

func InitSigner() error {
	var err error
	if err := getPass(); err != nil {
		return fmt.Errorf("InitSigner: %v", err)
	}
	signKey := jose.SigningKey{
		Algorithm: jose.HS512,
		Key: sharedPass,
	}
	signer, err = jose.NewSigner(signKey, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return fmt.Errorf("jose.NewSigner: %v", err)
	}
	return nil
}

func Verify(token string, receiver interface{}) error {
	jws, err := jose.ParseSigned(token)
	if err != nil {
		return fmt.Errorf("jose.ParseSigned: %v", err)
	}
	data, err := jws.Verify(sharedPass)
	if err != nil {
		return fmt.Errorf("jose.JSONWebSignature.Verify: %v", err)
	}
	if err = json.Unmarshal(data, receiver); err != nil {
		return fmt.Errorf("json.Unmarshal: %v", err)
	}
	return nil
}

func Sign() {
	fmt.Println(signer)
}
