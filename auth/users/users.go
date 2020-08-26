package users

type PrivClaims struct {
	Email string      `json:"email"`
	UID   interface{} `json:"uid"`
}
