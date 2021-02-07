package main

import (
	"fmt"
	"github.com/basilnsage/mwn-ticketapp/middleware"
	prometrics "github.com/basilnsage/prometheus-gin-metrics"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/stan.go"
	"net/http"
	"time"
)

type apiServer struct {
	orderDuration time.Duration
	tc            ticketsCRUD
	oc            ordersCRUD
	eBus          stan.Conn
	router        *gin.Engine
	v             *middleware.JWTValidator
}

func newApiServer(pass string, orderDuration time.Duration, r *gin.Engine, tc ticketsCRUD, oc ordersCRUD, stan stan.Conn) (*apiServer, error) {
	a := &apiServer{}

	jwtValidator, err := middleware.NewJWTValidator([]byte(pass), "HS256")
	if err != nil {
		return nil, fmt.Errorf("NewJWTValidator: %v", err)
	}
	a.v = jwtValidator

	a.orderDuration = orderDuration

	a.router = r
	if err := a.bindRoutes(); err != nil {
		return nil, fmt.Errorf("unable to bind routes to router: %v", err)
	}

	a.tc = tc
	a.oc = oc
	a.eBus = stan

	return a, nil
}

func (a *apiServer) bindRoutes() error {
	promRegistry := prometrics.NewRegistry()
	a.router.Use(promRegistry.ReportDuration(
		[]float64{0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0},
	))
	a.router.GET("/orders/metrics", promRegistry.DefaultHandler)

	userValidationMiddleware := middleware.UserValidator(a.v, "auth-jwt")
	ticketRoutes := a.router.Group("/api/orders")
	ticketRoutes.POST("/create", userValidationMiddleware, a.postOrder)
	ticketRoutes.GET("", userValidationMiddleware, a.getAllOrders)
	ticketRoutes.GET("/:id", userValidationMiddleware, a.getOrder)
	ticketRoutes.PUT("/:id", userValidationMiddleware, a.putOrder)

	return nil
}

func (a *apiServer) postOrder(c *gin.Context) {
	// get the ticket ID from the request
	req := OrderReq{}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResp{[]string{"Could not parse request"}})
		return
	}
	ticketId := req.TicketId

	// do we know about the ticket?
	ticket, err := a.tc.read(ticketId)
	if err != nil {
		ErrorLogger.Printf("failed to read ticket from DB: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal Server Error"}})
		return
	} else if ticket == nil {
		c.JSON(http.StatusNotFound, ErrorResp{[]string{"could not find ticket: " + ticketId}})
		return
	}

	// has the ticket been reserved?
	// find an order, whose status != reserved, that references the ticket
	//orders, err := a.oc.search(1, ticketId, Created, AwaitingPayment, Completed)
	orders, err := a.oc.searchTest(1, []string{ticketId}, []string{}, []orderStatus{Created, AwaitingPayment, Completed})
	if err != nil {
		ErrorLogger.Printf("reserved order search failed: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal Server Error"}})
		return
	}
	// if there exists a matching order then the ticket is already ordered --> we cannot order twice
	if len(orders) > 0 {
		c.JSON(http.StatusBadRequest, ErrorResp{[]string{"ticket already reserved"}})
		return
	}

	var userClaims middleware.UserClaims
	if err := userClaims.NewFromToken(a.v, c.GetHeader("auth-jwt")); err != nil {
		ErrorLogger.Printf("could not parse auth-jwt header: %v", err)
		c.Status(http.StatusForbidden)
		return
	}
	uid := userClaims.Id

	// create the order
	// by setting orderDuration == 0, we indicate that orders should expire immediately
	// so as a special case, when orderDuration == 0 set ExpiresAt to epoch
	var expiresAt time.Time
	if a.orderDuration == 0 {
		expiresAt = time.Unix(0, 0)
	} else {
		expiresAt = time.Now().Add(a.orderDuration)
	}
	order := Order{
		uid,
		Created,
		expiresAt,
		ticketId,
		"", // we can't know this until we save the order to the DB
	}

	// save the order
	orderId, err := a.oc.create(order)
	if err != nil {
		ErrorLogger.Printf("failed to save order: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal Server Error"}})
		return
	}
	order.Id = orderId
	InfoLogger.Printf("saved order with id: %v", orderId)

	// publish an event

	c.JSON(http.StatusCreated, OrderResp{
		order.Status,
		order.ExpiresAt,
		*ticket,
		order.Id,
	})
}

