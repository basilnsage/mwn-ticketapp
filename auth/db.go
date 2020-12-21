package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/basilnsage/mwn-ticketapp/auth/users"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var client *mongo.Client

type userColl struct {
	c *mongo.Collection
}

func InitMongo() error {
	var err error
	opts := options.Client().ApplyURI("mongodb://auth-mongo-svc:27017")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err = mongo.Connect(ctx, opts)
	if err != nil {
		return fmt.Errorf("unable to connect to mongo cluster: %v", err.Error())
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return fmt.Errorf("unable to ping mongo cluster: %v", err.Error())
	}
	return nil
}

func GetClient() *mongo.Client {
	return client
}

func CloseClient() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return client.Disconnect(ctx)
}

func GetDatabase(mc *mongo.Client, db string) *mongo.Database {
	return mc.Database(db)
}

func GetCollection(db *mongo.Database, coll string) *mongo.Collection {
	return db.Collection(coll)
}

func (uc userColl) Read(ctx context.Context, user users.User) ([]users.User, error) {
	var foundUsers []users.User
	var res []bson.M
	cursor, err := uc.c.Find(ctx, bson.M{"email": user.Email})
	if err != nil {
		return nil, fmt.Errorf("mongo.Collection.Find: %v", err)
	}

	if err = cursor.All(ctx, &res); err != nil {
		return nil, fmt.Errorf("mongo.Cursor.All: %v", err)
	}
	for _, result := range res {
		email, hash, uid, err := parseUserRes(result)
		if err != nil {
			return nil, err
		}
		foundUsers = append(foundUsers, users.User{
			Email: email,
			Hash: hash,
			Uid: uid,
		})
	}
	return foundUsers, nil
}

func parseUserRes(res bson.M) (string, []byte, string, error) {
	var email, uid string
	var hash []byte

	emailRes, ok := res["email"]
	if !ok {
		return "", nil, "", errors.New("did not required Email field from mongo result")
	}
	switch t := emailRes.(type) {
	case string:
		email = t
	default:
		return "", nil, "", errors.New("found Email field but could not cast it to string")
	}

	hashRes, ok := res["hash"]
	if !ok {
		return "", nil, "", errors.New("did not required Hash field from mongo result")
	}
	switch t := hashRes.(type) {
	case primitive.Binary:
		hash = t.Data
	default:
		return "", nil, "", errors.New("found Hash field but could not cast it to []byte")
	}

	uidRes, ok := res["_id"]
	if !ok {
		return "", nil, "", errors.New("did not required _id field from mongo result")
	}
	switch t := uidRes.(type) {
	case primitive.ObjectID:
		uid = t.Hex()
	default:
		return "", nil, "", errors.New("found _id field but could not cast it to string")
	}

	return email, hash, uid, nil
}

func (uc userColl) Write(ctx context.Context, user users.User) (interface{}, error) {
	res, err := uc.c.InsertOne(ctx, bson.M{"email": user.Email, "hash": user.Hash})
	if err != nil {
		return nil, fmt.Errorf("mongo.Collection.InsertOne: %v", err)
	}
	return res.InsertedID, nil
}
