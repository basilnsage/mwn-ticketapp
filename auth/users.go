package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/basilnsage/mwn-ticketapp/common/protos"
	"github.com/go-playground/validator"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/proto"
)

var v = validator.New()

type user struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required,passwd"`
}

func newUser(email string, password string) *user {
	return &user{
		email,
		password,
	}
}

func (u user) validate() error {
	_ = v.RegisterValidation("passwd", func(f validator.FieldLevel) bool {
		return len(f.Field().String()) >= 8
	})
	err := v.Struct(u)
	if err != nil {
		invalidFields := make([]string, 0)
		for _, err := range err.(validator.ValidationErrors) {
			invalidFields = append(invalidFields, err.Field())
		}
		if len(invalidFields) > 0 {
			invalidFieldString := fmt.Sprintf("unable to validate these fields: %v", strings.Join(invalidFields, ","))
			return errors.New(invalidFieldString)
		}
	}
	return nil
}

func (u user) passwdHash() ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
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
func (u user) write(mClient *mongo.Client) (*mongo.InsertOneResult, error) {
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
	return res, nil
}

func userFromPayload(data *[]byte) (userObj *user, err error) {
	userProto := &protos.SignIn{}
	err = proto.Unmarshal(*data, userProto)
	if err != nil {
		return
	}
	userObj = newUser(userProto.Username, userProto.Password)
	return
}
