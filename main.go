package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/graphql-go/graphql"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	LOGGER "github.com/sirupsen/logrus"
)

type key string

const (
	authHeaderKey          key = "Authorization"
	authorizationHeaderKey     = "Authorization"
)

type params struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

// Handler - AWS Lambda Execution invocation function point
//	- initialize the required dependencies for the handler
//	- get the request body and marshal into a params instance
//	- attempt to run the graphql query
//	- return the response of the query
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// add the Authorization header to the context which is passed to the query
	appCtx := context.WithValue(ctx, authHeaderKey, request.Headers[authorizationHeaderKey])
	if len(request.Body) == 0 {
		resp := new(apiResponse).
			WithReceivedAt(time.Now()).
			WithErrors("Request body is null").
			WithMessage("Handler() - The Request body is null. Cannot Process").
			ToJSON()
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       resp,
		}, nil
	}
	// initialize file upload manager config
	mgr, err := new(conf).init()
	if err != nil {
		resp := new(apiResponse).
			WithReceivedAt(time.Now()).
			WithErrors(err.Error()).
			WithMessage("Handler() - error occurred trying to initialize").
			ToJSON()
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       resp,
		}, nil
	}
	// log event
	mgr.loggerImpl().WithFields(LOGGER.Fields{
		"request_body":    request.Body,
		"request_method":  request.HTTPMethod,
		"request_headers": request.Headers,
	}).Info("Handler() - File Upload Request Received")
	// deserialize request body into params
	var reqParams = new(params)
	if err := json.Unmarshal([]byte(request.Body), &reqParams); err != nil {
		mgr.loggerImpl().WithFields(LOGGER.Fields{
			"deserialization_error": err.Error(),
		}).Error("Handler() - An error occurred while trying to deserialize the request body into the params")
		resp := new(apiResponse).
			WithReceivedAt(time.Now()).
			WithErrors(err.Error()).
			WithMessage("Handler() - An error occurred while trying to deserialize the request body into the params").
			ToJSON()
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       resp,
		}, nil
	}
	// run query against graphql instance to get result
	schema := mgr.schemaImpl()
	response := graphql.Do(graphql.Params{
		Schema:         *schema,
		RequestString:  reqParams.Query,
		VariableValues: reqParams.Variables,
		OperationName:  reqParams.OperationName,
		Context:        appCtx,
	})
	// check for errors
	if response.HasErrors() {
		mgr.loggerImpl().WithFields(LOGGER.Fields{
			"request_query":          reqParams.Query,
			"request_operation_name": reqParams.OperationName,
			"request_variables":      reqParams.Variables,
			"request_errors":         response.Errors,
		}).Error("Handler() - an error occurred trying to perform the graphql query operation")
		resp := new(apiResponse).
			WithReceivedAt(time.Now()).
			WithErrors(response.Errors).
			WithMessage("Handler() - an error occurred trying to perform the graphql query operation").
			ToJSON()
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       resp,
		}, nil
	}
	// parse response; serialize into JSON
	r, err := json.Marshal(response.Data)
	if err != nil {
		mgr.loggerImpl().WithFields(LOGGER.Fields{
			"request_query":          reqParams.Query,
			"request_operation_name": reqParams.OperationName,
			"request_variables":      reqParams.Variables,
			"request_errors":         err.Error(),
		}).Error("Handler() - an error occurred trying to marshal the graphql query response into json")
		resp := new(apiResponse).
			WithReceivedAt(time.Now()).
			WithErrors(err.Error()).
			WithMessage("Handler() - an error occurred trying to marshal the graphql query response into json").
			ToJSON()
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       resp,
		}, nil
	}
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(r),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}

func main() {
	lambda.Start(Handler)
}
