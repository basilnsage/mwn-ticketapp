package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// directly from https://pkg.go.dev/github.com/google/go-cmp@v0.5.4/cmp/cmpopts
var cmpStringSplitter cmp.Option = cmpopts.AcyclicTransformer("StringSplitter", func(s string) []string {
	return strings.Split(s, "\n")
})

type fakeMongoCollection struct {
	tickets map[string]*Ticket
	id int
}

func newFakeMongoCollection() *fakeMongoCollection {
	return &fakeMongoCollection{
		make(map[string]*Ticket),
		0,
	}
}

func (f *fakeMongoCollection) Create(title string, price float64) (interface{}, error) {
	if title == "should error" {
		return nil, errors.New("unable to create ticket")
	}
	currId := strconv.Itoa(f.id)
	f.id++
	f.tickets[currId] = &Ticket{title, price}
	return currId, nil
}

func (f *fakeMongoCollection) ReadOne(id interface{}) (*Ticket, error) {
	return nil, nil
}

func (f *fakeMongoCollection) ReadAll() ([]*Ticket, error) {
	return nil, nil
}

func (f *fakeMongoCollection) Update(id interface{}, title string, price float64) (interface{}, error) {
	return nil, nil
}

func TestCreate(t *testing.T) {
	fakeMongo := newFakeMongoCollection()
	resp := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	c, r := gin.CreateTestContext(resp)
	r.POST("/test", func(c *gin.Context) {
		serveCreate(c, fakeMongo)
	})

	tik := Ticket{"foo", 0.0}
	reqBody, _ := json.Marshal(tik)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(reqBody))
	r.ServeHTTP(resp, c.Request)
	if got, want := resp.Code, http.StatusCreated; got != want {
		t.Errorf("incorrect status code, got: %v want: %v", got, want)
	}

	respBody,_ := ioutil.ReadAll(resp.Body)
	respTik := &struct{
		Id string
		Title string
		Price float64
	}{}
	_ = json.Unmarshal(respBody, respTik)
	if got, want := respTik.Title, "foo"; got != want {
		t.Errorf("incorrect title, got: %v, want: %v", got, want)
	}
	if got, want := respTik.Price, 0.0; got != want {
		t.Errorf("incorrect price, got: %v, want: %v", got, want)
	}
	if got, want := respTik.Id, "0"; got != want {
		t.Errorf("incorrect ID, got: %v, want: %v", got, want)
	}

	tik = Ticket{"", -1.0}
	resp = httptest.NewRecorder()
	c, r = gin.CreateTestContext(resp)
	r.POST("/test", func(c *gin.Context) {
		serveCreate(c, fakeMongo)
	})
	reqBody, _ = json.Marshal(tik)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(reqBody))
	r.ServeHTTP(resp, c.Request)
	respBody, _ = ioutil.ReadAll(resp.Body)

	if got, want := resp.Code, http.StatusBadRequest; got != want {
		t.Errorf("incorrect status code, got: %v want: %v", got, want)
	}
	respStatus := &struct {
		Errors []string
	}{}
	_ = json.Unmarshal(respBody, respStatus)
	got := respStatus.Errors
	want := []string{
		"please specify a title",
		"price cannot be less than 0",
	}
	if diff := cmp.Diff(got, want, cmpStringSplitter); diff != "" {
		t.Errorf("incorrect errors reported, diff: %v", diff)
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
	tik := Ticket{"testing ticket", 1.0}
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
	resp = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/tickets/create", bytes.NewBuffer(tikJson))
	// user: foo@bar.com, id: 1
	req.Header.Add("auth-jwt", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImZvb0BiYXIuY29tIiwiaWQiOjF9.9DzMmA93ZeuJP1_tBm9yaznYUbtBwirW9YDt8KNDYBk")
	router.ServeHTTP(resp, req)
	if got, want := resp.Code, http.StatusCreated; got != want {
		t.Errorf("good jwt header should be authorized, resp code: %v, want %v", got, want)
	}
	respBody, _ = ioutil.ReadAll(resp.Body)
	type GoodResp struct {
		Title string
		Price float64
		Id string
	}
	goodRespJson := &GoodResp{}
	_ = json.Unmarshal(respBody, goodRespJson)
	goodGot, goodWant := goodRespJson, &GoodResp{"testing ticket", 1.0, "0"}
	if diff := cmp.Diff(goodGot, goodWant); diff != "" {
		t.Errorf("valid req did not return expected response: %v", diff)
	}
}