package main

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CRUD interface {
	Create(string, float64, string) (string, error)
	ReadOne(string) (*TicketResp, error)
	ReadAll() ([]TicketResp, error)
	Update(string, string, float64) (bool, error)
}

type MongoColl struct {
	coll    *mongo.Collection
	timeout time.Duration
}

func newCrud(timeout time.Duration, connStr, db, coll string) (CRUD, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, err := mongo.NewClient(options.Client().ApplyURI(connStr))
	if err != nil {
		return nil, err
	}
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return &MongoColl{client.Database(db).Collection(coll), timeout}, nil
}

func (c *MongoColl) Create(title string, price float64, owner string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	res, err := c.coll.InsertOne(ctx, bson.M{"title": title, "price": price, "owner": owner})
	if err != nil {
		return "", err
	}
	id := res.InsertedID.(primitive.ObjectID)
	return id.Hex(), nil
}

func (c *MongoColl) ReadOne(id string) (*TicketResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	mId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	tik := &TicketResp{}
	res := c.coll.FindOne(ctx, bson.M{"_id": mId})
	if err := res.Decode(tik); err == mongo.ErrNoDocuments {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return tik, nil
}

func (c *MongoColl) ReadAll() ([]TicketResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	var results []TicketResp
	cursor, err := c.coll.Find(ctx, bson.D{})
	if err != nil {
		return []TicketResp{}, err
	}

	if err := cursor.All(ctx, &results); err != nil {
		return []TicketResp{}, err
	}

	return results, nil
}

func (c *MongoColl) Update(id string, title string, price float64) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// convert hex ID string to mongo ObjectID
	objId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return false, err
	}

	res, err := c.coll.UpdateOne(ctx, bson.M{"_id": objId}, bson.M{"$set": bson.M{"title": title, "price": price}})
	if err != nil {
		return false, err
	}
	if res.MatchedCount == 0 || res.ModifiedCount == 0 {
		return false, nil
	}

	return true, nil
}
