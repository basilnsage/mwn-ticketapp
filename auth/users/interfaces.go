package users

import "context"

type CRUD interface {
	Read(context.Context, User) ([]User, error)
	Write(context.Context, User) (interface{}, error)
}

type Signer interface {
	Sign(map[string]interface{}) (string, error)
}