package routes

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/basilnsage/mwn-ticketapp/auth/jwt"
	"github.com/basilnsage/mwn-ticketapp/auth/users"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
)

var (
	sampleHeader = "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9"
	samplePayload = "eyJlbWFpbCI6ImZvb0BleGFtcGxlLmNvbSIsInVpZCI6IjVmNDdlYzJjODZlZDNlZjk5MWNkZmQ5NCJ9"
	sampleSig = "aF4HEksOE3zmyQiGNRS-yGV79oZEin-ESHsQx_WGcJNnuGKEXWyPTXUBYyL-wg7UEjrWp1MbMrMvPt4Yvw-mtg"
	sampleClaims = users.PrivClaims{
		Email: "foo@example.com",
		UID: "5f47ec2c86ed3ef991cdfd94",
	}
	sampleJWT = fmt.Sprintf("%s.%s.%s", sampleHeader, samplePayload, sampleSig)
	cookie = http.Cookie{
		Name: "auth-jwt",
		Value: sampleJWT,
	}
)

func setup() error {
	err := os.Setenv("JWT_SIGN_KEY", "password")
	if err != nil {
		return err
	}
	err = jwt.InitSigner()
	if err != nil {
		return err
	}
	return nil
}

func parseBody(receiver *users.PrivClaims, resp *http.Response) error {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ioutil.ReadAll: %v", err)
	}
	if err = json.Unmarshal(body, receiver); err != nil {
		return fmt.Errorf("json.Unmarshal: %v", err)
	}
	return nil
}

func TestWhoami(t *testing.T) {
	// initial setup to get JWT verification to work
	if err := setup(); err != nil {
		t.Fatalf("whoami_test.setup: %v", err)
	}

	// stand up a test router to actually invoke the Whoami route
	w := httptest.NewRecorder()
	eng := gin.Default()
	eng.GET("/test", Whoami)
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&cookie)
	eng.ServeHTTP(w, req)

	// check response
	resp := w.Result()
	if w.Code != 200 {
		t.Errorf("TestWhoami: got %v, want %v", w.Code, 200)
	}

	bodyJson := users.PrivClaims{}
	if err := parseBody(&bodyJson, resp); err != nil {
		t.Errorf("whoami_test.parseBody: %v", err)
	}
	if !cmp.Equal(bodyJson, sampleClaims) {
		t.Errorf("TestWhoami: got %v, want %v", bodyJson, sampleClaims)
	}
}
