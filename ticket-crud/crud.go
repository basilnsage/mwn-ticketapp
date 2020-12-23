package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/basilnsage/mwn-ticketapp/middleware"
	prometrics "github.com/basilnsage/prometheus-gin-metrics"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoCollection interface {
	Create(string, float64, string) (string, error)
	ReadOne(string) (*TicketResp, error)
	ReadAll() ([]*Ticket, error)
	Update(interface{}, string, float64) (interface{}, error)
}

type CRUD struct {
	coll *mongo.Collection
}

type Ticket struct {
	Title string
	Price float64
	Owner string `json:"id"`
}

type TicketReq struct {
	Title string
	Price float64
}

type TicketResp struct {
	Title string
	Price float64
	Owner string
	Id    string `bson:"_id"`
}

func newRouter(jwtKey string, crud MongoCollection) (*gin.Engine, error) {
	jwtValidator, err := middleware.NewJWTValidator([]byte(jwtKey), "HS256")
	if err != nil {
		return nil, fmt.Errorf("NewJWTValidator: %v", err)
	}

	r := gin.Default()
	promRegistry := prometrics.NewRegistry()
	r.Use(promRegistry.ReportDuration(
		[]float64{0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0},
	))

	r.GET("/tickets/metrics", promRegistry.DefaultHandler)
	ticketRoutes := r.Group("/api/tickets")
	ticketRoutes.POST(
		"/create",
		middleware.UserValidator(jwtValidator, "auth-jwt"),
		func(c *gin.Context) {
			serveCreate(c, crud, jwtValidator)
		},
	)
	ticketRoutes.GET("/:id", func(c *gin.Context) {
		serveReadOne(c, crud)
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

func (c *CRUD) Create(title string, price float64, owner string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res, err := c.coll.InsertOne(ctx, bson.M{"title": title, "price": price, "owner": owner})
	if err != nil {
		return "", err
	}
	id := res.InsertedID.(primitive.ObjectID)
	return id.Hex(), nil
}

func (c *CRUD) ReadOne(id string) (*TicketResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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

func (c *CRUD) ReadAll() ([]*Ticket, error) {
	return nil, nil
}

func (c *CRUD) Update(id interface{}, title string, price float64) (interface{}, error) {
	return nil, nil
}

func serveCreate(c *gin.Context, crud MongoCollection, v *middleware.JWTValidator) {
	// parse gin context for JSON body
	var tik TicketReq
	if err := c.BindJSON(&tik); err != nil {
		WarningLogger.Printf("could not parse body of request, err: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"errors": []string{"unable to process request"},
		})
		return
	}

	// parse user id from auth-jwt header
	jwtHeader := c.GetHeader("auth-jwt")
	if jwtHeader == "" {
		ErrorLogger.Print("no auth-jwt header found while creating ticket. This should never happen")
		c.JSON(http.StatusInternalServerError, gin.H{
			"errors": []string{"Internal server error"},
		})
		return
	}

	var userClaims middleware.UserClaims
	if err := userClaims.NewFromToken(v, jwtHeader); err != nil {
		ErrorLogger.Printf("could not parse auth-jwt header while creating ticket: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"errors": []string{"Internal server error"},
		})
	}
	uid := userClaims.Id

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
	tikId, err := crud.Create(tik.Title, tik.Price, uid)
	if err != nil {
		ErrorLogger.Printf("failed to write ticket to database, err: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"errors": []string{"unable to save ticket"},
		})
		return
	}

	// return object ID, title, price
	// TODO: define response struct somewhere for testing
	c.JSON(http.StatusCreated, TicketResp{
		Title: tik.Title,
		Price: tik.Price,
		Owner: uid,
		Id:    tikId,
	})
	InfoLogger.Printf("new ticket saved with id: %v", tikId)
}

func serveReadOne(c *gin.Context, crud MongoCollection) {
	id := c.Param("id")
	tik, err := crud.ReadOne(id)

	if err != nil {
		ErrorLogger.Printf("unable to fetch ticket from DB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"errors": []string{"Internal server error"},
		})
		return
	}

	if tik == nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, tik)
}
