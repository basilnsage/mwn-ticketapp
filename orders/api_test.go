package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/basilnsage/mwn-ticketapp-common/events"
	"github.com/basilnsage/mwn-ticketapp-common/subjects"
	"github.com/basilnsage/mwn-ticketapp/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func newTestInfra() (*apiServer, *fakeTicketsCollection, *fakeOrdersCollection, *fakeNatsConn, error) {
	fakeTC := newFakeTicketsCollection()
	fakeOC := newFakeOrdersCollection()
	fakeStan := newFakeNatsConn()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	server, err := newApiServer("password", 0, r, fakeTC, fakeOC, fakeStan)
	if err != nil {
		return nil, fakeTC, fakeOC, fakeStan, nil
	}

	return server, fakeTC, fakeOC, fakeStan, nil
}

func TestNewApiServer(t *testing.T) {
	t.Run("test newApiServer", func(tester *testing.T) {
		fakeTC := newFakeTicketsCollection()
		fakeOC := newFakeOrdersCollection()
		fakeStan := newFakeNatsConn()
		gin.SetMode(gin.TestMode)
		r := gin.New()

		server, err := newApiServer("password", 3*time.Second, r, fakeTC, fakeOC, fakeStan)
		if err != nil {
			tester.Fatalf("newApiServer: %v", err)
		}
		if server == nil {
			tester.Fatal("api server is nil")
		}

		if orderCreatedSubject == "" {
			tester.Fatal("orderCreatedSubject is empty")
		}
		if orderCancelledSubject == "" {
			tester.Fatal("orderCancelledSubject is empty")
		}
	})
}

type test struct {
	name         string
	method       string
	route        string
	body         interface{}
	headers      map[string]string
	expectedCode int
	expectedResp interface{}
	expectedErr  *ErrorResp
}

func runTest(tests []test, router *gin.Engine, t *testing.T) (err error) {
	for _, test := range tests {
		// if body is not nil convert it into bytes
		var body []byte
		if test.body != nil {
			if body, err = json.Marshal(test.body); err != nil {
				return err
			}
		}

		resp := httptest.NewRecorder()
		req := httptest.NewRequest(test.method, test.route, bytes.NewReader(body))
		for k, v := range test.headers {
			req.Header.Set(k, v)
		}
		router.ServeHTTP(resp, req)

		t.Run(test.name, func(currTest *testing.T) {
			// check resp code if expected value specified by test
			if test.expectedCode != -1 {
				if got, want := resp.Code, test.expectedCode; got != want {
					currTest.Fatalf("status code is %v, want %v", got, want)
				}
			}

			// check resp body if specified by test
			if test.expectedResp != nil {
				respBytes, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					currTest.Fatalf("ioutil.Readall: %v", err)
				}

				var diff string
				switch test.expectedResp.(type) {
				case OrderResp:
					var respBody OrderResp
					if err := json.Unmarshal(respBytes, &respBody); err != nil {
						currTest.Fatalf("json.Unmarshal: %v", err)
					}
					diff = cmp.Diff(test.expectedResp, respBody)
				case []OrderResp:
					var respBody []OrderResp
					if err := json.Unmarshal(respBytes, &respBody); err != nil {
						currTest.Fatalf("json.Unmarshal: %v", err)
					}
					diff = cmp.Diff(test.expectedResp, respBody)
				}
				if diff != "" {
					currTest.Fatalf("unexpected response: (-want, +got)\n%v", diff)
				}
			}

			// check resp body if an err is expected
			if test.expectedErr != nil {
				var respBody ErrorResp
				respBytes, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					currTest.Fatalf("ioutil.Readall: %v", err)
				}
				if err := json.Unmarshal(respBytes, &respBody); err != nil {
					currTest.Fatalf("json.Unmarshal: %v", err)
				}
				if diff := cmp.Diff(*test.expectedErr, respBody); diff != "" {
					currTest.Fatalf("unexpected error: (-want, +got)\n%v", diff)
				}
			}
		})
	}
	return nil
}

func TestPostOrder(t *testing.T) {
	server, fakeTC, fakeOC, _, err := newTestInfra()
	if err != nil {
		t.Fatalf("unable to complete pre-test tasks: %v", err)
	}

	testUserJWT, err := middleware.NewUserClaims("foo@bar.com", "1").Tokenize(server.v)
	if err != nil {
		t.Fatalf("unble to create test JWT: %v", err)
	}

	reservedTicket := fakeTC.createWrapper("i am reserved", 1.0, 0)
	_ = fakeOC.createWrapper("0", reservedTicket.Id, Created)
	availableTicket := fakeTC.createWrapper("reserve me", 1.0, 1)

	tests := []test{
		{
			"malformed request",
			http.MethodPost,
			"/api/orders/create",
			nil,
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusBadRequest,
			nil,
			&ErrorResp{[]string{"Could not parse request"}},
		},
		{
			"order a non existent ticket",
			http.MethodPost,
			"/api/orders/create",
			OrderReq{"-1"},
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusNotFound,
			nil,
			nil,
		},
		{
			"order a reserved ticket",
			http.MethodPost,
			"/api/orders/create",
			OrderReq{reservedTicket.Id},
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusBadRequest,
			nil,
			&ErrorResp{[]string{"ticket already reserved"}},
		},
		{
			"order an available ticket",
			http.MethodPost,
			"/api/orders/create",
			OrderReq{"1"},
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusCreated,
			OrderResp{
				Created,
				allBalls,
				availableTicket,
				"1",
			},
			nil,
		},
	}

	if err := runTest(tests, server.router, t); err != nil {
		t.Fatalf("error running tests: %v", err)
	}
}

