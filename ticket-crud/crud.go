package main

import (
	"context"
	"fmt"
	prometrics "github.com/basilnsage/prometheus-gin-metrics"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoCollection interface {
	Create(string, float64) (interface{}, error)
	ReadOne(interface{}) (*Ticket, error)
	ReadAll() ([]*Ticket, error)
	Update(interface{}, string, float64) (interface{}, error)
}

type CRUD struct {
	coll *mongo.Collection
}

type Ticket struct {
	Title string
	Price float64
}

func newRouter(jwtKey string, crud MongoCollection) (*gin.Engine, error) {
	userValidator, err := userAuthMiddleware(jwtKey)
	if err != nil {
		return nil, fmt.Errorf("unable to init user validation middleware: %v", err)
	}

	r := gin.Default()
	promRegistry := prometrics.NewRegistry()
	r.Use(promRegistry.ReportDuration(
		[]float64{0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0},
	))

	r.GET("/tickets/metrics", promRegistry.DefaultHandler)
	ticketRoutes := r.Group("/api/tickets")
	ticketRoutes.POST("/create", userValidator, func(c *gin.Context) {
		serveCreate(c, crud)
	})

	return r, nil
}

func newCrud(ctx context.Context, connStr string, db string, coll string) (MongoCollection, error) {
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

	return &CRUD{client.Database(db).Collection(coll)}, nil
}

func (c *CRUD) Create(title string, price float64) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()
	res, err := c.coll.InsertOne(ctx, bson.M{"title": title, "price": price})
	if err != nil {
		return nil, err
	}
	id := res.InsertedID.(primitive.ObjectID)
	return id.Hex(), nil
}

func (c *CRUD) ReadOne(id interface{}) (*Ticket, error) {
	return nil, nil
}

func (c *CRUD) ReadAll() ([]*Ticket, error) {
	return nil, nil
}

func (c *CRUD) Update(id interface{}, title string, price float64) (interface{}, error) {
	return nil, nil
}

func serveCreate(c *gin.Context, crud MongoCollection) {
	// parse gin context for JSON body
	tik := new(Ticket)
	if err := c.BindJSON(tik); err != nil {
		WarningLogger.Printf("could not parse body of request, err: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"errors": []string{"unable to process request"},
		})
		return
	}

	// validate fields
	var validationErrors []string
	if tik.Title == "" {
		validationErrors = append(validationErrors, "please specify a title")
	}
	if tik.Price < 0.0 {
		validationErrors = append(validationErrors, "price cannot be less than 0")
	}
	if len(validationErrors) > 0 {
		WarningLogger.Printf("ticket validation failed, err: %v", strings.Join(validationErrors, " | "))
		c.JSON(http.StatusBadRequest, gin.H{
			"errors": validationErrors,
		})
		return
	}

	// insert new ticket object into DB
	tikId, err := crud.Create(tik.Title, tik.Price)
	if err != nil {
		ErrorLogger.Printf("failed to write ticket to database, err: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"errors": []string{"unable to save ticket"},
		})
		return
	}

	// return object ID, title, price
	// TODO: define response struct somewhere for testing
	c.JSON(http.StatusCreated, gin.H{
		"id": tikId,
		"title": tik.Title,
		"price": tik.Price,
	})
	InfoLogger.Printf("new ticket saved with id: %v", tikId)
}