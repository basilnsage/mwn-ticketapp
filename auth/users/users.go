package users

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-playground/validator"
	"golang.org/x/crypto/bcrypt"
	"log"
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
	// do we really need the ID in the claims? why would the user need to see this?
	// if we do want to include the ID we should think about methods to create a new user
	// by fetching a matching user from the DB; match done via email address
	ID   string `json:"id"`
}

func (pc Claims) Valid() error {
	if pc.Email == "" {
		return errors.New("claims do not represent a user")
	}
	return nil
}

type User struct {
	Email    string `validate:"required,email,strnonblank"`
	Hash []byte
	// pulling user from mongo populates this field with the pass's bcrypt hash
	// when the user is created, populated with the plaintext password --> bad n sad
	// TODO: figure this out
	// unexport Password field
	// build validation into NewUser func
	// NewUser(email, password, hash)... perform check on password/hash to make sure they equate
	password string `validate:"required,passwd,strnonblank"`
	// issues with reintroducing Hash
	// how to mock? new hash generated every time the password is run through bcrypt
	Uid      interface{}
}

func validatePassword(password string) error {
	switch {
	case len(password) < 8:
		return errors.New("password is too short")
	}
	return nil
}

func NewUser(email string, password string, hash []byte) (*User, error) {
	// validate email
	type Email struct {
		Value string `validate:"required,email,strnonblank"`
	}
	if err := v.Struct(Email{email}); err != nil {
		return nil, fmt.Errorf("invalid email: %v", err)
	}

	// validate password
	if err := validatePassword(password); err != nil {
		return nil, fmt.Errorf("invalid password: %v", err)
	}

	// check that password and hash match
	if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil {
		return nil, fmt.Errorf("password and hash do not match: %v", err)
	}

	return &User{
		Email: email,
		Hash: hash,
		password: password,
		Uid: nil,
	}, nil
}

func (u *User) SetUID(uid interface{}){
	u.Uid = uid
}

func (u User) Exists(ctx context.Context, crud CRUD) (bool, error) {
	users, err := crud.Read(ctx, u)
	if err != nil {
		return true, fmt.Errorf("unable to fetch users from DB: %v", err)
	} else if len(users) == 0 {
		return false, nil
	} else {
		return true, nil
	}
}

func (u *User) Write(ctx context.Context, crud CRUD) (interface{}, error) {
	uid, err := crud.Write(ctx, *u)
	if err != nil {
		return nil, fmt.Errorf("could not persist user: %v", err)
	}
	u.SetUID(uid)
	return uid, nil
}

func (u User) DoesPassMatch(ctx context.Context, crud CRUD) (bool, error) {
	foundUsers, err := crud.Read(ctx, u)
	switch {
	case err != nil:
		return false, fmt.Errorf("unable to fetch users from DB: %v", err)
	case len(foundUsers) > 1:
		return false, fmt.Errorf("too many users found! only one user expected")
	case len(foundUsers) == 0:
		return false, fmt.Errorf("no users expected! only one user expected")
	}
	if err = bcrypt.CompareHashAndPassword([]byte(foundUsers[0].Hash), []byte(u.password)); err != nil {
		return false, fmt.Errorf("bcrypt.CompareHashAndPassword: %v", err)
	}
	return true, nil
}

func (u User) CreateSessionToken(ctx context.Context, c CRUD, signer Signer) (string, error) {
	if u.Uid == nil {
		res, err := c.Read(ctx, u)
		if err != nil {
			return "", fmt.Errorf("could not fetch user from DB: %v", err)
		}
		// WARNING: because this is a value receiver this will NOT update the UID of the underlying user
		u.Uid = res[0].Uid
	}

	claims := map[string]interface{}{
		"email": u.Email,
		"id": u.Uid,
	}
	token, err := signer.Sign(claims)
	if err != nil {
		return "", fmt.Errorf("unable to create session token: %v", err)
	}
	return token, nil
}