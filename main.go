package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/graphql-go/graphql"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	LOGGER "github.com/sirupsen/logrus"
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
	if len(request.Body) == 0 {
		fmt.Println("Handler() - The Request body is null. Cannot Process")
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Handler() - The Request body is null. Cannot Process",
		}, nil
	}
	// initialize file upload manager config
	mgr, err := new(conf).init()
	if err != nil {
		fmt.Printf("Handler() - error occurred trying to initialize :: %s\r\n", err.Error())
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       err.Error(),
		}, nil
	}
	// log event
	mgr.loggerImpl().WithFields(LOGGER.Fields{
		"request_body":   request.Body,
		"request_method": request.HTTPMethod,
	}).Info("Handler() - File Upload Request Received")
	// deserialize request body into params
	var reqParams = new(params)
	if err := json.Unmarshal([]byte(request.Body), &reqParams); err != nil {
		mgr.loggerImpl().WithFields(LOGGER.Fields{
			"deserialization_error": err.Error(),
		}).Info("Handler() - An error occurred while trying to deserialize the request body into the params")
	}
	// run query against graphql instance to get result
	schema := mgr.schemaImpl()
	response := graphql.Do(graphql.Params{
		Schema:         *schema,
		RequestString:  reqParams.Query,
		VariableValues: reqParams.Variables,
		OperationName:  reqParams.OperationName,
		Context:        ctx,
	})
	// check for errors
	if response.HasErrors() {
		mgr.loggerImpl().WithFields(LOGGER.Fields{
			"request_query":          reqParams.Query,
			"request_operation_name": reqParams.OperationName,
			"request_variables":      reqParams.Variables,
			"request_errors":         response.Errors,
		}).Error("Handler() - an error occurred trying to perform the graphql query operation")
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Handler() - An error occurred while trying to run the query. Please try again",
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
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Handler() - An error occurred while trying to serialize the query response",
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
