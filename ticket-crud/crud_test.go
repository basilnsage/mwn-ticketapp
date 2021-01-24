package main

import (
	"bytes"
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

func (f *fakeMongoCollection) Update(id string, title string, price float64) (bool, error) {
	item, ok := f.tickets[id]
	if !ok {
		return false, errors.New("no ticket with matching ID found")
	}
	item.Title = title
	item.Price = price
	f.tickets[id] = item
	return true, nil
}

func newTestInfra() (*gin.Engine, *middleware.JWTValidator, error) {
	fakeMongo := newFakeMongoCollection()
	gin.SetMode(gin.TestMode)
	router, err := newRouter("password", fakeMongo)
	if err != nil {
		return nil, nil, err
	}
	v, err := middleware.NewJWTValidator([]byte("password"), "HS256")
	if err != nil {
		return nil, nil, err
	}

	return router, v, nil
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
	router, v, err := newTestInfra()
	if err != nil {
		t.Fatalf("unable to complete pre-test tasks: %v", err)
	}

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

	if err := runTest(tests, router, t); err != nil {
		t.Fatalf("error running tests: %v", err)
	}
}

// test the route directly since it is a thin wrapper around serveReadOne
func TestReadOne(t *testing.T) {
	router, v, err := newTestInfra()
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

	if err := runTest(tests, router, t); err != nil {
		t.Fatalf("error running tests: %v", err)
	}
}

func TestReadAll(t *testing.T) {
	fakeMongo := newFakeMongoCollection()
	gin.SetMode(gin.TestMode)
	router, _ := newRouter("password", fakeMongo)

	for i := 0; i < 3; i++ {
		_, _ = fakeMongo.Create("testing", 1.0, "0")
	}

	resp := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/tickets", nil)
	router.ServeHTTP(resp, req)

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
	router, v, err := newTestInfra()
	if err != nil {
		t.Fatalf("unable to complete pre-test tasks: %v", err)
	}

	testUserJWT, _ := middleware.NewUserClaims("foo@bar.com", "1").Tokenize(v)
	badUserJWT, _ := middleware.NewUserClaims("bar@foo.com", "2").Tokenize(v)

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
			http.MethodPut,
			"/api/tickets/0",
			TicketReq{"test update", 2.0},
			map[string]string{"auth-jwt": badUserJWT},
			http.StatusUnauthorized,
			nil,
			&ErrorResp{[]string{"Unauthorized"}},
		},
		{
			"malformed update",
			http.MethodPut,
			"/api/tickets/0",
			TicketReq{"", -1.0},
			map[string]string{"auth-jwt": testUserJWT},
			http.StatusBadRequest,
			nil,
			&ErrorResp{[]string{"please specify a title", "price cannot be less than 0"}},
		},
		{
			"successful update",
			http.MethodPut,
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

	if err := runTest(tests, router, t); err != nil {
		t.Fatalf("error running tests: %v", err)
	}
}