func (a *apiServer) getOrder(c *gin.Context) {

}

func (a *apiServer) getAllOrders(c *gin.Context) {
	// get user id from auth-jwt header
	var userClaims middleware.UserClaims
	if err := userClaims.NewFromToken(a.v, c.GetHeader("auth-jwt")); err != nil {
		ErrorLogger.Printf("could not parse auth-jwt header: %v", err)
		c.Status(http.StatusForbidden)
		return
	}
	uid := userClaims.Id

	// search for all orders belonging to this user
	//orders, err := a.oc.search(
	// for each order, fetch the ticket and combine the two into the response

	// send the response
}

func (a *apiServer) putOrder(c *gin.Context) {

}

//func (a *apiServer) serveCreate(c *gin.Context, v *middleware.JWTValidator) {
//	// parse gin context for JSON body
//	var tik TicketReq
//	if err := c.BindJSON(&tik); err != nil {
//		WarningLogger.Printf("could not parse body of request, err: %v", err)
//		c.JSON(http.StatusBadRequest, ErrorResp{[]string{"unable to process request"}})
//		return
//	}
//
//	// parse user id from auth-jwt header
//	jwtHeader := c.GetHeader("auth-jwt")
//	if jwtHeader == "" {
//		ErrorLogger.Print("no auth-jwt header found while creating ticket. This should never happen")
//		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
//		return
//	}
//
//	var userClaims middleware.UserClaims
//	if err := userClaims.NewFromToken(v, jwtHeader); err != nil {
//		ErrorLogger.Printf("could not parse auth-jwt header while creating ticket: %v", err)
//		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
//	}
//	uid := userClaims.Id
//
//	// validate fields
//	if ok, validationErrors := tik.isValid(); !ok {
//		WarningLogger.Printf("ticket validation failed, err: %v", strings.Join(validationErrors, " | "))
//		c.JSON(http.StatusBadRequest, ErrorResp{validationErrors})
//		return
//	}
//
//	// insert new ticket object into DB
//	tikId, err := a.db.Create(tik.Title, tik.Price, uid)
//	if err != nil {
//		ErrorLogger.Printf("failed to write ticket to database, err: %v", err)
//		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"unable to save ticket"}})
//		return
//	}
//
//	resp := TicketResp{
//		Title: tik.Title,
//		Price: tik.Price,
//		Owner: uid,
//		Id:    tikId,
//	}
//
//	// publish new ticket to event bus
//	if err := resp.publish(a.eBus, events.Subject{}.CreateTicket()); err != nil {
//		ErrorLogger.Printf("unable to publish create ticket event: %v", err)
//		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
//		return
//	}
//
//	// return object ID, title, price
//	c.JSON(http.StatusCreated, resp)
//	InfoLogger.Printf("new ticket saved with id: %v", tikId)
//}
//
//func (a *apiServer) serveReadAll(c *gin.Context) {
//	tickets, err := a.db.ReadAll()
//	if err != nil {
//		ErrorLogger.Printf("unable to fetch all tickets from DB: %v", err)
//		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
//		return
//	}
//
//	c.JSON(http.StatusOK, gin.H{
//		"tickets": tickets,
//	})
//}
//
//func (a *apiServer) serveReadOne(c *gin.Context) {
//	id := c.Param("id")
//	tik, err := a.db.ReadOne(id)
//
//	if err != nil {
//		ErrorLogger.Printf("unable to fetch ticket from DB: %v", err)
//		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
//		return
//	}
//
//	if tik == nil {
//		c.Status(http.StatusNotFound)
//		return
//	}
//
//	c.JSON(http.StatusOK, tik)
//}
//
//func (a *apiServer) serveUpdate(c *gin.Context, v *middleware.JWTValidator) {
//	id := c.Param("id")
//	tik, err := a.db.ReadOne(id)
//	if tik == nil {
//		c.Status(http.StatusNotFound)
//		return
//	}
//	if err != nil {
//		ErrorLogger.Printf("could not read ticket from DB: %v", err)
//		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
//		return
//	}
//
//	// read ticket from DB without error
//	// make sure ticket owner matches originating user id
//	userJWT := c.GetHeader("auth-jwt")
//	if userJWT == "" {
//		ErrorLogger.Print("no auth-jwt header found! this should never happen")
//		c.JSON(http.StatusUnauthorized, ErrorResp{[]string{"Internal server error"}})
//		return
//	}
//
//	reqUser := new(middleware.UserClaims)
//	if err = reqUser.NewFromToken(v, userJWT); err != nil {
//		ErrorLogger.Printf("unable to parse auth-jwt header! This should never happen")
//		c.JSON(http.StatusUnauthorized, ErrorResp{[]string{"Unauthorized"}})
//		return
//	}
//
//	if tik.Owner != reqUser.Id {
//		c.JSON(http.StatusUnauthorized, ErrorResp{[]string{"Unauthorized"}})
//		return
//	}
//
//	var tikReq TicketReq
//	if err := c.BindJSON(&tikReq); err != nil {
//		WarningLogger.Printf("could not parse body of request, err: %v", err)
//		c.JSON(http.StatusBadRequest, ErrorResp{[]string{"unable to process request"}})
//		return
//	}
//
//	// validate fields
//	if ok, validationErrors := tikReq.isValid(); !ok {
//		WarningLogger.Printf("ticket validation failed, err: %v", strings.Join(validationErrors, " | "))
//		c.JSON(http.StatusBadRequest, ErrorResp{validationErrors})
//		return
//	}
//
//	ok, err := a.db.Update(id, tikReq.Title, tikReq.Price)
//	if !ok {
//		WarningLogger.Printf("no DB record modified")
//		c.Status(http.StatusNotFound)
//		return
//	}
//	if err != nil {
//		ErrorLogger.Printf("unable to update ticket in DB: %v", err)
//		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
//		return
//	}
//
//	resp := TicketResp{
//		Title: tikReq.Title,
//		Price: tikReq.Price,
//		Owner: tik.Owner,
//		Id:    tik.Id,
//	}
//	if err := resp.publish(a.eBus, events.Subject{}.UpdateTicket()); err != nil {
//		ErrorLogger.Printf("unable to publish update ticket event: %v", err)
//		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal server error"}})
//		return
//	}
//
//	c.JSON(http.StatusOK, resp)
//}
//
//type TicketReq struct {
//	Title string
//	Price float64
//}
//
//// checks a TicketReq struct to ensure all fields are non-empty and within proper bounds
//func (t TicketReq) isValid() (res bool, issues []string) {
//	res = true
//	if t.Title == "" {
//		issues = append(issues, "please specify a title")
//		res = false
//	}
//	if t.Price < 0.0 {
//		issues = append(issues, "price cannot be less than 0")
//		res = false
//	}
//	return res, issues
//}
//
//type TicketResp struct {
//	Title string
//	Price float64
//	Owner string
//	Id    string `bson:"_id"`
//}
//
//func ticketRespFromProto(data []byte) (*TicketResp, error) {
//	var resp events.CreateUpdateTicket
//	if err := proto.Unmarshal(data, &resp); err != nil {
//		return nil, err
//	}
//	return &TicketResp{
//		resp.Title,
//		resp.Price,
//		resp.Owner,
//		resp.Id,
//	}, nil
//}
//
//func (t TicketResp) publish(stan stan.Conn, subj string) error {
//	createEvent, err := proto.Marshal(&events.CreateUpdateTicket{
//		Title: t.Title,
//		Price: t.Price,
//		Owner: t.Owner,
//		Id:    t.Id,
//	})
//	if err != nil {
//		return err
//	}
//	if err := stan.Publish(subj, createEvent); err != nil {
//		return err
//	}
//	return nil
//}

type ErrorResp struct {
	Errors []string `json:"errors"`
}
