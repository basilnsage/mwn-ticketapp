package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/basilnsage/mwn-ticketapp/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
)

type fakeMongoCollection struct {
	tickets map[string]*TicketResp
	id      int
}

func newFakeMongoCollection() *fakeMongoCollection {
	return &fakeMongoCollection{
		make(map[string]*TicketResp),
		0,
	}
}

func (f *fakeMongoCollection) Create(title string, price float64, owner string) (string, error) {
	if title == "should error" {
		return "", errors.New("unable to create ticket")
	}
	currId := strconv.Itoa(f.id)
	f.id++
	f.tickets[currId] = &TicketResp{title, price, owner, currId}
	return currId, nil
}

func (f *fakeMongoCollection) ReadOne(id string) (*TicketResp, error) {
	tik, ok := f.tickets[id]
	if !ok {
		return nil, nil
	}
	return tik, nil
}

func (f *fakeMongoCollection) ReadAll() ([]TicketResp, error) {
	resp := make([]TicketResp, 0)
	for _, v := range f.tickets {
		resp = append(resp, *v)
	}
	return resp, nil
}

func (f *fakeMongoCollection) Update(id string, title string, price float64) (bool, bool, error) {
	item, ok := f.tickets[id]
	if !ok {
		return false, false, errors.New("no ticket with matching ID found")
	}
	isMod := !(item.Title == title && item.Price == price)
	item.Title = title
	item.Price = price
	f.tickets[id] = item
	return true, isMod, nil
}

func (f *fakeMongoCollection) Close(ctx context.Context) error {
	_ = ctx
	return nil
}

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

func newTestInfra() (*apiServer, *middleware.JWTValidator, error) {
	fakeMongo := newFakeMongoCollection()
	fakeStan := newFakeNatsConn()
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	server, err := newApiServer("password", r, fakeMongo, fakeStan)
	if err != nil {
		return nil, nil, err
	}
	v, err := middleware.NewJWTValidator([]byte("password"), "HS256")
	if err != nil {
		return nil, nil, err
	}

	return server, v, nil
}

type test struct {
	name         string
	method       string
	route        string
	body         interface{}
	headers      map[string]string
	expectedCode int
	expectedResp *TicketResp
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
					currTest.Fatalf("bad status code: got %v, want %v", got, want)
				}
			}

			// check resp body if specified by test
			if test.expectedResp != nil {
				var respBody TicketResp
				respBytes, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					currTest.Fatalf("ioutil.Readall: %v", err)
				}
				if err := json.Unmarshal(respBytes, &respBody); err != nil {
					currTest.Fatalf("json.Unmarshal: %v", err)
				}
				if diff := cmp.Diff(respBody, *test.expectedResp); diff != "" {
					currTest.Fatalf("bad response: %v", diff)
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
					currTest.Fatalf("bad error: %v", diff)
				}
			}
		})
	}
	return nil
}

func TestCreate(t *testing.T) {
	server, v, err := newTestInfra()
	if err != nil {
		t.Fatalf("unable to complete pre-test tasks: %v", err)
	}

	fakeStan := newFakeNatsConn()
	server.eBus = fakeStan

	testUserJWT, _ := middleware.NewUserClaims("foo@bar.com", "1").Tokenize(v)
	badUserJWT := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImZvb0BiYXIuY29tIiwiaWQiOjF9.f9FeG_FD2vOW6sGQwGxCoGYNIZv1P_Sb7WBgjq99KOs"

	tests := []test{
		{
			"create test ticket",
			http.MethodPost,
			"/api/tickets/create",
			TicketReq{"for testing", 0.0},
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusCreated,
			&TicketResp{"for testing", 0.0, "1", "0"},
			nil,
		},
		{
			"create ticket without jwt header",
			http.MethodPost,
			"/api/tickets/create",
			TicketReq{"new test ticket", 10.0},
			nil,
			http.StatusUnauthorized,
			nil,
			&ErrorResp{[]string{"User is not signed in"}},
		},
		{
			"create ticket with bad jwt header",
			http.MethodPost,
			"/api/tickets/create",
			TicketReq{"new test ticket", 100.0},
			map[string]string{"auth-jwt": badUserJWT},
			http.StatusUnauthorized,
			nil,
			&ErrorResp{[]string{"Unauthorized"}},
		},
		{
			"create ticket with bad payload",
			http.MethodPost,
			"/api/tickets/create",
			TicketReq{"", -1000.0},
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusBadRequest,
			nil,
			&ErrorResp{[]string{"please specify a title", "price cannot be less than 0"}},
		},
	}

	if err := runTest(tests, server.router, t); err != nil {
		t.Fatalf("error running tests: %v", err)
	}

	t.Run("create ticket event publish", func(currTest *testing.T) {
		// check that a ticket was published to our fake NATS client
		pbBytes := fakeStan.messages[createTicketSubject][0]
		resp, err := ticketRespFromProto(pbBytes)
		if err != nil {
			currTest.Fatal(err)
		}
		if diff := cmp.Diff(*resp, TicketResp{"for testing", 0.0, "1", "0"}); diff != "" {
			currTest.Fatalf("bad resp ticket: %v", diff)
		}
	})
}

