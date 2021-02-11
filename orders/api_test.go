package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/basilnsage/mwn-ticketapp/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeNatsConn struct {
	messages map[string][][]byte
}

func newFakeNatsConn() *fakeNatsConn {
	return &fakeNatsConn{
		make(map[string][][]byte),
	}
}

func (f *fakeNatsConn) Publish(subj string, data []byte) error {
	f.messages[subj] = append(f.messages[subj], data)
	return nil
}

func (f *fakeNatsConn) PublishAsync(subj string, data []byte, ah stan.AckHandler) (string, error) {
	_, _, _ = subj, data, ah
	return "", errors.New("not implemented")
}

func (f *fakeNatsConn) Subscribe(subj string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (stan.Subscription, error) {
	_, _, _ = subj, cb, opts
	return nil, errors.New("not implemented")
}

func (f *fakeNatsConn) QueueSubscribe(subj, qgroup string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (stan.Subscription, error) {
	_, _, _, _ = subj, qgroup, cb, opts
	return nil, errors.New("not implemented")
}

func (f *fakeNatsConn) Close() error {
	return nil
}

func (f *fakeNatsConn) NatsConn() *nats.Conn {
	return nil
}

func newTestInfra(tc ticketsCRUD, oc ordersCRUD, fs stan.Conn) (*apiServer, error) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	server, err := newApiServer("password", 0, r, tc, oc, fs)
	if err != nil {
		return nil, err
	}

	return server, nil
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
					currTest.Fatalf(" status code is %v, want %v", got, want)
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
				if diff := cmp.Diff(respBody, *test.expectedErr); diff != "" {
					currTest.Fatalf("unexpected error: %v", diff)
				}
			}
		})
	}
	return nil
}

