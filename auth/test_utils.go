package main

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/mock"
	"net/http"

	"github.com/basilnsage/mwn-ticketapp/auth/users"
)

var (
	key = []byte("password")
	// { alg: HS256, typ: JWT }
	header = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	// { email: foo@example.com, id: 5f47ec2c86ed3ef991cdfd94 }
	payload = "eyJlbWFpbCI6ImZvb0BleGFtcGxlLmNvbSIsImlkIjoiNWY0N2VjMmM4NmVkM2VmOTkxY2RmZDk0In0"
	// HS256 signature with key "password"
	sig       = "jQ4rq4Ih6SL2J6jx7yhQDj54PptcwBv1w38zvsbIygs"
	email     = "foo@example.com"
	pass      = "password"
	passHash  = []byte("$2a$10$XoknOf54X3cPka131Ga55.oN/YcnuJ8CfOWof5ZCTKIhmz7.95xn2")
	uid       = "5f47ec2c86ed3ef991cdfd94"
	claimsMap = map[string]interface{}{
		"email": email,
		"id":    uid,
	}
	sampleClaims = users.Claims{
		Email: email,
		ID:    uid,
	}
	jwtString = fmt.Sprintf("%s.%s.%s", header, payload, sig)
	cookie    = http.Cookie{
		Name:  "auth-jwt",
		Value: jwtString,
	}
)

type mockSigner struct {
	mock.Mock
}

type mockCRUD struct {
	mock.Mock
}

func (m *mockSigner) Sign(claims map[string]interface{}) (string, error) {
	args := m.Called(claims)
	return args.String(0), args.Error(1)
}

func (m *mockCRUD) Read(ctx context.Context, user users.User) ([]users.User, error) {
	args := m.Called(ctx, user)
	return args.Get(0).([]users.User), args.Error(1)
}

func (m *mockCRUD) Write(ctx context.Context, user users.User) (interface{}, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(interface{}), args.Error(1)
}

func checkTestUser(u users.User) bool {
	return u.Email == email
}
