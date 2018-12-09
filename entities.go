package main

import (
	"time"

	"github.com/graphql-go/graphql"
)

type baseMeta struct {
	MetaCreatedAt time.Time `json:"meta__created_at"`
	MetaUpdatedAt time.Time `json:"meta__updated_at"`
	MetaIsActive  bool      `json:"meta__is_active"`
}

type user struct {
	ID    string   `json:"id"`
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
			"id":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
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
		},
	})
)
