package main

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/square/go-jose.v2"
)

var sharedSigner jose.Signer

func initSigner() error {
	var err error
	pass, exists := os.LookupEnv("JWT_SIGN_KEY")
	if !exists {
		return errors.New("JWT_SIGN_KEY not set, cannot create JWT signer")
	}
	sharedSigner, err = jose.NewSigner(jose.SigningKey{Algorithm: jose.HS512, Key: []byte(pass)}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return fmt.Errorf("jose.NewSigner: %v", err)
	}
	return nil
}

