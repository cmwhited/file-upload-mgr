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

	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
	"github.com/graphql-go/graphql"
	LOGGER "github.com/sirupsen/logrus"
)

const dataKey = "data"

type config interface {
	initAwsConfig() error
	dynamoImpl() dynamodbiface.DynamoDBAPI
	s3Impl() s3iface.S3API
	initLoggerConfig()
	loggerImpl() *LOGGER.Logger
	initSchema() error
	schemaImpl() *graphql.Schema
	init() (config, error)
}

type conf struct {
	dynamo dynamodbiface.DynamoDBAPI
	s3     s3iface.S3API
	log    *LOGGER.Logger
	schema *graphql.Schema
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
	cfg.Region = endpoints.UsEast1RegionID
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
	log.SetLevel(LOGGER.InfoLevel)
	c.log = log
}

func (c *conf) loggerImpl() *LOGGER.Logger {
	return c.log
}

// schemaImpl() - init a graphql schema instance with the given:
//	* queries
//	* mutations
func (c *conf) initSchema() error {
	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name:        "RootQuery",
		Description: "Testing GraphQL impl",
		Fields: graphql.Fields{
			"hello": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
					return "World", nil
				},
			},
		},
	})
	schemaConfig := graphql.SchemaConfig{
		Query: rootQuery,
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

// init() - initialize all configurations
func (c *conf) init() (config, error) {
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
