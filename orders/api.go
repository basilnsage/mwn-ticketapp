package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/basilnsage/mwn-ticketapp/middleware"
	prometrics "github.com/basilnsage/prometheus-gin-metrics"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/stan.go"
)

type apiServer struct {
	orderDuration time.Duration
	ticketsCRUD   ticketsCRUD
	oc            ordersCRUD
	natsConn      stan.Conn
	router        *gin.Engine
	validator     *middleware.JWTValidator
}

func newApiServer(pass string, orderDuration time.Duration, r *gin.Engine, ticketsCRUD ticketsCRUD, oc ordersCRUD,
	natsConn stan.Conn) (*apiServer, error) {

	a := &apiServer{}

	if err := setOrderSubjects(); err != nil {
		return nil, fmt.Errorf("unable to set NATS subjects: %v", err)
	}

	jwtValidator, err := middleware.NewJWTValidator([]byte(pass), "HS256")
	if err != nil {
		return nil, fmt.Errorf("NewJWTValidator: %v", err)
	}
	a.validator = jwtValidator

	a.orderDuration = orderDuration

	a.router = r
	a.bindRoutes()

	a.ticketsCRUD = ticketsCRUD
	a.oc = oc
	a.natsConn = natsConn

	return a, nil
}

func (a *apiServer) bindRoutes() {
	promRegistry := prometrics.NewRegistry()
	a.router.Use(promRegistry.ReportDuration(
		[]float64{0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0},
	))
	a.router.GET("/orders/metrics", promRegistry.DefaultHandler)

	userValidationMiddleware := middleware.UserValidator(a.validator, "auth-jwt")
	orderRoutes := a.router.Group("/api/orders")
	orderRoutes.POST("/create", userValidationMiddleware, a.postOrder)
	orderRoutes.GET("", userValidationMiddleware, a.getAllOrders)
	orderRoutes.GET("/:id", userValidationMiddleware, a.getOrder)
	orderRoutes.PATCH("/:id", userValidationMiddleware, a.cancelOrder)
}

func (a *apiServer) postOrder(c *gin.Context) {
	// extract user ID from JWT
	var userClaims middleware.UserClaims
	if err := userClaims.NewFromToken(a.validator, c.GetHeader("auth-jwt")); err != nil {
		ErrorLogger.Printf("could not parse auth-jwt header: %v", err)
		c.Status(http.StatusForbidden)
		return
	}
	uid := userClaims.Id

	// get the ticket ID from the request
	req := OrderReq{}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResp{[]string{"Could not parse request"}})
		return
	}
	ticketId := req.TicketId

	// do we know about the ticket?
	ticket, err := a.ticketsCRUD.read(ticketId)
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
	orders, err := a.oc.search(1, []string{ticket.Id}, []string{}, []orderStatus{Created, AwaitingPayment, Completed})
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
		ticket.Id,
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

	// marshal the order created event
	createdEventBytes, err := marshalOrderCreated(*ticket, order)
	if err != nil {
		ErrorLogger.Printf("could not create order created event: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal Server Error"}})
		return
	}

	// publish event
	if err := a.natsConn.Publish(orderCreatedSubject, createdEventBytes); err != nil {
		ErrorLogger.Printf("could not publish created order event: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal Server Error"}})
		return
	}

	c.JSON(http.StatusCreated, OrderResp{
		order.Status,
		order.ExpiresAt,
		*ticket,
		order.Id,
	})
}

func (a *apiServer) getOrder(c *gin.Context) {
	// get user id from auth-jwt header
	var userClaims middleware.UserClaims
	if err := userClaims.NewFromToken(a.validator, c.GetHeader("auth-jwt")); err != nil {
		ErrorLogger.Printf("could not parse auth-jwt header: %v", err)
		c.Status(http.StatusForbidden)
		return
	}
	uid := userClaims.Id

	// fetch order with ID from URI param
	oid := c.Param("id")
	// if no oid (not sure how this would happen...)
	if oid == "" {
		c.JSON(http.StatusBadRequest, ErrorResp{[]string{"no order id found"}})
		return
	}
	order, err := a.oc.read(oid)
	if err != nil {
		ErrorLogger.Printf("unable to fetch single order: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal Server Error"}})
		return
	}
	if order == nil {
		c.JSON(http.StatusNotFound, ErrorResp{[]string{"no order found"}})
		return
	}
	if order.UserId != uid {
		c.JSON(http.StatusUnauthorized, ErrorResp{[]string{"unauthorized"}})
		return
	}

	// fetch corresponding ticket
	ticket, err := a.ticketsCRUD.read(order.TicketId)
	if err != nil {
		ErrorLogger.Printf("failed to read ticket from DB: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal Server Error"}})
		return
	}

	// return OrderResp if order exists
	c.JSON(http.StatusOK, OrderResp{
		order.Status,
		order.ExpiresAt,
		*ticket,
		order.Id,
	})
}