// test the route directly since it is a thin wrapper around serveReadOne
func TestReadOne(t *testing.T) {
	server, v, err := newTestInfra()
	if err != nil {
		t.Fatalf("unable to complete pre-test tasks: %v", err)
	}

	testUserJWT, _ := middleware.NewUserClaims("foo@bar.com", "1").Tokenize(v)

	tests := []test{
		{
			"create test ticket",
			http.MethodPost,
			"/api/tickets/create",
			TicketReq{"for testing", 0.0},
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusCreated,
			&TicketResp{"for testing", 0.0, "1", "0"},
			nil,
		},
		{
			"get test ticket",
			http.MethodGet,
			"/api/tickets/0",
			nil,
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusOK,
			&TicketResp{"for testing", 0.0, "1", "0"},
			nil,
		},
		{
			"get nonexistent ticket",
			http.MethodGet,
			"/api/tickets/1",
			nil,
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusNotFound,
			nil,
			nil,
		},
	}

	if err := runTest(tests, server.router, t); err != nil {
		t.Fatalf("error running tests: %v", err)
	}
}

func TestReadAll(t *testing.T) {
	server, _, err := newTestInfra()
	if err != nil {
		t.Fatalf("unable to complete pre-test tasks: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, _ = server.db.Create("testing", 1.0, "0")
	}

	resp := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/tickets", nil)
	server.router.ServeHTTP(resp, req)

	var tickets struct {
		Tickets []TicketResp
	}
	respBody, _ := ioutil.ReadAll(resp.Body)
	_ = json.Unmarshal(respBody, &tickets)

	if got, want := resp.Code, http.StatusOK; got != want {
		t.Errorf("ReadAll bad response code: %v, want %v", got, want)
	}
	if got, want := len(tickets.Tickets), 3; got != want {
		t.Errorf("did not fetch correct number of tickets: %v, want %v", got, want)
	}
}

func TestUpdate(t *testing.T) {
	server, v, err := newTestInfra()
	if err != nil {
		t.Fatalf("unable to complete pre-test tasks: %v", err)
	}

	testUserJWT, _ := middleware.NewUserClaims("foo@bar.com", "1").Tokenize(v)
	badUserJWT, _ := middleware.NewUserClaims("bar@foo.com", "2").Tokenize(v)

	fakeStan := newFakeNatsConn()
	server.eBus = fakeStan

	tests := []test{
		{
			"create test ticket",
			http.MethodPost,
			"/api/tickets/create",
			TicketReq{"for testing", 0.0},
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusCreated,
			&TicketResp{"for testing", 0.0, "1", "0"},
			nil,
		},
		{
			"unauth update",
			http.MethodPatch,
			"/api/tickets/0",
			TicketReq{"test update", 2.0},
			map[string]string{"auth-jwt": badUserJWT},
			http.StatusUnauthorized,
			nil,
			&ErrorResp{[]string{"Unauthorized"}},
		},
		{
			"malformed update",
			http.MethodPatch,
			"/api/tickets/0",
			TicketReq{"", -1.0},
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusBadRequest,
			nil,
			&ErrorResp{[]string{"please specify a title", "price cannot be less than 0"}},
		},
		{
			"successful update",
			http.MethodPatch,
			"/api/tickets/0",
			TicketReq{"this should be new", 10.0},
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusOK,
			&TicketResp{"this should be new", 10.0, "1", "0"},
			nil,
		},
		{
			"verify update persisted",
			http.MethodGet,
			"/api/tickets/0",
			nil,
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusOK,
			&TicketResp{"this should be new", 10.0, "1", "0"},
			nil,
		},
	}

	if err := runTest(tests, server.router, t); err != nil {
		t.Fatalf("error running tests: %v", err)
	}

	t.Run("update ticket event publish", func(currTest *testing.T) {
		// check that a ticket was published to our fake NATS client
		pbBytes := fakeStan.messages[updateTicketSubject][0]
		resp, err := ticketRespFromProto(pbBytes)
		if err != nil {
			currTest.Fatal(err)
		}
		if diff := cmp.Diff(*resp, TicketResp{"this should be new", 10.0, "1", "0"}); diff != "" {
			currTest.Fatalf("bad resp ticket: %v", diff)
		}
	})
}