func TestPostOrder(t *testing.T) {
	fakeTC := newFakeTicketsCollection()
	fakeOC := newFakeOrdersCollection()
	fakeStan := newFakeNatsConn()
	server, err := newTestInfra(fakeTC, fakeOC, fakeStan)
	if err != nil {
		t.Fatalf("unable to complete pre-test tasks: %v", err)
	}

	testUserJWT, err := middleware.NewUserClaims("foo@bar.com", "1").Tokenize(server.v)
	if err != nil {
		t.Fatalf("unble to create test JWT: %v", err)
	}

	reservedTicketId, _ := fakeTC.create(Ticket{
		Title:   "i am reserved",
		Price:   1.0,
		Version: 0,
	})
	_, _ = fakeOC.create(Order{
		UserId:    "0",
		Status:    Created,
		ExpiresAt: allBalls,
		TicketId:  reservedTicketId,
	})

	availableTicket := Ticket{
		Title:   "reserve me",
		Price:   1.0,
		Version: 1,
	}
	availableTicketId, _ := fakeTC.create(availableTicket)
	availableTicket.Id = availableTicketId

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
			OrderReq{reservedTicketId},
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

func TestGetAllOrders(t *testing.T) {
	// tests:
	// have 2 users, userA and userB
	// userA has 1 order
	// userB has 2 orders
	// ensure userA only gets their 1 order (with ticket)
	// ensure userB only gets their 2 orders (with tickets)

	fakeTC := newFakeTicketsCollection()
	fakeOC := newFakeOrdersCollection()
	fakeStan := newFakeNatsConn()
	server, err := newTestInfra(fakeTC, fakeOC, fakeStan)
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
			"get all orders many order",
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

//func TestCreate(t *testing.T) {
//	fakeTC := newFakeTicketsCollection()
//	fakeStan := newFakeNatsConn()
//	server, err := newTestInfra(fakeTC, nil, fakeStan)
//	if err != nil {
//		t.Fatalf("unable to complete pre-test tasks: %v", err)
//	}
//
//
//	testUserJWT, _ := middleware.NewUserClaims("foo@bar.com", "1").Tokenize(server.v)
//	badUserJWT := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImZvb0BiYXIuY29tIiwiaWQiOjF9.f9FeG_FD2vOW6sGQwGxCoGYNIZv1P_Sb7WBgjq99KOs"
//
//	tests := []test{
//		{
//			"create test ticket",
//			http.MethodPost,
//			"/api/tickets/create",
//			TicketReq{"for testing", 0.0},
//			map[string]string{"auth-jwt": testUserJWT},
//			http.StatusCreated,
//			&TicketResp{"for testing", 0.0, "1", "0"},
//			nil,
//		},
//		{
//			"create ticket without jwt header",
//			http.MethodPost,
//			"/api/tickets/create",
//			TicketReq{"new test ticket", 10.0},
//			nil,
//			http.StatusUnauthorized,
//			nil,
//			&ErrorResp{[]string{"User is not signed in"}},
//		},
//		{
//			"create ticket with bad jwt header",
//			http.MethodPost,
//			"/api/tickets/create",
//			TicketReq{"new test ticket", 100.0},
//			map[string]string{"auth-jwt": badUserJWT},
//			http.StatusUnauthorized,
//			nil,
//			&ErrorResp{[]string{"Unauthorized"}},
//		},
//		{
//			"create ticket with bad payload",
//			http.MethodPost,
//			"/api/tickets/create",
//			TicketReq{"", -1000.0},
//			map[string]string{"auth-jwt": testUserJWT},
//			http.StatusBadRequest,
//			nil,
//			&ErrorResp{[]string{"please specify a title", "price cannot be less than 0"}},
//		},
//	}
//
//	if err := runTest(tests, server.router, t); err != nil {
//		t.Fatalf("error running tests: %v", err)
//	}
//
//	t.Run("create ticket event publish", func(currTest *testing.T) {
//		// check that a ticket was published to our fake NATS client
//		pbBytes := fakeStan.messages[events.Subject{}.CreateTicket()][0]
//		resp, err := ticketRespFromProto(pbBytes)
//		if err != nil {
//			currTest.Fatal(err)
//		}
//		if diff := cmp.Diff(*resp, TicketResp{"for testing", 0.0, "1", "0"}); diff != "" {
//			currTest.Fatalf("bad resp ticket: %v", diff)
//		}
//	})
//}

// test the route directly since it is a thin wrapper around serveReadOne
//func TestReadOne(t *testing.T) {
//	server, v, err := newTestInfra()
//	if err != nil {
//		t.Fatalf("unable to complete pre-test tasks: %v", err)
//	}
//
//	testUserJWT, _ := middleware.NewUserClaims("foo@bar.com", "1").Tokenize(v)
//
//	tests := []test{
//		{
//			"create test ticket",
//			http.MethodPost,
//			"/api/tickets/create",
//			TicketReq{"for testing", 0.0},
//			map[string]string{"auth-jwt": testUserJWT},
//			http.StatusCreated,
//			&TicketResp{"for testing", 0.0, "1", "0"},
//			nil,
//		},
//		{
//			"get test ticket",
//			http.MethodGet,
//			"/api/tickets/0",
//			nil,
//			map[string]string{"auth-jwt": testUserJWT},
//			http.StatusOK,
//			&TicketResp{"for testing", 0.0, "1", "0"},
//			nil,
//		},
//		{
//			"get nonexistent ticket",
//			http.MethodGet,
//			"/api/tickets/1",
//			nil,
//			map[string]string{"auth-jwt": testUserJWT},
//			http.StatusNotFound,
//			nil,
//			nil,
//		},
//	}
//
//	if err := runTest(tests, server.router, t); err != nil {
//		t.Fatalf("error running tests: %v", err)
//	}
//}
//
//func TestReadAll(t *testing.T) {
//	server, _, err := newTestInfra()
//	if err != nil {
//		t.Fatalf("unable to complete pre-test tasks: %v", err)
//	}
//
//	for i := 0; i < 3; i++ {
//		_, _ = server.db.Create("testing", 1.0, "0")
//	}
//
//	resp := httptest.NewRecorder()
//	req := httptest.NewRequest("GET", "/api/tickets", nil)
//	server.router.ServeHTTP(resp, req)
//
//	var tickets struct {
//		Tickets []TicketResp
//	}
//	respBody, _ := ioutil.ReadAll(resp.Body)
//	_ = json.Unmarshal(respBody, &tickets)
//
//	if got, want := resp.Code, http.StatusOK; got != want {
//		t.Errorf("ReadAll bad response code: %v, want %v", got, want)
//	}
//	if got, want := len(tickets.Tickets), 3; got != want {
//		t.Errorf("did not fetch correct number of tickets: %v, want %v", got, want)
//	}
//}
//
//func TestUpdate(t *testing.T) {
//	server, v, err := newTestInfra()
//	if err != nil {
//		t.Fatalf("unable to complete pre-test tasks: %v", err)
//	}
//
//	testUserJWT, _ := middleware.NewUserClaims("foo@bar.com", "1").Tokenize(v)
//	badUserJWT, _ := middleware.NewUserClaims("bar@foo.com", "2").Tokenize(v)
//
//	fakeStan := newFakeNatsConn()
//	server.eBus = fakeStan
//
//	tests := []test{
//		{
//			"create test ticket",
//			http.MethodPost,
//			"/api/tickets/create",
//			TicketReq{"for testing", 0.0},
//			map[string]string{"auth-jwt": testUserJWT},
//			http.StatusCreated,
//			&TicketResp{"for testing", 0.0, "1", "0"},
//			nil,
//		},
//		{
//			"unauth update",
//			http.MethodPut,
//			"/api/tickets/0",
//			TicketReq{"test update", 2.0},
//			map[string]string{"auth-jwt": badUserJWT},
//			http.StatusUnauthorized,
//			nil,
//			&ErrorResp{[]string{"Unauthorized"}},
//		},
//		{
//			"malformed update",
//			http.MethodPut,
//			"/api/tickets/0",
//			TicketReq{"", -1.0},
//			map[string]string{"auth-jwt": testUserJWT},
//			http.StatusBadRequest,
//			nil,
//			&ErrorResp{[]string{"please specify a title", "price cannot be less than 0"}},
//		},
//		{
//			"successful update",
//			http.MethodPut,
//			"/api/tickets/0",
//			TicketReq{"this should be new", 10.0},
//			map[string]string{"auth-jwt": testUserJWT},
//			http.StatusOK,
//			&TicketResp{"this should be new", 10.0, "1", "0"},
//			nil,
//		},
//		{
//			"verify update persisted",
//			http.MethodGet,
//			"/api/tickets/0",
//			nil,
//			map[string]string{"auth-jwt": testUserJWT},
//			http.StatusOK,
//			&TicketResp{"this should be new", 10.0, "1", "0"},
//			nil,
//		},
//	}
//
//	if err := runTest(tests, server.router, t); err != nil {
//		t.Fatalf("error running tests: %v", err)
//	}
//
//	t.Run("update ticket event publish", func(currTest *testing.T) {
//		// check that a ticket was published to our fake NATS client
//		pbBytes := fakeStan.messages[events.Subject{}.UpdateTicket()][0]
//		resp, err := ticketRespFromProto(pbBytes)
//		if err != nil {
//			currTest.Fatal(err)
//		}
//		if diff := cmp.Diff(*resp, TicketResp{"this should be new", 10.0, "1", "0"}); diff != "" {
//			currTest.Fatalf("bad resp ticket: %v", diff)
//		}
//	})
//}