func (a *apiServer) getAllOrders(c *gin.Context) {
	// get user id from auth-jwt header
	var userClaims middleware.UserClaims
	if err := userClaims.NewFromToken(a.validator, c.GetHeader("auth-jwt")); err != nil {
		ErrorLogger.Printf("could not parse auth-jwt header: %v", err)
		c.Status(http.StatusForbidden)
		return
	}
	uid := userClaims.Id

	// search for all orders belonging to this user
	orders, err := a.oc.search(50, nil, []string{uid}, nil)
	if err != nil {
		ErrorLogger.Printf("error search for users orders: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal Server Error"}})
		return
	}

	// for each order, fetch the ticket and combine the two into the response
	resp := make([]OrderResp, 0)
	for _, order := range orders {
		tid := order.TicketId
		ticket, err := a.ticketsCRUD.read(tid)
		if err != nil {
			ErrorLogger.Printf("error reading ticket, id: %v, error: %v", tid, err)
			continue
		}
		resp = append(resp, OrderResp{
			order.Status,
			order.ExpiresAt,
			*ticket,
			order.Id,
		})
	}

	// send the response
	c.JSON(http.StatusOK, resp)
}

func (a *apiServer) cancelOrder(c *gin.Context) {
	// get user id from jwt header
	var userClaims middleware.UserClaims
	if err := userClaims.NewFromToken(a.validator, c.GetHeader("auth-jwt")); err != nil {
		ErrorLogger.Printf("could not parse auth-jwt header: %v", err)
		c.Status(http.StatusForbidden)
		return
	}
	uid := userClaims.Id

	// check if the order exists
	oid := c.Param("id")
	if oid == "" {
		ErrorLogger.Printf("no order id found, this should not happen, id: %v", oid)
		c.JSON(http.StatusBadRequest, ErrorResp{[]string{"please specify an order id"}})
		return
	}
	order, err := a.oc.read(oid)
	if err != nil {
		ErrorLogger.Printf("unable to fetch single order: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal Server Error"}})
		return
	}
	if order == nil {
		c.JSON(http.StatusNotFound, ErrorResp{[]string{"order not found"}})
		return
	}

	// check if the user owns the order
	if order.UserId != uid {
		c.JSON(http.StatusUnauthorized, ErrorResp{[]string{"unauthorized"}})
		return
	}

	// update status
	order.Status = Cancelled
	ok, err := a.oc.update(oid, *order)
	if err != nil {
		ErrorLogger.Printf("could not update order: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal Server Error"}})
		return
	}
	if !ok {
		ErrorLogger.Printf("order not found during update, this should not happen, id: %v", oid)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal Server Error"}})
		return
	}

	// publish event
	// could be dangerous if marshalOrderCancelled changes underneath without updating this function
	// it would then marshal zero-values that are probably do not match actual ticket values
	// ideally, this situation would be caught by unit tests
	eventBytes, err := marshalOrderCancelled(Ticket{Id: order.TicketId}, *order)
	if err != nil {
		ErrorLogger.Printf("unable to marshal orderCancelled event: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal Server Error"}})
		return
	}

	if err := a.natsConn.Publish(orderCancelledSubject, eventBytes); err != nil {
		ErrorLogger.Printf("unable to publish orderCancelled event: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResp{[]string{"Internal Server Error"}})
		return
	}

	// send response
	c.Status(http.StatusNoContent)
}

type ErrorResp struct {
	Errors []string `json:"errors"`
}