func TestPublishOrderCreated(t *testing.T) {
	server, fakeTC, fakeOC, fakeStan, err := newTestInfra()
	if err != nil {
		t.Fatalf("unable to complete pre-test tasks: %v", err)
	}

	testUserJWT, err := middleware.NewUserClaims("foo@bar.com", "1").Tokenize(server.v)
	if err != nil {
		t.Fatalf("unble to create test JWT: %v", err)
	}

	ticket := fakeTC.createWrapper("proto me", 1.0, 1)
	tests := []test{
		{
			"order the proto me ticket",
			http.MethodPost,
			"/api/orders/create",
			OrderReq{ticket.Id},
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusCreated,
			OrderResp{
				Created,
				allBalls,
				ticket,
				"0",
			},
			nil,
		},
	}

	if err := runTest(tests, server.router, t); err != nil {
		t.Fatalf("error running tests: %v", err)
	}

	order, _ := fakeOC.read("0")
	pbExpiresAt, _ := ptypes.TimestampProto(order.ExpiresAt)

	// an event should have been published
	// check our faked NATS conn (fakeStan) to verify this
	eventBytes := fakeStan.messages[orderCreatedSubject][0]
	var got events.OrderCreated
	if err := proto.Unmarshal(eventBytes, &got); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}

	want := &events.OrderCreated{
		Subject: subjects.Subject_ORDER_CREATED,
		Data: &events.CreatedData{
			Id:        order.Id,
			Status:    events.Status_Created,
			UserId:    order.UserId,
			ExpiresAt: pbExpiresAt,
			Ticket: &events.CreatedData_Ticket{
				Id:    ticket.Id,
				Price: ticket.Price,
			},
		},
	}
	if diff := cmp.Diff(want, &got, protocmp.Transform()); diff != "" {
		t.Fatalf("orderCreated event: (-want +got)\n%v", diff)
	}

}

func TestGetAnOrder(t *testing.T) {
	server, fakeTC, fakeOC, _, err := newTestInfra()
	if err != nil {
		t.Fatalf("unable to complete pre-test tasks: %v", err)
	}

	user1JWT, err := middleware.NewUserClaims("user1@example.com", "1").Tokenize(server.v)
	if err != nil {
		t.Fatalf("unble to create test JWT: %v", err)
	}
	user2JWT, err := middleware.NewUserClaims("user2@example.com", "2").Tokenize(server.v)
	if err != nil {
		t.Fatalf("unble to create test JWT: %v", err)
	}

	ticket1 := fakeTC.createWrapper("i am ordered by user1", 2.0, 2)
	ticket2 := fakeTC.createWrapper("i am ordered by user2", 3.0, 3)

	order1 := fakeOC.createWrapper("1", ticket1.Id, Created)
	order2 := fakeOC.createWrapper("2", ticket2.Id, Created)

	tests := []test{
		{
			"get order that dne",
			http.MethodGet,
			"/api/orders/-1",
			nil,
			map[string]string{"auth-jwt": user1JWT},
			http.StatusNotFound,
			nil,
			&ErrorResp{[]string{"no order found"}},
		},
		{
			"get someone elses order",
			http.MethodGet,
			"/api/orders/" + order1.Id,
			nil,
			map[string]string{"auth-jwt": user2JWT},
			http.StatusUnauthorized,
			nil,
			&ErrorResp{[]string{"unauthorized"}},
		},
		{
			"get order",
			http.MethodGet,
			"/api/orders/" + order2.Id,
			nil,
			map[string]string{"auth-jwt": user2JWT},
			http.StatusOK,
			OrderResp{
				order2.Status,
				order2.ExpiresAt,
				ticket2,
				order2.Id,
			},
			nil,
		},
	}

	if err := runTest(tests, server.router, t); err != nil {
		t.Fatalf("error running tests: %v", err)
	}
}

