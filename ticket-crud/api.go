package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/basilnsage/mwn-ticketapp/middleware"
	"github.com/basilnsage/mwn-ticketapp/ticket-crud/events"
	prometrics "github.com/basilnsage/prometheus-gin-metrics"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/stan.go"
	"google.golang.org/protobuf/proto"
)

type apiServer struct {
	db     CRUD
	eBus   stan.Conn
	router *gin.Engine
}

func newApiServer(pass string, r *gin.Engine, crud CRUD, stan stan.Conn) (*apiServer, error) {
	a := &apiServer{}

	a.router = r
	if err := a.bindRoutes(pass); err != nil {
		return nil, fmt.Errorf("unable to bind routes to router: %v", err)
	}

	a.db = crud
	a.eBus = stan

	return a, nil
}

func (a *apiServer) bindRoutes(jwtKey string) error {
	jwtValidator, err := middleware.NewJWTValidator([]byte(jwtKey), "HS256")
	if err != nil {
		return fmt.Errorf("NewJWTValidator: %v", err)
	}

	promRegistry := prometrics.NewRegistry()
	a.router.Use(promRegistry.ReportDuration(
		[]float64{0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0},
	))
	a.router.GET("/tickets/metrics", promRegistry.DefaultHandler)

	userValidationMiddleware := middleware.UserValidator(jwtValidator, "auth-jwt")
	ticketRoutes := a.router.Group("/api/tickets")
	ticketRoutes.POST(
		"/create",
		userValidationMiddleware,
		func(c *gin.Context) {
			a.serveCreate(c, jwtValidator)
		},
	)
	ticketRoutes.GET("", a.serveReadAll)
	ticketRoutes.GET("/:id", a.serveReadOne)
	ticketRoutes.PUT(
		"/:id",
		userValidationMiddleware,
		func(c *gin.Context) {
			a.serveUpdate(c, jwtValidator)
		},
	)

	return nil
}

func (a *apiServer) serveCreate(c *gin.Context, v *middleware.JWTValidator) {
	// parse gin context for JSON body
	var tik TicketReq
	if err := c.BindJSON(&tik); err != nil {
		WarningLogger.Printf("could not parse body of request, err: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResp{[]string{"unable to process request"}})
		return
	}

	// parse user id from auth-jwt header
	jwtHeader := c.GetHeader("auth-jwt")
	if jwtHeader == "" {
		ErrorLogger.Print("no auth-jwt header found while creating ticket. This should never happen")
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
		return
	}

	var userClaims middleware.UserClaims
	if err := userClaims.NewFromToken(v, jwtHeader); err != nil {
		ErrorLogger.Printf("could not parse auth-jwt header while creating ticket: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
	}
	uid := userClaims.Id

	// validate fields
	if ok, validationErrors := tik.isValid(); !ok {
		WarningLogger.Printf("ticket validation failed, err: %v", strings.Join(validationErrors, " | "))
		c.JSON(http.StatusBadRequest, ErrorResp{validationErrors})
		return
	}

	// insert new ticket object into DB
	tikId, err := a.db.Create(tik.Title, tik.Price, uid)
	if err != nil {
		ErrorLogger.Printf("failed to write ticket to database, err: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"unable to save ticket"}})
		return
	}

	// publish new ticket to event bus
	createEvent, err := proto.Marshal(&events.CreateUpdateTicket{
		Title: tik.Title,
		Price: tik.Price,
		Owner: uid,
		Id:    tikId,
	})
	if err != nil {
		ErrorLogger.Printf("failed to marshal create ticket event protobuf: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
		return
	}
	if err := a.eBus.Publish(events.Subject{}.CreateTicket(), createEvent); err != nil {
		ErrorLogger.Printf("unable to publish ticket to event bus: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
		return
	}

	// return object ID, title, price
	c.JSON(http.StatusCreated, TicketResp{
		Title: tik.Title,
		Price: tik.Price,
		Owner: uid,
		Id:    tikId,
	})
	InfoLogger.Printf("new ticket saved with id: %v", tikId)
}

func (a *apiServer) serveReadAll(c *gin.Context) {
	tickets, err := a.db.ReadAll()
	if err != nil {
		ErrorLogger.Printf("unable to fetch all tickets from DB: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tickets": tickets,
	})
}

func (a *apiServer) serveReadOne(c *gin.Context) {
	id := c.Param("id")
	tik, err := a.db.ReadOne(id)

	if err != nil {
		ErrorLogger.Printf("unable to fetch ticket from DB: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
		return
	}

	if tik == nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, tik)
}

func (a *apiServer) serveUpdate(c *gin.Context, v *middleware.JWTValidator) {
	id := c.Param("id")
	tik, err := a.db.ReadOne(id)
	if tik == nil {
		c.Status(http.StatusNotFound)
		return
	}
	if err != nil {
		ErrorLogger.Printf("could not read ticket from DB: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
		return
	}

	// read ticket from DB without error
	// make sure ticket owner matches originating user id
	userJWT := c.GetHeader("auth-jwt")
	if userJWT == "" {
		ErrorLogger.Print("no auth-jwt header found! this should never happen")
		c.JSON(http.StatusUnauthorized, ErrorResp{[]string{"Internal server error"}})
		return
	}

	reqUser := new(middleware.UserClaims)
	if err = reqUser.NewFromToken(v, userJWT); err != nil {
		ErrorLogger.Printf("unable to parse auth-jwt header! This should never happen")
		c.JSON(http.StatusUnauthorized, ErrorResp{[]string{"Unauthorized"}})
		return
	}

	if tik.Owner != reqUser.Id {
		c.JSON(http.StatusUnauthorized, ErrorResp{[]string{"Unauthorized"}})
		return
	}

	var tikReq TicketReq
	if err := c.BindJSON(&tikReq); err != nil {
		WarningLogger.Printf("could not parse body of request, err: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResp{[]string{"unable to process request"}})
		return
	}

	// validate fields
	if ok, validationErrors := tikReq.isValid(); !ok {
		WarningLogger.Printf("ticket validation failed, err: %v", strings.Join(validationErrors, " | "))
		c.JSON(http.StatusBadRequest, ErrorResp{validationErrors})
		return
	}

	ok, err := a.db.Update(id, tikReq.Title, tikReq.Price)
	if !ok {
		WarningLogger.Printf("no DB record modified")
		c.Status(http.StatusNotFound)
		return
	}
	if err != nil {
		ErrorLogger.Printf("unable to update ticket in DB: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
		return
	}

	c.JSON(http.StatusOK, TicketResp{
		Title: tikReq.Title,
		Price: tikReq.Price,
		Owner: tik.Owner,
		Id:    tik.Id,
	})

}

type TicketReq struct {
	Title string
	Price float64
}

// checks a TicketReq struct to ensure all fields are non-empty and within proper bounds
func (t TicketReq) isValid() (res bool, issues []string) {
	res = true
	if t.Title == "" {
		issues = append(issues, "please specify a title")
		res = false
	}
	if t.Price < 0.0 {
		issues = append(issues, "price cannot be less than 0")
		res = false
	}
	return res, issues
}

type TicketResp struct {
	Title string
	Price float64
	Owner string
	Id    string `bson:"_id"`
}

type ErrorResp struct {
	Errors []string `json:"errors"`
}
