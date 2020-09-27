package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"time"

	"github.com/basilnsage/mwn-ticketapp/auth/users"
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
	cursor, err := uc.c.Find(ctx, bson.M{"email": user.Email})
	if err != nil {
		return nil, fmt.Errorf("mongo.Collection.Find: %v", err)
	}

	if err = cursor.All(ctx, &foundUsers); err != nil {
		return nil, fmt.Errorf("mongo.Cursor.All: %v", err)
	}
	return foundUsers, nil
}

func (uc userColl) Write(ctx context.Context, user users.User) (interface{}, error) {
	res, err := uc.c.InsertOne(ctx, bson.M{"email": user.Email, "hash": user.Hash})
	if err != nil {
		return nil, fmt.Errorf("mongo.Collection.InsertOne: %v", err)
	}
	return res.InsertedID, nil
}
