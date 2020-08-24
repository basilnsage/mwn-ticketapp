package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

var v = validator.New()

func initValidator() error {
	err := v.RegisterValidation("passwd", func(fl validator.FieldLevel) bool {
		return len(fl.Field().String()) >= 8
	})
	if err != nil {
		return fmt.Errorf("validator.RegisterValidation: %v", err)
	}
	err = v.RegisterValidation("strnonblank", func(fl validator.FieldLevel) bool {
		return fl.Field().String() != ""
	})
	if err != nil {
		return fmt.Errorf("validator.RegisterValidation: %v", err)
	}
	return nil
}

type user struct {
	Email    string `validate:"required,email,strnonblank"`
	Password string `validate:"required,passwd,strnonblank"`
	uid      interface{}
}
type privClaims struct {
	Email string      `json:"email"`
	UID   interface{} `json:"uid"`
}

func newUser(email string, password string) *user {
	return &user{
		email,
		password,
		primitive.NilObjectID,
	}
}

func (u user) validate(exempt map[string]interface{}) error {
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
	claims := privClaims{
		u.Email,
		u.uid,
	}
	raw, err := jwt.Signed(sharedSigner).Claims(claims).CompactSerialize()
	if err != nil {
		return "", fmt.Errorf("user.jwt: %v", err)
	}
	return raw, nil
}

func verifyJWT(token string) (*privClaims, error) {
	jws, err := jose.ParseSigned(token)
	if err != nil {
		return nil, fmt.Errorf("jose.ParseSigned: %v", err)
	}
	data, err := jws.Verify(sharedPass)
	if err != nil {
		return nil, fmt.Errorf("jose.JSONWebSignature.Verify: %v", err)
	}
	claims := privClaims{}
	if err = json.Unmarshal(data, &claims); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %v", err)
	}
	return &claims, nil

}
