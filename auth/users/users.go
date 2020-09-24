package users

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log"
	"strings"
	"time"

	"github.com/go-playground/validator"
)

var v = validator.New()

func init() {
	if err := v.RegisterValidation("passwd", func(fl validator.FieldLevel) bool {
		return len(fl.Field().String()) >= 8
	}); err != nil {
		log.Fatalf("validator.RegisterValidation: %v", err)
	}
	if err := v.RegisterValidation("strnonblank", func(fl validator.FieldLevel) bool {
		return fl.Field().String() != ""
	}); err != nil {
		log.Fatalf("validator.RegisterValidation: %v", err)
	}
}

type Claims struct {
	Email string      `json:"email"`
	UID   string `json:"uid"`
}

func (pc Claims) Valid() error {
	if pc.Email == "" || pc.UID == "" {
		return errors.New("claims do not represent a user")
	}
	return nil
}

type User struct {
	Email    string `validate:"required,email,strnonblank"`
	Password string `validate:"required,passwd,strnonblank"`
	Hash []byte
	uid      interface{}
}

func NewUser(email string, password string) (*User, error) {
	hash, err := hashPass(password)
	if err != nil {
		return nil, err
	}
	return &User{
		email,
		password,
		hash,
		nil,
	}, nil
}

func hashPass(pass string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
}

func (u User) Validate(exempt map[string]interface{}) error {
	err := v.Struct(u)
	if err != nil {
		invalidTags := make([]string, 0)
		for _, err := range err.(validator.ValidationErrors) {
			tag := err.Tag()
			if _, ok := exempt[tag]; !ok {
				invalidTags = append(invalidTags, err.Tag())
			}
		}
		if len(invalidTags) > 0 {
			invalidFieldString := fmt.Sprintf("unable to validate these tags: %v", strings.Join(invalidTags, ","))
			return errors.New(invalidFieldString)
		}
	}
	return nil
}

func (u User) Exists(crud CRUD) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	users, err := crud.Read(ctx, u)
	if err != nil {
		return true, fmt.Errorf("unable to Read user from storage: %v", err)
	} else if len(users) > 1 {
		return true, fmt.Errorf("more than 1 user found: %v", len(users))
	}
	if len(users) == 0 {
		return false, nil
	} else {
		return true, nil
	}
}

func (u User) Write(crud CRUD) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	uid, err := crud.Write(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("could not persist user: %v", err)
	}
	u.uid = uid
	return uid, nil
}

func (u User) CreateSessionToken(signer Signer) (string, error) {
	claims := map[string]interface{}{
		"email": u.Email,
		"id": u.uid,
	}
	token, err := signer.Sign(claims)
	if err != nil {
		return "", fmt.Errorf("unable to create session token: %v", err)
	}
	return token, nil
}