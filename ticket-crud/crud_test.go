package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/basilnsage/mwn-ticketapp/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// directly from https://pkg.go.dev/github.com/google/go-cmp@v0.5.4/cmp/cmpopts
var cmpStringSplitter = cmpopts.AcyclicTransformer("StringSplitter", func(s string) []string {
	return strings.Split(s, "\n")
})

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

func (f *fakeMongoCollection) ReadAll() ([]*Ticket, error) {
	return nil, nil
}

func (f *fakeMongoCollection) Update(id interface{}, title string, price float64) (interface{}, error) {
	return nil, nil
}

func TestCreate(t *testing.T) {
	fakeMongo := newFakeMongoCollection()
	v, _ := middleware.NewJWTValidator([]byte("password"), "HS256")
	authHeader, _ := middleware.NewUserClaims("foo@bar.com", "0").Tokenize(v)

	gin.SetMode(gin.TestMode)
	resp := httptest.NewRecorder()
	_, r := gin.CreateTestContext(resp)
	r.POST("/test", func(c *gin.Context) {
		serveCreate(c, fakeMongo, v)
	})

	tik := TicketReq{"foo", 0.0}
	reqBody, _ := json.Marshal(tik)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(reqBody))
	req.Header.Add("auth-jwt", authHeader)
	r.ServeHTTP(resp, req)
	if got, want := resp.Code, http.StatusCreated; got != want {
		t.Errorf("incorrect status code, got: %v want: %v", got, want)
	}

	respBody, _ := ioutil.ReadAll(resp.Body)
	var respTik TicketResp
	_ = json.Unmarshal(respBody, &respTik)
	if diff := cmp.Diff(respTik, TicketResp{"foo", 0.0, "0", "0"}); diff != "" {
		t.Errorf("ticket response incorrect: %v", diff)
	}

	tik = TicketReq{"", -1.0}
	resp = httptest.NewRecorder()
	_, r = gin.CreateTestContext(resp)
	r.POST("/test", func(c *gin.Context) {
		serveCreate(c, fakeMongo, v)
	})
	reqBody, _ = json.Marshal(tik)
	req = httptest.NewRequest("POST", "/test", bytes.NewReader(reqBody))
	req.Header.Add("auth-jwt", authHeader)
	r.ServeHTTP(resp, req)
	respBody, _ = ioutil.ReadAll(resp.Body)

	if got, want := resp.Code, http.StatusBadRequest; got != want {
		t.Errorf("incorrect status code, got: %v want: %v", got, want)
	}
	var respStatus struct {
		Errors []string
	}
	_ = json.Unmarshal(respBody, &respStatus)
	if diff := cmp.Diff(respStatus.Errors, []string{
		"please specify a title",
		"price cannot be less than 0",
	}, cmpStringSplitter); diff != "" {
		t.Errorf("incorrect errors reported, diff: %v", diff)
	}
}

// test the route directly since it is a thin wrapper around serveReadOne
func TestReadOne(t *testing.T) {
	fakeMongo := newFakeMongoCollection()
	router, _ := newRouter("password", fakeMongo)
	gin.SetMode(gin.TestMode)

	id, _ := fakeMongo.Create("testing", 1.0, "0")
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/tickets/%v", id), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if got, want := resp.Code, http.StatusOK; got != want {
		t.Errorf("unexpected response code: %v, want %v", got, want)
	}

	var tik TicketResp
	respBody, _ := ioutil.ReadAll(resp.Body)
	_ = json.Unmarshal(respBody, &tik)
	if diff := cmp.Diff(tik, TicketResp{"testing", 1.0, "0", "0"}); diff != "" {
		t.Errorf("incorrect ticket returned: %v", diff)
	}

	req = httptest.NewRequest("GET", "/api/tickets/2", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if got, want := resp.Code, http.StatusNotFound; got != want {
		t.Errorf("request for non-existent ticket resp code: %v, want %v", got, want)
	}
}

// to test:
// 401 from req without a JWT header
// 401 from req with an invalid JWT header
// 201 from req with a valid JWT header
func TestIntegration(t *testing.T) {
	fakeMongo := newFakeMongoCollection()
	router, _ := newRouter("password", fakeMongo)
	gin.SetMode(gin.TestMode)

	// reuse the same JSON body for all requests
	tik := TicketReq{"testing ticket", 1.0}
	tikJson, _ := json.Marshal(&tik)
	type ErrResp struct {
		Errors []string
	}

	resp := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/tickets/create", bytes.NewBuffer(tikJson))

	// no auth-jwt header should result in a 401
	router.ServeHTTP(resp, req)
	if got, want := resp.Code, http.StatusUnauthorized; got != want {
		t.Errorf("no jwt header should be unauthorized, resp code: %v, want %v", got, want)
	}
	respBody, _ := ioutil.ReadAll(resp.Body)
	respJson := &ErrResp{}
	_ = json.Unmarshal(respBody, respJson)
	got, want := []string{"User is not signed in"}, respJson.Errors
	if diff := cmp.Diff(got, want, cmpStringSplitter); diff != "" {
		t.Errorf("no jwt header led to unexpected error status, diff: %v", diff)
	}

	// bad auth-jwt header should result in a 401
	resp = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/tickets/create", bytes.NewBuffer(tikJson))
	// user: foo@bar.com, id: 1
	req.Header.Add("auth-jwt", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImZvb0BiYXIuY29tIiwiaWQiOjF9.f9FeG_FD2vOW6sGQwGxCoGYNIZv1P_Sb7WBgjq99KOs")
	router.ServeHTTP(resp, req)
	if got, want := resp.Code, http.StatusUnauthorized; got != want {
		t.Errorf("no jwt header should be unauthorized, resp code: %v, want %v", got, want)
	}
	respBody, _ = ioutil.ReadAll(resp.Body)
	respJson = &ErrResp{}
	_ = json.Unmarshal(respBody, respJson)
	got, want = []string{"Unauthorized"}, respJson.Errors
	if diff := cmp.Diff(got, want, cmpStringSplitter); diff != "" {
		t.Errorf("bad jwt header led to unexpected error status, diff: %v", diff)
	}

	// good auth-jwt header should result in a 201
	v, _ := middleware.NewJWTValidator([]byte("password"), "HS256")
	authHeader, _ := middleware.NewUserClaims("foo@bar.com", "0").Tokenize(v)

	resp = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/tickets/create", bytes.NewBuffer(tikJson))
	req.Header.Add("auth-jwt", authHeader)
	router.ServeHTTP(resp, req)
	if got, want := resp.Code, http.StatusCreated; got != want {
		t.Errorf("good jwt header should be authorized, resp code: %v, want %v", got, want)
	}
	respBody, _ = ioutil.ReadAll(resp.Body)
	var goodResp TicketResp
	_ = json.Unmarshal(respBody, &goodResp)
	if diff := cmp.Diff(goodResp, TicketResp{"testing ticket", 1.0, "0", "0"}); diff != "" {
		t.Errorf("valid req did not return expected response: %v", diff)
	}
}
