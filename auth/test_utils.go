package main

import (
	"fmt"
	"net/http"

	"github.com/basilnsage/mwn-ticketapp/auth/users"
)

var (
	key = []byte("password")
	// { alg: HS256, typ: JWT }
	header = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	// { email: foo@example.com, uid: 5f47ec2c86ed3ef991cdfd94 }
	payload = "eyJlbWFpbCI6ImZvb0BleGFtcGxlLmNvbSIsInVpZCI6IjVmNDdlYzJjODZlZDNlZjk5MWNkZmQ5NCJ9"
	// HS256 signature with key "password"
	sig = "jrgWQhw5YFXm01UVbZ-ZWEpJgmM_iNXwwgPG4pJ6bcQ"
	email = "foo@example.com"
	uid = "5f47ec2c86ed3ef991cdfd94"
	claimsMap = map[string]interface{}{
		"email": email,
		"id": uid,
	}
	sampleClaims = users.Claims{
		Email: email,
		ID: uid,
	}
	jwtString = fmt.Sprintf("%s.%s.%s", header, payload, sig)
	cookie = http.Cookie{
		Name:  "auth-jwt",
		Value: jwtString,
	}
)
