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

	_ = fakeTC.createWrapper("my order does not exist", 1.0, 1)
	ticket2 := fakeTC.createWrapper("i am ordered by user1", 2.0, 2)
	ticket3 := fakeTC.createWrapper("i am ordered by user2", 3.0, 3)

	// skip order1; ticket1 should not have an order
	order2 := fakeOC.createWrapper("1", ticket2.Id, Created)
	order3 := fakeOC.createWrapper("2", ticket3.Id, Created)

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
			"/api/orders/" + order2.Id,
			nil,
			map[string]string{"auth-jwt": user2JWT},
			http.StatusUnauthorized,
			nil,
			&ErrorResp{[]string{"unauthorized"}},
		},
		{
			"get order that dne",
			http.MethodGet,
			"/api/orders/" + order3.Id,
			nil,
			map[string]string{"auth-jwt": user2JWT},
			http.StatusOK,
			OrderResp{
				order3.Status,
				order3.ExpiresAt,
				ticket3,
				order3.Id,
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
