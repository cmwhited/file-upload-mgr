package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Handler is executed by AWS Lambda in the main function. Once the request
// is processed, it returns an Amazon API Gateway response object to AWS Lambda
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// initialize file upload manager config
	if mgr, err := new(conf).init(); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       err.Error(),
		}, nil
	} else {
		mgr.loggerImpl().Info("Handler() - successfully initialized")
	}
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string("TESTING"),
	}, nil
}

func main() {
	lambda.Start(Handler)
}
