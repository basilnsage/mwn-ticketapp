package main

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Ticket struct {
	Title   string  `bson:"title"`
	Price   float64 `bson:"price"`
	Version uint    `bson:"version"`
	Id      string  `bson:"_id,omitempty"`
}

type ticketsCRUD interface {
	create(Ticket) (string, error)
	read(string) (*Ticket, error)
	update(string, Ticket) (bool, error)
}

type ticketsCollection struct {
	collection *mongo.Collection
	timeout    time.Duration
}

func newTicketCollection(collection *mongo.Collection, timeout time.Duration) ticketsCRUD {
	return ticketsCollection{
		collection,
		timeout,
	}
}

func (t ticketsCollection) create(ticket Ticket) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()

	res, err := t.collection.InsertOne(ctx, ticket)
	if err != nil {
		return "", err
	}

	switch t := res.InsertedID.(type) {
	case string:
		return t, nil
	case primitive.ObjectID:
		return res.InsertedID.(primitive.ObjectID).Hex(), nil
	default:
		return "", errors.New("InsertOne resulting ID neither string nor ObjectID")
	}
}

func (t ticketsCollection) read(ticketId string) (*Ticket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()

	mongoId, err := primitive.ObjectIDFromHex(ticketId)
	if err != nil {
		return nil, err
	}

	res := t.collection.FindOne(ctx, bson.M{"_id": mongoId})
	if res.Err() != nil && res.Err() != mongo.ErrNoDocuments {
		return nil, res.Err()
	} else if res.Err() == mongo.ErrNoDocuments {
		return nil, nil
	}

	var ticket Ticket
	if err := res.Decode(&ticket); err != nil {
		return nil, err
	}

	return &ticket, nil
}

func (t ticketsCollection) update(ticketId string, ticket Ticket) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()

	mongoId, err := primitive.ObjectIDFromHex(ticketId)
	if err != nil {
		return false, err
	}

	filter := bson.M{"_id": mongoId}
	update := bson.M{"$set": bson.M{"title": ticket.Title, "price": ticket.Price, "version": ticket.Version}}
	res, err := t.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return false, err
	}
	if res.MatchedCount > 0 {
		return true, nil
	}
	return false, nil
}

func (t Ticket) MarshalBSON() ([]byte, error) {
	doc := bson.M{
		"title":   t.Title,
		"price":   t.Price,
		"Version": t.Version,
	}
	if t.Id != "" {
		if oid, err := primitive.ObjectIDFromHex(t.Id); err != nil {
			return nil, err
		} else {
			doc["_id"] = oid
		}
	}
	return bson.Marshal(doc)
}
