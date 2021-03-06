/**
config - provides interface implementations to initiate and expose configuration resources required by the application.

	- AWS Configurations:
		- dynamodb
		- s3

	- Logging Framework: initialize and configure a logging framework that the handler will use

	- GraphQL Schema Instance: a built graphql schema with
		- queries:
			- get a user
			- get a list of sessions
			- get a session by id
		- mutations
			- register a new user
			- authenticate a user
			- init a new session
			- upload file(s) to the session
			- remove files from the session
*/
package main

import (
	"errors"
	"os"
	"strconv"

	"github.com/mitchellh/mapstructure"

	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
	"github.com/graphql-go/graphql"
	LOGGER "github.com/sirupsen/logrus"
)

const (
	dataKey              = "data"
	jwtSecretKey         = "JWT_SECRET"
	tokenExpiryMinKey    = "TOKEN_EXPIRY_MIN"
	tablesMapUserKey     = "USERS"
	usersTableNameKey    = "USERS_TABLE_NAME"
	tablesMapSessionKey  = "SESSIONS"
	sessionsTableNameKey = "SESSIONS_TABLE_NAME"
)

type config interface {
	initAwsConfig() error
	dynamoImpl() dynamodbiface.DynamoDBAPI
	s3Impl() s3iface.S3API
	initLoggerConfig()
	loggerImpl() *LOGGER.Logger
	initSchema() error
	schemaImpl() *graphql.Schema
	init() (config, error)
	tableNames() map[string]string
}

type conf struct {
	dynamo         dynamodbiface.DynamoDBAPI
	s3             s3iface.S3API
	log            *LOGGER.Logger
	schema         *graphql.Schema
	tableName      map[string]string
	jwtSecret      []byte
	tokenExpiryMin int
}

// initAwsConfig() - initialize the required AWS services
//	* load the configuration by using the user associated to this lambda
//	* use the configuration to instantiate a new dynamo service impl
//	* use the configuration to instantiate a new s3 service impl
func (c *conf) initAwsConfig() error {
	// establish the aws awsConfig with the env access key and secret
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
	}
	cfg.Region = endpoints.UsWest2RegionID
	// instantiate service impl
	c.dynamo = dynamodb.New(cfg)
	c.s3 = s3.New(cfg)
	return nil
}

func (c *conf) dynamoImpl() dynamodbiface.DynamoDBAPI {
	return c.dynamo
}

func (c *conf) s3Impl() s3iface.S3API {
	return c.s3
}

// initLoggerConfig() - instantiate a logger instance with given configurations
func (c *conf) initLoggerConfig() {
	log := LOGGER.New()
	log.SetFormatter(&LOGGER.JSONFormatter{
		PrettyPrint: true,
		DataKey:     dataKey,
	})
	log.SetOutput(os.Stdout)
	log.SetReportCaller(true)
	log.SetLevel(LOGGER.DebugLevel)
	c.log = log
}

func (c *conf) loggerImpl() *LOGGER.Logger {
	return c.log
}

func (c *conf) buildRootQuery() *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name:        "RootQuery",
		Description: "Hello World impl for testing purposes",
		Fields: graphql.Fields{
			"hello": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
					return "World", nil
				},
			},
			"getUserByEmail": &graphql.Field{
				Type:        userType,
				Description: "find a user record by its email",
				Args: graphql.FieldConfigArgument{
					"email": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
					email := p.Args["email"].(string)
					return findUserByEmail(email, c.tableNames()[tablesMapUserKey], c.dynamoImpl(), c.loggerImpl())
				},
			},
			"getAuthUser": &graphql.Field{
				Type:        userType,
				Description: "Get the currently authenticated user by getting their info from the Auth header in the request",
				Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
					// attempt to validate token
					email, err := validateToken(p.Context.Value(authHeaderKey), c.jwtSecret, c.loggerImpl())
					if err != nil {
						return nil, err
					}
					return findUserByEmail(*email, c.tableNames()[tablesMapUserKey], c.dynamoImpl(), c.loggerImpl())
				},
			},
			"getSession": &graphql.Field{
				Type:        sessionType,
				Description: "Get the session by the id and email keys",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
					id := p.Args["id"].(string)
					email, err := validateToken(p.Context.Value(authHeaderKey), c.jwtSecret, c.loggerImpl())
					if err != nil {
						return nil, err
					}
					return findSessionByID(id, *email, c.tableNames()[tablesMapSessionKey], c.dynamoImpl(), c.loggerImpl())
				},
			},
			"getSessions": &graphql.Field{
				Type:        graphql.NewList(sessionType),
				Description: "Get all sessions associated with the given email",
				Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
					email, err := validateToken(p.Context.Value(authHeaderKey), c.jwtSecret, c.loggerImpl())
					if err != nil {
						return nil, err
					}
					return findSessions(*email, c.tableNames()[tablesMapSessionKey], c.dynamoImpl(), c.loggerImpl())
				},
			},
		},
	})
}

