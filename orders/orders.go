package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Order struct {
	UserId    string      `bson:"userId"`
	Status    orderStatus `bson:"status"`
	ExpiresAt time.Time   `bson:"expiresAt"`
	TicketId  string      `bson:"ticketId"`
	Id        string      `bson:"_id,omitempty"`
}

type Orders []Order

func (o Orders) Len() int {
	return len(o)
}

func (o Orders) Less(i, j int) bool {
	return o[i].Id < o[j].Id
}

func (o Orders) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

type OrderReq struct {
	TicketId string `json:"ticketId"`
}

type OrderResp struct {
	Status    orderStatus
	ExpiresAt time.Time
	Ticket    Ticket
	Id        string
}

type ordersCRUD interface {
	create(Order) (string, error)
	read(string) (*Order, error)
	search(int64, []string, []string, []orderStatus) ([]Order, error)
	update(string, Order) (bool, error)
}

//func (o ordersCollection) searchBy(limit int64, ticketIds, userIds []string, statuses []orderStatus) ([]Order, error) {

type ordersCollection struct {
	collection *mongo.Collection
	timeout    time.Duration
}

func newOrdersCollection(collection *mongo.Collection, timeout time.Duration) ordersCRUD {
	return ordersCollection{
		collection,
		timeout,
	}
}

func (o ordersCollection) create(order Order) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), o.timeout)
	defer cancel()

	res, err := o.collection.InsertOne(ctx, order)
	if err != nil {
		return "", err
	}

	return res.InsertedID.(primitive.ObjectID).Hex(), nil
}

func (o ordersCollection) read(id string) (*Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), o.timeout)
	defer cancel()

	mongoId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	res := o.collection.FindOne(ctx, bson.M{"_id": mongoId})
	if res.Err() != nil && res.Err() != mongo.ErrNoDocuments {
		return nil, err
	} else if res.Err() == mongo.ErrNoDocuments {
		return nil, nil
	}

	var order Order
	if err := res.Decode(&order); err != nil {
		return nil, err
	}

	return &order, nil
}

func (o ordersCollection) search(limit int64, ticketIds, userIds []string, statuses []orderStatus) ([]Order, error) {
	filter := bson.M{}

	if len(ticketIds) == 1 {
		filter["ticketId"] = ticketIds[0]
	} else if len(ticketIds) > 1 {
		filter["ticketId"] = bson.M{"$in": ticketIds}
	}

	if len(userIds) == 1 {
		filter["userId"] = userIds[0]
	} else if len(userIds) > 1 {
		filter["userId"] = bson.M{"$in": userIds}
	}

	if len(statuses) == 1 {
		filter["status"] = statuses[0].String()
	} else if len(statuses) > 1 {
		var statusStrings []string
		for _, s := range statuses {
			statusStrings = append(statusStrings, s.String())
		}
		filter["status"] = bson.M{"$in": statusStrings}
	}

	ctx, cancel := context.WithTimeout(context.Background(), o.timeout)
	defer cancel()

	findOpts := options.Find().SetLimit(limit).SetSort(bson.M{"_id": 1})
	cursor, err := o.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}

	var orders []Order
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, err
	}

	return orders, nil
}

// can only update statuses for now
func (o ordersCollection) update(id string, order Order) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), o.timeout)
	defer cancel()

	mongoId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return false, err
	}

	filter := bson.M{"_id": mongoId}
	update := bson.M{"$set": bson.M{"status": order.Status.String()}}
	res, err := o.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return false, err
	}
	if res.MatchedCount > 0 {
		return true, nil
	}
	return false, nil
}

type orderStatus int

const (
	Created orderStatus = iota
	Cancelled
	AwaitingPayment
	Completed
)

func (s orderStatus) String() string {
	return []string{
		"Created",
		"Cancelled",
		"AwaitingPayment",
		"Completed",
	}[s]
}

func statusFromString(s string) (*orderStatus, error) {
	var status orderStatus
	var err error
	switch {
	case s == "Created":
		status = Created
	case s == "Cancelled":
		status = Cancelled
	case s == "AwaitingPayment":
		status = AwaitingPayment
	case s == "Completed":
		status = Completed
	default:
		err = fmt.Errorf("invalid status: %v", s)
	}
	return &status, err
}

func (s orderStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *orderStatus) UnmarshalJSON(b []byte) error {
	var status string
	if err := json.Unmarshal(b, &status); err != nil {
		return err
	}

	if os, err := statusFromString(status); err != nil {
		return err
	} else {
		*s = *os
	}
	return nil
}

func (s orderStatus) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bson.MarshalValue(s.String())
}

func (s *orderStatus) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	rv := bson.RawValue{Type: t, Value: b}
	var status string
	if err := rv.Unmarshal(&status); err != nil {
		return err
	}

	if os, err := statusFromString(status); err != nil {
		return err
	} else {
		*s = *os
	}
	return nil
}
