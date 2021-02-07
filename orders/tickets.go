package main

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
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
	return "", nil
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
