package main

import (
	"encoding/json"
	"time"

	"github.com/graphql-go/graphql"
)

type apiResponseBuilder interface {
	WithReceivedAt(receivedAt time.Time) apiResponseBuilder
	WithErrors(errors interface{}) apiResponseBuilder
	WithMessage(msg string) apiResponseBuilder
	ToJSON() string
}

type apiResponse struct {
	ReceivedAt time.Time   `json:"received_at"`
	Errors     interface{} `json:"errors"`
	Message    string      `json:"message"`
}

func (api *apiResponse) WithReceivedAt(receivedAt time.Time) apiResponseBuilder {
	api.ReceivedAt = receivedAt
	return api
}

func (api *apiResponse) WithErrors(errors interface{}) apiResponseBuilder {
	api.Errors = errors
	return api
}

func (api *apiResponse) WithMessage(msg string) apiResponseBuilder {
	api.Message = msg
	return api
}

func (api *apiResponse) ToJSON() string {
	r, _ := json.Marshal(api)
	return string(r)
}

type baseMeta struct {
	MetaCreatedAt time.Time `json:"meta__created_at"`
	MetaUpdatedAt time.Time `json:"meta__updated_at"`
	MetaIsActive  bool      `json:"meta__is_active"`
}

type user struct {
	Email string   `json:"email"`
	Pwd   string   `json:"pwd"`
	Name  string   `json:"name"`
	Role  string   `json:"role"`
	Meta  baseMeta `json:"meta"`
}

type auth struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Token     string `json:"token,omitempty"`
	ExpiresAt int64  `json:"expiresAt,omitempty"`
	User      *user  `json:"user,omitempty"`
}

var (
	baseMetaType = graphql.NewObject(graphql.ObjectConfig{
		Name: "Meta",
		Fields: graphql.Fields{
			"meta__created_at": &graphql.Field{Type: graphql.NewNonNull(graphql.DateTime)},
			"meta__updated_at": &graphql.Field{Type: graphql.NewNonNull(graphql.DateTime)},
		},
	})
	userType = graphql.NewObject(graphql.ObjectConfig{
		Name:        "User",
		Description: "Describes fields for a User record",
		Fields: graphql.Fields{
			"email": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"name":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"role":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"meta":  &graphql.Field{Type: baseMetaType},
		},
	})
	authType = graphql.NewObject(graphql.ObjectConfig{
		Name:        "Auth",
		Description: "The return of an authentication request",
		Fields: graphql.Fields{
			"success":   &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"message":   &graphql.Field{Type: graphql.String},
			"token":     &graphql.Field{Type: graphql.String},
			"expiresAt": &graphql.Field{Type: graphql.Float},
			"user":      &graphql.Field{Type: userType},
		},
	})
)