func TestGetAllOrders(t *testing.T) {
	server, fakeTC, fakeOC, _, err := newTestInfra()
	if err != nil {
		t.Fatalf("unable to complete pre-test tasks: %v", err)
	}

	user1JWT, err := middleware.NewUserClaims("user1@example.com", "1").Tokenize(server.v)
	if err != nil {
		t.Fatalf("unble to create test JWT: %v", err)
	}
	user2JWT, err := middleware.NewUserClaims("user2@example.com", "2").Tokenize(server.v)
	if err != nil {
		t.Fatalf("unble to create test JWT: %v", err)
	}
	user3JWT, err := middleware.NewUserClaims("user3@example.com", "3").Tokenize(server.v)
	if err != nil {
		t.Fatalf("unble to create test JWT: %v", err)
	}

	user1Ticket1 := fakeTC.createWrapper("user1 ticket1", 1.0, 1)
	user2Ticket1 := fakeTC.createWrapper("user2 ticket1", 2.0, 2)
	user2Ticket2 := fakeTC.createWrapper("user2 ticket2", 3.0, 3)

	user1Order1 := fakeOC.createWrapper("1", "0", Created)
	user2Order1 := fakeOC.createWrapper("2", "1", Created)
	user2Order2 := fakeOC.createWrapper("2", "2", Created)

	tests := []test{
		{
			"get all orders single order",
			http.MethodGet,
			"/api/orders",
			nil,
			map[string]string{"auth-jwt": user1JWT},
			http.StatusOK,
			[]OrderResp{
				{
					Created,
					allBalls,
					user1Ticket1,
					user1Order1.Id,
				},
			},
			nil,
		},
		{
			"get all orders many orders",
			http.MethodGet,
			"/api/orders",
			nil,
			map[string]string{"auth-jwt": user2JWT},
			http.StatusOK,
			[]OrderResp{
				{
					Created,
					allBalls,
					user2Ticket1,
					user2Order1.Id,
				},
				{
					Created,
					allBalls,
					user2Ticket2,
					user2Order2.Id,
				},
			},
			nil,
		},
		{
			"get all orders no order",
			http.MethodGet,
			"/api/orders",
			nil,
			map[string]string{"auth-jwt": user3JWT},
			http.StatusOK,
			[]OrderResp{},
			nil,
		},
	}

	if err := runTest(tests, server.router, t); err != nil {
		t.Fatalf("error running tests: %v", err)
	}
}

func TestCancelOrder(t *testing.T) {
	server, fakeTC, fakeOC, _, err := newTestInfra()
	if err != nil {
		t.Fatalf("unable to init new server: %v", err)
	}

	testUserJWT, err := middleware.NewUserClaims("foo@bar.com", "1").Tokenize(server.v)
	if err != nil {
		t.Fatalf("unble to create test JWT: %v", err)
	}

	user0Order := fakeOC.createWrapper("0", "0", Created)
	user1Ticket := fakeTC.createWrapper("cancel me", 1.0, 1)
	user1Order := fakeOC.createWrapper("1", user1Ticket.Id, Created)

	// test various failure conditions as well as successful patch
	patchTests := []test{
		{
			"cancel a DNE order",
			http.MethodPatch,
			"/api/orders/-1",
			nil,
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusNotFound,
			nil,
			&ErrorResp{[]string{"order not found"}},
		},
		{
			"cancel a different users order",
			http.MethodPatch,
			"/api/orders/" + user0Order.Id,
			nil,
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusUnauthorized,
			nil,
			&ErrorResp{[]string{"unauthorized"}},
		},
		{
			"cancel an order",
			http.MethodPatch,
			"/api/orders/" + user1Order.Id,
			nil,
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusNoContent,
			nil,
			nil,
		},
	}

	if err := runTest(patchTests, server.router, t); err != nil {
		t.Fatalf("error running tests: %v", err)
	}

	// test the test: make sure the successful patch was saved
	sanityTests := []test{
		{
			"ensure cancelled order is cancelled",
			http.MethodGet,
			"/api/orders/" + user1Order.Id,
			nil,
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusOK,
			OrderResp{
				Cancelled,
				allBalls,
				user1Ticket,
				user1Order.Id,
			},
			nil,
		},
	}

	if err := runTest(sanityTests, server.router, t); err != nil {
		t.Fatalf("error running tests: %v", err)
	}
}

func TestPublishOrderCancelled(t *testing.T) {
	server, fakeTC, fakeOC, fakeStan, err := newTestInfra()
	if err != nil {
		t.Fatalf("unable to complete pre-test tasks: %v", err)
	}

	testUserJWT, err := middleware.NewUserClaims("foo@bar.com", "1").Tokenize(server.v)
	if err != nil {
		t.Fatalf("unble to create test JWT: %v", err)
	}

	ticket := fakeTC.createWrapper("proto me", 1.0, 1)
	order := fakeOC.createWrapper("1", ticket.Id, Created)

	tests := []test{
		{
			"cancel the ticket",
			http.MethodPatch,
			"/api/orders/" + order.Id,
			nil,
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusNoContent,
			nil,
			nil,
		},
	}

	if err := runTest(tests, server.router, t); err != nil {
		t.Fatalf("error running tests: %v", err)
	}

	// an event should have been published
	// check our faked NATS conn (fakeStan) to verify this
	eventBytes := fakeStan.messages[orderCancelledSubject][0]
	var got events.OrderCancelled
	if err := proto.Unmarshal(eventBytes, &got); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}

	want := &events.OrderCancelled{
		Subject: subjects.Subject_ORDER_CANCELLED,
		Data: &events.CancelledData{
			Id: order.Id,
			Ticket: &events.CancelledData_Ticket{
				Id: ticket.Id,
			},
		},
	}
	if diff := cmp.Diff(want, &got, protocmp.Transform()); diff != "" {
		t.Fatalf("orderCancelled event: (-want +got)\n%v", diff)
	}

}
