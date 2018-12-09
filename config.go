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
	"os"
	"strconv"

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
	dataKey           = "data"
	jwtSecretKey      = "JWT_SECRET"
	tokenExpiryMinKey = "TOKEN_EXPIRY_MIN"
	tablesMapUserKey  = "USERS"
	usersTableNameKey = "USERS_TABLE_NAME"
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
	jwtSecret      string
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
			"getUserById": &graphql.Field{
				Type:        userType,
				Description: "find a user record by its id",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
					id := p.Args["id"].(string)
					return findUserBydId(id, c.tableNames()[tablesMapUserKey], c.dynamoImpl(), c.loggerImpl())
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
					return findUserBydEmail(email, c.tableNames()[tablesMapUserKey], c.dynamoImpl(), c.loggerImpl())
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
					// log request
					c.loggerImpl().WithFields(LOGGER.Fields{
						"email": email,
					}).Info("mutation.authenticate() - attempting to authenticate user")
					// attempt to authenticate user
					return authenticate(email, pwd, c.tableNames()[tablesMapUserKey], c.jwtSecret, c.tokenExpiryMin, c.dynamoImpl(), c.loggerImpl()), nil
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
	c.tableName = map[string]string{
		tablesMapUserKey: usersTableName,
	}
	c.jwtSecret = os.Getenv(jwtSecretKey)          // get the jwt secret key from the env
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
