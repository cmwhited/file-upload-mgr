package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHandler(t *testing.T) {

	request := events.APIGatewayProxyRequest{}
	expectedResponse := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body: "TESTING",
	}

	response, err := Handler(request)

	assert.Contains(t, response.Body, expectedResponse.Body)
	assert.Equal(t, err, nil)

}