func (c *conf) buildRootMutation() *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "RootMutation",
		Fields: graphql.Fields{
			"register": &graphql.Field{
				Type:        graphql.NewNonNull(userType),
				Description: "Register a new user instance",
				Args: graphql.FieldConfigArgument{
					"email": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"pwd":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"name":  &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"role":  &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
					// get input args
					email := p.Args["email"].(string)
					pwd := p.Args["pwd"].(string)
					name := p.Args["name"].(string)
					role := p.Args["role"].(string)
					// attempt to register user
					return registerUser(email, pwd, name, role, c.tableNames()[tablesMapUserKey], c.dynamoImpl(), c.loggerImpl())
				},
			},
			"authenticate": &graphql.Field{
				Type:        graphql.NewNonNull(authType),
				Description: "Attempt to authenticate a user with their email and password",
				Args: graphql.FieldConfigArgument{
					"email": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"pwd":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
					// get input args
					email := p.Args["email"].(string)
					pwd := p.Args["pwd"].(string)
					// attempt to authenticate user
					return authenticate(email, pwd, c.tableNames()[tablesMapUserKey], c.jwtSecret, c.tokenExpiryMin, c.dynamoImpl(), c.loggerImpl()), nil
				},
			},
			"saveSession": &graphql.Field{
				Type:        sessionType,
				Description: "Save a session instance",
				Args: graphql.FieldConfigArgument{
					"sess": &graphql.ArgumentConfig{Type: graphql.NewNonNull(sessionInputType)},
				},
				Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
					sess := p.Args["sess"]
					sessMap, ok := sess.(map[string]interface{}) // convert the input type to a User
					if !ok {
						e = errors.New("unable to convert input object to session record")
						return nil, e
					}
					var s = new(session)
					e = mapstructure.Decode(sessMap, &s) // decode map into session instance
					if e != nil {
						return nil, e
					}
					return saveSession(*s, c.tableNames()[tablesMapSessionKey], c.dynamoImpl(), c.loggerImpl())
				},
			},
		},
	})
}

// schemaImpl() - init a graphql schema instance with the given:
//	* queries
//	* mutations
func (c *conf) initSchema() error {
	schemaConfig := graphql.SchemaConfig{
		Query:    c.buildRootQuery(),
		Mutation: c.buildRootMutation(),
	}
	schema, err := graphql.NewSchema(schemaConfig)
	if err != nil {
		return err
	}
	c.schema = &schema
	return nil
}

func (c *conf) schemaImpl() *graphql.Schema {
	return c.schema
}

func (c *conf) tableNames() map[string]string {
	return c.tableName
}

// init() - initialize all configurations
func (c *conf) init() (config, error) {
	// load table names from env variables
	usersTableName := os.Getenv(usersTableNameKey)
	sessionsTableName := os.Getenv(sessionsTableNameKey)
	c.tableName = map[string]string{
		tablesMapUserKey:    usersTableName,
		tablesMapSessionKey: sessionsTableName,
	}
	jwtSecret := os.Getenv(jwtSecretKey)           // get the jwt secret key from the env
	c.jwtSecret = []byte(jwtSecret)                // set as byte array; required by signer
	tokenExpiryVal := os.Getenv(tokenExpiryMinKey) // get the jwt expiry value from the env
	tokenExpiry, _ := strconv.Atoi(tokenExpiryVal) // convert to int
	c.tokenExpiryMin = tokenExpiry
	c.initLoggerConfig() // initialize logger instance
	// initialize aws config
	if err := c.initAwsConfig(); err != nil {
		return c, err
	}
	// initialize graphql schema config
	if err := c.initSchema(); err != nil {
		return c, err
	}
	return c, nil
}
