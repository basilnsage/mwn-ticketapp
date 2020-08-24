package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-playground/validator/non-standard/validators"
	"strings"
	"time"

	"github.com/go-playground/validator"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/square/go-jose.v2/jwt"
)

var v = validator.New()

type user struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required,passwd"`
	uid interface{}
}

func newUser(email string, password string) *user {
	return &user{
		email,
		password,
		primitive.NilObjectID,
	}
}

func (u user) validate(exempt map[string]interface{}) error {
	_ = v.RegisterValidation("passwd", func(f validator.FieldLevel) bool {
		return len(f.Field().String()) >= 8
	})
	_ = v.RegisterValidation("nonblank", validators.NotBlank)
	err := v.Struct(u)
	if err != nil {
		invalidTags := make([]string, 0)
		for _, err := range err.(validator.ValidationErrors) {
			tag := err.Tag()
			if _, ok := exempt[tag]; ok {
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

func (u user) passwdHash() ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
}

func (u user) passOk(mClient *mongo.Client) (bool, error) {
	temp := user{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	users := mClient.Database(authDB).Collection(authCollection)
	err := users.FindOne(ctx, bson.M{"username": u.Email}).Decode(&temp)
	if err == mongo.ErrNoDocuments {
		return false, errors.New("user.passOk: no matching user record found")
	} else if err != nil {
		return false, fmt.Errorf("user.checkPassword: %v", err)
	}
	if err = bcrypt.CompareHashAndPassword([]byte(temp.Password), []byte(u.Password)); err != nil {
		return false, fmt.Errorf("user.passOk: %v", err)
	}
	return true, nil
}


func (u user) exists(mClient *mongo.Client) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	users := mClient.Database(authDB).Collection(authCollection)
	err := users.FindOne(ctx, bson.M{"username": u.Email}).Decode(&user{})
	if err == mongo.ErrNoDocuments {
		return false, nil
	} else if err != nil {
		return true, err
	} else {
		return true, nil
	}
}

// write/persist user to DB
func (u *user) write(mClient *mongo.Client) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	users := mClient.Database(authDB).Collection(authCollection)
	hash, err := u.passwdHash()
	if err != nil {
		return &mongo.InsertOneResult{}, err
	}
	fmt.Println(hash)
	res, err := users.InsertOne(ctx, bson.M{"username": u.Email, "password": hash})
	if err != nil {
		return &mongo.InsertOneResult{}, err
	}
	u.uid = res.InsertedID
	return res, nil
}

func (u user) jwt() (string, error) {
	if sharedSigner == nil {
		return "", errors.New("user.jwt: JWT signer uninitialized, cannot sign JWT")
	}
	privClaims := struct {
		Email string `json:"email"`
		UID interface{} `json:"uid"`
	}{
		u.Email,
		u.uid,
	}
	raw, err := jwt.Signed(sharedSigner).Claims(privClaims).CompactSerialize()
	if err != nil {
		return "", fmt.Errorf("user.jwt: %v", err)
	}
	return raw, nil
}
